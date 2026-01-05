## vimm

`vimm` is a Go-based CLI for downloading ROMs you own from Vimm’s Lair and verifying them against the hashes published for each title.

### Features
- Search by name or download by vault ID
- Bulk downloads per system with count checks
- Concurrent downloads with retries, rate limiting, and resume
- Automatically uses the format selected by the site when multiple formats are offered
- CRC/MD5/SHA1 verification using `Vimm’s Lair.txt`
- Existing files are always checksum-verified before skipping
- Region filtering with variant-based excludes (defaults to USA, excludes Virtual Console and LodgeNet; use `--include-variants` to allow them)
- JSON or plain text output

### Installation
Requires Go 1.25.5.

```bash
go build -o vimm ./cmd/vimm
```

Or install directly with Go:

```bash
go install github.com/emelander/vimm-cli/cmd/vimm@latest
```

### Usage
```bash
# List systems
./vimm systems

# Search within a system code (region filter defaults to USA)
./vimm search --system N64 --query "*mario*"

# Include non-USA regions or special variants
./vimm search --system N64 --query "*fox*" --region all --include-variants all

# Download a few matches
./vimm download --system N64 --query "*mario*" --output-dir ./roms

# Download all ROMs for a system code (confirmation required)
./vimm download --system N64 --all

# Download all USA LodgeNet variants
./vimm download --system N64 --all --region USA --include-variants LodgeNet

# Verify local files by system code
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
- `VIMM_CONCURRENCY` (default: `1`)
- `VIMM_RETRIES` (default: `5`)
- `VIMM_TIMEOUT` (default: `60s`)
- `VIMM_MAX_RPS` (default: `0`, unlimited)
- `VIMM_NO_COLOR` (default: `false`)
