package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"vimm-cli/internal/vault"
)

const systemsUsage = `vimm systems - list available systems and codes

USAGE:
  vimm systems [--class <console|handheld>]

FLAGS:
  --class <console|handheld>   Filter by system class
  --no-header                  Hide column headers
  -h, --help                   Show help
`

func runSystems(cfg *Config, args []string) int {
	var (
		class string
		noHdr bool
		help  bool
	)
	fs := flag.NewFlagSet("systems", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&class, "class", "", "system class")
	fs.BoolVar(&noHdr, "no-header", false, "hide column headers")
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
	systems, err := client.Systems(ctx)
	if err != nil {
		printError(os.Stderr, err)
		return exitNetwork
	}

	if class != "" {
		filtered := make([]vault.System, 0, len(systems))
		for _, sys := range systems {
			if sys.Class == class {
				filtered = append(filtered, sys)
			}
		}
		systems = filtered
	}

	for i := range systems {
		count, err := client.SystemCount(ctx, systems[i].Slug)
		if err == nil {
			systems[i].Titles = count
		}
	}

	sort.Slice(systems, func(i, j int) bool {
		return systems[i].Name < systems[j].Name
	})

	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(systems); err != nil {
			printError(os.Stderr, err)
			return exitGeneric
		}
		return exitOK
	}

	if cfg.Plain {
		for _, sys := range systems {
			fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%d\n", sys.Slug, sys.Name, sys.Class, sys.Titles)
		}
		return exitOK
	}

	codeHeader := "CODE"
	nameHeader := "NAME"
	classHeader := "CLASS"
	titlesHeader := "TITLES"
	codeWidth := len(codeHeader)
	nameWidth := len(nameHeader)
	classWidth := len(classHeader)
	titlesWidth := len(titlesHeader)
	for _, sys := range systems {
		if len(sys.Slug) > codeWidth {
			codeWidth = len(sys.Slug)
		}
		if len(sys.Name) > nameWidth {
			nameWidth = len(sys.Name)
		}
		if len(sys.Class) > classWidth {
			classWidth = len(sys.Class)
		}
		titlesLen := len(fmt.Sprintf("%d", sys.Titles))
		if titlesLen > titlesWidth {
			titlesWidth = titlesLen
		}
	}

	if !noHdr {
		fmt.Fprintf(os.Stdout, "%-*s  %-*s  %-*s  %*s\n", codeWidth, codeHeader, nameWidth, nameHeader, classWidth, classHeader, titlesWidth, titlesHeader)
	}
	for _, sys := range systems {
		fmt.Fprintf(os.Stdout, "%-*s  %-*s  %-*s  %*d\n", codeWidth, sys.Slug, nameWidth, sys.Name, classWidth, sys.Class, titlesWidth, sys.Titles)
	}
	return exitOK
}
