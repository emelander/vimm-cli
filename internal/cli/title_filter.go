package cli

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/emelander/vimm-cli/internal/vault"
)

type titleFilters struct {
	Region         string
	ExcludeTags    []string
	IncludeTags    []string
	IncludeAllTags bool
}

var titleTagRe = regexp.MustCompile(`\(([^)]+)\)`)

func defaultExcludedTags() []string {
	return []string{"virtual console", "lodgenet"}
}

var knownExtensions = map[string]struct{}{
	".7z":   {},
	".bin":  {},
	".cia":  {},
	".chd":  {},
	".cue":  {},
	".gcm":  {},
	".gcz":  {},
	".gb":   {},
	".gba":  {},
	".gbc":  {},
	".gen":  {},
	".img":  {},
	".iso":  {},
	".mdf":  {},
	".mds":  {},
	".n64":  {},
	".nds":  {},
	".nes":  {},
	".pce":  {},
	".rom":  {},
	".sfc":  {},
	".sgx":  {},
	".smc":  {},
	".sms":  {},
	".v64":  {},
	".wad":  {},
	".wbfs": {},
	".z64":  {},
	".zip":  {},
}

var knownRegions = map[string]struct{}{
	"usa":         {},
	"japan":       {},
	"europe":      {},
	"world":       {},
	"asia":        {},
	"korea":       {},
	"china":       {},
	"taiwan":      {},
	"australia":   {},
	"brazil":      {},
	"canada":      {},
	"uk":          {},
	"france":      {},
	"germany":     {},
	"italy":       {},
	"spain":       {},
	"netherlands": {},
	"sweden":      {},
	"norway":      {},
	"denmark":     {},
	"finland":     {},
}

func filterSearchResults(ctx context.Context, client *vault.Client, results []vault.ROMEntry, filters titleFilters, offset, limit int) ([]vault.ROMEntry, error) {
	if limit == 0 {
		return nil, nil
	}
	filtered := make([]vault.ROMEntry, 0, len(results))
	skipped := 0
	for _, entry := range results {
		displayTitle, err := resolveLatestMediaTitle(ctx, client, entry.ID)
		if err != nil {
			return nil, fmt.Errorf("fetch rom %d: %w", entry.ID, err)
		}
		if displayTitle != "" {
			entry.Title = displayTitle
		}
		if !matchesRegion(entry.Title, filters.Region) {
			continue
		}
		if isExcludedByTags(entry.Title, filters.ExcludeTags, filters.IncludeTags, filters.IncludeAllTags) {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		filtered = append(filtered, entry)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func resolveLatestMediaTitle(ctx context.Context, client *vault.Client, id int) (string, error) {
	page, err := client.ROMPage(ctx, id)
	if err != nil {
		return "", err
	}
	if len(page.Media) == 0 {
		return "", nil
	}
	selected, _, err := selectMedia(page.Media, true, "", page.DownloadAlt)
	if err != nil {
		selected = page.Media[0]
	}
	title := selected.DecodedTitle()
	if title == "" {
		return "", nil
	}
	return trimMediaTitle(title), nil
}

func trimMediaTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	ext := path.Ext(title)
	if ext == "" {
		return title
	}
	if _, ok := knownExtensions[strings.ToLower(ext)]; !ok {
		return title
	}
	return strings.TrimSuffix(title, ext)
}

func parseIncludeTags(value string) ([]string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	lower := strings.ToLower(value)
	if lower == "all" || lower == "*" {
		return nil, true
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, strings.ToLower(part))
	}
	return out, false
}

func normalizeRegion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	switch lower {
	case "all", "any", "*":
		return ""
	case "us", "u.s.", "u.s.a.", "usa":
		return "usa"
	case "jp", "jpn", "japan":
		return "japan"
	case "eu", "eur", "europe":
		return "europe"
	case "uk", "gb", "great britain", "united kingdom":
		return "uk"
	default:
		return lower
	}
}

func matchesRegion(title, region string) bool {
	if region == "" {
		return true
	}
	regions := extractRegionTags(title)
	if len(regions) == 0 {
		return true
	}
	for _, tag := range regions {
		if tag == region {
			return true
		}
	}
	return false
}

func isExcludedByTags(title string, excludes, includes []string, includeAll bool) bool {
	if includeAll || len(excludes) == 0 {
		return false
	}
	tags := extractTitleTags(title)
	for _, tag := range tags {
		normalized := strings.ToLower(tag)
		excluded := false
		for _, ex := range excludes {
			if ex == "" {
				continue
			}
			if strings.Contains(normalized, ex) {
				excluded = true
				break
			}
		}
		if !excluded {
			continue
		}
		if tagAllowed(normalized, includes) {
			continue
		}
		return true
	}
	return false
}

func tagAllowed(normalized string, includes []string) bool {
	if len(includes) == 0 {
		return false
	}
	for _, inc := range includes {
		if inc == "" {
			continue
		}
		if strings.Contains(normalized, inc) {
			return true
		}
	}
	return false
}

func extractRegionTags(title string) []string {
	tags := extractTitleTags(title)
	regions := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := normalizeRegion(tag)
		if normalized == "" {
			continue
		}
		if _, ok := knownRegions[normalized]; ok {
			regions = append(regions, normalized)
		}
	}
	return regions
}

func extractTitleTags(title string) []string {
	title = trimMediaTitle(title)
	if title == "" {
		return nil
	}
	matches := titleTagRe.FindAllStringSubmatch(title, -1)
	if len(matches) == 0 {
		return nil
	}
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tags = append(tags, part)
		}
	}
	return tags
}
