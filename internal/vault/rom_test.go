package vault

import "testing"

func TestDownloadAvailableVariants(t *testing.T) {
	media := Media{
		Zipped:     "1",
		AltZipped:  "0",
		AltZipped2: "5",
	}

	if !media.DownloadAvailable("standard") {
		t.Fatalf("expected standard variant to be available")
	}
	if media.DownloadAvailable("alt") {
		t.Fatalf("expected alt variant to be unavailable")
	}
	if !media.DownloadAvailable("alt2") {
		t.Fatalf("expected alt2 variant to be available")
	}
}
