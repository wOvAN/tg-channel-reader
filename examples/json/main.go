// JSON example: fetch messages and output as JSON for piping to jq.
//
// Demonstrates encoding Message structs to JSON and writing to stdout.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tg "github.com/wOvAN/tg-channel-reader"
)

func main() {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(msgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
