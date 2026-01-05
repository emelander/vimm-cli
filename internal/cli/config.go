package cli

import (
	"os"
	"strings"
)

const defaultBaseURL = "https://vimm.net/vault"

func resolveBaseURL() string {
	base := strings.TrimSpace(os.Getenv("VIMM_BASE_URL"))
	if base == "" {
		base = defaultBaseURL
	}
	return strings.TrimRight(base, "/")
}
