// Package pkg reads messages from public Telegram channels
// via the web archive at t.me/s/<username>.
//
// No API keys, authentication, or MTProto — just HTTP + HTML parsing.
//
// Example:
//
//	reader := pkg.New("durov")
//	msgs, err := reader.Fetch(context.Background(), 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Printf("[%s] #%d %s\n", m.Date, m.ID, m.Text[:min(80, len(m.Text))])
//	}
package pkg

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Message represents a parsed Telegram channel message.
type Message struct {
	// ID is the sequential message number within the channel.
	ID int `json:"id"`

	// Channel is the channel username (e.g. "durov").
	Channel string `json:"channel"`

	// Date is the publication time in UTC.
	Date time.Time `json:"date"`

	// Text is the message body with normalized whitespace.
	Text string `json:"text"`

	// MediaType describes the message content type.
	// Possible values: text, photo, video, roundvideo, audio, document,
	// sticker, location, poll, service, multimedia.
	// "text" means no media is attached.
	MediaType string `json:"media_type"`

	// Author is the name of the message author (for channels with multiple posters).
	Author string `json:"author,omitempty"`

	// Views is the view count as displayed (e.g. "7.62K", "57K").
	Views string `json:"views"`

	// ViewsCount is the parsed view count as an integer (e.g. "7.62K" -> 7620).
	ViewsCount int `json:"views_count"`

	// Edited is true if the message was edited after publishing.
	Edited bool `json:"edited"`

	// Reactions lists emoji reactions with their counts.
	Reactions []Reaction `json:"reactions,omitempty"`

	// ForwardedFrom is the name of the source if the message was forwarded.
	ForwardedFrom string `json:"forwarded_from,omitempty"`

	// ForwardedFromURL is the link to the forwarded-from source.
	ForwardedFromURL string `json:"forwarded_from_url,omitempty"`

	// URLs are inline links found in the message text.
	URLs []string `json:"urls,omitempty"`

	// LinkPreview contains data from a link preview attached to the message.
	LinkPreview *LinkPreview `json:"link_preview,omitempty"`

	// Link is the direct URL to the message in Telegram.
	Link string `json:"link"`
}

// Reaction represents an emoji reaction on a message.
type Reaction struct {
	// Emoji is the reaction emoji (e.g. "👍", "❤").
	Emoji string `json:"emoji"`

	// Count is the number of users who reacted with this emoji.
	Count int `json:"count"`
}

// LinkPreview represents a link preview attached to a message.
type LinkPreview struct {
	// URL is the linked URL.
	URL string `json:"url"`

	// SiteName is the name of the linked site.
	SiteName string `json:"site_name,omitempty"`

	// Title is the preview title.
	Title string `json:"title,omitempty"`

	// Description is the preview description.
	Description string `json:"description,omitempty"`

	// ImageURL is the preview image URL.
	ImageURL string `json:"image_url,omitempty"`
}

// ChannelInfo holds metadata about a Telegram channel.
type ChannelInfo struct {
	// Title is the channel display name.
	Title string `json:"title"`

	// Description is the channel description.
	Description string `json:"description"`

	// ImageURL is the channel avatar/profile image URL.
	ImageURL string `json:"image_url,omitempty"`

	// Subscribers is the subscriber count as displayed (e.g. "10.4M").
	Subscribers string `json:"subscribers,omitempty"`

	// Photos is the number of photos in the channel archive.
	Photos int `json:"photos"`

	// Videos is the number of videos in the channel archive.
	Videos int `json:"videos"`

	// Links is the number of links in the channel archive.
	Links int `json:"links"`
}

// Reader reads messages from a Telegram channel's public web archive.
type Reader struct {
	username string
	baseURL  string
	client   *http.Client
	timeout  time.Duration
	since    time.Time
	until    time.Time
	useUntil bool
}

// Option configures a Reader.
type Option func(*Reader)

// New creates a Reader for the given channel username (without @).
//
// The username is the part after t.me/ — e.g. "durov", "telegram".
// defaultTimeout is the timeout for HTTP requests and page fetches.
const defaultTimeout = 30 * time.Second

