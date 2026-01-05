package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"vimm-download/internal/vault"
)

const searchUsage = `vimm search - search ROMs by name pattern

USAGE:
  vimm search --system <code> --query <pattern>
  vimm search --class <console|handheld> --query <pattern>

FLAGS:
  --system <code>               System code (e.g., N64)
  --class <console|handheld>    Search across system class
  --query <pattern>             Search pattern (quote globs like "*mario*")
  --match <auto|glob|prefix|contains|regex>  Match mode (default: auto)
  --limit <n>                   Max results (default: 100)
  --offset <n>                  Skip first N results (default: 0)
  --region <code>               Preferred region (default: USA, use "all" to disable)
  --include-tags <tags>         Include excluded tags (comma-separated, default exclusions: "Virtual Console,LodgeNet"; use "all" to disable exclusions)
  --no-header                   Hide column headers
  -h, --help                    Show help
`

func runSearch(cfg *Config, args []string) int {
	var (
		system      string
		class       string
		query       string
		match       string
		limit       int
		offset      int
		region      string
		includeTags string
		noHdr       bool
		help        bool
	)
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system code")
	fs.StringVar(&class, "class", "", "system class")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.IntVar(&limit, "limit", 100, "limit results")
	fs.IntVar(&offset, "offset", 0, "offset results")
	fs.StringVar(&region, "region", "USA", "preferred region")
	fs.StringVar(&includeTags, "include-tags", "", "include tags")
	fs.BoolVar(&noHdr, "no-header", false, "hide column headers")
	fs.BoolVar(&help, "h", false, "show help")
	fs.BoolVar(&help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, searchUsage)
		return exitUsage
	}
	if help {
		printUsage(os.Stdout, searchUsage)
		return exitOK
	}
	if system == "" && class == "" {
		printError(os.Stderr, fmt.Errorf("--system or --class is required"))
		printUsage(os.Stderr, searchUsage)
		return exitUsage
	}
	if system != "" && class != "" {
		printError(os.Stderr, fmt.Errorf("--system and --class are mutually exclusive"))
		return exitUsage
	}
	if query == "" {
		printError(os.Stderr, fmt.Errorf("--query is required"))
		printUsage(os.Stderr, searchUsage)
		return exitUsage
	}
	if err := validateMatch(match); err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}
	if err := validateClass(class); err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}
	if limit < 0 || offset < 0 {
		printError(os.Stderr, fmt.Errorf("--limit and --offset must be non-negative"))
		return exitUsage
	}

	matcher, err := newMatcher(match, query)
	if err != nil {
		printError(os.Stderr, fmt.Errorf("invalid matcher: %w", err))
		return exitUsage
	}

	runtimeCfg, err := loadRuntimeConfig(cfg.ConfigPath)
	if err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}
	if cfg.NoColor == false && runtimeCfg.NoColor != nil {
		cfg.NoColor = *runtimeCfg.NoColor
	}
	maxRPS := 0
	if runtimeCfg.MaxRPS != nil {
		maxRPS = *runtimeCfg.MaxRPS
	}
	limiter := newRateLimiter(maxRPS)
	if limiter != nil {
		defer limiter.Stop()
	}

	client := vault.NewClient(runtimeCfg.BaseURL)
	client.Limiter = limiter
	ctx := context.Background()

	systems, err := resolveSystemsForSearch(ctx, client, system, class)
	if err != nil {
		printError(os.Stderr, err)
		return exitNetwork
	}

	searchQuery := queryHint(query, match)
	if searchQuery == "" {
		searchQuery = query
	}

	var results []vault.ROMEntry
	for _, sys := range systems {
		entries, err := client.Search(ctx, sys.Slug, searchQuery)
		if err != nil {
			printError(os.Stderr, err)
			return exitNetwork
		}
		for _, entry := range entries {
			ok, err := matcher.Match(entry.Title)
			if err != nil {
				printError(os.Stderr, err)
				return exitUsage
			}
			if ok {
				results = append(results, entry)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].System == results[j].System {
			return results[i].Title < results[j].Title
		}
		return results[i].System < results[j].System
	})

	includeList, includeAll := parseIncludeTags(includeTags)
	filters := titleFilters{
		Region:         normalizeRegion(region),
		ExcludeTags:    defaultExcludedTags(),
		IncludeTags:    includeList,
		IncludeAllTags: includeAll,
	}
	filtered, err := filterSearchResults(ctx, client, results, filters, offset, limit)
	if err != nil {
		printError(os.Stderr, err)
		return exitNetwork
	}

	if err := outputSearchResults(cfg, filtered, noHdr); err != nil {
		printError(os.Stderr, err)
		return exitGeneric
	}
	return exitOK
}

func resolveSystemsForSearch(ctx context.Context, client *vault.Client, system, class string) ([]vault.System, error) {
	if system != "" {
		return []vault.System{{Slug: system}}, nil
	}
	systems, err := client.Systems(ctx)
	if err != nil {
		return nil, err
	}
	if class == "" {
		return systems, nil
	}
	filtered := make([]vault.System, 0, len(systems))
	for _, sys := range systems {
		if sys.Class == class {
			filtered = append(filtered, sys)
		}
	}
	return filtered, nil
}
