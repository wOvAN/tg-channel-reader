// Filter example: read messages from a specific date onward.
//
// Shows how to use WithSince to narrow results,
// and how to aggregate reactions across messages.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tg "github.com/wOvAN/tg-channel-reader"
)

func main() {
	// Read messages from May 6, 2026 onward.
	since := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)

	reader := tg.New("durov", tg.WithSince(since))

	msgs, err := reader.Fetch(context.Background(), 5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Messages since 2026-05-06: %d\n\n", len(msgs))

	// Aggregate reactions.
	reactionSum := make(map[string]int)
	for _, m := range msgs {
		for _, r := range m.Reactions {
			reactionSum[r.Emoji] += r.Count
		}
	}

	fmt.Println("Total reactions:")
	for emoji, count := range reactionSum {
		fmt.Printf("  %s %d\n", emoji, count)
	}
}
