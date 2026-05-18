package app

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/aaronflorey/genignore/internal/api"
	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

type APIClient interface {
	AvailableProviders(ctx context.Context) ([]string, error)
	FetchTemplate(ctx context.Context, providers []string) (api.TemplateResponse, error)
}

type Service struct {
	CWD       string
	Config    Config
	Client    APIClient
	Manager   *gitignore.Manager
	Detectors map[string]provider.Detector
}

type namedDetector struct {
	key      string
	detector provider.Detector
}

type DetectOptions struct {
	Include []string
	Exclude []string
	DryRun  bool
	Verbose bool
}

type AddOptions struct {
	Keys    []string
	DryRun  bool
	Verbose bool
}

func NewService(cwd string, cfg Config) *Service {
	return &Service{
		CWD:       cwd,
		Config:    cfg,
		Client:    api.NewClientWithOptions(api.Options{Offline: cfg.Runtime.Offline}),
		Manager:   gitignore.NewManager(cwd),
		Detectors: provider.Registry(),
	}
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (CommandResult, error) {
	includeInput := opts.Include
	if len(includeInput) == 0 {
		includeInput = s.Config.Defaults.Providers
	}

	include, includeWarnings := sanitizeKeys(includeInput)
	exclude, excludeWarnings := sanitizeKeys(opts.Exclude)
	warnings := append(includeWarnings, excludeWarnings...)

	targetResult, err := s.detectTarget(ctx, s.CWD, s.Manager, include, exclude, opts.DryRun)
	if err != nil {
		return CommandResult{}, err
	}

	return CommandResult{
		Command:                "detect",
		CWD:                    s.CWD,
		DetectedProviders:      targetResult.DetectedProviders,
		IncludedProviders:      include,
		ExcludedProviders:      exclude,
		FinalProviders:         targetResult.FinalProviders,
		UnsupportedKeyWarnings: warnings,
		RuntimeWarnings:        runtimeWarnings(s.Config.Runtime.Offline, targetResult.FinalProviders),
		RemoteProviderWarnings: targetResult.RemoteProviderWarnings,
		DetectionResults:       targetResult.DetectionResults,
		FileAction:             targetResult.FileAction,
		TemplateProviderCount:  targetResult.TemplateProviderCount,
	}, nil
}
func (s *Service) detectTarget(ctx context.Context, targetPath string, manager *gitignore.Manager, include []string, exclude []string, dryRun bool) (TargetResult, error) {
	targetResult, err := s.scanTarget(ctx, targetPath)
	if err != nil {
		return TargetResult{}, err
	}

	finalProviders, err := s.detectFinalProviders(targetResult.DetectedProviders, include, exclude)
	if err != nil {
		return TargetResult{}, err
	}

	template, err := s.Client.FetchTemplate(ctx, finalProviders)
	if err != nil {
		return TargetResult{}, err
	}
	block := gitignore.BuildManagedBlock(finalProviders, template.Content, s.Config.Defaults.IgnoreRules)
	action, err := manager.UpsertManagedBlock(block, dryRun)
	if err != nil {
		return TargetResult{}, err
	}
	remoteWarnings := remoteWarningsFromTemplate(template)

	relPath, relErr := filepath.Rel(s.CWD, targetPath)
	if relErr != nil {
		relPath = targetPath
	}

	return TargetResult{
		Path:                   relPath,
		DetectedProviders:      targetResult.DetectedProviders,
		FinalProviders:         finalProviders,
		DetectionResults:       targetResult.DetectionResults,
		RemoteProviderWarnings: remoteWarnings,
		FileAction:             action,
		TemplateProviderCount:  len(template.Providers),
	}, nil
}

func (s *Service) scanTarget(ctx context.Context, targetPath string) (TargetResult, error) {
	detections := make([]provider.Result, 0, len(s.Detectors))
	detected := makeSet(nil)
	for _, entry := range sortedDetectors(s.Detectors) {
		result := entry.detector.Detect(ctx, targetPath)
		if result.Key == "" {
			result.Key = entry.key
		}
		detections = append(detections, result)
		if result.Matched {
			detected[result.Key] = struct{}{}
		}
	}
	sortDetectionResults(detections)

	relPath, relErr := filepath.Rel(s.CWD, targetPath)
	if relErr != nil {
		relPath = targetPath
	}

	return TargetResult{
		Path:              relPath,
		DetectedProviders: filterSupportedKeys(mapKeysSorted(detected)),
		DetectionResults:  detections,
	}, nil
}

func (s *Service) detectFinalProviders(detectedProviders []string, include []string, exclude []string) ([]string, error) {
	final := makeSet(detectedProviders)
	for _, key := range include {
		final[key] = struct{}{}
	}
	for _, key := range exclude {
		delete(final, key)
	}

	finalProviders := mapKeysSorted(final)
	if len(finalProviders) == 0 {
		return nil, fmt.Errorf("no providers selected after include/exclude")
	}

	return finalProviders, nil
}

func sortedDetectors(detectors map[string]provider.Detector) []namedDetector {
	keys := make([]string, 0, len(detectors))
	for key := range detectors {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sorted := make([]namedDetector, 0, len(keys))
	for _, key := range keys {
		sorted = append(sorted, namedDetector{key: key, detector: detectors[key]})
	}

	return sorted
}

func sortDetectionResults(results []provider.Result) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Key < results[j].Key
	})
}

