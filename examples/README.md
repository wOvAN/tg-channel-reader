# Examples

Each example is a standalone Go project with its own `go.mod`.

## basic

Fetch and print the latest messages from a channel.

```bash
cd basic && go run .
```

## filter

Read messages from a specific date onward and aggregate reactions.

```bash
cd filter && go run .
```

## json

Output messages as JSON — useful for piping to `jq`.

```bash
cd json && go run . | jq '.[0].text'
```

## stats

Compute channel statistics: average views, media frequency, top reactions.

```bash
cd stats && go run .
```

## proxy

Read messages through an HTTP(S) or SOCKS5 proxy.

```bash
cd proxy && go run . -proxy http://proxy.example.com:8080
cd proxy && go run . -proxy socks5://127.0.0.1:1080
cd proxy && go run . -proxy http://user:pass@proxy:8080 -channel telegram
```
