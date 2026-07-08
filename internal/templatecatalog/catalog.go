package templatecatalog

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

//go:embed github-gitignore
var templateFS embed.FS

var (
	providers []string
	byPath    map[string]string
	byContent map[string]string
	initErr   error
)

func init() {
	providers = []string{}
	byPath = map[string]string{}
	byContent = map[string]string{}

	keys, paths, contents, err := load()
	if err != nil {
		initErr = err
		return
	}

	providers = keys
	byPath = paths
	byContent = contents
}

func InitError() error {
	return initErr
}

func Providers() []string {
	return append([]string(nil), providers...)
}

func HasProvider(provider string) bool {
	_, ok := byPath[provider]
	return ok
}

func Path(provider string) (string, error) {
	if initErr != nil {
		return "", initErr
	}

	templatePath, ok := byPath[provider]
	if !ok {
		return "", fmt.Errorf("embedded upstream template not found: %s", provider)
	}
	return templatePath, nil
}

func Content(provider string) (string, error) {
	if initErr != nil {
		return "", initErr
	}

	content, ok := byContent[provider]
	if !ok {
		return "", fmt.Errorf("embedded upstream template not found: %s", provider)
	}
	return content, nil
}

func load() ([]string, map[string]string, map[string]string, error) {
	return loadFromFS(templateFS, "github-gitignore")
}

func loadFromFS(rootFS fs.FS, root string) ([]string, map[string]string, map[string]string, error) {
	templatePaths := []string{}

	err := fs.WalkDir(rootFS, root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, ok := strings.CutPrefix(filePath, root+"/")
		if !ok || !isTemplatePath(relativePath) {
			return nil
		}

		templatePaths = append(templatePaths, relativePath)
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	paths, err := ProviderPaths(templatePaths)
	if err != nil {
		return nil, nil, nil, err
	}
	contents := make(map[string]string, len(paths))

	for provider, relativePath := range paths {
		raw, err := fs.ReadFile(rootFS, root+"/"+relativePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read embedded upstream template %q: %w", relativePath, err)
		}
		contents[provider] = strings.Trim(string(raw), "\n")
	}
	if len(paths) == 0 {
		return nil, nil, nil, fmt.Errorf("no embedded upstream templates found")
	}

	keys := make([]string, 0, len(paths))
	for provider := range paths {
		keys = append(keys, provider)
	}
	slices.Sort(keys)

	return keys, paths, contents, nil
}

// ProviderPaths applies the embedded catalog path eligibility and provider-key
// derivation rules to a set of upstream template paths.
func ProviderPaths(templatePaths []string) (map[string]string, error) {
	eligiblePaths := make([]string, 0, len(templatePaths))
	for _, templatePath := range templatePaths {
		if isTemplatePath(templatePath) {
			eligiblePaths = append(eligiblePaths, templatePath)
		}
	}

	slices.Sort(eligiblePaths)
	rootKeys := rootTemplateKeys(eligiblePaths)
	paths := make(map[string]string, len(eligiblePaths))
	for _, templatePath := range eligiblePaths {
		provider, err := providerKey(templatePath, rootKeys)
		if err != nil {
			return nil, err
		}
		if existingPath, exists := paths[provider]; exists {
			return nil, fmt.Errorf("duplicate embedded upstream template provider %q for %q and %q", provider, existingPath, templatePath)
		}
		paths[provider] = templatePath
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no embedded upstream templates found")
	}

	return paths, nil
}

func isTemplatePath(templatePath string) bool {
	if !strings.HasSuffix(templatePath, ".gitignore") {
		return false
	}
	if !strings.Contains(templatePath, "/") {
		return true
	}
	parts := strings.Split(templatePath, "/")
	if len(parts) == 2 && parts[0] == "Global" {
		return true
	}
	return len(parts) == 3 && parts[0] == "community"
}

func providerKey(templatePath string, rootKeys map[string]struct{}) (string, error) {
	if !isTemplatePath(templatePath) {
		return "", fmt.Errorf("unsupported embedded upstream template path %q", templatePath)
	}

	parts := strings.Split(templatePath, "/")
	name := strings.TrimSuffix(parts[len(parts)-1], ".gitignore")
	if len(parts) == 3 {
		name = parts[1] + "/" + name
	} else if len(parts) == 2 && parts[0] == "Global" {
		key := strings.ToLower(name)
		if _, exists := rootKeys[key]; exists {
			name = parts[0] + "/" + name
		}
	}

	return strings.ToLower(name), nil
}

func rootTemplateKeys(templatePaths []string) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, templatePath := range templatePaths {
		if strings.Contains(templatePath, "/") {
			continue
		}
		keys[strings.ToLower(strings.TrimSuffix(templatePath, ".gitignore"))] = struct{}{}
	}
	return keys
}
