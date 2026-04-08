package app

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/aaronflorey/gitignore-gen/internal/api"
	"github.com/aaronflorey/gitignore-gen/internal/gitignore"
	"github.com/aaronflorey/gitignore-gen/internal/provider"
)

type APIClient interface {
	AvailableProviders(ctx context.Context) ([]string, error)
	FetchTemplate(ctx context.Context, providers []string) (api.TemplateResponse, error)
}

type Service struct {
	CWD       string
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

func NewService(cwd string) *Service {
	return &Service{
		CWD:       cwd,
		Client:    api.NewClient(),
		Manager:   gitignore.NewManager(cwd),
		Detectors: provider.Registry(),
	}
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (CommandResult, error) {
	availableRemote, err := s.Client.AvailableProviders(ctx)
	if err != nil {
		return CommandResult{}, err
	}
	remoteSet := makeSet(availableRemote)
	remoteWarnings := remoteDiffWarnings(remoteSet)

	detections := make([]provider.Result, 0, len(s.Detectors))
	detected := makeSet(nil)
	for _, entry := range sortedDetectors(s.Detectors) {
		result := entry.detector.Detect(ctx, s.CWD)
		if result.Key == "" {
			result.Key = entry.key
		}
		detections = append(detections, result)
		if result.Matched {
			detected[result.Key] = struct{}{}
		}
	}
	sortDetectionResults(detections)
	detectedProviders := mapKeysSorted(detected)

	include, includeWarnings := sanitizeKeys(opts.Include)
	exclude, excludeWarnings := sanitizeKeys(opts.Exclude)
	warnings := append(includeWarnings, excludeWarnings...)

	final := makeSet(detectedProviders)
	for _, key := range include {
		final[key] = struct{}{}
	}
	for _, key := range exclude {
		delete(final, key)
	}
	finalProviders := mapKeysSorted(final)
	if len(finalProviders) == 0 {
		return CommandResult{}, fmt.Errorf("no providers selected after include/exclude")
	}

	template, err := s.Client.FetchTemplate(ctx, finalProviders)
	if err != nil {
		return CommandResult{}, err
	}
	block := gitignore.BuildManagedBlock(finalProviders, template.Content)
	action, err := s.Manager.UpsertManagedBlock(block, opts.DryRun)
	if err != nil {
		return CommandResult{}, err
	}

	return CommandResult{
		Command:                "detect",
		CWD:                    s.CWD,
		DetectedProviders:      detectedProviders,
		IncludedProviders:      include,
		ExcludedProviders:      exclude,
		FinalProviders:         finalProviders,
		UnsupportedKeyWarnings: warnings,
		RemoteProviderWarnings: remoteWarnings,
		DetectionResults:       detections,
		FileAction:             action,
		TemplateProviderCount:  len(template.Providers),
	}, nil
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
	availableRemote, err := s.Client.AvailableProviders(ctx)
	if err != nil {
		return CommandResult{}, err
	}
	remoteSet := makeSet(availableRemote)
	remoteWarnings := remoteDiffWarnings(remoteSet)

	keys, warnings := sanitizeKeys(opts.Keys)
	existing, err := s.Manager.ReadManagedProviders()
	if err != nil {
		return CommandResult{}, err
	}
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
	block := gitignore.BuildManagedBlock(finalProviders, template.Content)
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
		RemoteProviderWarnings: remoteWarnings,
		FileAction:             action,
		TemplateProviderCount:  len(template.Providers),
	}, nil
}

func sanitizeKeys(keys []string) ([]string, []string) {
	set := make(map[string]struct{})
	warnings := make([]string, 0)
	for _, key := range keys {
		if !provider.IsSupported(key) {
			warnings = append(warnings, fmt.Sprintf("unsupported provider key: %s", key))
			continue
		}
		set[key] = struct{}{}
	}
	slices.Sort(warnings)
	clean := mapKeysSorted(set)
	return clean, warnings
}

func remoteDiffWarnings(remote map[string]struct{}) []string {
	warnings := make([]string, 0)
	for _, key := range provider.SupportedKeys {
		if _, ok := remote[key]; !ok {
			warnings = append(warnings, fmt.Sprintf("supported provider missing remotely: %s", key))
		}
	}
	slices.Sort(warnings)
	return warnings
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
