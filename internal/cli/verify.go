package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"vimm-download/internal/vault"
)

const verifyUsage = `vimm verify - verify local ROMs

USAGE:
  vimm verify --system <code> [--query <pattern>] --output-dir <path>
  vimm verify --id <vault_id> [--id <vault_id> ...] --output-dir <path>

FLAGS:
  --system <code>               System code
  --id <vault_id>               Verify specific ROMs by id (repeatable)
  --query <pattern>             Search pattern (quote globs like "*mario*")
  --match <auto|glob|prefix|contains|regex>  Match mode (default: auto)
  --output-dir <path>           Output directory (default: .)
  --strict-hashes               Fail if any hash missing (default: true)
  -h, --help                    Show help
`

func runVerify(cfg *Config, args []string) int {
	var (
		system    string
		query     string
		match     string
		outputDir string
		ids       []string
		strict    bool
		help      bool
	)

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system code")
	fs.Var(&stringSliceFlag{values: &ids}, "id", "vault id (repeatable)")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.StringVar(&outputDir, "output-dir", ".", "output directory")
	fs.BoolVar(&strict, "strict-hashes", true, "strict hash checks")
	fs.BoolVar(&help, "h", false, "show help")
	fs.BoolVar(&help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, verifyUsage)
		return exitUsage
	}
	if help {
		printUsage(os.Stdout, verifyUsage)
		return exitOK
	}

	if err := validateMatch(match); err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}

	if len(ids) > 0 {
		if system != "" || query != "" {
			printError(os.Stderr, fmt.Errorf("--id cannot be combined with --system or --query"))
			return exitUsage
		}
	} else {
		if system == "" {
			printError(os.Stderr, fmt.Errorf("--system or --id is required"))
			return exitUsage
		}
	}

	if outputDir == "" {
		printError(os.Stderr, fmt.Errorf("--output-dir is required"))
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
	if runtimeCfg.OutputDir != nil && *runtimeCfg.OutputDir != "" && outputDir == "." {
		outputDir = *runtimeCfg.OutputDir
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

	entries, err := selectVerifyTargets(ctx, client, system, query, match, ids)
	if err != nil {
		printError(os.Stderr, err)
		return exitNetwork
	}

	summary := downloadSummary{}
	for _, entry := range entries {
		item := downloadItem{
			ID:     entry.ID,
			Title:  entry.Title,
			System: entry.System,
		}

		media, expected, _, title, _, _, alt, err := prepareMedia(ctx, client, entry.ID, downloadOptions{Latest: true})
		if err != nil {
			item.Error = err.Error()
			summary.Failed++
			summary.Items = append(summary.Items, item)
			continue
		}
		if item.Title == "" && title != "" {
			item.Title = title
		}
		item.Format = formatLabel(alt)

		zipName := zipNameFromMedia(media)
		if zipName == "" {
			zipName = fmt.Sprintf("%d.zip", media.ID)
		}
		path := filepath.Join(outputDir, zipName)
		item.Path = path

		if _, err := os.Stat(path); err != nil {
			item.Error = err.Error()
			summary.Failed++
			summary.Items = append(summary.Items, item)
			continue
		}

		if err := verifyZip(path, expected, strict); err != nil {
			item.Error = err.Error()
			summary.Failed++
			summary.Items = append(summary.Items, item)
			continue
		}

		item.Verified = true
		summary.Verified++
		summary.Items = append(summary.Items, item)
	}

	if err := outputVerifySummary(cfg, summary); err != nil {
		printError(os.Stderr, err)
		return exitGeneric
	}
	if summary.Failed > 0 {
		return exitVerify
	}
	return exitOK
}

func selectVerifyTargets(ctx context.Context, client *vault.Client, system, query, match string, ids []string) ([]vault.ROMEntry, error) {
	if len(ids) > 0 {
		entries := make([]vault.ROMEntry, 0, len(ids))
		for _, raw := range ids {
			id, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid id %q", raw)
			}
			entries = append(entries, vault.ROMEntry{ID: id})
		}
		return entries, nil
	}

	if system == "" {
		return nil, fmt.Errorf("--system or --id is required")
	}

	if query == "" {
		return client.ListAll(ctx, system)
	}

	matcher, err := newMatcher(match, query)
	if err != nil {
		return nil, err
	}

	searchQuery := queryHint(query, match)
	if searchQuery == "" {
		searchQuery = query
	}

	entries, err := client.Search(ctx, system, searchQuery)
	if err != nil {
		return nil, err
	}

	results := make([]vault.ROMEntry, 0, len(entries))
	for _, entry := range entries {
		ok, err := matcher.Match(entry.Title)
		if err != nil {
			return nil, err
		}
		if ok {
			results = append(results, entry)
		}
	}
	return results, nil
}
