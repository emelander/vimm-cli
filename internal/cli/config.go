package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const defaultBaseURL = "https://vimm.net/vault"

type runtimeConfig struct {
	BaseURL     string
	OutputDir   *string
	Concurrency *int
	Retries     *int
	Timeout     *time.Duration
	MaxRPS      *int
	NoColor     *bool
}

type fileConfig struct {
	BaseURL     *string `toml:"base_url"`
	OutputDir   *string `toml:"output_dir"`
	Concurrency *int    `toml:"concurrency"`
	Retries     *int    `toml:"retries"`
	Timeout     *string `toml:"timeout"`
	MaxRPS      *int    `toml:"max_rps"`
	NoColor     *bool   `toml:"no_color"`
}

func loadRuntimeConfig(explicitPath string) (runtimeConfig, error) {
	cfg := runtimeConfig{BaseURL: defaultBaseURL}

	userPath, err := userConfigPath()
	if err != nil {
		return cfg, err
	}
	if userPath != "" {
		if err := applyConfigFile(&cfg, userPath, false); err != nil {
			return cfg, err
		}
	}
	if err := applyConfigFile(&cfg, ".vimm.toml", false); err != nil {
		return cfg, err
	}
	if explicitPath != "" {
		if err := applyConfigFile(&cfg, explicitPath, true); err != nil {
			return cfg, err
		}
	}

	if val := strings.TrimSpace(os.Getenv("VIMM_BASE_URL")); val != "" {
		cfg.BaseURL = strings.TrimRight(val, "/")
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_OUTPUT_DIR")); val != "" {
		cfg.OutputDir = &val
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_CONCURRENCY")); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return cfg, fmt.Errorf("invalid VIMM_CONCURRENCY: %w", err)
		}
		cfg.Concurrency = &parsed
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_RETRIES")); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return cfg, fmt.Errorf("invalid VIMM_RETRIES: %w", err)
		}
		cfg.Retries = &parsed
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_TIMEOUT")); val != "" {
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("invalid VIMM_TIMEOUT: %w", err)
		}
		cfg.Timeout = &parsed
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_MAX_RPS")); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return cfg, fmt.Errorf("invalid VIMM_MAX_RPS: %w", err)
		}
		cfg.MaxRPS = &parsed
	}
	if val := strings.TrimSpace(os.Getenv("VIMM_NO_COLOR")); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return cfg, fmt.Errorf("invalid VIMM_NO_COLOR: %w", err)
		}
		cfg.NoColor = &parsed
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}

func userConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vimm", "config.toml"), nil
}

func applyConfigFile(cfg *runtimeConfig, path string, required bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil
		}
		return err
	}
	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if fc.BaseURL != nil && strings.TrimSpace(*fc.BaseURL) != "" {
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(*fc.BaseURL), "/")
	}
	if fc.OutputDir != nil {
		cfg.OutputDir = fc.OutputDir
	}
	if fc.Concurrency != nil {
		cfg.Concurrency = fc.Concurrency
	}
	if fc.Retries != nil {
		cfg.Retries = fc.Retries
	}
	if fc.Timeout != nil && strings.TrimSpace(*fc.Timeout) != "" {
		parsed, err := time.ParseDuration(strings.TrimSpace(*fc.Timeout))
		if err != nil {
			return fmt.Errorf("invalid timeout in %s: %w", path, err)
		}
		cfg.Timeout = &parsed
	}
	if fc.MaxRPS != nil {
		cfg.MaxRPS = fc.MaxRPS
	}
	if fc.NoColor != nil {
		cfg.NoColor = fc.NoColor
	}
	return nil
}
