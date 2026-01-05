//go:build integration

package cli

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"vimm-cli/internal/vault"
)

func TestStarFox64DownloadEndpoint(t *testing.T) {
	if os.Getenv("VIMM_INTEGRATION") == "" {
		t.Skip("set VIMM_INTEGRATION=1 to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := vault.NewClient("https://vimm.net/vault")
	page, err := client.ROMPage(ctx, 2754)
	if err != nil {
		t.Fatalf("ROMPage error: %v", err)
	}

	media, alt, err := selectMedia(page.Media, true, "", page.DownloadAlt)
	if err != nil {
		t.Fatalf("selectMedia error: %v", err)
	}

	base := page.DownloadBase
	if base == "" {
		base = "https://dl2.vimm.net"
	}
	params := url.Values{}
	params.Set("mediaId", strconv.Itoa(media.ID))
	if alt > 0 {
		params.Set("alt", strconv.Itoa(alt))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/?"+params.Encode(), nil)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	req.Header.Set("User-Agent", vault.DefaultUserAgent)
	req.Header.Set("Referer", "https://vimm.net/vault/2754")
	req.Header.Set("Range", "bytes=0-0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %s", resp.Status)
	}
}
