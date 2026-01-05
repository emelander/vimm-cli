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
		script, err := completionScript(shell)
		if err != nil {
			printError(os.Stderr, err)
			return exitGeneric
		}
		fmt.Fprint(os.Stdout, script)
		return exitOK
	default:
		printError(os.Stderr, fmt.Errorf("unsupported shell: %s", shell))
		return exitUsage
	}
}

func completionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashCompletion, nil
	case "zsh":
		return zshCompletion, nil
	case "fish":
		return fishCompletion, nil
	case "powershell":
		return powershellCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

const bashCompletion = `# bash completion for vimm
_vimm() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "systems search download verify completion help" -- "$cur") )
    return 0
  fi

  local cmd="${COMP_WORDS[1]}"
  case "$cmd" in
    systems)
      COMPREPLY=( $(compgen -W "--class --no-header --help -h --json --plain --quiet -q --verbose -v --no-color --no-input --config --version" -- "$cur") )
      ;;
    search)
      COMPREPLY=( $(compgen -W "--system --class --query --match --limit --offset --region --include-tags --no-header --help -h --json --plain --quiet -q --verbose -v --no-color --no-input --config --version" -- "$cur") )
      ;;
    download)
      COMPREPLY=( $(compgen -W "--system --query --match --all --id --output-dir --tmp-dir --concurrency -c --retries --retry-backoff --timeout --max-rps --resume --overwrite --dry-run --latest --revision --region --include-tags --verify --skip-verify --strict-hashes --count-check --strict-count --allow-mismatch --force --help -h --json --plain --quiet -q --verbose -v --no-color --no-input --config --version" -- "$cur") )
      ;;
    verify)
      COMPREPLY=( $(compgen -W "--system --id --query --match --output-dir --strict-hashes --help -h --json --plain --quiet -q --verbose -v --no-color --no-input --config --version" -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W "bash zsh fish powershell --help -h" -- "$cur") )
      ;;
    *)
      COMPREPLY=()
      ;;
  esac
  return 0
}
complete -F _vimm vimm
`

const zshCompletion = `# zsh completion for vimm
_vimm() {
  local -a commands
  commands=(
    "systems:List available systems"
    "search:Search ROMs by name pattern"
    "download:Download and verify ROMs"
    "verify:Verify local ROMs"
    "completion:Shell completion scripts"
    "help:Show help"
  )

  if (( CURRENT == 2 )); then
    _describe -t commands "vimm command" commands
    return
  fi

  local cmd=${words[2]}
  case $cmd in
    systems)
      _arguments "--class[Filter by system class]" "--no-header[Hide column headers]" "--help[Show help]" ;;
    search)
      _arguments "--system[System code]" "--class[System class]" "--query[Search pattern]" "--match[Match mode]" "--limit[Limit]" "--offset[Offset]" "--region[Preferred region]" "--include-tags[Include tags]" "--no-header[Hide column headers]" "--help[Show help]" ;;
    download)
      _arguments "--system[System code]" "--query[Search pattern]" "--match[Match mode]" "--all[Download all]" "--id[Vault id]" "--output-dir[Output directory]" "--tmp-dir[Temp directory]" "-c[Concurrency]" "--concurrency[Concurrency]" "--retries[Retries]" "--retry-backoff[Retry strategy]" "--timeout[Timeout]" "--max-rps[Rate limit]" "--resume[Resume]" "--overwrite[Overwrite]" "--dry-run[Dry run]" "--latest[Prefer latest]" "--revision[Revision override]" "--region[Preferred region]" "--include-tags[Include tags]" "--verify[Verify hashes]" "--skip-verify[Skip verification]" "--strict-hashes[Strict hashes]" "--count-check[Count check]" "--strict-count[Strict count]" "--allow-mismatch[Allow mismatch]" "--force[Skip confirmation]" "--help[Show help]" ;;
    verify)
      _arguments "--system[System code]" "--id[Vault id]" "--query[Search pattern]" "--match[Match mode]" "--output-dir[Output directory]" "--strict-hashes[Strict hashes]" "--help[Show help]" ;;
    completion)
      _arguments "1: :((bash zsh fish powershell))" "--help[Show help]" ;;
  esac
}
compdef _vimm vimm
`

