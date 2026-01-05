package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system slug")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.BoolVar(&all, "all", false, "download all")
	fs.Var(&stringSliceFlag{values: &ids}, "id", "vault id (repeatable)")

	fs.StringVar(&outputDir, "output-dir", ".", "output directory")
	fs.StringVar(&tmpDir, "tmp-dir", "", "temp directory")
	fs.IntVar(&concurrency, "concurrency", 4, "concurrency")
	fs.IntVar(&concurrency, "c", 4, "concurrency")
	fs.IntVar(&retries, "retries", 5, "retry count")
	fs.StringVar(&retryBackoff, "retry-backoff", "exponential", "retry backoff")
	fs.DurationVar(&timeout, "timeout", 60*time.Second, "timeout")
	fs.IntVar(&maxRPS, "max-rps", 0, "rate limit")
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

	logger := &downloadLogger{cfg: cfg}
	client := vault.NewClient(resolveBaseURL())
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
		for _, entry := range entries {
			fmt.Fprintf(os.Stdout, "%d\t%s\n", entry.ID, entry.Title)
		}
		return exitOK
	}

	limiter := newRateLimiter(maxRPS)

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
	for res := range results {
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

	fmt.Fprintf(os.Stdout, "downloaded=%d skipped=%d failed=%d verified=%d\n", downloaded, skipped, failed, verified)

	if failed > 0 {
		if verifyFailed {
			return exitVerify
		}
		return exitPartial
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
		path, skipped, verified, verifyFailed, err := downloadOne(ctx, client, httpClient, opts, limiter, logger, entry)
		res.Path = path
		res.Skipped = skipped
		res.Verified = verified
		res.VerifyFailed = verifyFailed
		res.Err = err
		results <- res
	}
}

func downloadOne(ctx context.Context, client *vault.Client, httpClient *http.Client, opts downloadOptions, limiter *rateLimiter, logger *downloadLogger, entry vault.ROMEntry) (string, bool, bool, bool, error) {
	media, expected, referer, err := prepareMedia(ctx, client, entry.ID, opts)
	if err != nil {
		return "", false, false, false, err
	}

	zipName := zipNameFromMedia(media)
	if zipName == "" {
		zipName = fmt.Sprintf("%d.zip", media.ID)
	}
	outputPath := filepath.Join(opts.OutputDir, zipName)

	if !opts.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			if !opts.Verify {
				return outputPath, true, false, false, nil
			}
			logger.verbosef("verifying existing %s\n", outputPath)
			if err := verifyZip(outputPath, expected, opts.StrictHashes); err == nil {
				return outputPath, true, true, false, nil
			}
		}
	}

	var verifyFailed bool
	attempt := func() error {
		path, err := downloadZip(ctx, httpClient, limiter, referer, media, outputPath, opts.TmpDir)
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
		return outputPath, false, false, verifyFailed, err
	}
	return outputPath, false, opts.Verify, false, nil
}

func prepareMedia(ctx context.Context, client *vault.Client, id int, opts downloadOptions) (vault.Media, vault.Hashes, string, error) {
	mediaList, err := client.ROMMedia(ctx, id)
	if err != nil {
		return vault.Media{}, vault.Hashes{}, "", err
	}
	selected, err := selectMedia(mediaList, opts.Latest, opts.Revision)
	if err != nil {
		return vault.Media{}, vault.Hashes{}, "", err
	}
	return selected, selected.ExpectedHashes(), fmt.Sprintf("%s/%d", opts.RefererBase, id), nil
}

func selectMedia(media []vault.Media, latest bool, revision string) (vault.Media, error) {
	available := make([]vault.Media, 0, len(media))
	for _, item := range media {
		if item.ZippedAvailable() {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		return vault.Media{}, fmt.Errorf("no downloadable media found")
	}

	if revision != "" {
		for _, item := range available {
			if matchesRevision(item, revision) {
				return item, nil
			}
		}
		return vault.Media{}, fmt.Errorf("revision %q not found", revision)
	}

	if !latest {
		return available[0], nil
	}

	sort.Slice(available, func(i, j int) bool {
		return mediaNewer(available[i], available[j])
	})

	return available[0], nil
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
	title := media.DecodedTitle()
	if title == "" {
		return ""
	}
	ext := filepath.Ext(title)
	base := strings.TrimSuffix(title, ext)
	return base + ".zip"
}

func downloadZip(ctx context.Context, httpClient *http.Client, limiter *rateLimiter, referer string, media vault.Media, outputPath, tmpDir string) (string, error) {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return outputPath, err
		}
	}

	url := fmt.Sprintf("https://dl2.vimm.net/?mediaId=%d", media.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return outputPath, err
	}
	req.Header.Set("User-Agent", "vimm-cli/0.1")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return outputPath, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return outputPath, fmt.Errorf("download failed: %s", resp.Status)
	}

	if header := resp.Header.Get("Content-Disposition"); header != "" {
		if _, params, err := mime.ParseMediaType(header); err == nil {
			if name := strings.TrimSpace(params["filename"]); name != "" {
				outputPath = filepath.Join(filepath.Dir(outputPath), name)
			}
		}
	}

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return outputPath, err
	}

	tmpPath := filepath.Join(tmpDir, filepath.Base(outputPath)+".part")
	out, err := os.Create(tmpPath)
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

	if err := os.Rename(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)
		return outputPath, err
	}
	return outputPath, nil
}

func verifyZip(path string, expected vault.Hashes, strict bool) error {
	lair, lairErr := vault.ReadLairHashesFromZip(path)
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

	actual, _, err := vault.ComputeROMHashesFromZip(path)
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
	ch <-chan time.Time
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
	return &rateLimiter{ch: ticker.C}
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
