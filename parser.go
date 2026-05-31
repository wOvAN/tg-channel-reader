package pkg

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var postRe = regexp.MustCompile(`^([^/]+)/(\d+)$`)
var whitespaceRe = regexp.MustCompile(`\s+`)
var viewsRe = regexp.MustCompile(`^([0-9.]+)([KMB])?$`)

var (
	errNoPostAttr      = fmt.Errorf("channel: no data-post attribute")
	errInvalidPost     = fmt.Errorf("channel: invalid data-post format")
	errNoMessagesFound = fmt.Errorf("channel: no messages found (rate-limited?)")
)

func (r *Reader) parseDoc(doc *goquery.Document) ([]Message, error) {
	// Detect Telegram rate-limit fake page
	if doc.Find("div.tme_no_messages_found").Length() > 0 {
		return nil, errNoMessagesFound
	}

	var messages []Message

	doc.Find("div.tgme_widget_message_wrap").Each(func(_ int, s *goquery.Selection) {
		// Skip the "load more" centered wrapper
		if s.Find("div.tgme_widget_message_centered").Length() > 0 {
			return
		}

		msg, err := parseMessage(s)
		if err != nil {
			return // skip unparseable messages
		}

		// Apply date filters
		if !r.since.IsZero() && msg.Date.Before(r.since) {
			return
		}
		if r.useUntil && msg.Date.After(r.until) {
			return
		}

		messages = append(messages, msg)
	})

	return messages, nil
}

func parseMessage(s *goquery.Selection) (Message, error) {
	msgDiv := s.Find("div.tgme_widget_message")

	// Extract post ID
	postAttr, exists := msgDiv.Attr("data-post")
	if !exists {
		return Message{}, errNoPostAttr
	}

	matches := postRe.FindStringSubmatch(postAttr)
	if matches == nil {
		return Message{}, errInvalidPost
	}

	ch := matches[1]
	id, _ := strconv.Atoi(matches[2])

	// Extract date
	msgDate := parseDateTime(msgDiv)

	// Extract text content
	textDiv := msgDiv.Find("div.tgme_widget_message_text")
	text := cleanText(textDiv.Text())

	// Extract inline URLs from message text
	urls := parseURLs(textDiv)

	// Check if edited
	edited := false
	msgDiv.Find("span.tgme_widget_message_meta").Each(func(_ int, sel *goquery.Selection) {
		if strings.Contains(sel.Text(), "edited") {
			edited = true
		}
	})

	// Extract views
	views := msgDiv.Find("span.tgme_widget_message_views").Text()
	viewsCount := parseViewsCount(views)

	// Extract reactions
	reactions := parseReactions(msgDiv)

	// Extract author
	author := parseAuthor(msgDiv)

	// Extract media type
	mediaType := parseMediaType(msgDiv)

	// Extract forwarded-from info
	forwardedFrom, forwardedFromURL := parseForwardedFrom(msgDiv)

	// Extract link preview
	linkPreview := parseLinkPreview(msgDiv)

	link := "https://t.me/" + ch + "/" + matches[2]

	msg := Message{
		ID:               id,
		Channel:          ch,
		Date:             msgDate,
		Text:             text,
		MediaType:        mediaType,
		Author:           author,
		Views:            views,
		ViewsCount:       viewsCount,
		Edited:           edited,
		Reactions:        reactions,
		ForwardedFrom:    forwardedFrom,
		ForwardedFromURL: forwardedFromURL,
		URLs:             urls,
		LinkPreview:      linkPreview,
		Link:             link,
	}

	return msg, nil
}

func parseDateTime(msgDiv *goquery.Selection) time.Time {
	var t time.Time

	msgDiv.Find("time.time").Each(func(_ int, sel *goquery.Selection) {
		dt, exists := sel.Attr("datetime")
		if !exists {
			return
		}
		parsed, err := time.Parse(time.RFC3339, dt)
		if err != nil {
			parsed, _ = time.Parse("2006-01-02T15:04:05Z07:00", dt)
		}
		t = parsed
	})

	return t
}

