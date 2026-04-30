package customtemplate

import (
	"embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed templates/*.gitignore
var templateFS embed.FS

var (
	providerKeys []string
	byKey        map[string]string
	initErr      error
)

func init() {
	providerKeys = []string{}
	byKey = map[string]string{}

	keys, templates, err := loadDefinitions()
	if err != nil {
		initErr = err
		return
	}
	providerKeys = keys
	byKey = templates
}

func InitError() error {
	return initErr
}

func ProviderKeys() []string {
	return append([]string(nil), providerKeys...)
}

func HasProvider(key string) bool {
	_, ok := byKey[key]
	return ok
}

func ContentForProviders(keys []string) (string, error) {
	if initErr != nil {
		return "", initErr
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		content, ok := byKey[key]
		if !ok {
			return "", fmt.Errorf("embedded custom template not found: %s", key)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n"), nil
}

func loadDefinitions() ([]string, map[string]string, error) {
	keys := make([]string, 0, len(Definitions))
	templates := make(map[string]string, len(Definitions))
	for _, def := range Definitions {
		if def.Key == "" {
			return nil, nil, fmt.Errorf("embedded custom template has empty key")
		}
		if def.Path == "" {
			return nil, nil, fmt.Errorf("embedded custom template path is empty for key %s", def.Key)
		}
		if _, exists := templates[def.Key]; exists {
			return nil, nil, fmt.Errorf("duplicate embedded custom template key: %s", def.Key)
		}

		raw, err := templateFS.ReadFile(def.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("read embedded custom template %q: %w", def.Path, err)
		}

		templates[def.Key] = strings.Trim(string(raw), "\n")
		keys = append(keys, def.Key)
	}

	slices.Sort(keys)
	return keys, templates, nil
}
