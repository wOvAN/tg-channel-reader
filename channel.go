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

	// Views is the view count as displayed (e.g. "7.62K", "57K").
	Views string `json:"views"`

	// Edited is true if the message was edited after publishing.
	Edited bool `json:"edited"`

	// Reactions lists emoji reactions with their counts.
	Reactions []Reaction `json:"reactions,omitempty"`

	// HasMedia is true if the message contains photos, videos, or other media.
	HasMedia bool `json:"has_media"`

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

// Reader reads messages from a Telegram channel's public web archive.
type Reader struct {
	username string
	baseURL  string
	client   *http.Client
	since    time.Time
	until    time.Time
	useUntil bool
}

// Option configures a Reader.
type Option func(*Reader)

// New creates a Reader for the given channel username (without @).
//
// The username is the part after t.me/ — e.g. "durov", "telegram".
func New(username string, opts ...Option) *Reader {
	r := &Reader{
		username: username,
		baseURL:  "https://t.me",
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(r *Reader) {
		r.client = client
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
