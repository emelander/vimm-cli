## vimm

`vimm` is a Go-based CLI for downloading ROMs you own from Vimm’s Lair and verifying them against the hashes published for each title.

### Features
- Search by name or download by vault ID
- Bulk downloads per system with count checks
- Concurrent downloads with retries, rate limiting, and resume
- CRC/MD5/SHA1 verification using `Vimm’s Lair.txt`
- JSON or plain text output

### Installation
Requires Go 1.25.5.

```bash
go build -o vimm ./cmd/vimm
```

Or install directly with Go:

```bash
go install github.com/emelander/vimm-download/cmd/vimm@latest
```

### Usage
```bash
# List systems
./vimm systems

# Search within a system
./vimm search --system N64 --query "*mario*"

# Download a few matches
./vimm download --system N64 --query "*mario*" --output-dir ./roms

# Download all ROMs for a system (confirmation required)
./vimm download --system N64 --all

# Verify local files
./vimm verify --system N64 --output-dir ./roms --query "*fox*"
```

### Configuration
Precedence: flags > env > project config > user config

Config files:
- `./.vimm.toml`
- `~/.config/vimm/config.toml`

Environment variables (defaults in parentheses):
- `VIMM_BASE_URL` (default: `https://vimm.net/vault`)
- `VIMM_OUTPUT_DIR` (default: `.`)
- `VIMM_CONCURRENCY` (default: `4`)
- `VIMM_RETRIES` (default: `5`)
- `VIMM_TIMEOUT` (default: `60s`)
- `VIMM_MAX_RPS` (default: `0`, unlimited)
- `VIMM_NO_COLOR` (default: `false`)
