package vault

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://vimm.net/vault"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Fetch(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "vimm-cli/0.1")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) Systems(ctx context.Context) ([]System, error) {
	body, err := c.Fetch(ctx, "")
	if err != nil {
		return nil, err
	}
	return ParseSystems(strings.NewReader(body))
}

func (c *Client) SystemCount(ctx context.Context, slug string) (int, error) {
	body, err := c.Fetch(ctx, "/"+slug)
	if err != nil {
		return 0, err
	}
	return ParseSystemCount(strings.NewReader(body))
}

func (c *Client) Search(ctx context.Context, slug, query string) ([]ROMEntry, error) {
	q := url.Values{}
	q.Set("p", "list")
	q.Set("system", slug)
	if query != "" {
		q.Set("q", query)
	}
	body, err := c.Fetch(ctx, "/?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return ParseSearchResults(strings.NewReader(body), slug)
}

func (c *Client) ROMMedia(ctx context.Context, id int) ([]Media, error) {
	body, err := c.Fetch(ctx, "/"+strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	return ParseMediaFromPage(body)
}

func (c *Client) ListAll(ctx context.Context, slug string) ([]ROMEntry, error) {
	sections := []string{"number"}
	for letter := 'A'; letter <= 'Z'; letter++ {
		sections = append(sections, string(letter))
	}

	entries := make([]ROMEntry, 0, 1024)
	seen := make(map[int]struct{})
	for _, section := range sections {
		sectionEntries, err := c.ListSection(ctx, slug, section)
		if err != nil {
			return nil, err
		}
		for _, entry := range sectionEntries {
			if _, ok := seen[entry.ID]; ok {
				continue
			}
			seen[entry.ID] = struct{}{}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (c *Client) ListSection(ctx context.Context, slug, section string) ([]ROMEntry, error) {
	var path string
	if section == "number" {
		q := url.Values{}
		q.Set("p", "list")
		q.Set("system", slug)
		q.Set("section", "number")
		path = "/?" + q.Encode()
	} else {
		path = "/" + slug + "/" + section
	}
	body, err := c.Fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	return ParseSearchResults(strings.NewReader(body), slug)
}
