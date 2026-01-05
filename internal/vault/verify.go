package vault

import (
	"fmt"
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
