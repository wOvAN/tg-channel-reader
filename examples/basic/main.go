// Basic example: fetch and print the latest messages from a channel.
package main

import (
	"context"
	"fmt"
	"log"

	tg "github.com/wOvAN/tg-channel-reader"
)

func main() {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 5)
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range msgs {
		fmt.Printf("[%s] #%d %s\n", m.Date.Format("15:04"), m.ID, m.Text[:min(80, len(m.Text))])
	}
}
