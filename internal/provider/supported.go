package provider

import (
	"fmt"
	"slices"

	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/providercatalog"
)

var (
	SupportedKeys []string
	supportedSet  map[string]struct{}
	initErr       error
)

func init() {
	SupportedKeys = []string{}
	supportedSet = map[string]struct{}{}

	if err := customtemplate.InitError(); err != nil {
		initErr = fmt.Errorf("load embedded custom templates: %w", err)
		return
	}

	remoteSupportedKeys := providercatalog.RemoteSupportedKeys()
	remoteSet := make(map[string]struct{}, len(remoteSupportedKeys))
	for _, key := range remoteSupportedKeys {
		remoteSet[key] = struct{}{}
	}

	SupportedKeys = append([]string(nil), remoteSupportedKeys...)
	for _, key := range customtemplate.ProviderKeys() {
		if _, exists := remoteSet[key]; exists {
			initErr = fmt.Errorf("embedded custom template key collides with remote provider key: %s", key)
			SupportedKeys = []string{}
			supportedSet = map[string]struct{}{}
			return
		}
		SupportedKeys = append(SupportedKeys, key)
	}
	slices.Sort(SupportedKeys)
	SupportedKeys = slices.Compact(SupportedKeys)

	supportedSet = make(map[string]struct{}, len(SupportedKeys))
	for _, key := range SupportedKeys {
		supportedSet[key] = struct{}{}
	}
}

func InitError() error {
	return initErr
}

func RemoteSupportedKeys() []string {
	return providercatalog.RemoteSupportedKeys()
}

func IsSupported(key string) bool {
	_, ok := supportedSet[key]
	return ok
}
