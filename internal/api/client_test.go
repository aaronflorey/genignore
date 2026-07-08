package api

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/providercatalog"
)

func TestNewClientWithOptionsPreservesDiagnosticsConfig(t *testing.T) {
	t.Parallel()

	client := NewClientWithOptions(Options{Offline: true, UpstreamCommit: "1234567890abcdef1234567890abcdef12345678"})
	if !client.offline {
		t.Fatal("expected offline flag to be preserved for diagnostics")
	}
	if client.upstreamCommit != "1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected upstream commit: %q", client.upstreamCommit)
	}
	if _, ok := any(NewEmbeddedClientWithOptions(Options{})).(*EmbeddedClient); !ok {
		t.Fatal("expected embedded client constructor to remain available")
	}
}

func TestDefaultUpstreamCommitMatchesEmbeddedCatalogPin(t *testing.T) {
	t.Parallel()

	const submoduleCommit = "dcc0fc7bc2b5ba480cf117ad1be31bafceeaff46"
	if DefaultUpstreamCommit != submoduleCommit {
		t.Fatalf("DefaultUpstreamCommit = %q, want %q", DefaultUpstreamCommit, submoduleCommit)
	}
}

func TestAvailableProvidersUsesCanonicalProviderCatalog(t *testing.T) {
	t.Parallel()

	client := NewClientWithOptions(Options{UpstreamCommit: "1234567890abcdef1234567890abcdef12345678"})
	got, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	if !slices.Equal(got, providercatalog.RemoteSupportedKeys()) {
		t.Fatalf("unexpected canonical provider list")
	}
	gotAgain, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders second call failed: %v", err)
	}
	if !slices.Equal(gotAgain, got) {
		t.Fatalf("expected deterministic list ordering, got %v then %v", got, gotAgain)
	}
}

func TestFetchTemplateUsesEmbeddedUpstreamTemplateWithoutHTTPServer(t *testing.T) {
	t.Parallel()

	client := NewClientWithOptions(Options{UpstreamCommit: "1234567890abcdef1234567890abcdef12345678"})

	resp, err := client.FetchTemplate(context.Background(), []string{"go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if !strings.Contains(resp.Content, "go.work") {
		t.Fatalf("unexpected embedded upstream template content: %q", resp.Content)
	}
	if !slices.Equal(resp.AvailableProviders, providercatalog.RemoteSupportedKeys()) {
		t.Fatalf("unexpected available providers: %v", resp.AvailableProviders)
	}
}

func TestFetchTemplatePreservesRequestedProviderOrderAcrossSources(t *testing.T) {
	t.Parallel()

	client := NewClient()

	resp, err := client.FetchTemplate(context.Background(), []string{"wrangler", "go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if !strings.HasPrefix(resp.Content, "# Cloudflare Wrangler\n.wrangler/") {
		t.Fatalf("expected custom template first in merged content: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "\n\n# If you prefer the allow list template") {
		t.Fatalf("expected embedded upstream template after custom template: %q", resp.Content)
	}
	if !slices.Equal(resp.Providers, []string{"wrangler", "go"}) {
		t.Fatalf("unexpected provider order: %v", resp.Providers)
	}
}

func TestFetchTemplateSupportsCustomOnlySelection(t *testing.T) {
	t.Parallel()

	resp, err := NewClient().FetchTemplate(context.Background(), []string{"ai-agents"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if !strings.Contains(resp.Content, ".agents/") || !strings.Contains(resp.Content, ".claude/") || !strings.Contains(resp.Content, ".cursor/") {
		t.Fatalf("unexpected embedded custom template content: %q", resp.Content)
	}
}

func TestFetchTemplateRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	_, err := NewClient().FetchTemplate(context.Background(), []string{"definitely-not-a-provider"})
	if err == nil || err.Error() != "embedded upstream template not found: definitely-not-a-provider" {
		t.Fatalf("unexpected error: %v", err)
	}
}
