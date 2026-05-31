package pkg

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const userAgent = "Mozilla/5.0 (compatible; ChannelReader/1.0)"

func (r *Reader) fetchDoc(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// pageTimeout is the maximum time to wait for a single page fetch + parse.
const pageTimeout = 30 * time.Second

// fetchPage fetches a single page with a timeout.
func (r *Reader) fetchPage(ctx context.Context, url string) (*goquery.Document, error) {
	ctx, cancel := context.WithTimeout(ctx, pageTimeout)
	defer cancel()

	return r.fetchDoc(ctx, url)
}
