package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const systemsUsage = `vimm systems - list available systems

USAGE:
  vimm systems [--class <console|handheld>]

FLAGS:
  --class <console|handheld>   Filter by system class
  -h, --help                   Show help
`

func runSystems(cfg *Config, args []string) int {
	var (
		class string
		help  bool
	)
	fs := flag.NewFlagSet("systems", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&class, "class", "", "system class")
	fs.BoolVar(&help, "h", false, "show help")
	fs.BoolVar(&help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, systemsUsage)
		return exitUsage
	}
	if help {
		printUsage(os.Stdout, systemsUsage)
		return exitOK
	}
	if err := validateClass(class); err != nil {
		printError(os.Stderr, err)
		return exitUsage
	}

	_ = cfg
	fmt.Fprintln(os.Stderr, "systems: not implemented yet")
	return exitGeneric
}
