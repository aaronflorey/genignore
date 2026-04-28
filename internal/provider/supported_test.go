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

func TestRemoteSupportedKeysExcludeEmbeddedCustomProviders(t *testing.T) {
	t.Parallel()

	if slices.Contains(RemoteSupportedKeys(), "ai-agents") {
		t.Fatalf("expected remote provider list to exclude embedded custom provider key")
	}
	if slices.Contains(RemoteSupportedKeys(), "wrangler") {
		t.Fatalf("expected remote provider list to exclude wrangler embedded custom provider key")
	}
}
