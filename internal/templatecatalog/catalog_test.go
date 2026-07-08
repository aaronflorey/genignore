package templatecatalog

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInitError(t *testing.T) {
	t.Parallel()

	if err := InitError(); err != nil {
		t.Fatalf("InitError() = %v", err)
	}
}

func TestProvidersAreSorted(t *testing.T) {
	t.Parallel()

	got := Providers()
	if len(got) == 0 {
		t.Fatalf("Providers() returned no providers")
	}
	if !slices.IsSorted(got) {
		t.Fatalf("Providers() not sorted: %v", got)
	}
	if !slices.Contains(got, "go") {
		t.Fatalf("Providers() missing go")
	}
	if !slices.Contains(got, "macos") {
		t.Fatalf("Providers() missing macos")
	}
	if !slices.Contains(got, "global/al") {
		t.Fatalf("Providers() missing global/al fallback key")
	}
	if !slices.Contains(got, "javascript/vue") {
		t.Fatalf("Providers() missing javascript/vue")
	}
}

func TestEmbeddedCatalogInvariants(t *testing.T) {
	t.Parallel()

	if err := InitError(); err != nil {
		t.Fatalf("InitError() = %v", err)
	}

	got := Providers()
	if len(got) == 0 {
		t.Fatalf("Providers() returned no providers")
	}
	if !slices.IsSorted(got) {
		t.Fatalf("Providers() not sorted: %v", got)
	}
	if len(got) != len(byPath) {
		t.Fatalf("len(Providers()) = %d, len(byPath) = %d", len(got), len(byPath))
	}
	if len(got) != len(byContent) {
		t.Fatalf("len(Providers()) = %d, len(byContent) = %d", len(got), len(byContent))
	}

	seen := make(map[string]struct{}, len(got))
	var rootCount, globalCount, communityCount int
	for _, provider := range got {
		if provider == "" {
			t.Fatalf("Providers() contains empty provider key")
		}
		if _, exists := seen[provider]; exists {
			t.Fatalf("Providers() contains duplicate provider key %q", provider)
		}
		seen[provider] = struct{}{}

		path, err := Path(provider)
		if err != nil {
			t.Fatalf("Path(%q) failed: %v", provider, err)
		}
		if path == "" {
			t.Fatalf("Path(%q) returned empty path", provider)
		}

		content, err := Content(provider)
		if err != nil {
			t.Fatalf("Content(%q) failed: %v", provider, err)
		}
		if strings.TrimSpace(content) == "" {
			t.Fatalf("Content(%q) returned empty content", provider)
		}

		switch {
		case strings.HasPrefix(path, "community/"):
			communityCount++
		case strings.HasPrefix(path, "Global/"):
			globalCount++
		case !strings.Contains(path, "/"):
			rootCount++
		default:
			t.Fatalf("Path(%q) returned unsupported template path %q", provider, path)
		}
	}

	if rootCount == 0 {
		t.Fatalf("Providers() missing root template coverage")
	}
	if globalCount == 0 {
		t.Fatalf("Providers() missing Global/ template coverage")
	}
	if communityCount == 0 {
		t.Fatalf("Providers() missing community/ template coverage")
	}
}

