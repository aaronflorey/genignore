package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/providercatalog"
)

const (
	DefaultUpstreamCommit  = "3780fff86c705155792fb3e1787cebd6281ba8cf"
	defaultListURLTemplate = "https://api.github.com/repos/github/gitignore/git/trees/%s?recursive=1"
	defaultTemplateURLTmpl = "https://raw.githubusercontent.com/github/gitignore/%s/"
	cacheMetadataVersion   = 1
	cacheFreshnessWindow   = 7 * 24 * time.Hour
)

type TemplateResponse struct {
	Providers          []string `json:"providers"`
	Content            string   `json:"content"`
	AvailableProviders []string `json:"-"`
}

type RuntimeDiagnostics struct {
	UpstreamCommit    string
	Offline           bool
	RemoteProviders   []string
	EmbeddedProviders []string
	CacheEntries      []CacheEntryStatus
	Decisions         []string
}

type CacheEntryStatus struct {
	Provider string
	State    string
	Detail   string
}

type Options struct {
	Offline        bool
	UpstreamCommit string
}

type Client struct {
	httpClient     *http.Client
	listURL        string
	templateURL    string
	offline        bool
	cacheDir       string
	upstreamCommit string

	catalogMu sync.Mutex
	catalog   map[string]string
}

func NewClient() *Client {
	return NewClientWithOptions(Options{})
}

var userCacheDir = os.UserCacheDir

func NewClientWithOptions(opts Options) *Client {
	cacheDir, _ := defaultCacheDir()
	upstreamCommit := strings.TrimSpace(opts.UpstreamCommit)
	if upstreamCommit == "" {
		upstreamCommit = DefaultUpstreamCommit
	}
	return &Client{
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		listURL:        fmt.Sprintf(defaultListURLTemplate, upstreamCommit),
		templateURL:    fmt.Sprintf(defaultTemplateURLTmpl, upstreamCommit),
		offline:        opts.Offline,
		cacheDir:       cacheDir,
		upstreamCommit: upstreamCommit,
	}
}

type cacheMetadata struct {
	Version        int       `json:"version"`
	UpstreamCommit string    `json:"upstream_commit"`
	ETag           string    `json:"etag,omitempty"`
	FetchedAt      time.Time `json:"fetched_at"`
	SHA256         string    `json:"sha256"`
}

type cacheEntry struct {
	Body     []byte
	Metadata cacheMetadata
	Stale    bool
}

func (c *Client) AvailableProviders(ctx context.Context) ([]string, error) {
	return providercatalog.RemoteSupportedKeys(), nil
}

func (c *Client) InspectRuntime(providers []string) RuntimeDiagnostics {
	remoteProviders, customProviders := splitProvidersBySource(providers)
	diagnostics := RuntimeDiagnostics{
		UpstreamCommit:    c.upstreamCommit,
		Offline:           c.offline,
		RemoteProviders:   remoteProviders,
		EmbeddedProviders: customProviders,
		CacheEntries:      make([]CacheEntryStatus, 0, len(remoteProviders)),
	}
	for _, key := range remoteProviders {
		diagnostics.CacheEntries = append(diagnostics.CacheEntries, c.inspectTemplateCache(key))
	}
	diagnostics.Decisions = append(diagnostics.Decisions, runtimeDecisions(c.offline, remoteProviders, customProviders)...)
	return diagnostics
}