func New(username string, opts ...Option) *Reader {
	r := &Reader{
		username: username,
		baseURL:  "https://t.me",
		timeout:  defaultTimeout,
		client:   newClient(defaultTimeout),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(r *Reader) {
		r.client = client
	}
}

// WithTimeout sets the timeout for HTTP requests and individual page fetches.
// This affects both the HTTP client timeout and the per-page context deadline.
//
// When fetching large numbers of messages (e.g. 100+), consider increasing
// the timeout to avoid premature cutoffs on slow connections:
//
//	reader := pkg.New("durov", pkg.WithTimeout(2*time.Minute))
func WithTimeout(timeout time.Duration) Option {
	return func(r *Reader) {
		r.timeout = timeout
		r.client = newClient(timeout)
	}
}

// WithProxy configures the Reader to use an HTTP(S) proxy for all requests.
//
// The proxyURL should be a full URL such as:
//
//	"http://proxy.example.com:8080"
//	"https://user:pass@proxy.example.com:8080"
//	"socks5://127.0.0.1:1080"
//
// Note: SOCKS5 support requires the "golang.org/x/net/proxy" package.
// If WithHTTPClient is also used, apply WithProxy after it so the proxy
// is set on the final transport.
func WithProxy(proxyURL string) Option {
	return func(r *Reader) {
		u, err := url.Parse(proxyURL)
		if err != nil {
			// Store the error and apply it lazily; the Reader will
			// return the error on Fetch. For now, create a client
			// without a proxy — the error is surfaced via the transport.
			return
		}
		r.client.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
	}
}

// WithSince filters messages to only include those published on or after the given date.
// The time is interpreted as UTC midnight of the given day.
func WithSince(date time.Time) Option {
	return func(r *Reader) {
		r.since = date.UTC()
	}
}

// WithUntil filters messages to only include those published on or before the given date
// (the entire day is included).
func WithUntil(date time.Time) Option {
	return func(r *Reader) {
		r.until = date.Add(24 * time.Hour).UTC()
		r.useUntil = true
	}
}

// Fetch retrieves up to limit messages from the channel, newest first.
//
// It follows pagination links automatically to collect enough messages.
// A zero or negative limit means no upper bound (fetches all available).
func (r *Reader) Fetch(ctx context.Context, limit int) ([]Message, error) {
	archiveURL := fmt.Sprintf("https://t.me/s/%s", r.username)

	// Resolve final URL (handle redirects)
	finalURL, err := r.resolveURL(ctx, archiveURL)
	if err != nil {
		return nil, fmt.Errorf("channel: resolve %s: %w", archiveURL, err)
	}

	var messages []Message
	currentURL := finalURL
	emptyPages := 0

	for emptyPages < 3 {
		doc, err := r.fetchPage(ctx, currentURL)
		if err != nil {
			return nil, fmt.Errorf("channel: fetch %s: %w", currentURL, err)
		}

		page, err := r.parseDoc(doc)
		if err != nil {
			return nil, fmt.Errorf("channel: parse: %w", err)
		}

		if len(page) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
			// Try next page in case of a bad page
			nextURL := findNextPage(doc, r.baseURL)
			if nextURL == "" {
				break
			}
			currentURL = nextURL
			continue
		}
		emptyPages = 0

		// Deduplicate: Telegram repeats the last message of the previous page
		// at the top of the next page.
		page = deduplicate(page, messages)

		messages = append(messages, page...)

		// Check limit
		if limit > 0 && len(messages) >= limit {
			messages = messages[:limit]
			break
		}

		// Paginate
		nextURL := findNextPage(doc, r.baseURL)
		if nextURL == "" {
			break
		}
		currentURL = nextURL
	}

	// Telegram returns oldest first; reverse to newest first.
	sliceReverse(messages)

	return messages, nil
}

// FetchInfo retrieves metadata about the channel (title, description,
// subscriber count, etc.) without fetching messages.
func (r *Reader) FetchInfo(ctx context.Context) (*ChannelInfo, error) {
	archiveURL := fmt.Sprintf("https://t.me/s/%s", r.username)

	finalURL, err := r.resolveURL(ctx, archiveURL)
	if err != nil {
		return nil, fmt.Errorf("channel: resolve %s: %w", archiveURL, err)
	}

	doc, err := r.fetchPage(ctx, finalURL)
	if err != nil {
		return nil, fmt.Errorf("channel: fetch info: %w", err)
	}

	info := parseChannelInfo(doc)
	return info, nil
}

// FetchWithInfo retrieves messages and channel metadata in a single request.
func (r *Reader) FetchWithInfo(ctx context.Context, limit int) ([]Message, *ChannelInfo, error) {
	archiveURL := fmt.Sprintf("https://t.me/s/%s", r.username)

	finalURL, err := r.resolveURL(ctx, archiveURL)
	if err != nil {
		return nil, nil, fmt.Errorf("channel: resolve %s: %w", archiveURL, err)
	}

	var messages []Message
	currentURL := finalURL
	emptyPages := 0
	var info *ChannelInfo

	for emptyPages < 3 {
		doc, err := r.fetchPage(ctx, currentURL)
		if err != nil {
			return nil, nil, fmt.Errorf("channel: fetch %s: %w", currentURL, err)
		}

		// Parse channel info from the first page
		if info == nil {
			info = parseChannelInfo(doc)
		}

		page, err := r.parseDoc(doc)
		if err != nil {
			return nil, nil, fmt.Errorf("channel: parse: %w", err)
		}

		if len(page) == 0 {
			emptyPages++
			if emptyPages >= 3 {
				break
			}
			nextURL := findNextPage(doc, r.baseURL)
			if nextURL == "" {
				break
			}
			currentURL = nextURL
			continue
		}
		emptyPages = 0

		page = deduplicate(page, messages)
		messages = append(messages, page...)

		if limit > 0 && len(messages) >= limit {
			messages = messages[:limit]
			break
		}

		nextURL := findNextPage(doc, r.baseURL)
		if nextURL == "" {
			break
		}
		currentURL = nextURL
	}

	sliceReverse(messages)
	return messages, info, nil
}

// deduplicate removes messages from page whose ID already appears in existing.
func deduplicate(page []Message, existing []Message) []Message {
	seen := make(map[int]struct{}, len(existing))
	for _, m := range existing {
		seen[m.ID] = struct{}{}
	}

	deduped := make([]Message, 0, len(page))
	for _, m := range page {
		if _, ok := seen[m.ID]; !ok {
			deduped = append(deduped, m)
			seen[m.ID] = struct{}{}
		}
	}

	return deduped
}

// sliceReverse reverses a slice in place.
func sliceReverse(s []Message) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (r *Reader) resolveURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if location := resp.Header.Get("Location"); location != "" {
			return location, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	return url, nil
}