func TestEmbeddedCatalogMatchesSubmoduleWalk(t *testing.T) {
	t.Parallel()

	if err := InitError(); err != nil {
		t.Fatalf("InitError() = %v", err)
	}

	diskProviders, diskPaths, diskContents, err := loadEmbedEligibleFromDisk("github-gitignore")
	if err != nil {
		t.Fatalf("loadEmbedEligibleFromDisk(%q) failed: %v", "github-gitignore", err)
	}

	embeddedCount, err := countTemplateFiles(templateFS, "github-gitignore")
	if err != nil {
		t.Fatalf("countTemplateFiles(templateFS) failed: %v", err)
	}
	diskCount, err := countTemplateFiles(os.DirFS("."), "github-gitignore")
	if err != nil {
		t.Fatalf("countTemplateFiles(os.DirFS(%q)) failed: %v", ".", err)
	}

	got := Providers()
	if !slices.Equal(got, diskProviders) {
		t.Fatalf("Providers() mismatch with submodule walk\n got: %v\nwant: %v", got, diskProviders)
	}
	if embeddedCount != len(got) {
		t.Fatalf("embedded template count = %d, len(Providers()) = %d", embeddedCount, len(got))
	}
	if diskCount != len(got) {
		t.Fatalf("submodule template count = %d, len(Providers()) = %d", diskCount, len(got))
	}

	for _, provider := range got {
		path, err := Path(provider)
		if err != nil {
			t.Fatalf("Path(%q) failed: %v", provider, err)
		}
		if path != diskPaths[provider] {
			t.Fatalf("Path(%q) = %q, want %q", provider, path, diskPaths[provider])
		}

		content, err := Content(provider)
		if err != nil {
			t.Fatalf("Content(%q) failed: %v", provider, err)
		}
		if content != diskContents[provider] {
			t.Fatalf("Content(%q) mismatch with submodule file", provider)
		}
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	got, err := Path("go")
	if err != nil {
		t.Fatalf("Path(go) failed: %v", err)
	}
	if got != "Go.gitignore" {
		t.Fatalf("Path(go) = %q, want %q", got, "Go.gitignore")
	}

	got, err = Path("macos")
	if err != nil {
		t.Fatalf("Path(macos) failed: %v", err)
	}
	if got != "Global/macOS.gitignore" {
		t.Fatalf("Path(macos) = %q, want %q", got, "Global/macOS.gitignore")
	}

	got, err = Path("global/al")
	if err != nil {
		t.Fatalf("Path(global/al) failed: %v", err)
	}
	if got != "Global/AL.gitignore" {
		t.Fatalf("Path(global/al) = %q, want %q", got, "Global/AL.gitignore")
	}

	got, err = Path("javascript/vue")
	if err != nil {
		t.Fatalf("Path(javascript/vue) failed: %v", err)
	}
	if got != "community/JavaScript/Vue.gitignore" {
		t.Fatalf("Path(javascript/vue) = %q, want %q", got, "community/JavaScript/Vue.gitignore")
	}
}

func TestContent(t *testing.T) {
	t.Parallel()

	got, err := Content("go")
	if err != nil {
		t.Fatalf("Content(go) failed: %v", err)
	}
	if !strings.Contains(got, "*.test") {
		t.Fatalf("Content(go) missing expected test ignore rule")
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatalf("Content(go) should be trimmed of trailing newlines")
	}

	got, err = Content("macos")
	if err != nil {
		t.Fatalf("Content(macos) failed: %v", err)
	}
	if !strings.Contains(got, ".DS_Store") {
		t.Fatalf("Content(macos) missing expected macOS ignore rule")
	}

	got, err = Content("global/al")
	if err != nil {
		t.Fatalf("Content(global/al) failed: %v", err)
	}
	if !strings.Contains(got, "*.code-workspace") {
		t.Fatalf("Content(global/al) missing expected AL ignore rule")
	}

	got, err = Content("javascript/vue")
	if err != nil {
		t.Fatalf("Content(javascript/vue) failed: %v", err)
	}
	if !strings.Contains(got, "docs/_book") {
		t.Fatalf("Content(javascript/vue) missing expected Vue ignore rule")
	}
}

func TestUnknownProvider(t *testing.T) {
	t.Parallel()

	if _, err := Path("__missing__"); err == nil {
		t.Fatalf("Path(__missing__) expected error")
	}
	if _, err := Content("__missing__"); err == nil {
		t.Fatalf("Content(__missing__) expected error")
	}
	if _, err := Path("vue"); err == nil {
		t.Fatalf("Path(vue) expected error for non-derived community key")
	}
}

func TestProviderKeyExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{path: "Go.gitignore", want: "go"},
		{path: "Global/macOS.gitignore", want: "macos"},
		{path: "Global/AL.gitignore", want: "global/al"},
		{path: "community/JavaScript/Vue.gitignore", want: "javascript/vue"},
	}
	rootKeys := rootTemplateKeys([]string{"AL.gitignore", "Go.gitignore"})

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got, err := providerKey(tt.path, rootKeys)
			if err != nil {
				t.Fatalf("providerKey(%q) failed: %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("providerKey(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadFromFSRejectsDuplicateDerivedKeys(t *testing.T) {
	t.Parallel()

	_, _, _, err := loadFromFS(fstest.MapFS{
		"root/AL.gitignore":                  &fstest.MapFile{Data: []byte("bin\n")},
		"root/Global/AL.gitignore":           &fstest.MapFile{Data: []byte("pkg\n")},
		"root/community/global/AL.gitignore": &fstest.MapFile{Data: []byte("obj\n")},
	}, "root")
	if err == nil {
		t.Fatalf("loadFromFS() expected duplicate provider key error")
	}
	if !strings.Contains(err.Error(), "duplicate embedded upstream template provider \"global/al\"") {
		t.Fatalf("loadFromFS() error = %v, want duplicate provider key", err)
	}
}

func TestLoadFromFSSortsProviders(t *testing.T) {
	t.Parallel()

	got, _, _, err := loadFromFS(fstest.MapFS{
		"root/Global/macOS.gitignore":             &fstest.MapFile{Data: []byte(".DS_Store\n")},
		"root/community/JavaScript/Vue.gitignore": &fstest.MapFile{Data: []byte("node_modules\n")},
		"root/Go.gitignore":                       &fstest.MapFile{Data: []byte("*.test\n")},
	}, "root")
	if err != nil {
		t.Fatalf("loadFromFS() failed: %v", err)
	}

	want := []string{"go", "javascript/vue", "macos"}
	if !slices.Equal(got, want) {
		t.Fatalf("loadFromFS() providers = %v, want %v", got, want)
	}
}

func TestLoadFromFSSkipsUnsupportedPaths(t *testing.T) {
	t.Parallel()

	got, paths, contents, err := loadFromFS(fstest.MapFS{
		"root/community/JavaScript/Vue.gitignore":         &fstest.MapFile{Data: []byte("node_modules\n")},
		"root/community/JavaScript/nested/Skip.gitignore": &fstest.MapFile{Data: []byte("skip\n")},
		"root/docs/readme.md":                             &fstest.MapFile{Data: []byte("docs\n")},
	}, "root")
	if err != nil {
		t.Fatalf("loadFromFS() failed: %v", err)
	}

	if !slices.Equal(got, []string{"javascript/vue"}) {
		t.Fatalf("loadFromFS() providers = %v, want [javascript/vue]", got)
	}
	if _, ok := paths["javascript/vue"]; !ok {
		t.Fatalf("loadFromFS() missing path for javascript/vue")
	}
	if _, ok := contents["javascript/vue"]; !ok {
		t.Fatalf("loadFromFS() missing content for javascript/vue")
	}
}

func countTemplateFiles(rootFS fs.FS, root string) (int, error) {
	count := 0
	err := fs.WalkDir(rootFS, root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		relativePath, ok := strings.CutPrefix(filePath, root+"/")
		if ok && isTemplatePath(relativePath) {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func loadEmbedEligibleFromDisk(root string) ([]string, map[string]string, map[string]string, error) {
	paths := make(map[string]string)
	contents := make(map[string]string)
	templatePaths := []string{}

	err := fs.WalkDir(os.DirFS("."), root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		relativePath, ok := strings.CutPrefix(filePath, root+"/")
		if !ok || !isTemplatePath(relativePath) {
			return nil
		}

		templatePaths = append(templatePaths, relativePath)
		raw, err := fs.ReadFile(os.DirFS("."), filePath)
		if err != nil {
			return err
		}
		contents[relativePath] = strings.Trim(string(raw), "\n")
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	slices.Sort(templatePaths)
	rootKeys := rootTemplateKeys(templatePaths)
	providers := make([]string, 0, len(templatePaths))
	for _, relativePath := range templatePaths {
		provider, err := providerKey(relativePath, rootKeys)
		if err != nil {
			return nil, nil, nil, err
		}
		if existingPath, exists := paths[provider]; exists {
			return nil, nil, nil, fmt.Errorf("duplicate embedded upstream template provider %q for %q and %q", provider, existingPath, relativePath)
		}
		providers = append(providers, provider)
		paths[provider] = relativePath
		contents[provider] = contents[relativePath]
		delete(contents, relativePath)
	}
	slices.Sort(providers)

	return providers, paths, contents, nil
}
