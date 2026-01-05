package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"

	"vimm-download/internal/vault"
)

const downloadUsage = `vimm download - download and verify ROMs

USAGE:
  vimm download --system <slug> --query <pattern>
  vimm download --system <slug> --all
  vimm download --id <vault_id> [--id <vault_id> ...]

SELECTION FLAGS:
  --system <slug>               System vault slug (required for --query or --all)
  --query <pattern>             Search pattern (quote globs like "*mario*")
  --match <auto|glob|prefix|contains|regex>  Match mode (default: auto)
  --all                         Download all titles for system
  --id <vault_id>               Download specific ROM by vault id (repeatable)

DOWNLOAD FLAGS:
  --output-dir <path>           Output directory (default: .)
  --tmp-dir <path>              Temp directory (default: <output-dir>/.vimm.tmp)
  -c, --concurrency <n>          Concurrent downloads (default: 4)
  --retries <n>                 Retry count (default: 5)
  --retry-backoff <linear|exponential>  Retry strategy (default: exponential)
  --timeout <duration>          Per-request timeout (default: 60s)
  --max-rps <n>                 Optional requests/second limit
  --resume                      Resume partial downloads (default: true)
  --overwrite                   Always re-download
  --dry-run                     Plan only (no downloads)

VERSION FLAGS:
  --latest                      Prefer latest revision (default: true)
  --revision <value>            Override latest (e.g., rev2 or 2021-05-01)

VERIFICATION FLAGS:
  --verify                      Verify CRC/MD5/SHA1 (default: true)
  --skip-verify                 Disable verification
  --strict-hashes               Fail if any hash missing (default: true)

COUNT CHECK FLAGS (when --all):
  --count-check                 Compare against system title count (default: true)
  --strict-count                Fail if count cannot be read
  --allow-mismatch              Ignore mismatched count

SAFETY FLAGS:
  --force                       Skip confirmation for --all
  -h, --help                    Show help
`

type downloadOptions struct {
	OutputDir    string
	TmpDir       string
	Retries      int
	RetryBackoff string
	Timeout      time.Duration
	MaxRPS       int
	Resume       bool
	Overwrite    bool
	Verify       bool
	StrictHashes bool
	Latest       bool
	Revision     string
	DryRun       bool
	RefererBase  string
}

type downloadResult struct {
	Entry        vault.ROMEntry
	Path         string
	Skipped      bool
	Verified     bool
	VerifyFailed bool
	Err          error
	Title        string
	Format       string
}

