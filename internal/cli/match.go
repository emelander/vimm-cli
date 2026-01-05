package cli

import (
	"path"
	"regexp"
	"strings"
)

type matcher struct {
	mode    string
	pattern string
	regex   *regexp.Regexp
}

func newMatcher(mode, pattern string) (*matcher, error) {
	resolved := resolveMatchMode(mode, pattern)
	m := &matcher{
		mode:    resolved,
		pattern: pattern,
	}
	switch resolved {
	case "regex":
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, err
		}
		m.regex = re
	}
	return m, nil
}

func (m *matcher) Match(value string) (bool, error) {
	switch m.mode {
	case "prefix":
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(m.pattern)), nil
	case "contains":
		return strings.Contains(strings.ToLower(value), strings.ToLower(m.pattern)), nil
	case "glob":
		return path.Match(strings.ToLower(m.pattern), strings.ToLower(value))
	case "regex":
		return m.regex.MatchString(value), nil
	default:
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(m.pattern)), nil
	}
}

func resolveMatchMode(mode, pattern string) string {
	if mode == "" || mode == "auto" {
		if strings.ContainsAny(pattern, "*?[]") {
			return "glob"
		}
		return "prefix"
	}
	return mode
}

func queryHint(pattern, mode string) string {
	resolved := resolveMatchMode(mode, pattern)
	switch resolved {
	case "glob", "regex":
		return longestLiteral(pattern)
	default:
		return strings.TrimSpace(pattern)
	}
}

func longestLiteral(pattern string) string {
	longest := ""
	current := strings.Builder{}
	flush := func() {
		if current.Len() > len(longest) {
			longest = current.String()
		}
		current.Reset()
	}
	for _, r := range pattern {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return strings.TrimSpace(longest)
}
