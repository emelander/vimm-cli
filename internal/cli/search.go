package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const searchUsage = `vimm search - search ROMs by name pattern

USAGE:
  vimm search --system <slug> --query <pattern>
  vimm search --class <console|handheld> --query <pattern>

FLAGS:
  --system <slug>               System vault slug (e.g., N64)
  --class <console|handheld>    Search across system class
  --query <pattern>             Search pattern (quote globs like "*mario*")
  --match <auto|glob|prefix|contains|regex>  Match mode (default: auto)
  --limit <n>                   Max results (default: 100)
  --offset <n>                  Skip first N results (default: 0)
  -h, --help                    Show help
`

func runSearch(cfg *Config, args []string) int {
	var (
		system string
		class  string
		query  string
		match  string
		limit  int
		offset int
		help   bool
	)
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&system, "system", "", "system slug")
	fs.StringVar(&class, "class", "", "system class")
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&match, "match", "auto", "match mode")
	fs.IntVar(&limit, "limit", 100, "limit results")
	fs.IntVar(&offset, "offset", 0, "offset results")
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

	_ = cfg
	fmt.Fprintln(os.Stderr, "search: not implemented yet")
	return exitGeneric
}
