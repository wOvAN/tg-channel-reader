# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

tg-channel-reader is a Go library that reads messages from public Telegram channel web archives (`t.me/s/<username>`) using HTTP requests and HTML parsing — no authentication, API keys, or bots required.

- **Module**: `github.com/wOvAN/tg-channel-reader`
- **Go**: 1.25.0
- **Key dependency**: `github.com/PuerkitoBio/goquery` (CSS selector-based HTML parsing)

## Development Commands

All commands are run from the project root with standard Go tooling — no Makefile or build scripts.

```bash
go build                    # Build the library
go test -v                  # Run all tests (integration tests hitting live Telegram)
go test -v -run TestFetch_Basic  # Run a single test by name
```

Examples are standalone Go modules with a `replace` directive pointing to `../..`:

```bash
cd examples/basic && go run .   # Fetch and print messages
cd examples/filter && go run .  # Date filtering + reaction aggregation
cd examples/json && go run .    # JSON output for piping
cd examples/proxy && go run .   # Proxy support
cd examples/stats && go run .   # Channel statistics
```

## Architecture

Four Go source files, each with a single responsibility:

- **`channel.go`** — Public API. `Reader` type with functional options (`WithHTTPClient`, `WithProxy`, `WithSince`, `WithUntil`). The `Fetch(ctx, limit)` method orchestrates pagination, deduplication, date filtering, and reversal (Telegram returns oldest-first; library reverses to newest-first).
- **`fetcher.go`** — HTTP fetching with user-agent, context timeout (30s per page), and status code validation.
- **`parser.go`** — goquery-based HTML parsing. Extracts messages from `div.tgme_widget_message_wrap`, dates from `time.time`, reactions from `span.tgme_reaction`, view counts, and media indicators.
- **`channel_test.go`** — Integration tests that fetch from live Telegram (8 tests covering basic fetch, date filtering, reactions, pagination, nonexistent channels).

### Key Data Types

- `Message` — ID, Channel, Date, Text, Views, Edited, Reactions, HasMedia, Link
- `Reaction` — Emoji, Count

### Pagination Detail

Telegram repeats the last message of the previous page at the top of the next page. The `Fetch` method deduplicates by tracking the last message ID. The loop breaks after 3 consecutive empty pages (no new messages found).

### Constraints

- Only works with **public channels** (those with `t.me/s/<username>`)
- Telegram limits archive depth — typically ~3000 messages
- Frequent requests may trigger CAPTCHA (HTTP 429/403)