func parseReactions(msgDiv *goquery.Selection) []Reaction {
	var reactions []Reaction

	msgDiv.Find("span.tgme_reaction").Each(func(_ int, sel *goquery.Selection) {
		emoji := sel.Find("i.emoji b").Text()
		if emoji == "" {
			emoji = sel.Find("i.emoji").Text()
		}

		// Get count: strip emoji from full text to isolate the number
		countStr := strings.Replace(strings.TrimSpace(sel.Text()), emoji, "", 1)
		countStr = strings.TrimSpace(countStr)
		count, _ := strconv.Atoi(countStr)

		if emoji != "" || count > 0 {
			reactions = append(reactions, Reaction{Emoji: emoji, Count: count})
		}
	})

	return reactions
}

// parseAuthor extracts the message author name from channels with multiple posters.
func parseAuthor(msgDiv *goquery.Selection) string {
	author := msgDiv.Find("span.tgme_widget_message_from_author").Text()
	return strings.TrimSpace(author)
}

// parseMediaType determines the message content type.
// Returns one of: text, photo, video, roundvideo, audio, document,
// sticker, location, poll, service, multimedia.
func parseMediaType(msgDiv *goquery.Selection) string {
	// Check for service messages
	if msgDiv.Find("div.tgme_widget_message_service").Length() > 0 {
		return "service"
	}

	// Check for poll
	if msgDiv.Find("div.tgme_widget_message_poll").Length() > 0 {
		return "poll"
	}

	// Check for location
	if msgDiv.Find("a.tgme_widget_message_location_wrap").Length() > 0 {
		return "location"
	}

	// Check for sticker
	if msgDiv.Find("i.tgme_widget_message_sticker").Length() > 0 {
		return "sticker"
	}

	// Check for document
	if msgDiv.Find("div.tgme_widget_message_document").Length() > 0 {
		return "document"
	}

	// Check for audio
	if msgDiv.Find("audio").Length() > 0 {
		return "audio"
	}

	// Check for round video
	if msgDiv.Find("video.tgme_widget_message_roundvideo").Length() > 0 {
		return "roundvideo"
	}

	// Check for video
	if msgDiv.Find("div.tgme_widget_message_video_player").Length() > 0 ||
		msgDiv.Find("video").Length() > 0 {
		return "video"
	}

	// Check for photo
	if msgDiv.Find("img.tgme_widget_message_photo").Length() > 0 ||
		msgDiv.Find("div.tgme_widget_message_photo_wrap").Length() > 0 {
		return "photo"
	}

	// Check for multimedia (multiple media items)
	if msgDiv.Find("div.media_supported_cont").Length() > 0 {
		return "multimedia"
	}

	return "text"
}

// parseForwardedFrom extracts forwarded-from info.
func parseForwardedFrom(msgDiv *goquery.Selection) (name, url string) {
	msgDiv.Find("a.tgme_widget_message_forwarded_from_name").Each(func(_ int, sel *goquery.Selection) {
		name = strings.TrimSpace(sel.Text())
		if h, exists := sel.Attr("href"); exists {
			url = h
		}
	})
	return name, url
}

// parseURLs extracts inline links from the message text.
func parseURLs(textDiv *goquery.Selection) []string {
	var urls []string
	textDiv.Find("a").Each(func(_ int, sel *goquery.Selection) {
		if h, exists := sel.Attr("href"); exists && h != "" {
			urls = append(urls, h)
		}
	})
	return urls
}

// parseLinkPreview extracts link preview data.
func parseLinkPreview(msgDiv *goquery.Selection) *LinkPreview {
	var lp LinkPreview

	msgDiv.Find("a.tgme_widget_message_link_preview").Each(func(_ int, sel *goquery.Selection) {
		if h, exists := sel.Attr("href"); exists && h != "" {
			lp.URL = h
		}

		lp.SiteName = cleanText(sel.Find("span.tgme_widget_message_link_preview_site_name").Text())
		lp.Title = cleanText(sel.Find("span.tgme_widget_message_link_preview_title").Text())
		lp.Description = cleanText(sel.Find("span.tgme_widget_message_link_preview_description").Text())

		// Extract preview image from background-image CSS
		sel.Find("i.link_preview_image").Each(func(_ int, img *goquery.Selection) {
			if style, exists := img.Attr("style"); exists {
				lp.ImageURL = extractImageURL(style)
			}
		})
	})

	// Only return if we found at least the URL
	if lp.URL != "" {
		return &lp
	}
	return nil
}

