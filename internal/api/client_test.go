package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/providercatalog"
)

func TestClientUsesFixtures(t *testing.T) {
	t.Parallel()
	templateFixture, err := os.ReadFile(filepath.Join("testdata", "template.txt"))
	if err != nil {
		t.Fatalf("read template fixture failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Go.gitignore","type":"blob"},{"path":"Node.gitignore","type":"blob"},{"path":"Global/macOS.gitignore","type":"blob"}]}`))
		case "/templates/Node.gitignore":
			_, _ = w.Write(templateFixture)
		case "/templates/Go.gitignore":
			_, _ = w.Write([]byte("bin/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	list, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	for _, key := range []string{"go", "macos", "node"} {
		if !slices.Contains(list, key) {
			t.Fatalf("expected canonical provider list to contain %q: %v", key, list)
		}
	}
	listAgain, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders second call failed: %v", err)
	}
	if !reflect.DeepEqual(listAgain, list) {
		t.Fatalf("expected deterministic list ordering, got %v then %v", list, listAgain)
	}

	template, err := client.FetchTemplate(context.Background(), []string{"node"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if template.Content != strings.TrimSpace(string(templateFixture)) {
		t.Fatalf("unexpected template content")
	}
}

func TestAvailableProvidersUsesCanonicalProviderCatalog(t *testing.T) {
	t.Parallel()

	client := NewClient()
	got, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	if !reflect.DeepEqual(got, providercatalog.RemoteSupportedKeys()) {
		t.Fatalf("unexpected canonical provider list")
	}
}

func TestFetchTemplateReturnsStableErrorOnNonOKResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Node.gitignore","type":"blob"}]}`))
		case "/templates/Node.gitignore":
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	_, err := client.FetchTemplate(context.Background(), []string{"node"})
	if err == nil || err.Error() != "template API returned status 502" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplateOfflineUsesCachedRemoteTemplate(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	live := NewClientWithOptions(Options{})
	live.cacheDir = cacheDir
	offline := NewClientWithOptions(Options{Offline: true})
	offline.cacheDir = cacheDir

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Node.gitignore","type":"blob"}]}`))
		case "/templates/Node.gitignore":
			_, _ = w.Write([]byte("node_modules/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	live.listURL = server.URL + "/catalog"
	live.templateURL = server.URL + "/templates/"

	if _, err := live.FetchTemplate(context.Background(), []string{"node"}); err != nil {
		t.Fatalf("live fetch failed: %v", err)
	}

	resp, err := offline.FetchTemplate(context.Background(), []string{"node"})
	if err != nil {
		t.Fatalf("offline fetch failed: %v", err)
	}
	if resp.Content != "node_modules/" {
		t.Fatalf("unexpected cached template content: %q", resp.Content)
	}
}

func TestFetchTemplateOfflineRequiresCachedRemoteTemplate(t *testing.T) {
	t.Parallel()

	client := NewClientWithOptions(Options{Offline: true})
	client.cacheDir = t.TempDir()

	_, err := client.FetchTemplate(context.Background(), []string{"node"})
	if err == nil || err.Error() != "offline mode requires cached template for provider: node" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplateSupportsEmbeddedCustomProviderWithoutRemoteRequest(t *testing.T) {
	t.Parallel()

	hitRemoteTemplate := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/templates/") {
			hitRemoteTemplate = true
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient()
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"ai-agents"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if hitRemoteTemplate {
		t.Fatalf("expected custom-only template fetch to skip remote API call")
	}
	if !strings.Contains(resp.Content, ".agents/") || !strings.Contains(resp.Content, ".claude/") || !strings.Contains(resp.Content, ".cursor/") {
		t.Fatalf("unexpected embedded custom template content: %q", resp.Content)
	}
}

func TestFetchTemplateSupportsWranglerEmbeddedCustomProviderWithoutRemoteRequest(t *testing.T) {
	t.Parallel()

	hitRemoteTemplate := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/templates/") {
			hitRemoteTemplate = true
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient()
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"wrangler"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if hitRemoteTemplate {
		t.Fatalf("expected wrangler-only template fetch to skip remote API call")
	}
	for _, fragment := range []string{".wrangler/", ".dev.vars*", "!.dev.vars.example"} {
		if !strings.Contains(resp.Content, fragment) {
			t.Fatalf("missing %q in wrangler template content: %q", fragment, resp.Content)
		}
	}
}

func TestFetchTemplateMergesRemoteAndEmbeddedCustomTemplates(t *testing.T) {
	t.Parallel()

	requestPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Go.gitignore","type":"blob"}]}`))
		case "/templates/Go.gitignore":
			requestPath = r.URL.Path
			_, _ = w.Write([]byte("bin/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"ai-agents", "go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if requestPath != "/templates/Go.gitignore" {
		t.Fatalf("expected remote request to include only remote providers, got %q", requestPath)
	}
	if !strings.Contains(resp.Content, "bin/") {
		t.Fatalf("expected remote template content in merge: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, ".agents/") {
		t.Fatalf("expected embedded custom template content in merge: %q", resp.Content)
	}
}

func TestFetchTemplateUsesSingleCatalogLookupForRemoteProviders(t *testing.T) {
	t.Parallel()

	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			catalogRequests++
			_, _ = w.Write([]byte(`{"tree":[{"path":"Go.gitignore","type":"blob"}]}`))
		case "/templates/Go.gitignore":
			_, _ = w.Write([]byte("bin/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"ai-agents", "go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if catalogRequests != 1 {
		t.Fatalf("expected one catalog request, got %d", catalogRequests)
	}
	if !reflect.DeepEqual(resp.AvailableProviders, []string{"go"}) {
		t.Fatalf("unexpected available providers: %v", resp.AvailableProviders)
	}
}

func TestFetchTemplateMergesRemoteAndWranglerEmbeddedTemplates(t *testing.T) {
	t.Parallel()

	requestPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Go.gitignore","type":"blob"}]}`))
		case "/templates/Go.gitignore":
			requestPath = r.URL.Path
			_, _ = w.Write([]byte("bin/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"go", "wrangler"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if requestPath != "/templates/Go.gitignore" {
		t.Fatalf("expected remote request to include only remote providers, got %q", requestPath)
	}
	if !strings.Contains(resp.Content, "bin/") {
		t.Fatalf("expected remote template content in merge: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, ".wrangler/") {
		t.Fatalf("expected wrangler template content in merge: %q", resp.Content)
	}
}

func TestFetchTemplateResolvesGlobalTemplatePathsAndPreservesRequestedOrder(t *testing.T) {
	t.Parallel()

	requestPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			_, _ = w.Write([]byte(`{"tree":[{"path":"Go.gitignore","type":"blob"},{"path":"Global/macOS.gitignore","type":"blob"}]}`))
		case "/templates/Global/macOS.gitignore":
			requestPaths = append(requestPaths, r.URL.Path)
			_, _ = w.Write([]byte(".DS_Store\n"))
		case "/templates/Go.gitignore":
			requestPaths = append(requestPaths, r.URL.Path)
			_, _ = w.Write([]byte("bin/\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/catalog"
	client.templateURL = server.URL + "/templates/"

	resp, err := client.FetchTemplate(context.Background(), []string{"macos", "go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if !reflect.DeepEqual(requestPaths, []string{"/templates/Global/macOS.gitignore", "/templates/Go.gitignore"}) {
		t.Fatalf("unexpected request order: %v", requestPaths)
	}
	if resp.Content != ".DS_Store\n\nbin/" {
		t.Fatalf("unexpected merged content: %q", resp.Content)
	}
}
