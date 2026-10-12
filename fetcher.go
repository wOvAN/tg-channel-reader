package tgchannelreader

import (
	"context"
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

const userAgent = "Mozilla/5.0 (compatible; ChannelReader/1.0)"

// get issues a GET request with the library user agent.
// The caller owns the response body and status handling.
func (r *Reader) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	return r.client.Do(req)
}

func (r *Reader) fetchDoc(ctx context.Context, url string) (*goquery.Document, error) {
	resp, err := r.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// fetchPage fetches a single page with a timeout.
func (r *Reader) fetchPage(ctx context.Context, url string) (*goquery.Document, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	return r.fetchDoc(ctx, url)
}