func runDownload(cfg *Config, args []string) int {
	var (
		system        string
		query         string
		match         string
		all           bool
		ids           []string
		outputDir     string
		tmpDir        string
		concurrency   int
		retries       int
		retryBackoff  string
		timeout       time.Duration
		maxRPS        int
		resume        bool
		overwrite     bool
		dryRun        bool
		latest        bool
		revision      string
		verify        bool
		skipVerify    bool
		strictHashes  bool
		countCheck    bool
		strictCount   bool
		allowMismatch bool
		force         bool
		help          bool
	)

	runtimeCfg, err := loadRuntimeConfig(cfg.ConfigPath)
	if err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}
	if cfg.NoColor == false && runtimeCfg.NoColor != nil {
		cfg.NoColor = *runtimeCfg.NoColor
	}

	outputDirDefault := "."
	if runtimeCfg.OutputDir != nil && *runtimeCfg.OutputDir != "" {
		outputDirDefault = *runtimeCfg.OutputDir
	}
	concurrencyDefault := 4
	if runtimeCfg.Concurrency != nil {
		concurrencyDefault = *runtimeCfg.Concurrency
	}
	retriesDefault := 5
	if runtimeCfg.Retries != nil {
		retriesDefault = *runtimeCfg.Retries
	}
	timeoutDefault := 60 * time.Second
	if runtimeCfg.Timeout != nil {
		timeoutDefault = *runtimeCfg.Timeout
	}
	maxRPSDefault := 0
	if runtimeCfg.MaxRPS != nil {
		maxRPSDefault = *runtimeCfg.MaxRPS
	}

	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system slug")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.BoolVar(&all, "all", false, "download all")
	fs.Var(&stringSliceFlag{values: &ids}, "id", "vault id (repeatable)")

	fs.StringVar(&outputDir, "output-dir", outputDirDefault, "output directory")
	fs.StringVar(&tmpDir, "tmp-dir", "", "temp directory")
	fs.IntVar(&concurrency, "concurrency", concurrencyDefault, "concurrency")
	fs.IntVar(&concurrency, "c", concurrencyDefault, "concurrency")
	fs.IntVar(&retries, "retries", retriesDefault, "retry count")
	fs.StringVar(&retryBackoff, "retry-backoff", "exponential", "retry backoff")
	fs.DurationVar(&timeout, "timeout", timeoutDefault, "timeout")
	fs.IntVar(&maxRPS, "max-rps", maxRPSDefault, "rate limit")
	fs.BoolVar(&resume, "resume", true, "resume partial downloads")
	fs.BoolVar(&overwrite, "overwrite", false, "overwrite existing")
	fs.BoolVar(&dryRun, "dry-run", false, "dry run")

	fs.BoolVar(&latest, "latest", true, "prefer latest")
	fs.StringVar(&revision, "revision", "", "revision override")

	fs.BoolVar(&verify, "verify", true, "verify hashes")
	fs.BoolVar(&skipVerify, "skip-verify", false, "skip verification")
	fs.BoolVar(&strictHashes, "strict-hashes", true, "strict hash checks")

	fs.BoolVar(&countCheck, "count-check", true, "count check")
	fs.BoolVar(&strictCount, "strict-count", false, "strict count check")
	fs.BoolVar(&allowMismatch, "allow-mismatch", false, "allow count mismatch")

	fs.BoolVar(&force, "force", false, "skip confirmation")
	fs.BoolVar(&help, "h", false, "show help")
	fs.BoolVar(&help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, downloadUsage)
		return exitUsage
	}
	if help {
		printUsage(os.Stdout, downloadUsage)
		return exitOK
	}

	if err := validateMatch(match); err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}

	if len(ids) > 0 {
		if all || query != "" || system != "" {
			printError(os.Stderr, fmt.Errorf("--id cannot be combined with --all, --query, or --system"))
			return exitUsage
		}
	} else if all {
		if system == "" {
			printError(os.Stderr, fmt.Errorf("--system is required with --all"))
			return exitUsage
		}
		if query != "" {
			printError(os.Stderr, fmt.Errorf("--all and --query are mutually exclusive"))
			return exitUsage
		}
	} else if query != "" {
		if system == "" {
			printError(os.Stderr, fmt.Errorf("--system is required with --query"))
			return exitUsage
		}
	} else {
		printError(os.Stderr, fmt.Errorf("must specify --id, --query, or --all"))
		printUsage(os.Stderr, downloadUsage)
		return exitUsage
	}

	if skipVerify {
		verify = false
	}
	if revision != "" {
		latest = false
	}

	if concurrency < 1 {
		printError(os.Stderr, fmt.Errorf("--concurrency must be >= 1"))
		return exitUsage
	}
	if retries < 0 {
		printError(os.Stderr, fmt.Errorf("--retries must be >= 0"))
		return exitUsage
	}
	if maxRPS < 0 {
		printError(os.Stderr, fmt.Errorf("--max-rps must be >= 0"))
		return exitUsage
	}
	if retryBackoff != "linear" && retryBackoff != "exponential" {
		printError(os.Stderr, fmt.Errorf("invalid --retry-backoff value: %s", retryBackoff))
		return exitUsage
	}
	if tmpDir == "" {
		tmpDir = filepath.Join(outputDir, ".vimm.tmp")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		printError(os.Stderr, err)
		return exitGeneric
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		printError(os.Stderr, err)
		return exitGeneric
	}

	limiter := newRateLimiter(maxRPS)
	if limiter != nil {
		defer limiter.Stop()
	}

	logger := &downloadLogger{cfg: cfg}
	client := vault.NewClient(runtimeCfg.BaseURL)
	client.Limiter = limiter
	ctx := context.Background()

	entries, expectedCount, err := selectDownloadTargets(ctx, client, system, query, match, all, ids)
	if err != nil {
		printError(os.Stderr, err)
		return exitNetwork
	}
	if all && countCheck {
		if expectedCount == 0 {
			if strictCount {
				printError(os.Stderr, fmt.Errorf("system count unavailable"))
				return exitCountMismatch
			}
		} else if len(entries) != expectedCount && !allowMismatch {
			printError(os.Stderr, fmt.Errorf("count mismatch: expected %d, got %d", expectedCount, len(entries)))
			return exitCountMismatch
		}
	}

	if dryRun {
		items := make([]downloadItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, downloadItem{
				ID:     entry.ID,
				Title:  entry.Title,
				System: entry.System,
			})
		}
		if cfg.JSON {
			if err := outputDownloadSummary(cfg, downloadSummary{Items: items}); err != nil {
				printError(os.Stderr, err)
				return exitGeneric
			}
			return exitOK
		}
		for _, entry := range entries {
			fmt.Fprintf(os.Stdout, "%d\t%s\n", entry.ID, entry.Title)
		}
		return exitOK
	}

	if all && !force {
		if err := confirmAll(cfg, system, expectedCount); err != nil {
			if errors.Is(err, errAborted) {
				return exitOK
			}
			printError(os.Stderr, err)
			return exitUsage
		}
	}

	opts := downloadOptions{
		OutputDir:    outputDir,
		TmpDir:       tmpDir,
		Retries:      retries,
		RetryBackoff: retryBackoff,
		Timeout:      timeout,
		MaxRPS:       maxRPS,
		Resume:       resume,
		Overwrite:    overwrite,
		Verify:       verify,
		StrictHashes: strictHashes,
		Latest:       latest,
		Revision:     revision,
		DryRun:       dryRun,
		RefererBase:  client.BaseURL,
	}

	jobs := make(chan vault.ROMEntry)
	results := make(chan downloadResult)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerDownload(ctx, client, opts, limiter, logger, jobs, results)
		}()
	}

	go func() {
		for _, entry := range entries {
			jobs <- entry
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var downloaded, skipped, failed, verified int
	verifyFailed := false
	var items []downloadItem
	for res := range results {
		items = append(items, downloadItem{
			ID:       res.Entry.ID,
			Title:    res.Title,
			System:   res.Entry.System,
			Path:     res.Path,
			Skipped:  res.Skipped,
			Verified: res.Verified,
			Error:    errString(res.Err),
			Format:   res.Format,
		})
		if res.Err != nil {
			failed++
			if res.VerifyFailed {
				verifyFailed = true
			}
			logger.printf("failed %d: %v\n", res.Entry.ID, res.Err)
			continue
		}
		if res.Skipped {
			skipped++
		} else {
			downloaded++
		}
		if res.Verified {
			verified++
		}
	}

	summary := downloadSummary{
		Downloaded: downloaded,
		Skipped:    skipped,
		Failed:     failed,
		Verified:   verified,
		Items:      items,
	}
	if err := outputDownloadSummary(cfg, summary); err != nil {
		printError(os.Stderr, err)
		return exitGeneric
	}

	if failed > 0 {
		if verifyFailed {
			return exitVerify
		}
		return exitPartial
	}
	if all && countCheck && expectedCount > 0 && !allowMismatch {
		completed := verified
		if !verify {
			completed = downloaded + skipped
		}
		if completed != expectedCount {
			printError(os.Stderr, fmt.Errorf("count mismatch: expected %d, got %d", expectedCount, completed))
			return exitCountMismatch
		}
	}
	return exitOK
}

type downloadLogger struct {
	cfg *Config
	mu  sync.Mutex
}

func (l *downloadLogger) printf(format string, args ...any) {
	if l == nil || l.cfg == nil || l.cfg.Quiet {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(os.Stderr, format, args...)
}

func (l *downloadLogger) verbosef(format string, args ...any) {
	if l == nil || l.cfg == nil || l.cfg.Verbose == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(os.Stderr, format, args...)
}

func selectDownloadTargets(ctx context.Context, client *vault.Client, system, query, match string, all bool, ids []string) ([]vault.ROMEntry, int, error) {
	if len(ids) > 0 {
		entries := make([]vault.ROMEntry, 0, len(ids))
		for _, raw := range ids {
			id, err := strconv.Atoi(raw)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid id %q", raw)
			}
			entries = append(entries, vault.ROMEntry{ID: id})
		}
		return entries, 0, nil
	}

	if all {
		entries, err := client.ListAll(ctx, system)
		if err != nil {
			return nil, 0, err
		}
		count, _ := client.SystemCount(ctx, system)
		return entries, count, nil
	}

	matcher, err := newMatcher(match, query)
	if err != nil {
		return nil, 0, err
	}

	systems, err := resolveSystemsForSearch(ctx, client, system, "")
	if err != nil {
		return nil, 0, err
	}

	searchQuery := queryHint(query, match)
	if searchQuery == "" {
		searchQuery = query
	}

	var results []vault.ROMEntry
	for _, sys := range systems {
		entries, err := client.Search(ctx, sys.Slug, searchQuery)
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range entries {
			ok, err := matcher.Match(entry.Title)
			if err != nil {
				return nil, 0, err
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

	return results, 0, nil
}

func workerDownload(ctx context.Context, client *vault.Client, opts downloadOptions, limiter *rateLimiter, logger *downloadLogger, jobs <-chan vault.ROMEntry, results chan<- downloadResult) {
	httpClient := &http.Client{Timeout: opts.Timeout}
	for entry := range jobs {
		res := downloadResult{Entry: entry}
		path, title, format, skipped, verified, verifyFailed, err := downloadOne(ctx, client, httpClient, opts, limiter, logger, entry)
		res.Path = path
		res.Skipped = skipped
		res.Verified = verified
		res.VerifyFailed = verifyFailed
		res.Err = err
		res.Format = format
		if title != "" {
			res.Title = title
		} else if entry.Title != "" {
			res.Title = entry.Title
		}
		results <- res
	}
}

func downloadOne(ctx context.Context, client *vault.Client, httpClient *http.Client, opts downloadOptions, limiter *rateLimiter, logger *downloadLogger, entry vault.ROMEntry) (string, string, string, bool, bool, bool, error) {
	media, expected, referer, title, downloadBase, downloadMethod, alt, err := prepareMedia(ctx, client, entry.ID, opts)
	if err != nil {
		return "", "", "", false, false, false, err
	}
	if entry.Title == "" && title != "" {
		entry.Title = title
	}
	format := formatLabel(alt)

	zipName := zipNameFromMedia(media)
	if zipName == "" {
		zipName = sanitizeFilename(fmt.Sprintf("%d.zip", media.ID))
	}
	outputPath := filepath.Join(opts.OutputDir, zipName)

	if !opts.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			if !opts.Verify {
				return outputPath, entry.Title, format, true, false, false, nil
			}
			logger.verbosef("verifying existing %s\n", outputPath)
			if err := verifyZip(outputPath, expected, opts.StrictHashes); err == nil {
				return outputPath, entry.Title, format, true, true, false, nil
			}
		}
	}

	var verifyFailed bool
	attempt := func() error {
		path, err := downloadZip(ctx, httpClient, limiter, referer, downloadBase, downloadMethod, media, outputPath, opts.TmpDir, opts.Resume, alt, logger)
		if err != nil {
			return err
		}
		outputPath = path
		if opts.Verify {
			if err := verifyZip(outputPath, expected, opts.StrictHashes); err != nil {
				verifyFailed = true
				_ = os.Remove(outputPath)
				return err
			}
			return nil
		}
		return nil
	}

	err = withRetries(attempt, opts.Retries, opts.RetryBackoff)
	if err != nil {
		return outputPath, entry.Title, format, false, false, verifyFailed, err
	}
	return outputPath, entry.Title, format, false, opts.Verify, false, nil
}

func prepareMedia(ctx context.Context, client *vault.Client, id int, opts downloadOptions) (vault.Media, vault.Hashes, string, string, string, string, int, error) {
	page, err := client.ROMPage(ctx, id)
	if err != nil {
		return vault.Media{}, vault.Hashes{}, "", "", "", "", 0, err
	}
	selected, alt, err := selectMedia(page.Media, opts.Latest, opts.Revision, page.DownloadAlt)
	if err != nil {
		return vault.Media{}, vault.Hashes{}, "", "", "", "", 0, err
	}
	return selected, selected.ExpectedHashes(), fmt.Sprintf("%s/%d", opts.RefererBase, id), selected.DecodedTitle(), page.DownloadBase, page.DownloadMethod, alt, nil
}

func selectMedia(media []vault.Media, latest bool, revision string, alt int) (vault.Media, int, error) {
	alt = normalizeAlt(alt)
	selectWithAlt := func(targetAlt int) (vault.Media, bool, error) {
		available := make([]vault.Media, 0, len(media))
		for _, item := range media {
			if item.DownloadAvailableAlt(targetAlt) {
				available = append(available, item)
			}
		}
		if len(available) == 0 {
			return vault.Media{}, false, fmt.Errorf("no downloadable media found for alt=%d", targetAlt)
		}

		if revision != "" {
			for _, item := range available {
				if matchesRevision(item, revision) {
					return item, true, nil
				}
			}
			return vault.Media{}, false, fmt.Errorf("revision %q not found", revision)
		}

		if !latest {
			return available[0], true, nil
		}

		sort.Slice(available, func(i, j int) bool {
			return mediaNewer(available[i], available[j])
		})

		return available[0], true, nil
	}

	selected, ok, err := selectWithAlt(alt)
	if err == nil {
		return selected, alt, nil
	}
	if alt != 0 {
		fallback, okFallback, errFallback := selectWithAlt(0)
		if errFallback == nil && okFallback {
			return fallback, 0, nil
		}
		if ok || okFallback {
			return vault.Media{}, 0, errFallback
		}
	}
	return vault.Media{}, 0, err
}

func matchesRevision(media vault.Media, revision string) bool {
	if strings.EqualFold(media.VersionString, revision) || strings.EqualFold(media.Version, revision) {
		return true
	}
	if media.GoodDate != nil && media.GoodDate.Date != "" {
		date := strings.Split(media.GoodDate.Date, " ")[0]
		if date == revision {
			return true
		}
	}
	return false
}

func mediaNewer(a, b vault.Media) bool {
	va, oka := parseVersion(firstNonEmptyLocal(a.VersionString, a.Version))
	vb, okb := parseVersion(firstNonEmptyLocal(b.VersionString, b.Version))
	if oka && okb {
		cmp := compareVersion(va, vb)
		if cmp != 0 {
			return cmp > 0
		}
	}
	ta, oka := parseMediaDate(a)
	tb, okb := parseMediaDate(b)
	if oka && okb {
		if !ta.Equal(tb) {
			return ta.After(tb)
		}
	}
	return a.ID > b.ID
}

type version struct {
	parts []int
}

func parseVersion(value string) (version, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return version{}, false
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if len(parts) == 0 {
		return version{}, false
	}
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return version{}, false
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return version{}, false
	}
	return version{parts: out}, true
}

func firstNonEmptyLocal(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func compareVersion(a, b version) int {
	maxLen := int(math.Max(float64(len(a.parts)), float64(len(b.parts))))
	for i := 0; i < maxLen; i++ {
		av := 0
		if i < len(a.parts) {
			av = a.parts[i]
		}
		bv := 0
		if i < len(b.parts) {
			bv = b.parts[i]
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}

func normalizeAlt(alt int) int {
	switch alt {
	case 1, 2:
		return alt
	default:
		return 0
	}
}

func formatLabel(alt int) string {
	switch normalizeAlt(alt) {
	case 1:
		return "alt"
	case 2:
		return "alt2"
	default:
		return "standard"
	}
}

func parseMediaDate(media vault.Media) (time.Time, bool) {
	if media.GoodDate == nil || media.GoodDate.Date == "" {
		return time.Time{}, false
	}
	date := strings.TrimSpace(media.GoodDate.Date)
	if len(date) >= 19 {
		date = date[:19]
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", date)
	if err == nil {
		return parsed, true
	}
	parsed, err = time.Parse("2006-01-02", date)
	if err == nil {
		return parsed, true
	}
	return time.Time{}, false
}

func zipNameFromMedia(media vault.Media) string {
	title := sanitizeFilename(media.DecodedTitle())
	if title == "" {
		return ""
	}
	ext := filepath.Ext(title)
	base := strings.TrimSuffix(title, ext)
	name := base + ".zip"
	return sanitizeFilename(name)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	return name
}

func downloadZip(ctx context.Context, httpClient *http.Client, limiter *rateLimiter, referer, downloadBase, downloadMethod string, media vault.Media, outputPath, tmpDir string, resume bool, alt int, logger *downloadLogger) (string, error) {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return outputPath, err
		}
	}

	params := url.Values{}
	params.Set("mediaId", strconv.Itoa(media.ID))
	if alt > 0 {
		params.Set("alt", strconv.Itoa(alt))
	}
	base := strings.TrimRight(downloadBase, "/")
	if base == "" {
		base = "https://dl2.vimm.net"
	}
	tmpPath := filepath.Join(tmpDir, filepath.Base(outputPath)+".part")
	var resumeFrom int64
	if resume {
		if info, err := os.Stat(tmpPath); err == nil {
			resumeFrom = info.Size()
		}
	}
	method := strings.ToUpper(strings.TrimSpace(downloadMethod))
	if method == "" {
		method = http.MethodGet
	}
	methods := []string{method}
	if method == http.MethodPost {
		methods = []string{http.MethodGet, http.MethodPost}
	}

	var resp *http.Response
	var err error
	for i, m := range methods {
		resumeSupported := m == http.MethodGet
		if resumeFrom > 0 && !resumeSupported {
			if logger != nil {
				logger.verbosef("resume not supported for %s; restarting download\n", m)
			}
			_ = os.Remove(tmpPath)
			resumeFrom = 0
		}

		var req *http.Request
		if m == http.MethodPost {
			requestURL := base + "/"
			body := strings.NewReader(params.Encode())
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, requestURL, body)
			if err != nil {
				return outputPath, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			requestURL := base + "/?" + params.Encode()
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
			if err != nil {
				return outputPath, err
			}
		}
		req.Header.Set("User-Agent", vault.DefaultUserAgent)
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		if resumeFrom > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
		}

		resp, err = httpClient.Do(req)
		if err != nil {
			return outputPath, err
		}
		if resumeFrom > 0 && resp.StatusCode == http.StatusOK {
			// Server ignored range, restart from scratch.
			_ = os.Remove(tmpPath)
			resumeFrom = 0
		}
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			_ = os.Remove(tmpPath)
			resp.Body.Close()
			return outputPath, fmt.Errorf("resume failed: range not satisfiable")
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		resp.Body.Close()
		if i == len(methods)-1 {
			return outputPath, fmt.Errorf("download failed: %s", resp.Status)
		}
	}
	if resp == nil {
		return outputPath, fmt.Errorf("download failed: no response")
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return outputPath, err
	}

	var out *os.File
	if resumeFrom > 0 && resp.StatusCode == http.StatusPartialContent {
		out, err = os.OpenFile(tmpPath, os.O_APPEND|os.O_WRONLY, 0o644)
	} else {
		out, err = os.Create(tmpPath)
	}
	if err != nil {
		return outputPath, err
	}

	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return outputPath, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return outputPath, closeErr
	}

	if err := moveFile(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)
		return outputPath, err
	}
	return outputPath, nil
}

