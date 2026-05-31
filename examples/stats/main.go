// Stats example: compute channel statistics from recent messages.
//
// Shows how to use the package to gather metrics like average views,
// reaction distribution, and media frequency.
package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	tg "github.com/wOvAN/tg-channel-reader"
)

// Stats holds aggregated channel statistics.
type Stats struct {
	MessageCount int
	MediaCount   int
	TotalViews   float64
	Reactions    map[string]int
}

func main() {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 10)
	if err != nil {
		log.Fatal(err)
	}

	stats := computeStats(msgs)
	printStats(stats)
}

func computeStats(msgs []tg.Message) Stats {
	s := Stats{
		MessageCount: len(msgs),
		Reactions:    make(map[string]int),
	}

	for _, m := range msgs {
		if m.HasMedia {
			s.MediaCount++
		}

		views, _ := parseViews(m.Views)
		s.TotalViews += views

		for _, r := range m.Reactions {
			s.Reactions[r.Emoji] += r.Count
		}
	}

	return s
}

// parseViews converts "7.62K" to 7620, "15M" to 15000000, etc.
func parseViews(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	var multiplier float64 = 1
	switch strings.ToUpper(s[len(s)-1:]) {
	case "K":
		multiplier = 1_000
		s = s[:len(s)-1]
	case "M":
		multiplier = 1_000_000
		s = s[:len(s)-1]
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return f * multiplier, nil
}

func printStats(s Stats) {
	fmt.Printf("Channel Stats (%d messages)\n", s.MessageCount)
	fmt.Println(strings.Repeat("=", 40))

	if s.MessageCount > 0 {
		avgViews := s.TotalViews / float64(s.MessageCount)
		fmt.Printf("Average views:  %.0f\n", avgViews)
		fmt.Printf("With media:     %d (%.0f%%)\n",
			s.MediaCount, float64(s.MediaCount)/float64(s.MessageCount)*100)
	}

	// Sort reactions by count descending.
	type kv struct {
		Emoji string
		Count int
	}
	var sorted []kv
	for emoji, count := range s.Reactions {
		sorted = append(sorted, kv{emoji, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	if len(sorted) > 0 {
		fmt.Printf("\nTop reactions:\n")
		for _, kv := range sorted {
			fmt.Printf("  %s %d\n", kv.Emoji, kv.Count)
		}
	}
}
