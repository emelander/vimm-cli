package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"vimm-download/internal/vault"
)

func outputSearchResults(cfg *Config, results []vault.ROMEntry) error {
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	for _, entry := range results {
		fmt.Fprintf(os.Stdout, "%s\t%d\t%s\n", entry.Title, entry.ID, entry.System)
	}
	return nil
}
