package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
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

func runDownload(cfg *Config, args []string) int {
	var (
		system       string
		query        string
		match        string
		all          bool
		ids          []string
		outputDir    string
		tmpDir       string
		concurrency  int
		retries      int
		retryBackoff string
		timeout      time.Duration
		maxRPS       int
		resume       bool
		overwrite    bool
		dryRun       bool
		latest       bool
		revision     string
		verify       bool
		skipVerify   bool
		strictHashes bool
		countCheck   bool
		strictCount  bool
		allowMismatch bool
		force        bool
		help         bool
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
		tmpDir = outputDir + string(os.PathSeparator) + ".vimm.tmp"
	}

	_ = cfg
	_ = verify
	_ = strictHashes
	_ = countCheck
	_ = strictCount
	_ = allowMismatch
	_ = force
	_ = latest
	_ = revision
	_ = resume
	_ = overwrite
	_ = dryRun
	_ = timeout
	_ = tmpDir

	fmt.Fprintln(os.Stderr, "download: not implemented yet")
	return exitGeneric
}