func (s *Service) Add(ctx context.Context, opts AddOptions) (CommandResult, error) {
	inputKeys := make([]string, 0, len(s.Config.Defaults.Providers)+len(opts.Keys))
	inputKeys = append(inputKeys, s.Config.Defaults.Providers...)
	inputKeys = append(inputKeys, opts.Keys...)

	keys, warnings := sanitizeKeys(inputKeys)
	existing, err := s.Manager.ReadManagedProviders()
	if err != nil {
		return CommandResult{}, err
	}
	existing = filterSupportedKeys(existing)
	final := makeSet(existing)
	added := make([]string, 0)
	for _, key := range keys {
		if _, ok := final[key]; !ok {
			added = append(added, key)
		}
		final[key] = struct{}{}
	}

	finalProviders := mapKeysSorted(final)
	if len(finalProviders) == 0 {
		return CommandResult{}, fmt.Errorf("no providers available to add")
	}

	template, err := s.Client.FetchTemplate(ctx, finalProviders)
	if err != nil {
		return CommandResult{}, err
	}
	block := gitignore.BuildManagedBlock(finalProviders, template.Content, s.Config.Defaults.IgnoreRules)
	action, err := s.Manager.UpsertManagedBlock(block, opts.DryRun)
	if err != nil {
		return CommandResult{}, err
	}

	sort.Strings(added)
	return CommandResult{
		Command:                "add",
		CWD:                    s.CWD,
		AddedProviders:         added,
		FinalProviders:         finalProviders,
		UnsupportedKeyWarnings: warnings,
		RuntimeWarnings:        runtimeWarnings(s.Config.Runtime.Offline, finalProviders),
		RemoteProviderWarnings: remoteWarningsFromTemplate(template),
		FileAction:             action,
		TemplateProviderCount:  len(template.Providers),
	}, nil
}

func sanitizeKeys(keys []string) ([]string, []string) {
	set := make(map[string]struct{})
	warnings := make([]string, 0)
	for _, key := range keys {
		if key == "" {
			continue
		}
		if !provider.IsSupported(key) {
			warnings = append(warnings, fmt.Sprintf("unsupported provider key: %s", key))
			continue
		}
		set[key] = struct{}{}
	}
	slices.Sort(warnings)
	return mapKeysSorted(set), warnings
}

func filterSupportedKeys(keys []string) []string {
	filtered := make([]string, 0, len(keys))
	for _, key := range keys {
		if provider.IsSupported(key) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}

func remoteWarningsFromTemplate(template api.TemplateResponse) []string {
	if len(template.AvailableProviders) == 0 {
		return nil
	}
	return remoteDiffWarnings(makeSet(template.AvailableProviders))
}

func remoteDiffWarnings(remote map[string]struct{}) []string {
	warnings := make([]string, 0)
	for _, key := range provider.RemoteSupportedKeys() {
		if _, ok := remote[key]; !ok {
			warnings = append(warnings, fmt.Sprintf("supported provider missing remotely: %s", key))
		}
	}
	slices.Sort(warnings)
	return warnings
}

func runtimeWarnings(offline bool, providers []string) []string {
	if !offline || !hasRemoteProviders(providers) {
		return nil
	}
	return []string{"runtime.offline is enabled; remote templates were loaded from the local cache without a live GitHub refresh"}
}

func hasRemoteProviders(providers []string) bool {
	for _, key := range providers {
		if !customtemplate.HasProvider(key) {
			return true
		}
	}
	return false
}

func makeSet(keys []string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		m[key] = struct{}{}
	}
	return m
}

func mapKeysSorted(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
