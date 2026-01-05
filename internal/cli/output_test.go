package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"vimm-download/internal/vault"
)

func TestOutputSearchResultsHeader(t *testing.T) {
	cfg := &Config{}
	results := []vault.ROMEntry{{Title: "Star Fox 64", ID: 2754, System: "N64"}}

	out := captureStdout(t, func() {
		if err := outputSearchResults(cfg, results, false); err != nil {
			t.Fatalf("outputSearchResults error: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + row, got %q", out)
	}
	if !strings.HasPrefix(lines[0], "TITLE") {
		t.Fatalf("expected header line, got %q", lines[0])
	}
}

func TestOutputSearchResultsNoHeader(t *testing.T) {
	cfg := &Config{}
	results := []vault.ROMEntry{{Title: "Star Fox 64", ID: 2754, System: "N64"}}

	out := captureStdout(t, func() {
		if err := outputSearchResults(cfg, results, true); err != nil {
			t.Fatalf("outputSearchResults error: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected single row, got %q", out)
	}
	if strings.HasPrefix(lines[0], "TITLE") {
		t.Fatalf("expected no header line, got %q", lines[0])
	}
}

func TestOutputSearchResultsPlain(t *testing.T) {
	cfg := &Config{Plain: true}
	results := []vault.ROMEntry{{Title: "Star Fox 64", ID: 2754, System: "N64"}}

	out := captureStdout(t, func() {
		if err := outputSearchResults(cfg, results, false); err != nil {
			t.Fatalf("outputSearchResults error: %v", err)
		}
	})
	if !strings.Contains(out, "\t") {
		t.Fatalf("expected tab-separated output, got %q", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig
	output := <-done
	_ = r.Close()
	return output
}
