package cli

import (
	"strings"
	"testing"
	"time"
)

func TestFormatTitleTruncates(t *testing.T) {
	got := formatTitle("123456789", 5)
	if got != "1234…" {
		t.Fatalf("expected truncation, got %q", got)
	}
	if len([]rune(got)) != 5 {
		t.Fatalf("expected width 5, got %d", len([]rune(got)))
	}
}

func TestFormatBarWidth(t *testing.T) {
	bar := formatBar(50, 30)
	if len(bar) != 30 {
		t.Fatalf("expected bar length 30, got %d", len(bar))
	}
	if !strings.Contains(bar, "#") || !strings.Contains(bar, ".") {
		t.Fatalf("expected mixed bar, got %q", bar)
	}
}

func TestFormatProgressLineIncludesETA(t *testing.T) {
	line := formatProgressLine("Game", 50, 100, 1024*1024, time.Now(), time.Now().Add(-time.Second))
	if !strings.Contains(line, "ETA") {
		t.Fatalf("expected ETA in line, got %q", line)
	}
}
