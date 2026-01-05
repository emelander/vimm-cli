package vault

import (
	"strings"
	"testing"
)

func TestParseSystemCount(t *testing.T) {
	html := `<html><body><table><tr><td>Have 1147 of 1147 media (100%)</td></tr></table></body></html>`
	count, err := ParseSystemCount(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseSystemCount error: %v", err)
	}
	if count != 1147 {
		t.Fatalf("expected 1147, got %d", count)
	}
}

func TestParseSystems(t *testing.T) {
	html := `
	<html><body>
	  <table><caption>Consoles</caption>
	    <tr><td><a href="/vault/N64">Nintendo 64</a></td></tr>
	  </table>
	  <table><caption>Handhelds</caption>
	    <tr><td><a href="/vault/GBA">Game Boy Adv</a></td></tr>
	  </table>
	</body></html>`
	systems, err := ParseSystems(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseSystems error: %v", err)
	}
	if len(systems) != 2 {
		t.Fatalf("expected 2 systems, got %d", len(systems))
	}
	bySlug := map[string]System{}
	for _, sys := range systems {
		bySlug[sys.Slug] = sys
	}
	if bySlug["N64"].Class != "console" || bySlug["N64"].Name != "Nintendo 64" {
		t.Fatalf("unexpected N64 system: %+v", bySlug["N64"])
	}
	if bySlug["GBA"].Class != "handheld" || bySlug["GBA"].Name != "Game Boy Adv" {
		t.Fatalf("unexpected GBA system: %+v", bySlug["GBA"])
	}
}

func TestParseSearchResults(t *testing.T) {
	html := `
	<html><body>
	  <table class="rounded hovertable striped">
	    <tr><td><a href="/vault/123">Mario Kart 64</a></td></tr>
	    <tr><td><a href="/vault/999">Super Mario 64</a></td></tr>
	    <tr><td><a href="/vault/?p=rating&amp;id=123">8.1</a></td></tr>
	  </table>
	</body></html>`
	results, err := ParseSearchResults(strings.NewReader(html), "N64")
	if err != nil {
		t.Fatalf("ParseSearchResults error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].System != "N64" || results[1].System != "N64" {
		t.Fatalf("expected system N64, got %+v", results)
	}
}

func TestSlugFromHref(t *testing.T) {
	if got := slugFromHref("/vault/N64"); got != "N64" {
		t.Fatalf("expected N64, got %q", got)
	}
	if got := slugFromHref("/vault/123"); got != "" {
		t.Fatalf("expected empty for numeric slug, got %q", got)
	}
	if got := slugFromHref("/vault/N64/A"); got != "" {
		t.Fatalf("expected empty for nested slug, got %q", got)
	}
}

func TestVaultIDFromHref(t *testing.T) {
	if id, ok := vaultIDFromHref("/vault/123"); !ok || id != 123 {
		t.Fatalf("expected id 123, got %d ok=%v", id, ok)
	}
	if id, ok := vaultIDFromHref("/vault/123?x=1"); !ok || id != 123 {
		t.Fatalf("expected id 123 with query, got %d ok=%v", id, ok)
	}
	if _, ok := vaultIDFromHref("/vault/N64"); ok {
		t.Fatalf("expected non-numeric href to fail")
	}
}
