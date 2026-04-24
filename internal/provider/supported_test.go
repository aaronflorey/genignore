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

func TestRemoteSupportedKeysExcludeEmbeddedCustomProviders(t *testing.T) {
	t.Parallel()

	if slices.Contains(RemoteSupportedKeys(), "ai-agents") {
		t.Fatalf("expected remote provider list to exclude embedded custom provider key")
	}
	if slices.Contains(RemoteSupportedKeys(), "wrangler") {
		t.Fatalf("expected remote provider list to exclude wrangler embedded custom provider key")
	}
}
