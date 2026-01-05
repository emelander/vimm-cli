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

func TestParseMediaFromPage(t *testing.T) {
	html := `
	<html><head><script>
	const media=[{"ID":123,"GoodTitle":"U3VwZXIgTWFyaW8gNjQgKFVTQSkuemY0","Version":"1.0","VersionString":"1.0","Zipped":"100","GoodHash":"ABCDEF12","GoodMd5":"CAFEBABE","GoodSha1":"DEADBEEF"}];
	</script></head></html>`
	media, err := ParseMediaFromPage(html)
	if err != nil {
		t.Fatalf("ParseMediaFromPage error: %v", err)
	}
	if len(media) != 1 {
		t.Fatalf("expected 1 media, got %d", len(media))
	}
	if media[0].ID != 123 {
		t.Fatalf("expected id 123, got %d", media[0].ID)
	}
	hashes := media[0].ExpectedHashes()
	if hashes.CRC != "abcdef12" || hashes.MD5 != "cafebabe" || hashes.SHA1 != "deadbeef" {
		t.Fatalf("unexpected hashes: %+v", hashes)
	}
}

func TestParseLairTxt(t *testing.T) {
	data := []byte("CRC:   b1fcaa9c\nMD5:   caf9a78db13ee00002ff63a3c0c5eabb\nSHA-1: d8b1088520f7c5f81433292a9258c1184afa1457\n")
	hashes, err := ParseLairTxt(data)
	if err != nil {
		t.Fatalf("ParseLairTxt error: %v", err)
	}
	if hashes.CRC != "b1fcaa9c" || hashes.MD5 != "caf9a78db13ee00002ff63a3c0c5eabb" || hashes.SHA1 != "d8b1088520f7c5f81433292a9258c1184afa1457" {
		t.Fatalf("unexpected hashes: %+v", hashes)
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
