package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClientUsesFixtures(t *testing.T) {
	t.Parallel()
	listFixture, err := os.ReadFile(filepath.Join("testdata", "list.json"))
	if err != nil {
		t.Fatalf("read list fixture failed: %v", err)
	}
	templateFixture, err := os.ReadFile(filepath.Join("testdata", "template.txt"))
	if err != nil {
		t.Fatalf("read template fixture failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/list":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(listFixture)
		case "/templates/node,go":
			_, _ = w.Write(templateFixture)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL + "/list"
	client.templateURL = server.URL + "/templates/"

	list, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	if !reflect.DeepEqual(list, []string{"go", "macos", "node"}) {
		t.Fatalf("unexpected list: %v", list)
	}
	listAgain, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders second call failed: %v", err)
	}
	if !reflect.DeepEqual(listAgain, list) {
		t.Fatalf("expected deterministic list ordering, got %v then %v", list, listAgain)
	}

	template, err := client.FetchTemplate(context.Background(), []string{"node", "go"})
	if err != nil {
		t.Fatalf("FetchTemplate failed: %v", err)
	}
	if template.Content != string(templateFixture) {
		t.Fatalf("unexpected template content")
	}
}

func TestAvailableProvidersReturnsStableErrorOnNonOKResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL

	_, err := client.AvailableProviders(context.Background())
	if err == nil || err.Error() != "list API returned status 502" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplateReturnsStableErrorOnNonOKResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/templates/node" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient()
	client.templateURL = server.URL + "/templates/"

	_, err := client.FetchTemplate(context.Background(), []string{"node"})
	if err == nil || err.Error() != "template API returned status 502" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAvailableProvidersSupportsLegacyListShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gitignores":["node","go"]}`))
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL

	providers, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	if !reflect.DeepEqual(providers, []string{"go", "node"}) {
		t.Fatalf("unexpected providers: %v", providers)
	}
}

func TestAvailableProvidersSupportsEmptyLegacyListShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gitignores":[]}`))
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL

	providers, err := client.AvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("AvailableProviders failed: %v", err)
	}
	if !reflect.DeepEqual(providers, []string{}) {
		t.Fatalf("unexpected providers: %v", providers)
	}
}

func TestAvailableProvidersRejectsInvalidResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient()
	client.listURL = server.URL

	_, err := client.AvailableProviders(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decode list API response") {
		t.Fatalf("unexpected error: %v", err)
	}
}
