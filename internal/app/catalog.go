package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/provider"
)

type providerCatalog interface {
	AvailableProviders(ctx context.Context) ([]string, error)
}

func ListProviders(ctx context.Context, client providerCatalog) ([]string, error) {
	return supportedProviders()
}

func SearchProviders(ctx context.Context, client providerCatalog, term string) ([]string, error) {
	providers, err := supportedProviders()
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(term)
	filtered := make([]string, 0)
	for _, key := range providers {
		if strings.Contains(strings.ToLower(key), needle) {
			filtered = append(filtered, key)
		}
	}
	slices.Sort(filtered)
	return filtered, nil
}

func supportedProviders() ([]string, error) {
	if err := runtimeInitError(); err != nil {
		return nil, err
	}

	return provider.AllSupportedKeys(), nil
}

func runtimeInitError() error {
	if err := customtemplate.InitError(); err != nil {
		return fmt.Errorf("initialize embedded templates: %w", err)
	}
	if err := provider.InitError(); err != nil {
		return fmt.Errorf("initialize provider registry: %w", err)
	}
	return nil
}
