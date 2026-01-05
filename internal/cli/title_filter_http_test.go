package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"vimm-cli/internal/vault"
)

func TestFilterSearchResultsUsesLatestAndRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1":
			media := []vault.Media{
				makeMedia(101, "Star Fox 64 (USA).z64", "1.0"),
				makeMedia(102, "Star Fox 64 (USA) (Rev 1).z64", "1.1"),
			}
			writeROMPage(w, media)
		case "/2":
			media := []vault.Media{
				makeMedia(201, "Star Fox 64 (Japan).z64", "1.0"),
			}
			writeROMPage(w, media)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := vault.NewClient(server.URL)
	results := []vault.ROMEntry{
		{ID: 1, Title: "Star Fox 64"},
		{ID: 2, Title: "Star Fox 64"},
	}
	filters := titleFilters{
		Region:         "usa",
		ExcludeTags:    defaultExcludedTags(),
		IncludeTags:    nil,
		IncludeAllTags: false,
	}

	filtered, err := filterSearchResults(context.Background(), client, results, filters, 0, -1)
	if err != nil {
		t.Fatalf("filterSearchResults error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Title != "Star Fox 64 (USA) (Rev 1)" {
		t.Fatalf("unexpected title %q", filtered[0].Title)
	}
}

func TestFilterSearchResultsIncludeVariants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/3" {
			media := []vault.Media{
				makeMedia(301, "Example Game (USA) (LodgeNet).z64", "1.0"),
			}
			writeROMPage(w, media)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := vault.NewClient(server.URL)
	results := []vault.ROMEntry{{ID: 3, Title: "Example Game"}}
	filters := titleFilters{
		Region:         "usa",
		ExcludeTags:    defaultExcludedTags(),
		IncludeTags:    []string{"lodgenet"},
		IncludeAllTags: false,
	}

	filtered, err := filterSearchResults(context.Background(), client, results, filters, 0, -1)
	if err != nil {
		t.Fatalf("filterSearchResults error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected LodgeNet variant to be included, got %d results", len(filtered))
	}
}

func TestFilterSearchResultsErrorOnROMPageFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := vault.NewClient(server.URL)
	results := []vault.ROMEntry{{ID: 9, Title: "Broken"}}
	filters := titleFilters{
		Region:         "usa",
		ExcludeTags:    defaultExcludedTags(),
		IncludeTags:    nil,
		IncludeAllTags: false,
	}

	if _, err := filterSearchResults(context.Background(), client, results, filters, 0, -1); err == nil {
		t.Fatalf("expected error on ROMPage failure")
	}
}

func writeROMPage(w http.ResponseWriter, media []vault.Media) {
	payload, _ := json.Marshal(media)
	fmt.Fprintf(w, "<html><script>const media=%s;</script></html>", payload)
}

func makeMedia(id int, title, version string) vault.Media {
	return vault.Media{
		ID:            id,
		GoodTitle:     base64.StdEncoding.EncodeToString([]byte(title)),
		Version:       version,
		VersionString: version,
		Zipped:        "1",
	}
}