func (c *Client) FetchTemplate(ctx context.Context, providers []string) (TemplateResponse, error) {
	if len(providers) == 0 {
		return TemplateResponse{}, fmt.Errorf("providers must not be empty")
	}
	var availableProviders []string
	remoteProviders, customProviders := splitProvidersBySource(providers)

	parts := make([]string, 0, 2)
	if len(remoteProviders) > 0 {
		if c.offline {
			availableProviders = providercatalog.RemoteSupportedKeys()
			for _, key := range remoteProviders {
				content, err := c.readCachedTemplate(key)
				if err != nil {
					return TemplateResponse{}, err
				}
				if content != "" {
					parts = append(parts, content)
				}
			}
		} else {
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

func splitProvidersBySource(providers []string) ([]string, []string) {
	remoteProviders := make([]string, 0, len(providers))
	customProviders := make([]string, 0, len(providers))
	for _, key := range providers {
		if customtemplate.HasProvider(key) {
			customProviders = append(customProviders, key)
			continue
		}
		remoteProviders = append(remoteProviders, key)
	}
	return remoteProviders, customProviders
}

func (c *Client) fetchProviderCatalog(ctx context.Context) (map[string]string, error) {
	c.catalogMu.Lock()
	if c.catalog != nil {
		catalog := cloneCatalog(c.catalog)
		c.catalogMu.Unlock()
		return catalog, nil
	}
	c.catalogMu.Unlock()

	cachedCatalog, cacheErr := c.readCachedCatalog(false)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build list request: %w", err)
	}
	if cacheErr == nil && cachedCatalog.Metadata.ETag != "" {
		req.Header.Set("If-None-Match", cachedCatalog.Metadata.ETag)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request list API: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode == http.StatusNotModified {
		if cacheErr != nil {
			return nil, fmt.Errorf("list API returned status 304 without a valid cached provider catalog")
		}
		catalog, err := decodeProviderCatalog(cachedCatalog.Body)
		if err != nil {
			return nil, fmt.Errorf("decode cached provider catalog: %w", err)
		}
		if err := c.refreshCachedCatalog(cachedCatalog, res.Header.Get("ETag")); err != nil {
			return nil, err
		}

		c.catalogMu.Lock()
		c.catalog = cloneCatalog(catalog)
		c.catalogMu.Unlock()
		return catalog, nil
	}
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
	if err := c.writeCachedCatalog(body, res.Header.Get("ETag")); err != nil {
		return nil, err
	}

	c.catalogMu.Lock()
	c.catalog = cloneCatalog(catalog)
	c.catalogMu.Unlock()

	return catalog, nil
}

func (c *Client) fetchTemplatePart(ctx context.Context, key string, templatePath string) (string, error) {
	cachedTemplate, cacheErr := c.readCachedTemplateEntry(key, false)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.templateURL+templatePath, nil)
	if err != nil {
		return "", fmt.Errorf("build template request: %w", err)
	}
	if cacheErr == nil && cachedTemplate.Metadata.ETag != "" {
		req.Header.Set("If-None-Match", cachedTemplate.Metadata.ETag)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request template API: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode == http.StatusNotModified {
		if cacheErr != nil {
			return "", fmt.Errorf("template API returned status 304 without a valid cached template for provider %s", key)
		}
		if err := c.refreshCachedTemplate(key, cachedTemplate, res.Header.Get("ETag")); err != nil {
			return "", err
		}
		return strings.Trim(string(cachedTemplate.Body), "\n"), nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("template API returned status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read template API response: %w", err)
	}
	content := strings.Trim(string(body), "\n")
	if err := c.writeCachedTemplate(key, content, res.Header.Get("ETag")); err != nil {
		return "", err
	}
	return content, nil
}

func defaultCacheDir() (string, error) {
	cacheHome, err := userCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve cache home: %w", err)
	}
	return filepath.Join(cacheHome, "genignore", "github-gitignore"), nil
}

func (c *Client) readCachedTemplate(key string) (string, error) {
	entry, err := c.readCachedTemplateEntry(key, true)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("offline mode requires cached template for provider: %s", key)
		}
		return "", err
	}
	return strings.Trim(string(entry.Body), "\n"), nil
}

func (c *Client) writeCachedTemplate(key string, content string, etag string) error {
	return c.writeCacheEntry(c.templateCachePath(key), c.templateMetadataPath(key), []byte(content), etag, fmt.Sprintf("template for provider %s", key))
}

func (c *Client) writeCachedCatalog(body []byte, etag string) error {
	return c.writeCacheEntry(c.catalogCachePath(), c.catalogMetadataPath(), body, etag, "provider catalog")
}

func (c *Client) refreshCachedTemplate(key string, entry cacheEntry, etag string) error {
	return c.refreshCacheEntry(c.templateMetadataPath(key), entry, etag, fmt.Sprintf("template for provider %s", key))
}

func (c *Client) refreshCachedCatalog(entry cacheEntry, etag string) error {
	return c.refreshCacheEntry(c.catalogMetadataPath(), entry, etag, "provider catalog")
}

func (c *Client) writeCacheEntry(bodyPath string, metadataPath string, body []byte, etag string, subject string) error {
	if c.cacheDir == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o755); err != nil {
		return fmt.Errorf("create cache directory for %s: %w", subject, err)
	}
	metadata := cacheMetadata{
		Version:        cacheMetadataVersion,
		UpstreamCommit: c.upstreamCommit,
		ETag:           etag,
		FetchedAt:      time.Now().UTC(),
		SHA256:         checksum(body),
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode cached %s metadata: %w", subject, err)
	}
	if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
		return fmt.Errorf("write cached %s body: %w", subject, err)
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		return fmt.Errorf("write cached %s metadata: %w", subject, err)
	}
	return nil
}

func (c *Client) templateCachePath(key string) string {
	return filepath.Join(c.cacheDir, "templates", url.PathEscape(key)+".gitignore")
}

