package cli

import "fmt"

func validateMatch(value string) error {
	switch value {
	case "", "auto", "glob", "prefix", "contains", "regex":
		return nil
	default:
		return fmt.Errorf("invalid --match value: %s", value)
	}
}

func validateClass(value string) error {
	if value == "" {
		return nil
	}
	if value == "console" || value == "handheld" {
		return nil
	}
	return fmt.Errorf("invalid --class value: %s", value)
}
