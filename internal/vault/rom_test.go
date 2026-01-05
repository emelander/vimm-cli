package vault

import "testing"

func TestDownloadAvailableAlt(t *testing.T) {
	media := Media{
		Zipped:     "1",
		AltZipped:  "0",
		AltZipped2: "5",
	}

	if !media.DownloadAvailableAlt(0) {
		t.Fatalf("expected standard format to be available")
	}
	if media.DownloadAvailableAlt(1) {
		t.Fatalf("expected alt format to be unavailable")
	}
	if !media.DownloadAvailableAlt(2) {
		t.Fatalf("expected alt2 format to be available")
	}
}

func TestParseDownloadBaseFromPage(t *testing.T) {
	html := `<form action="//dl3.vimm.net/" method="POST" id="dl_form"></form>`
	got := ParseDownloadBaseFromPage(html)
	if got != "https://dl3.vimm.net" {
		t.Fatalf("expected https://dl3.vimm.net, got %q", got)
	}
}

func TestParseDownloadAltFromPage(t *testing.T) {
	html := `<select id="dl_format"><option value="0">JB</option><option value="1" selected>.dec.iso</option></select>`
	if got := ParseDownloadAltFromPage(html); got != 1 {
		t.Fatalf("expected alt=1, got %d", got)
	}
	html = `<select id="dl_format"><option value="2">alt2</option><option value="0">std</option></select>`
	if got := ParseDownloadAltFromPage(html); got != 2 {
		t.Fatalf("expected alt=2 from first option, got %d", got)
	}
	if got := ParseDownloadAltFromPage("<html></html>"); got != 0 {
		t.Fatalf("expected alt=0 when selector missing, got %d", got)
	}
}