var errAborted = errors.New("aborted")

func confirmAll(cfg *Config, system string, count int) error {
	if cfg == nil {
		return fmt.Errorf("confirmation required for --all")
	}
	if cfg.NoInput || !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("confirmation required for --all; use --force to proceed")
	}
	label := system
	if count > 0 {
		label = fmt.Sprintf("%s (%d titles)", system, count)
	}
	fmt.Fprintf(os.Stderr, "Download all titles for %s? [y/N]: ", label)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		return nil
	}
	fmt.Fprintln(os.Stderr, "aborted")
	return errAborted
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if linkErr, ok := err.(*os.LinkError); !ok || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

func verifyZip(path string, expected vault.Hashes, strict bool) error {
	lair, lairErr := vault.ReadLairHashesFromArchive(path)
	if lairErr != nil && strict {
		return lairErr
	}
	if !expected.Empty() && !lair.Empty() && !hashesMatch(expected, lair) && strict {
		return fmt.Errorf("hash mismatch between page and Vimm's Lair.txt")
	}

	resolved := expected
	if resolved.Empty() {
		resolved = lair
	}
	if strict && (resolved.CRC == "" || resolved.MD5 == "" || resolved.SHA1 == "") {
		return fmt.Errorf("missing hashes for verification")
	}

	actual, _, err := vault.ComputeROMHashesFromArchive(path)
	if err != nil {
		return err
	}

	if err := compareHashes(resolved, actual); err != nil {
		return err
	}
	return nil
}

