package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const verifyUsage = `vimm verify - verify local ROMs

USAGE:
  vimm verify --system <slug> [--query <pattern>] --output-dir <path>
  vimm verify --id <vault_id> [--id <vault_id> ...] --output-dir <path>

FLAGS:
  --system <slug>               System vault slug
  --id <vault_id>               Verify specific ROMs by id (repeatable)
  --query <pattern>             Search pattern (quote globs like "*mario*")
  --match <auto|glob|prefix|contains|regex>  Match mode (default: auto)
  --output-dir <path>           Output directory (default: .)
  -h, --help                    Show help
`

func runVerify(cfg *Config, args []string) int {
	var (
		system    string
		query     string
		match     string
		outputDir string
		ids       []string
		help      bool
	)

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system slug")
	fs.Var(&stringSliceFlag{values: &ids}, "id", "vault id (repeatable)")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.StringVar(&outputDir, "output-dir", ".", "output directory")
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

	_ = cfg
	fmt.Fprintln(os.Stderr, "verify: not implemented yet")
	return exitGeneric
}
