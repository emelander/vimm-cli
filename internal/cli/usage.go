package cli

import (
	"fmt"
	"io"
)

const rootUsage = `vimm - download and verify ROMs you own from Vimm's Lair

USAGE:
  vimm [global flags] <subcommand> [flags]

SUBCOMMANDS:
  systems     List available systems and codes
  search      Search ROMs by name pattern
  download    Download and verify ROMs
  verify      Re-verify local ROMs
  completion  Shell completion scripts

GLOBAL FLAGS:
  -h, --help          Show help
  --version           Print version
  -q, --quiet         Errors only
  -v, --verbose       Increase verbosity (repeatable)
  --json              Machine-readable output (implies --plain)
  --plain             Stable line-based output
  --no-color          Disable color
  --no-input          Disable prompts
  --config <path>     Config file path
`

func printUsage(w io.Writer, text string) {
	fmt.Fprint(w, text)
}
