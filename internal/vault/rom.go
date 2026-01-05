package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type GoodDate struct {
	Date string `json:"date"`
}

type Media struct {
	ID            int       `json:"ID"`
	GoodDate      *GoodDate `json:"GoodDate"`
	GoodTitle     string    `json:"GoodTitle"`
	Serial        string    `json:"Serial"`
	SortOrder     int       `json:"SortOrder"`
	Version       string    `json:"Version"`
	VersionString string    `json:"VersionString"`
	Zipped        string    `json:"Zipped"`
	AltZipped     string    `json:"AltZipped"`
	AltZipped2    string    `json:"AltZipped2"`
	GoodHash      string    `json:"GoodHash"`
	GoodMd5       string    `json:"GoodMd5"`
	GoodSha1      string    `json:"GoodSha1"`
	Crc           string    `json:"Crc"`
	Md5           string    `json:"Md5"`
	Sha1          string    `json:"Sha1"`
}

func ParseMediaFromPage(html string) ([]Media, error) {
	const marker = "const media="
	start := strings.Index(html, marker)
	if start == -1 {
		return nil, fmt.Errorf("media array not found")
	}
	start += len(marker)
	end := strings.Index(html[start:], "];")
	if end == -1 {
		return nil, fmt.Errorf("media array terminator not found")
	}
	jsonBlob := strings.TrimSpace(html[start : start+end+1])
	var media []Media
	if err := json.Unmarshal([]byte(jsonBlob), &media); err != nil {
		return nil, fmt.Errorf("parse media json: %w", err)
	}
	return media, nil
}

var dlFormRe = regexp.MustCompile(`(?i)<form[^>]*id="dl_form"[^>]*>`)
var actionAttrRe = regexp.MustCompile(`(?i)\baction="([^"]+)"`)

func ParseDownloadBaseFromPage(html string) string {
	tag := dlFormRe.FindString(html)
	if tag == "" {
		return ""
	}
	match := actionAttrRe.FindStringSubmatch(tag)
	if len(match) < 2 {
		return ""
	}
	action := strings.TrimSpace(match[1])
	if action == "" {
		return ""
	}
	if strings.HasPrefix(action, "//") {
		action = "https:" + action
	}
	action = strings.TrimRight(action, "/")
	return action
}

func ParseDownloadAltFromPage(htmlText string) int {
	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return 0
	}
	var selectNode *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if selectNode != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "select" {
			for _, attr := range n.Attr {
				if strings.EqualFold(attr.Key, "id") && attr.Val == "dl_format" {
					selectNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if selectNode == nil {
		return 0
	}

	first := -1
	selected := -1
	for c := selectNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "option" {
			continue
		}
		value := ""
		isSelected := false
		for _, attr := range c.Attr {
			switch strings.ToLower(attr.Key) {
			case "value":
				value = strings.TrimSpace(attr.Val)
			case "selected":
				isSelected = true
			}
		}
		if value == "" {
			continue
		}
		alt, err := strconv.Atoi(value)
		if err != nil {
			continue
		}
		if first == -1 {
			first = alt
		}
		if isSelected {
			selected = alt
		}
	}
	if selected != -1 {
		return normalizeAlt(selected)
	}
	if first != -1 {
		return normalizeAlt(first)
	}
	return 0
}

func (m Media) ExpectedHashes() Hashes {
	crc := firstNonEmpty(m.GoodHash, m.Crc)
	md5 := firstNonEmpty(m.GoodMd5, m.Md5)
	sha1 := firstNonEmpty(m.GoodSha1, m.Sha1)
	return Hashes{
		CRC:  strings.ToLower(crc),
		MD5:  strings.ToLower(md5),
		SHA1: strings.ToLower(sha1),
	}
}

func (m Media) DecodedTitle() string {
	if m.GoodTitle == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(m.GoodTitle)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (m Media) ZippedAvailable() bool {
	return parseNumeric(m.Zipped) > 0
}

func (m Media) DownloadAvailableAlt(alt int) bool {
	switch alt {
	case 1:
		return parseNumeric(m.AltZipped) > 0
	case 2:
		return parseNumeric(m.AltZipped2) > 0
	default:
		return parseNumeric(m.Zipped) > 0
	}
}

func parseNumeric(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func normalizeAlt(alt int) int {
	switch alt {
	case 1, 2:
		return alt
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