func hashesMatch(a, b vault.Hashes) bool {
	return strings.EqualFold(a.CRC, b.CRC) && strings.EqualFold(a.MD5, b.MD5) && strings.EqualFold(a.SHA1, b.SHA1)
}

func compareHashes(expected, actual vault.Hashes) error {
	if expected.CRC != "" && !strings.EqualFold(expected.CRC, actual.CRC) {
		return fmt.Errorf("CRC mismatch")
	}
	if expected.MD5 != "" && !strings.EqualFold(expected.MD5, actual.MD5) {
		return fmt.Errorf("MD5 mismatch")
	}
	if expected.SHA1 != "" && !strings.EqualFold(expected.SHA1, actual.SHA1) {
		return fmt.Errorf("SHA1 mismatch")
	}
	return nil
}

func withRetries(fn func() error, retries int, mode string) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if attempt == retries {
			break
		}
		time.Sleep(retryDelay(attempt, mode))
	}
	return err
}

func retryDelay(attempt int, mode string) time.Duration {
	base := 1 * time.Second
	jitter := time.Duration(rand.Intn(250)) * time.Millisecond
	if mode == "linear" {
		return time.Duration(attempt+1)*base + jitter
	}
	return time.Duration(1<<attempt)*base + jitter
}

type rateLimiter struct {
	ticker *time.Ticker
	ch     <-chan time.Time
}

func newRateLimiter(maxRPS int) *rateLimiter {
	if maxRPS <= 0 {
		return nil
	}
	interval := time.Second / time.Duration(maxRPS)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	return &rateLimiter{ticker: ticker, ch: ticker.C}
}

func (r *rateLimiter) Wait(ctx context.Context) error {
	if r == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ch:
		return nil
	}
}

func (r *rateLimiter) Stop() {
	if r == nil || r.ticker == nil {
		return
	}
	r.ticker.Stop()
}
