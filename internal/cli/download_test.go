package cli

import (
	"net/http"
	"testing"
)

func TestParseContentRangeTotal(t *testing.T) {
	if got := parseContentRangeTotal("bytes 0-99/1000"); got != 1000 {
		t.Fatalf("expected 1000, got %d", got)
	}
	if got := parseContentRangeTotal("bytes */100"); got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}
	if got := parseContentRangeTotal("invalid"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestTotalFromResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode:    http.StatusPartialContent,
		ContentLength: 100,
		Header:        http.Header{"Content-Range": []string{"bytes 0-99/1000"}},
	}
	if got := totalFromResponse(resp, 0); got != 1000 {
		t.Fatalf("expected 1000, got %d", got)
	}

	resp = &http.Response{
		StatusCode:    http.StatusPartialContent,
		ContentLength: 100,
		Header:        http.Header{},
	}
	if got := totalFromResponse(resp, 50); got != 150 {
		t.Fatalf("expected 150, got %d", got)
	}

	resp = &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 80,
		Header:        http.Header{},
	}
	if got := totalFromResponse(resp, 0); got != 80 {
		t.Fatalf("expected 80, got %d", got)
	}
}
