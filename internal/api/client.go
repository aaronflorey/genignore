package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aaronflorey/genignore/internal/customtemplate"
)

const (
	defaultListURL     = "https://api.github.com/repos/github/gitignore/git/trees/main?recursive=1"
	defaultTemplateURL = "https://raw.githubusercontent.com/github/gitignore/main/"
)

type TemplateResponse struct {
	Providers          []string `json:"providers"`
	Content            string   `json:"content"`
	AvailableProviders []string `json:"-"`
}

type Client struct {
	httpClient  *http.Client
	listURL     string
	templateURL string

	catalogMu sync.Mutex
	catalog   map[string]string
}

func NewClient() *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		listURL:     defaultListURL,
		templateURL: defaultTemplateURL,
	}
}

func (c *Client) AvailableProviders(ctx context.Context) ([]string, error) {
	catalog, err := c.fetchProviderCatalog(ctx)
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(catalog))
	for key := range catalog {
		providers = append(providers, key)
	}
	slices.Sort(providers)
	return providers, nil
}

func (c *Client) FetchTemplate(ctx context.Context, providers []string) (TemplateResponse, error) {
	if len(providers) == 0 {
		return TemplateResponse{}, fmt.Errorf("providers must not be empty")
	}
	var availableProviders []string
	remoteProviders := make([]string, 0, len(providers))
	customProviders := make([]string, 0, len(providers))
	for _, key := range providers {
		if customtemplate.HasProvider(key) {
			customProviders = append(customProviders, key)
			continue
		}
		remoteProviders = append(remoteProviders, key)
	}

	parts := make([]string, 0, 2)
	if len(remoteProviders) > 0 {
		catalog, err := c.fetchProviderCatalog(ctx)
		if err != nil {
			return TemplateResponse{}, err
		}
		availableProviders = sortedCatalogProviders(catalog)
		for _, key := range remoteProviders {
			templatePath, ok := catalog[key]
			if !ok {
				return TemplateResponse{}, fmt.Errorf("template catalog missing provider: %s", key)
			}
			content, err := c.fetchTemplatePart(ctx, key, templatePath)
			if err != nil {
				return TemplateResponse{}, err
			}
			if content != "" {
				parts = append(parts, content)
			}
		}
	}

	customContent, err := customtemplate.ContentForProviders(customProviders)
	if err != nil {
		return TemplateResponse{}, err
	}
	if customContent != "" {
		parts = append(parts, customContent)
	}

	return TemplateResponse{Providers: providers, Content: strings.Join(parts, "\n\n"), AvailableProviders: availableProviders}, nil
}

func (c *Client) fetchProviderCatalog(ctx context.Context) (map[string]string, error) {
	c.catalogMu.Lock()
	if c.catalog != nil {
		catalog := cloneCatalog(c.catalog)
		c.catalogMu.Unlock()
		return catalog, nil
	}
	c.catalogMu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build list request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request list API: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("list API returned status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read list API response: %w", err)
	}
	catalog, err := decodeProviderCatalog(body)
	if err != nil {
		return nil, fmt.Errorf("decode list API response: %w", err)
	}

	c.catalogMu.Lock()
	c.catalog = cloneCatalog(catalog)
	c.catalogMu.Unlock()

	return catalog, nil
}

func (c *Client) fetchTemplatePart(ctx context.Context, key string, templatePath string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.templateURL+templatePath, nil)
	if err != nil {
		return "", fmt.Errorf("build template request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request template API: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("template API returned status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read template API response: %w", err)
	}
	return strings.Trim(string(body), "\n"), nil
}

func decodeProviderCatalog(body []byte) (map[string]string, error) {
	var treeResponse struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(body, &treeResponse); err != nil {
		return nil, err
	}
	if treeResponse.Tree == nil {
		return nil, fmt.Errorf("missing tree entries")
	}

	catalog := make(map[string]string)
	for _, entry := range treeResponse.Tree {
		if entry.Type != "blob" || !strings.HasSuffix(entry.Path, ".gitignore") || !isCatalogTemplatePath(entry.Path) {
			continue
		}
		key := strings.ToLower(strings.TrimSuffix(path.Base(entry.Path), ".gitignore"))
		if existing, ok := catalog[key]; ok && !preferCatalogPath(entry.Path, existing) {
			continue
		}
		catalog[key] = entry.Path
	}
	if len(catalog) == 0 {
		return nil, fmt.Errorf("no gitignore templates found")
	}
	return catalog, nil
}

func sortedCatalogProviders(catalog map[string]string) []string {
	providers := make([]string, 0, len(catalog))
	for key := range catalog {
		providers = append(providers, key)
	}
	slices.Sort(providers)
	return providers
}

func isCatalogTemplatePath(templatePath string) bool {
	if !strings.Contains(templatePath, "/") {
		return true
	}
	parts := strings.Split(templatePath, "/")
	return len(parts) == 2 && parts[0] == "Global"
}

func preferCatalogPath(candidate string, existing string) bool {
	if strings.Count(candidate, "/") != strings.Count(existing, "/") {
		return strings.Count(candidate, "/") < strings.Count(existing, "/")
	}
	return candidate < existing
}

func cloneCatalog(catalog map[string]string) map[string]string {
	cloned := make(map[string]string, len(catalog))
	for key, value := range catalog {
		cloned[key] = value
	}
	return cloned
}
