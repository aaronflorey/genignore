package provider

import (
	"slices"
	"testing"
)

func TestEmbeddedCustomProviderIsSupported(t *testing.T) {
	t.Parallel()

	if !IsSupported("ai-agents") {
		t.Fatalf("expected embedded custom provider key to be supported")
	}
	if !IsSupported("wrangler") {
		t.Fatalf("expected wrangler embedded custom provider key to be supported")
	}
}

func TestGitHubBackedKeysAreSupported(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"go", "macos", "nextjs", "visualstudiocode"} {
		if !IsSupported(key) {
			t.Fatalf("expected GitHub-backed provider key %q to be supported", key)
		}
	}

	for _, key := range []string{"react", "dotnetcore", "androidstudio"} {
		if IsSupported(key) {
			t.Fatalf("expected legacy non-GitHub provider key %q to be unsupported", key)
		}
	}
}

func TestAllSupportedKeysIncludesEmbeddedUpstreamAndCustomProviders(t *testing.T) {
	t.Parallel()

	got := AllSupportedKeys()
	if len(got) == 0 {
		t.Fatalf("AllSupportedKeys() returned no providers")
	}
	if !slices.IsSorted(got) {
		t.Fatalf("AllSupportedKeys() not sorted: %v", got)
	}
	for _, key := range []string{"ai-agents", "go", "macos", "wrangler"} {
		if !slices.Contains(got, key) {
			t.Fatalf("AllSupportedKeys() missing %q", key)
		}
	}
}

func TestAllSupportedKeysReturnsCopy(t *testing.T) {
	t.Parallel()

	got := AllSupportedKeys()
	want := append([]string(nil), got...)
	got[0] = "mutated"

	if !slices.Equal(AllSupportedKeys(), want) {
		t.Fatalf("AllSupportedKeys() changed after caller mutation")
	}
}

func TestRemoteSupportedKeysExcludeEmbeddedCustomProviders(t *testing.T) {
	t.Parallel()

	if slices.Contains(RemoteSupportedKeys(), "ai-agents") {
		t.Fatalf("expected remote provider list to exclude embedded custom provider key")
	}
	if slices.Contains(RemoteSupportedKeys(), "wrangler") {
		t.Fatalf("expected remote provider list to exclude wrangler embedded custom provider key")
	}
}
