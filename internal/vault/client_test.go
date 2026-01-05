package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSectionNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/N64/Z" {
			http.NotFound(w, r)
			return
		}
		t.Fatalf("unexpected request path: %s", r.URL.Path)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	entries, err := client.ListSection(context.Background(), "N64", "Z")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}
