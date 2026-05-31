package pkg_test

import (
	"context"
	"testing"
	"time"

	tg "github.com/wOvAN/tg-channel-reader"
)

func TestFetch_Basic(t *testing.T) {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 5)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}

	// Check first message structure
	m := msgs[0]
	if m.ID == 0 {
		t.Error("expected non-zero message ID")
	}
	if m.Channel != "durov" {
		t.Errorf("expected channel durov, got %s", m.Channel)
	}
	if m.Date.IsZero() {
		t.Error("expected non-zero date")
	}
	if m.Text == "" {
		t.Error("expected non-empty text")
	}
	if m.Views == "" {
		t.Error("expected non-empty views")
	}
	if m.Link == "" {
		t.Error("expected non-empty link")
	}

	// Messages should be in descending order (newest first)
	for i := 1; i < len(msgs); i++ {
		if msgs[i].ID >= msgs[i-1].ID {
			t.Errorf("messages not in descending order: #%d >= #%d", msgs[i].ID, msgs[i-1].ID)
		}
	}
}

func TestFetch_WithSince(t *testing.T) {
	since := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	reader := tg.New("durov", tg.WithSince(since))

	msgs, err := reader.Fetch(context.Background(), 5)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatal("expected some messages since 2026-05-06")
	}

	for _, m := range msgs {
		if m.Date.Before(since) {
			t.Errorf("message #%d date %v is before since %v", m.ID, m.Date, since)
		}
	}
}

func TestFetch_WithUntil(t *testing.T) {
	until := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	reader := tg.New("durov", tg.WithUntil(until))

	msgs, err := reader.Fetch(context.Background(), 5)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatal("expected some messages until 2026-05-06")
	}

	endOfDay := until.Add(24 * time.Hour)
	for _, m := range msgs {
		if m.Date.After(endOfDay) {
			t.Errorf("message #%d date %v is after until %v", m.ID, m.Date, endOfDay)
		}
	}
}

func TestFetch_Reactions(t *testing.T) {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 10)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Check that reactions field is populated (may be empty for some channels).
	for _, m := range msgs {
		for _, r := range m.Reactions {
			if r.Emoji == "" {
				t.Errorf("message #%d has reaction with empty emoji", m.ID)
			}
		}
	}
}

func TestFetch_Durov(t *testing.T) {
	reader := tg.New("durov")

	msgs, err := reader.Fetch(context.Background(), 3)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	for _, m := range msgs {
		if m.Channel != "durov" {
			t.Errorf("expected channel durov, got %s", m.Channel)
		}
	}
}

func TestFetch_Pagination(t *testing.T) {
	reader := tg.New("durov")

	// Fetch 30 messages — enough to trigger pagination (one page ~20-25 msgs)
	msgs, err := reader.Fetch(context.Background(), 30)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(msgs) != 30 {
		t.Fatalf("expected 30 messages (with pagination), got %d", len(msgs))
	}

	// All IDs should be unique
	seen := make(map[int]bool, len(msgs))
	for _, m := range msgs {
		if seen[m.ID] {
			t.Errorf("duplicate message ID: %d", m.ID)
		}
		seen[m.ID] = true
	}
}

func TestWithProxy_InvalidURL(t *testing.T) {
	// WithProxy with an invalid URL should not panic and the Reader
	// should still work (proxy is simply not applied).
	reader := tg.New("durov", tg.WithProxy("://invalid"))

	msgs, err := reader.Fetch(context.Background(), 2)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestFetch_NonexistentChannel(t *testing.T) {
	reader := tg.New("this_channel_does_not_exist_12345")

	_, err := reader.Fetch(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for nonexistent channel")
	}
}
