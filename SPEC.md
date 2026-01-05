# vimm CLI Specification

## Overview
`vimm` is a Go-based CLI for asynchronously downloading ROMs you own from Vimm’s Lair (`https://vimm.net/vault`).
It supports search-by-name, bulk downloads per system, robust retry logic, and post-download verification using CRC/MD5/SHA1
values published in `Vimm’s Lair.txt` on each ROM page.

## Goals
- Download ROMs concurrently with retry/backoff and timeouts.
- Select ROMs by search pattern, explicit vault ID, or `--all` for a system.
- Verify downloads using CRC, MD5, and SHA1 from the ROM page (`Vimm’s Lair.txt`).
- Prefer the latest available version of a ROM by a deterministic rule.
- Count-check `--all` downloads against the system’s reported title count.
- Human-first UX with script-friendly `--plain`/`--json` outputs.

## Non-goals
- Extracting or modifying ROM archives.
- Managing emulator configuration or libraries.
- Downloading content you do not own.

## Command Tree
- `vimm systems` — list available systems and vault slugs
- `vimm search` — search ROMs by name pattern
- `vimm download` — download + verify ROMs
- `vimm verify` — re-verify local ROMs
- `vimm completion` — shell completion scripts

## Global Flags
- `-h, --help`
- `--version`
- `-q, --quiet` (errors only)
- `-v, --verbose` (repeatable; `-vv` for debug)
- `--json` machine output (implies `--plain`)
- `--plain` stable line-based text (no color, no progress)
- `--no-color`
- `--no-input` never prompt
- `--config <path>`

## I/O Contract
- **stdout**: primary data (lists/summaries/JSON)
- **stderr**: progress, retries, warnings, errors

## Exit Codes
- `0` success
- `1` generic failure
- `2` invalid usage/validation
- `3` network/HTTP error after retries
- `4` checksum verification failed
- `5` `--all` count mismatch or count unavailable with `--strict-count`
- `6` partial success (some downloads failed)

## Systems
Systems are identified by vault slugs (e.g., `N64` for `https://vimm.net/vault/N64`).

`vimm systems` output:
- plain: `SLUG<TAB>NAME<TAB>CLASS<TAB>TITLES`
- json: array of `{slug,name,class,titles}`

## Search Semantics
`vimm search` and `vimm download --query` share matching rules:

Flags:
- `--query <pattern>` required
- `--match <auto|glob|prefix|contains|regex>` default: `auto`

Rules:
- `auto`: if pattern contains glob chars (`*?[]`) → `glob`, else → `prefix`
- Users should **quote** globs, e.g. `"*mario*"`

Search output:
- plain: `NAME<TAB>VAULT_ID<TAB>SYSTEM`
- json: array of `{name, vault_id, system}`

## Download Selection
`vimm download` selection flags are mutually exclusive:
- `--id <vault_id>` (repeatable)
- `--system <slug> --query <pattern>`
- `--system <slug> --all`

## Download Robustness
Flags:
- `--output-dir <path>` default: `.` (cwd)
- `--tmp-dir <path>` default: `<output-dir>/.vimm.tmp`
- `-c, --concurrency <n>` default: 4
- `--retries <n>` default: 5
- `--retry-backoff <linear|exponential>` default: `exponential` (with jitter)
- `--timeout <duration>` default: `60s`
- `--max-rps <n>` optional rate limit
- `--resume` default: true
- `--overwrite` always re-download
- `--dry-run` plan only

## Version Selection
Default behavior is `--latest`:
- Prefer highest revision number when present.
- Else prefer newest dated entry.
- Else prefer the last listed media file.

Override:
- `--revision <value>` (e.g., `rev2` or `2021-05-01`)

## Verification
Default behavior is `--verify`:
- Parse CRC/MD5/SHA1 from `Vimm’s Lair.txt` on the ROM page.
- Verify the downloaded file against all three hashes.
- Missing hashes are a failure when `--strict-hashes` is enabled (default).

Flags:
- `--verify` (default: true)
- `--skip-verify`
- `--strict-hashes` (default: true)

Behavior:
- Existing files that match all hashes are skipped unless `--overwrite`.
- On mismatch: delete temp file, mark failure, continue; overall exit `4` or `6`.

## `--all` Completion Check
When `--all` is used:
- Compare total verified files (existing + newly downloaded) to the system’s reported title count.

Flags:
- `--count-check` (default: true)
- `--strict-count` fail if count cannot be read
- `--allow-mismatch` ignore mismatched counts

## Config and Environment
Precedence: flags > env > project config > user config

Config files:
- Project: `./.vimm.toml`
- User: `~/.config/vimm/config.toml`

Environment variables:
- `VIMM_BASE_URL` default `https://vimm.net/vault`
- `VIMM_OUTPUT_DIR`
- `VIMM_CONCURRENCY`
- `VIMM_RETRIES`
- `VIMM_TIMEOUT`
- `VIMM_MAX_RPS`
- `VIMM_NO_COLOR`

## Examples
```bash
# list systems
vimm systems

# prefix search (auto)
vimm search --system N64 --query mario

# contains search (glob)
vimm search --system N64 --query "*mario*"

# download a few matches
vimm download --system N64 --query "*mario*" --output-dir ./roms

# download all for a system (confirmation required)
vimm download --system N64 --all

# non-interactive bulk download
vimm download --system N64 --all --force --no-input

# download specific ROMs by ID
vimm download --id 2754 --id 1234

# verify local files
vimm verify --system N64 --output-dir ./roms --query "*fox*"
```
