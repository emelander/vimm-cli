package vault

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var systemCountRe = regexp.MustCompile(`Have\s+(\d+)\s+of\s+(\d+)\s+media`)

func ParseSystemCount(r io.Reader) (int, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return 0, err
	}
	text := textContent(doc)
	matches := systemCountRe.FindStringSubmatch(text)
	if len(matches) < 3 {
		return 0, fmt.Errorf("system count not found")
	}
	count, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, fmt.Errorf("invalid count: %w", err)
	}
	return count, nil
}

func ParseSystems(r io.Reader) ([]System, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]System)
	var walkTable func(*html.Node)
	walkTable = func(n *html.Node) {
		if n.Type != html.ElementNode || n.Data != "table" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkTable(c)
			}
			return
		}

		class := tableClassFromCaption(n)
		if class == "" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkTable(c)
			}
			return
		}

		var walkLinks func(*html.Node)
		walkLinks = func(node *html.Node) {
			if node.Type == html.ElementNode && node.Data == "a" {
				href := strings.TrimSpace(attrValue(node, "href"))
				slug := slugFromHref(href)
				if slug != "" {
					name := strings.TrimSpace(textContent(node))
					if name != "" {
						seen[slug] = System{
							Slug:  slug,
							Name:  name,
							Class: class,
						}
					}
				}
			}
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				walkLinks(c)
			}
		}
		walkLinks(n)

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkTable(c)
		}
	}
	walkTable(doc)

	systems := make([]System, 0, len(seen))
	for _, sys := range seen {
		systems = append(systems, sys)
	}
	return systems, nil
}

func ParseSearchResults(r io.Reader, system string) ([]ROMEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []ROMEntry
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" && hasClass(n, "hovertable") {
			collectResults(n, system, &results)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results, nil
}

func collectResults(table *html.Node, system string, results *[]ROMEntry) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := strings.TrimSpace(attrValue(n, "href"))
			if id, ok := vaultIDFromHref(href); ok {
				title := strings.TrimSpace(textContent(n))
				if title != "" {
					*results = append(*results, ROMEntry{
						ID:     id,
						Title:  title,
						System: system,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
}

func tableClassFromCaption(table *html.Node) string {
	for c := table.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "caption" {
			caption := strings.ToLower(strings.TrimSpace(textContent(c)))
			switch caption {
			case "consoles":
				return "console"
			case "handhelds":
				return "handheld"
			}
		}
	}
	return ""
}

func attrValue(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func hasClass(n *html.Node, class string) bool {
	classes := strings.Fields(attrValue(n, "class"))
	for _, c := range classes {
		if c == class {
			return true
		}
	}
	return false
}

func slugFromHref(href string) string {
	if !strings.HasPrefix(href, "/vault/") {
		return ""
	}
	path := strings.TrimPrefix(href, "/vault/")
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.Split(path, "?")[0]
	path = strings.Split(path, "#")[0]
	path = strings.Trim(path, "/")
	if path == "" || strings.Contains(path, "/") {
		return ""
	}
	if isNumeric(path) {
		return ""
	}
	return path
}

func vaultIDFromHref(href string) (int, bool) {
	if !strings.HasPrefix(href, "/vault/") {
		return 0, false
	}
	path := strings.TrimPrefix(href, "/vault/")
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	path = strings.Split(path, "?")[0]
	path = strings.Split(path, "#")[0]
	path = strings.Trim(path, "/")
	if !isNumeric(path) {
		return 0, false
	}
	id, err := strconv.Atoi(path)
	if err != nil {
		return 0, false
	}
	return id, true
}

func isNumeric(val string) bool {
	for _, r := range val {
		if r < '0' || r > '9' {
			return false
		}
	}
	return val != ""
}
