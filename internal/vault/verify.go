package vault

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Hashes struct {
	CRC  string
	MD5  string
	SHA1 string
}

func (h Hashes) Empty() bool {
	return h.CRC == "" && h.MD5 == "" && h.SHA1 == ""
}

func ParseLairTxt(data []byte) (Hashes, error) {
	lines := strings.Split(string(data), "\n")
	var hashes Hashes
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "CRC:"):
			hashes.CRC = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "CRC:")))
		case strings.HasPrefix(line, "MD5:"):
			hashes.MD5 = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "MD5:")))
		case strings.HasPrefix(line, "SHA-1:"):
			hashes.SHA1 = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "SHA-1:")))
		}
	}
	if hashes.Empty() {
		return Hashes{}, fmt.Errorf("hashes not found in Vimm's Lair.txt")
	}
	return hashes, nil
}

func ExtractRomNameFromLair(data []byte) string {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, ":") {
			continue
		}
		ext := filepath.Ext(line)
		if len(ext) < 2 || len(ext) > 6 {
			continue
		}
		return line
	}
	return ""
}
