package gitignore

import (
	"reflect"
	"strings"
	"testing"
)

func FuzzParseManagedProvidersRoundTrip(f *testing.F) {
	for _, seed := range []struct {
		before string
		csv    string
		after  string
	}{
		{"", "go,node", ""},
		{"# user rule\n", "macos", "\n*.log\n"},
		{"before\n", "terraform,windows", "after\n"},
	} {
		f.Add(seed.before, seed.csv, seed.after)
	}

	f.Fuzz(func(t *testing.T, before, csv, after string) {
		providers := fuzzProviders(csv)
		block := BuildManagedBlock(providers, "bin/\n")

		got := ParseManagedProviders(before + block + after)
		if !reflect.DeepEqual(got, providers) {
			t.Fatalf("expected providers %v, got %v", providers, got)
		}
	})
}

func FuzzMergeManagedBlock(f *testing.F) {
	block := BuildManagedBlock([]string{"go", "node"}, "bin/\nnode_modules/\n")

	for _, seed := range []string{
		"",
		"*.log\n",
		strings.Join([]string{StartMarker, "old", EndMarker, ""}, "\n"),
		strings.Join([]string{"before", StartMarker, "old", EndMarker, "after", ""}, "\n"),
		strings.Join([]string{"before", StartMarker, "old", "after", ""}, "\n"),
		strings.Join([]string{"before", StartMarker, "old", EndMarker, StartMarker, "duplicate", EndMarker, ""}, "\n"),
		strings.Join([]string{"before", EndMarker, "old", StartMarker, "after", ""}, "\n"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, existing string) {
		_, _, _, malformed := managedMarkerBounds(existing)

		updated, err := mergeManagedBlock(existing, block)
		if malformed {
			if err == nil {
				t.Fatalf("expected malformed input to fail")
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected merge error: %v", err)
		}

		_, _, hasMarkers, malformed := managedMarkerBounds(updated)
		if malformed || !hasMarkers {
			t.Fatalf("expected merged content to contain one managed block")
		}

		got := ParseManagedProviders(updated)
		if !reflect.DeepEqual(got, []string{"go", "node"}) {
			t.Fatalf("expected providers [go node], got %v", got)
		}
	})
}

func fuzzProviders(csv string) []string {
	fields := strings.FieldsFunc(csv, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r == '-', r == '_', r == '+':
			return false
		default:
			return true
		}
	})

	providers := make([]string, 0, len(fields))
	for _, field := range fields {
		provider := strings.ToLower(strings.TrimSpace(field))
		if provider == "" {
			continue
		}
		providers = append(providers, provider)
	}
	if len(providers) == 0 {
		return []string{"generic"}
	}
	return providers
}
