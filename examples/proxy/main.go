// Proxy example: read messages through an HTTP(S) proxy.
//
// Shows how to use WithProxy to route all requests through
// a proxy server. Supports http://, https://, and socks5:// URLs.
//
// Usage:
//
//	go run . -proxy http://proxy.example.com:8080
//	go run . -proxy socks5://127.0.0.1:1080
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	tg "github.com/wOvAN/tg-channel-reader"
)

func main() {
	proxyURL := flag.String("proxy", "", "Proxy URL (e.g. http://proxy:8080, socks5://127.0.0.1:1080)")
	channel := flag.String("channel", "durov", "Channel username")
	limit := flag.Int("limit", 5, "Number of messages")
	flag.Parse()

	if *proxyURL == "" {
		fmt.Println("Usage: go run . -proxy <proxy-url>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run . -proxy http://proxy.example.com:8080")
		fmt.Println("  go run . -proxy socks5://127.0.0.1:1080")
		fmt.Println("  go run . -proxy http://user:pass@proxy:8080 -channel telegram")
		return
	}

	reader := tg.New(*channel, tg.WithProxy(*proxyURL))

	msgs, err := reader.Fetch(context.Background(), *limit)
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range msgs {
		fmt.Printf("[%s] #%d %s\n", m.Date.Format("15:04"), m.ID, m.Text[:min(80, len(m.Text))])
	}
}
