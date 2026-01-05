package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func Run(args []string) int {
	cfg := &Config{}
	var (
		showHelp    bool
		showVersion bool
	)

	fs := flag.NewFlagSet("vimm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.Var(&countFlag{value: &cfg.Verbose}, "v", "increase verbosity")
	fs.Var(&countFlag{value: &cfg.Verbose}, "verbose", "increase verbosity")
	fs.BoolVar(&cfg.Quiet, "q", false, "errors only")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "errors only")
	fs.BoolVar(&cfg.JSON, "json", false, "machine-readable output")
	fs.BoolVar(&cfg.Plain, "plain", false, "plain output")
	fs.BoolVar(&cfg.NoColor, "no-color", false, "disable color")
	fs.BoolVar(&cfg.NoInput, "no-input", false, "disable prompts")
	fs.StringVar(&cfg.ConfigPath, "config", "", "config path")
	fs.BoolVar(&showVersion, "version", false, "print version")
	fs.BoolVar(&showHelp, "h", false, "show help")
	fs.BoolVar(&showHelp, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		printError(os.Stderr, err)
		printUsage(os.Stderr, rootUsage)
		return exitUsage
	}

	if showHelp {
		printUsage(os.Stdout, rootUsage)
		return exitOK
	}

	if showVersion {
		fmt.Fprintln(os.Stdout, Version)
		return exitOK
	}

	if cfg.JSON {
		cfg.Plain = true
	}
	if cfg.Quiet && cfg.Verbose > 0 {
		printError(os.Stderr, fmt.Errorf("cannot use --quiet and --verbose together"))
		return exitUsage
	}

	if fs.NArg() == 0 {
		printUsage(os.Stdout, rootUsage)
		return exitUsage
	}

	subcmd := fs.Arg(0)
	subargs := fs.Args()[1:]

	switch subcmd {
	case "systems":
		return runSystems(cfg, subargs)
	case "search":
		return runSearch(cfg, subargs)
	case "download":
		return runDownload(cfg, subargs)
	case "verify":
		return runVerify(cfg, subargs)
	case "completion":
		return runCompletion(cfg, subargs)
	case "help":
		printUsage(os.Stdout, rootUsage)
		return exitOK
	default:
		printError(os.Stderr, fmt.Errorf("unknown subcommand: %s", subcmd))
		printUsage(os.Stderr, rootUsage)
		return exitUsage
	}
}

func printError(w io.Writer, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(w, "vimm: error: %v\n", err)
}
