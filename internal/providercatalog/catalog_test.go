package providercatalog

import (
	"slices"
	"testing"

	"github.com/aaronflorey/genignore/internal/templatecatalog"
)

func TestInitError(t *testing.T) {
	t.Parallel()

	if err := InitError(); err != nil {
		t.Fatalf("InitError() = %v", err)
	}
}

func TestRemoteSupportedKeysMatchesTemplateCatalog(t *testing.T) {
	t.Parallel()

	got := RemoteSupportedKeys()
	want := templatecatalog.Providers()
	if !slices.Equal(got, want) {
		t.Fatalf("RemoteSupportedKeys() = %v, want %v", got, want)
	}

	if len(got) == 0 {
		t.Fatalf("RemoteSupportedKeys() returned no providers")
	}
	if !slices.Contains(got, "go") {
		t.Fatalf("RemoteSupportedKeys() missing go")
	}
	if !slices.Contains(got, "global/al") {
		t.Fatalf("RemoteSupportedKeys() missing global/al")
	}
	if !slices.Contains(got, "javascript/vue") {
		t.Fatalf("RemoteSupportedKeys() missing javascript/vue")
	}
	if !slices.IsSorted(got) {
		t.Fatalf("RemoteSupportedKeys() not sorted: %v", got)
	}

	got[0] = "mutated"
	if slices.Equal(got, RemoteSupportedKeys()) {
		t.Fatalf("RemoteSupportedKeys() returned mutable shared backing slice")
	}
	if slices.Equal(got, templatecatalog.Providers()) {
		t.Fatalf("templatecatalog.Providers() changed after mutating providercatalog result")
	}
	if RemoteSupportedKeys()[0] != want[0] {
		t.Fatalf("RemoteSupportedKeys() changed after mutation")
	}
	if templatecatalog.Providers()[0] != want[0] {
		t.Fatalf("templatecatalog.Providers() changed after mutation")
	}
}
