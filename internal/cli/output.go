package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"vimm-download/internal/vault"
)

type downloadSummary struct {
	Downloaded int            `json:"downloaded"`
	Skipped    int            `json:"skipped"`
	Failed     int            `json:"failed"`
	Verified   int            `json:"verified"`
	Items      []downloadItem `json:"items,omitempty"`
}

type downloadItem struct {
	ID       int    `json:"id"`
	Title    string `json:"title,omitempty"`
	System   string `json:"system,omitempty"`
	Path     string `json:"path,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Error    string `json:"error,omitempty"`
}

func outputSearchResults(cfg *Config, results []vault.ROMEntry) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	for _, entry := range results {
		fmt.Fprintf(os.Stdout, "%s\t%d\t%s\n", entry.Title, entry.ID, entry.System)
	}
	return nil
}

func outputDownloadSummary(cfg *Config, summary downloadSummary) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}
	fmt.Fprintf(os.Stdout, "downloaded=%d skipped=%d failed=%d verified=%d\n", summary.Downloaded, summary.Skipped, summary.Failed, summary.Verified)
	return nil
}

func outputVerifySummary(cfg *Config, summary downloadSummary) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}
	fmt.Fprintf(os.Stdout, "verified=%d failed=%d\n", summary.Verified, summary.Failed)
	for _, item := range summary.Items {
		status := "ok"
		if item.Error != "" {
			status = "error"
		}
		fmt.Fprintf(os.Stdout, "%s\t%s\t%d\t%s\n", status, item.Path, item.ID, item.Title)
	}
	return nil
}