func (c *Client) templateMetadataPath(key string) string {
	return filepath.Join(c.cacheDir, "templates", url.PathEscape(key)+".metadata.json")
}

func (c *Client) catalogCachePath() string {
	return filepath.Join(c.cacheDir, "catalog.json")
}

func (c *Client) catalogMetadataPath() string {
	return filepath.Join(c.cacheDir, "catalog.metadata.json")
}

func (c *Client) readCachedTemplateEntry(key string, requireFresh bool) (cacheEntry, error) {
	return c.readCacheEntry(c.templateCachePath(key), c.templateMetadataPath(key), fmt.Sprintf("template for provider %s", key), requireFresh)
}

func (c *Client) readCachedCatalog(requireFresh bool) (cacheEntry, error) {
	return c.readCacheEntry(c.catalogCachePath(), c.catalogMetadataPath(), "provider catalog", requireFresh)
}

func (c *Client) readCacheEntry(bodyPath string, metadataPath string, subject string, requireFresh bool) (cacheEntry, error) {
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cacheEntry{}, err
		}
		return cacheEntry{}, fmt.Errorf("read cached %s body: %w", subject, err)
	}
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cacheEntry{}, fmt.Errorf("cached %s metadata is missing", subject)
		}
		return cacheEntry{}, fmt.Errorf("read cached %s metadata: %w", subject, err)
	}

	var metadata cacheMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return cacheEntry{}, fmt.Errorf("decode cached %s metadata: %w", subject, err)
	}
	if metadata.Version != cacheMetadataVersion {
		return cacheEntry{}, fmt.Errorf("cached %s metadata version %d is unsupported", subject, metadata.Version)
	}
	if metadata.UpstreamCommit == "" {
		return cacheEntry{}, fmt.Errorf("cached %s is missing upstream commit metadata", subject)
	}
	if metadata.UpstreamCommit != c.upstreamCommit {
		return cacheEntry{}, fmt.Errorf("cached %s was created for upstream commit %s, expected %s", subject, metadata.UpstreamCommit, c.upstreamCommit)
	}
	if metadata.FetchedAt.IsZero() {
		return cacheEntry{}, fmt.Errorf("cached %s is missing fetched timestamp metadata", subject)
	}
	if metadata.SHA256 == "" {
		return cacheEntry{}, fmt.Errorf("cached %s is missing integrity metadata", subject)
	}
	if checksum(body) != metadata.SHA256 {
		return cacheEntry{}, fmt.Errorf("cached %s failed integrity validation", subject)
	}

	stale := time.Since(metadata.FetchedAt) > cacheFreshnessWindow
	if requireFresh && stale {
		return cacheEntry{}, fmt.Errorf("cached %s is stale; rerun without runtime.offline to refresh it", subject)
	}

	return cacheEntry{Body: body, Metadata: metadata, Stale: stale}, nil
}

func (c *Client) inspectTemplateCache(key string) CacheEntryStatus {
	entry, err := c.readCachedTemplateEntry(key, false)
	if err == nil {
		state := "fresh"
		if entry.Stale {
			state = "stale"
		}
		return CacheEntryStatus{Provider: key, State: state}
	}
	if _, statErr := os.Stat(c.templateCachePath(key)); os.IsNotExist(statErr) {
		return CacheEntryStatus{Provider: key, State: "missing"}
	}
	return CacheEntryStatus{Provider: key, State: "invalid", Detail: err.Error()}
}

func runtimeDecisions(offline bool, remoteProviders []string, customProviders []string) []string {
	decisions := []string{"supported providers are validated against the checked-in GitHub catalog snapshot plus embedded exceptions"}
	if len(remoteProviders) == 0 {
		return append(decisions, "no remote template fetch is required because the selection is satisfied entirely by embedded providers")
	}
	if offline {
		return append(decisions, "runtime.offline is enabled, so remote templates must come from the local cache without a live GitHub refresh")
	}
	decisions = append(decisions, "remote templates are fetched from github/gitignore and cached metadata can reuse ETags on unchanged responses")
	if len(customProviders) > 0 {
		decisions = append(decisions, "embedded providers are merged with remote templates after provider resolution")
	}
	return decisions
}

func (c *Client) refreshCacheEntry(metadataPath string, entry cacheEntry, etag string, subject string) error {
	if c.cacheDir == "" {
		return nil
	}
	metadata := entry.Metadata
	metadata.FetchedAt = time.Now().UTC()
	if etag != "" {
		metadata.ETag = etag
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode cached %s metadata: %w", subject, err)
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		return fmt.Errorf("write cached %s metadata: %w", subject, err)
	}
	return nil
}

func checksum(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%x", sum)
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
