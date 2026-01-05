package cli

import "testing"

func TestTrimMediaTitle(t *testing.T) {
	title := "Star Fox 64 (USA) (Rev 1).z64"
	got := trimMediaTitle(title)
	if got != "Star Fox 64 (USA) (Rev 1)" {
		t.Fatalf("trimMediaTitle = %q", got)
	}
	if trimMediaTitle("Dr. Mario") != "Dr. Mario" {
		t.Fatalf("expected title without extension to remain unchanged")
	}
}

func TestExtractTitleTags(t *testing.T) {
	title := "Star Fox 64 (USA) (Rev 1) (Wii Virtual Console).z64"
	tags := extractTitleTags(title)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %#v", len(tags), tags)
	}
	if tags[0] != "USA" || tags[1] != "Rev 1" || tags[2] != "Wii Virtual Console" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestNormalizeRegion(t *testing.T) {
	cases := map[string]string{
		"USA":            "usa",
		"U.S.A.":         "usa",
		"Japan":          "japan",
		"EUR":            "europe",
		"all":            "",
		"any":            "",
		"United Kingdom": "uk",
	}
	for input, expected := range cases {
		if got := normalizeRegion(input); got != expected {
			t.Fatalf("normalizeRegion(%q) = %q, expected %q", input, got, expected)
		}
	}
}

func TestMatchesRegion(t *testing.T) {
	title := "Star Fox 64 (USA) (Rev 1)"
	if !matchesRegion(title, "usa") {
		t.Fatalf("expected USA to match")
	}
	if matchesRegion(title, "japan") {
		t.Fatalf("expected Japan to not match")
	}
	if !matchesRegion("Mystery Game", "usa") {
		t.Fatalf("expected unknown region to pass")
	}
}

func TestParseIncludeTags(t *testing.T) {
	tags, all := parseIncludeTags("all")
	if !all || len(tags) != 0 {
		t.Fatalf("expected include-all for all")
	}
	tags, all = parseIncludeTags("Virtual Console,LodgeNet")
	if all || len(tags) != 2 {
		t.Fatalf("expected two tags, got %#v (all=%v)", tags, all)
	}
}

func TestIsExcludedByTags(t *testing.T) {
	title := "Star Fox 64 (Japan) (Wii Virtual Console)"
	ex := defaultExcludedTags()
	if !isExcludedByTags(title, ex, nil, false) {
		t.Fatalf("expected tag to be excluded")
	}
	if isExcludedByTags("Star Fox 64 (USA) (Rev 1)", ex, nil, false) {
		t.Fatalf("did not expect exclusion")
	}
	if isExcludedByTags(title, ex, []string{"virtual console"}, false) {
		t.Fatalf("expected include override")
	}
	if isExcludedByTags(title, ex, nil, true) {
		t.Fatalf("expected include-all override")
	}
}
