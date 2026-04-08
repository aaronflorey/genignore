package app

import (
	"slices"
	"strings"

	"github.com/aaronflorey/gitignore-gen/internal/provider"
)

func ListProviders() []string {
	providers := append([]string(nil), provider.SupportedKeys...)
	slices.Sort(providers)
	return providers
}

func SearchProviders(term string) []string {
	needle := strings.ToLower(term)
	providers := make([]string, 0)
	for _, key := range provider.SupportedKeys {
		if strings.Contains(strings.ToLower(key), needle) {
			providers = append(providers, key)
		}
	}
	slices.Sort(providers)
	return providers
}