// parseViewsCount parses view count strings like "7.62K", "1.2M" into integers.
func parseViewsCount(views string) int {
	views = strings.TrimSpace(views)
	if views == "" {
		return 0
	}

	matches := viewsRe.FindStringSubmatch(views)
	if matches == nil {
		return 0
	}

	num, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(matches[2]) {
	case "K":
		return int(num * 1000)
	case "M":
		return int(num * 1_000_000)
	default:
		return int(num)
	}
}

// extractImageURL extracts a URL from a CSS background-image property.
func extractImageURL(style string) string {
	start := strings.Index(style, "url(")
	if start == -1 {
		return ""
	}
	start += 4 // skip "url("

	// Remove quotes
	if style[start] == '\'' || style[start] == '"' {
		start++
	}

	end := strings.Index(style[start:], ")")
	if end == -1 {
		return ""
	}

	u := style[start : start+end]

	// Fix protocol-relative URLs
	if strings.HasPrefix(u, "//") {
		u = "https:" + u
	}

	return u
}

func cleanText(text string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(text, " "))
}

// findNextPage finds the URL of the next page in the archive.
// Priority: <link rel="prev"> > centered wrapper > ?before= pattern.
func findNextPage(doc *goquery.Document, baseURL string) string {
	var nextURL string

	// Primary: <link rel="prev"> — most reliable
	doc.Find("link[rel=prev]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			nextURL = href
		}
	})

	// Fallback: centered message wrapper
	if nextURL == "" {
		doc.Find("div.tgme_widget_message_centered").Each(func(_ int, s *goquery.Selection) {
			if href, exists := s.Find("a").First().Attr("href"); exists && href != "" {
				nextURL = href
			}
		})
	}

	// Fallback: ?before= pattern
	if nextURL == "" {
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists && strings.Contains(href, "?before=") {
				nextURL = href
			}
		})
	}

	// Resolve relative URLs
	if nextURL != "" && !strings.HasPrefix(nextURL, "http") {
		if base, err := url.Parse(baseURL); err == nil {
			if resolved, err := base.Parse(nextURL); err == nil {
				nextURL = resolved.String()
			}
		}
	}

	return nextURL
}

// parseChannelInfo extracts channel metadata from the archive page.
func parseChannelInfo(doc *goquery.Document) *ChannelInfo {
	info := &ChannelInfo{}

	// Extract og: meta tags
	doc.Find("meta[property]").Each(func(_ int, s *goquery.Selection) {
		prop, _ := s.Attr("property")
		content, _ := s.Attr("content")

		switch prop {
		case "og:title":
			info.Title = content
		case "og:description":
			info.Description = content
		case "og:image":
			info.ImageURL = content
		}
	})

	// Extract channel counters (photos, videos, links, subscribers)
	doc.Find("div.tgme_channel_info_counters a, div.tgme_channel_info_counter a").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		href, _ := s.Attr("href")

		switch {
		case strings.Contains(href, "/photos"):
			info.Photos = parseCounter(text)
		case strings.Contains(href, "/videos"):
			info.Videos = parseCounter(text)
		case strings.Contains(href, "/links"):
			info.Links = parseCounter(text)
		case strings.Contains(href, "/members"):
			info.Subscribers = text
		}
	})

	// If counters are in a single div (alternative layout)
	if info.Photos == 0 && info.Videos == 0 && info.Links == 0 {
		doc.Find("div.tgme_channel_info_counters").Each(func(_ int, s *goquery.Selection) {
			s.Find("a").Each(func(_ int, a *goquery.Selection) {
				text := strings.TrimSpace(a.Text())
				href, _ := a.Attr("href")

				switch {
				case strings.Contains(href, "/photos"):
					info.Photos = parseCounter(text)
				case strings.Contains(href, "/videos"):
					info.Videos = parseCounter(text)
				case strings.Contains(href, "/links"):
					info.Links = parseCounter(text)
				}
			})
		})
	}

	return info
}

// parseCounter parses a counter string like "1.2K" or "57" into an integer.
func parseCounter(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	n, err := strconv.Atoi(s)
	if err == nil {
		return n
	}

	// Try with K/M suffix
	matches := viewsRe.FindStringSubmatch(s)
	if matches == nil {
		return 0
	}

	num, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(matches[2]) {
	case "K":
		return int(num * 1000)
	case "M":
		return int(num * 1_000_000)
	default:
		return int(num)
	}
}
