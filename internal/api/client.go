// Package api retains its historical name for the embedded template/catalog
// loader used by the app service. It no longer performs runtime HTTP fetches;
// provider content comes from checked-in catalogs and embedded custom
// templates.
package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/providercatalog"
	"github.com/aaronflorey/genignore/internal/templatecatalog"
)

const DefaultUpstreamCommit = "dcc0fc7bc2b5ba480cf117ad1be31bafceeaff46"

type TemplateResponse struct {
	Providers          []string `json:"providers"`
	Content            string   `json:"content"`
	AvailableProviders []string `json:"-"`
}

type RuntimeDiagnostics struct {
	// UpstreamCommit and Offline are retained as internal compatibility fields so
	// diagnostics and provenance stay stable while template loading remains fully
	// embedded.
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
	// Offline and UpstreamCommit are compatibility-only inputs for diagnostics
	// and provenance. They do not enable runtime network retrieval.
	Offline        bool
	UpstreamCommit string
}

type Client struct {
	offline        bool
	upstreamCommit string
}

// EmbeddedClient loads provider metadata and template bodies from embedded
// catalogs. The api package name is kept as an internal compatibility detail
// so callers do not need broad package churn after runtime fetching was
// removed.
type EmbeddedClient struct {
	*Client
}

func NewClient() *Client {
	return NewClientWithOptions(Options{})
}

func NewClientWithOptions(opts Options) *Client {
	upstreamCommit := strings.TrimSpace(opts.UpstreamCommit)
	if upstreamCommit == "" {
		upstreamCommit = DefaultUpstreamCommit
	}

	return &Client{
		offline:        opts.Offline,
		upstreamCommit: upstreamCommit,
	}
}

func NewEmbeddedClientWithOptions(opts Options) *EmbeddedClient {
	return &EmbeddedClient{Client: NewClientWithOptions(opts)}
}

func (c *Client) AvailableProviders(context.Context) ([]string, error) {
	return providercatalog.RemoteSupportedKeys(), nil
}

func (c *Client) InspectRuntime(providers []string) RuntimeDiagnostics {
	remoteProviders, customProviders := splitProvidersBySource(providers)
	decisions := []string{"supported providers are validated against the checked-in GitHub catalog snapshot plus embedded exceptions"}
	if len(remoteProviders) == 0 {
		decisions = append(decisions, "no upstream template lookup is required because the selection is satisfied entirely by embedded providers")
	} else {
		decisions = append(decisions, "upstream templates are loaded from the checked-in github/gitignore snapshot")
	}
	if len(customProviders) > 0 {
		decisions = append(decisions, "embedded custom providers are merged with upstream templates in requested provider order")
	}
	if c.offline {
		decisions = append(decisions, "legacy offline compatibility settings do not change template loading because provider content is already checked in")
	}

	return RuntimeDiagnostics{
		UpstreamCommit:    c.upstreamCommit,
		Offline:           c.offline,
		RemoteProviders:   remoteProviders,
		EmbeddedProviders: customProviders,
		Decisions:         decisions,
	}
}

func (c *Client) FetchTemplate(_ context.Context, providers []string) (TemplateResponse, error) {
	if len(providers) == 0 {
		return TemplateResponse{}, fmt.Errorf("providers must not be empty")
	}

	parts := make([]string, 0, len(providers))
	for _, key := range providers {
		content, err := contentForProvider(key)
		if err != nil {
			return TemplateResponse{}, err
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, content)
	}

	return TemplateResponse{
		Providers:          providers,
		Content:            strings.Join(parts, "\n\n"),
		AvailableProviders: providercatalog.RemoteSupportedKeys(),
	}, nil
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

func contentForProvider(key string) (string, error) {
	if customtemplate.HasProvider(key) {
		return customtemplate.Content(key)
	}
	return templatecatalog.Content(key)
}
