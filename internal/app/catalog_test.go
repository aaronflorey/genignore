package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/provider"
)

func TestListProviders(t *testing.T) {
	t.Parallel()

	got, err := ListProviders(context.Background(), stubCatalogClient{providers: []string{"go", "macos", "node"}})
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	want := provider.AllSupportedKeys()

	if !slices.Equal(got, want) {
		t.Fatalf("ListProviders() = %v, want %v", got, want)
	}
}

func TestListProvidersIgnoresCatalogClientState(t *testing.T) {
	t.Parallel()

	got, err := ListProviders(context.Background(), stubCatalogClient{err: errors.New("network unavailable")})
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}

	if !slices.Equal(got, provider.AllSupportedKeys()) {
		t.Fatalf("ListProviders() = %v, want embedded supported providers", got)
	}
}

func TestSearchProviders(t *testing.T) {
	t.Parallel()

	got, err := SearchProviders(context.Background(), stubCatalogClient{err: errors.New("network unavailable")}, "go")
	if err != nil {
		t.Fatalf("SearchProviders failed: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least one provider match")
	}
	if !slices.IsSorted(got) {
		t.Fatalf("expected sorted provider matches")
	}
	for _, key := range got {
		if !strings.Contains(strings.ToLower(key), "go") {
			t.Fatalf("provider %q did not match query", key)
		}
	}
}

func TestSearchProvidersNoMatches(t *testing.T) {
	t.Parallel()

	got, err := SearchProviders(context.Background(), stubCatalogClient{err: errors.New("network unavailable")}, "__no_match__")
	if err != nil {
		t.Fatalf("SearchProviders failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no provider matches, got %v", got)
	}
}

func TestSanitizeKeysUsesEmbeddedSupportedProviders(t *testing.T) {
	t.Parallel()

	got, warnings := sanitizeKeys([]string{"wrangler", "go", "unsupported", "go"})
	want := []string{"go", "wrangler"}

	if !slices.Equal(got, want) {
		t.Fatalf("sanitizeKeys() keys = %v, want %v", got, want)
	}
	if !slices.Equal(warnings, []string{"unsupported provider key: unsupported"}) {
		t.Fatalf("sanitizeKeys() warnings = %v", warnings)
	}
}
