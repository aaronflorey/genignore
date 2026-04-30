package app

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestListProviders(t *testing.T) {
	t.Parallel()

	got, err := ListProviders(context.Background(), stubCatalogClient{providers: []string{"go", "macos", "node"}})
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	want := []string{"ai-agents", "go", "macos", "node", "wrangler"}
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected list providers output")
	}
}

func TestSearchProviders(t *testing.T) {
	t.Parallel()

	got, err := SearchProviders(context.Background(), stubCatalogClient{providers: []string{"go", "goland", "macos", "node"}}, "go")
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

	got, err := SearchProviders(context.Background(), stubCatalogClient{providers: []string{"go", "macos", "node"}}, "__no_match__")
	if err != nil {
		t.Fatalf("SearchProviders failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no provider matches, got %v", got)
	}
}
