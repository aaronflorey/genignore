package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	defaultListURL     = "https://www.toptal.com/developers/gitignore/api/list?format=json"
	defaultTemplateURL = "https://www.toptal.com/developers/gitignore/api/"
)

type TemplateResponse struct {
	Providers []string `json:"providers"`
	Content   string   `json:"content"`
}

type Client struct {
	httpClient  *http.Client
	listURL     string
	templateURL string
}

func NewClient() *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		listURL:     defaultListURL,
		templateURL: defaultTemplateURL,
	}
}

func (c *Client) AvailableProviders(ctx context.Context) ([]string, error) {
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
	providers, err := decodeAvailableProviders(body)
	if err != nil {
		return nil, fmt.Errorf("decode list API response: %w", err)
	}
	slices.Sort(providers)
	return providers, nil
}

func (c *Client) FetchTemplate(ctx context.Context, providers []string) (TemplateResponse, error) {
	if len(providers) == 0 {
		return TemplateResponse{}, fmt.Errorf("providers must not be empty")
	}
	joined := strings.Join(providers, ",")
	u := c.templateURL + url.PathEscape(joined)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return TemplateResponse{}, fmt.Errorf("build template request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return TemplateResponse{}, fmt.Errorf("request template API: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return TemplateResponse{}, fmt.Errorf("template API returned status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return TemplateResponse{}, fmt.Errorf("read template API response: %w", err)
	}
	return TemplateResponse{Providers: providers, Content: string(body)}, nil
}

func decodeAvailableProviders(body []byte) ([]string, error) {
	var listShape struct {
		Gitignores *[]string `json:"gitignores"`
	}
	if err := json.Unmarshal(body, &listShape); err == nil && listShape.Gitignores != nil {
		return append([]string{}, (*listShape.Gitignores)...), nil
	}

	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(catalog))
	for key := range catalog {
		providers = append(providers, key)
	}
	return providers, nil
}
