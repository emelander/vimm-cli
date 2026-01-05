package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vimm-cli/internal/vault"
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
	Format   string `json:"format,omitempty"`
}

func outputSearchResults(cfg *Config, results []vault.ROMEntry, noHeader bool) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if cfg.Plain {
		for _, entry := range results {
			fmt.Fprintf(os.Stdout, "%s\t%d\t%s\n", entry.Title, entry.ID, entry.System)
		}
		return nil
	}

	titleHeader := "TITLE"
	idHeader := "ID"
	systemHeader := "SYSTEM"
	titleWidth := len(titleHeader)
	idWidth := len(idHeader)
	systemWidth := len(systemHeader)
	for _, entry := range results {
		if len(entry.Title) > titleWidth {
			titleWidth = len(entry.Title)
		}
		idLen := len(fmt.Sprintf("%d", entry.ID))
		if idLen > idWidth {
			idWidth = idLen
		}
		if len(entry.System) > systemWidth {
			systemWidth = len(entry.System)
		}
	}
	if !noHeader {
		fmt.Fprintf(os.Stdout, "%-*s  %*s  %-*s\n", titleWidth, titleHeader, idWidth, idHeader, systemWidth, systemHeader)
	}
	for _, entry := range results {
		fmt.Fprintf(os.Stdout, "%-*s  %*d  %-*s\n", titleWidth, entry.Title, idWidth, entry.ID, systemWidth, entry.System)
	}
	return nil
}

func outputDownloadSummary(cfg *Config, summary downloadSummary) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}
	formatSummary := formatCounts(summary.Items)
	if formatSummary != "" {
		fmt.Fprintf(os.Stdout, "downloaded=%d skipped=%d failed=%d verified=%d formats=%s\n", summary.Downloaded, summary.Skipped, summary.Failed, summary.Verified, formatSummary)
		return nil
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
	formatSummary := formatCounts(summary.Items)
	if formatSummary != "" {
		fmt.Fprintf(os.Stdout, "verified=%d failed=%d formats=%s\n", summary.Verified, summary.Failed, formatSummary)
	} else {
		fmt.Fprintf(os.Stdout, "verified=%d failed=%d\n", summary.Verified, summary.Failed)
	}
	for _, item := range summary.Items {
		status := "ok"
		if item.Error != "" {
			status = "error"
		}
		fmt.Fprintf(os.Stdout, "%s\t%s\t%d\t%s\n", status, item.Path, item.ID, item.Title)
	}
	return nil
}

func formatCounts(items []downloadItem) string {
	counts := map[string]int{}
	for _, item := range items {
		if item.Format == "" {
			continue
		}
		counts[item.Format]++
	}
	if len(counts) == 0 {
		return ""
	}
	labels := []string{"standard", "alt", "alt2"}
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		if count := counts[label]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", label, count))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}
