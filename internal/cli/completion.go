package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const completionUsage = `vimm completion - shell completion scripts

USAGE:
  vimm completion <bash|zsh|fish|powershell>

FLAGS:
  -h, --help   Show help
`

func runCompletion(cfg *Config, args []string) int {
	var help bool
	fs := flag.NewFlagSet("completion", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&help, "h", false, "show help")
	fs.BoolVar(&help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, completionUsage)
		return exitUsage
	}
	if help {
		printUsage(os.Stdout, completionUsage)
		return exitOK
	}
	if fs.NArg() != 1 {
		printError(os.Stderr, fmt.Errorf("shell is required"))
		printUsage(os.Stderr, completionUsage)
		return exitUsage
	}

	shell := fs.Arg(0)
	switch shell {
	case "bash", "zsh", "fish", "powershell":
		_ = cfg
		fmt.Fprintln(os.Stderr, "completion: not implemented yet")
		return exitGeneric
	default:
		printError(os.Stderr, fmt.Errorf("unsupported shell: %s", shell))
		return exitUsage
	}
}
