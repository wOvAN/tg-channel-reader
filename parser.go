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

var (
	errNoPostAttr = fmt.Errorf("channel: no data-post attribute")
	errInvalidPost = fmt.Errorf("channel: invalid data-post format")
)

func (r *Reader) parseDoc(doc *goquery.Document) ([]Message, error) {
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
	text := cleanText(msgDiv.Find("div.tgme_widget_message_text").Text())

	// Check if edited
	edited := false
	msgDiv.Find("span.tgme_widget_message_meta").Each(func(_ int, sel *goquery.Selection) {
		if strings.Contains(sel.Text(), "edited") {
			edited = true
		}
	})

	// Extract views
	views := msgDiv.Find("span.tgme_widget_message_views").Text()

	// Extract reactions
	reactions := parseReactions(msgDiv)

	// Check for media
	hasMedia := hasMedia(msgDiv)

	link := "https://t.me/" + ch + "/" + matches[2]

	return Message{
		ID:        id,
		Channel:   ch,
		Date:      msgDate,
		Text:      text,
		Views:     views,
		Edited:    edited,
		Reactions: reactions,
		HasMedia:  hasMedia,
		Link:      link,
	}, nil
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

func hasMedia(msgDiv *goquery.Selection) bool {
	return msgDiv.Find("div.media_supported_cont").Length() > 0 ||
		msgDiv.Find("img.tgme_widget_message_photo").Length() > 0 ||
		msgDiv.Find("div.tgme_widget_message_video_player").Length() > 0
}

func cleanText(text string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(text, " "))
}

// findNextPage finds the URL of the next page in the archive.
func findNextPage(doc *goquery.Document, baseURL string) string {
	var nextURL string

	// The pagination link is in the centered message wrapper
	doc.Find("div.tgme_widget_message_centered").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Find("a").First().Attr("href"); exists && href != "" {
			nextURL = href
		}
	})

	// Fallback: look for ?before= pattern
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