const fishCompletion = `# fish completion for vimm
complete -c vimm -f -n '__fish_use_subcommand' -a 'systems search download verify completion help'
complete -c vimm -n '__fish_seen_subcommand_from systems' -l class -d 'Filter by system class'
complete -c vimm -n '__fish_seen_subcommand_from systems' -l no-header -d 'Hide column headers'
complete -c vimm -n '__fish_seen_subcommand_from search' -l system -d 'System code'
complete -c vimm -n '__fish_seen_subcommand_from search' -l class -d 'System class'
complete -c vimm -n '__fish_seen_subcommand_from search' -l query -d 'Search pattern'
complete -c vimm -n '__fish_seen_subcommand_from search' -l match -d 'Match mode'
complete -c vimm -n '__fish_seen_subcommand_from search' -l limit -d 'Limit results'
complete -c vimm -n '__fish_seen_subcommand_from search' -l offset -d 'Offset results'
complete -c vimm -n '__fish_seen_subcommand_from search' -l region -d 'Preferred region'
complete -c vimm -n '__fish_seen_subcommand_from search' -l include-tags -d 'Include tags'
complete -c vimm -n '__fish_seen_subcommand_from search' -l no-header -d 'Hide column headers'
complete -c vimm -n '__fish_seen_subcommand_from download' -l system -d 'System code'
complete -c vimm -n '__fish_seen_subcommand_from download' -l query -d 'Search pattern'
complete -c vimm -n '__fish_seen_subcommand_from download' -l match -d 'Match mode'
complete -c vimm -n '__fish_seen_subcommand_from download' -l all -d 'Download all'
complete -c vimm -n '__fish_seen_subcommand_from download' -l id -d 'Vault id'
complete -c vimm -n '__fish_seen_subcommand_from download' -l output-dir -d 'Output directory'
complete -c vimm -n '__fish_seen_subcommand_from download' -l tmp-dir -d 'Temp directory'
complete -c vimm -n '__fish_seen_subcommand_from download' -l concurrency -s c -d 'Concurrency'
complete -c vimm -n '__fish_seen_subcommand_from download' -l retries -d 'Retries'
complete -c vimm -n '__fish_seen_subcommand_from download' -l retry-backoff -d 'Retry backoff'
complete -c vimm -n '__fish_seen_subcommand_from download' -l timeout -d 'Timeout'
complete -c vimm -n '__fish_seen_subcommand_from download' -l max-rps -d 'Rate limit'
complete -c vimm -n '__fish_seen_subcommand_from download' -l resume -d 'Resume partial downloads'
complete -c vimm -n '__fish_seen_subcommand_from download' -l overwrite -d 'Overwrite existing'
complete -c vimm -n '__fish_seen_subcommand_from download' -l dry-run -d 'Dry run'
complete -c vimm -n '__fish_seen_subcommand_from download' -l latest -d 'Prefer latest'
complete -c vimm -n '__fish_seen_subcommand_from download' -l revision -d 'Revision override'
complete -c vimm -n '__fish_seen_subcommand_from download' -l region -d 'Preferred region'
complete -c vimm -n '__fish_seen_subcommand_from download' -l include-tags -d 'Include tags'
complete -c vimm -n '__fish_seen_subcommand_from download' -l verify -d 'Verify hashes'
complete -c vimm -n '__fish_seen_subcommand_from download' -l skip-verify -d 'Skip verification'
complete -c vimm -n '__fish_seen_subcommand_from download' -l strict-hashes -d 'Strict hashes'
complete -c vimm -n '__fish_seen_subcommand_from download' -l count-check -d 'Count check'
complete -c vimm -n '__fish_seen_subcommand_from download' -l strict-count -d 'Strict count'
complete -c vimm -n '__fish_seen_subcommand_from download' -l allow-mismatch -d 'Allow mismatch'
complete -c vimm -n '__fish_seen_subcommand_from download' -l force -d 'Skip confirmation'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l system -d 'System code'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l id -d 'Vault id'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l query -d 'Search pattern'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l match -d 'Match mode'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l output-dir -d 'Output directory'
complete -c vimm -n '__fish_seen_subcommand_from verify' -l strict-hashes -d 'Strict hashes'
complete -c vimm -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish powershell'
`

const powershellCompletion = `# powershell completion for vimm
using namespace System.Management.Automation

Register-ArgumentCompleter -CommandName vimm -ScriptBlock {
  param($commandName, $wordToComplete, $cursorPosition)

  $commands = @('systems','search','download','verify','completion','help')
  $flags = @(
    '--help','-h','--version','--json','--plain','--quiet','-q','--verbose','-v','--no-color','--no-input','--config'
  )

  if ($wordToComplete -like '-*') {
    $flags | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {
      [CompletionResult]::new($_, $_, 'ParameterName', $_)
    }
    return
  }

  $commands | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {
    [CompletionResult]::new($_, $_, 'Command', $_)
  }
}
`
