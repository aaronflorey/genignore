package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/provider"
)

func TestListProviders(t *testing.T) {
	t.Parallel()

	got := ListProviders()
	want := append([]string(nil), provider.SupportedKeys...)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected list providers output")
	}
}

func TestSearchProviders(t *testing.T) {
	t.Parallel()

	got := SearchProviders("go")
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

	got := SearchProviders("__no_match__")
	if len(got) != 0 {
		t.Fatalf("expected no provider matches, got %v", got)
	}
}
