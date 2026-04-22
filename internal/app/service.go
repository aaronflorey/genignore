package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/aaronflorey/genignore/internal/api"
	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
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

var detectCarryForwardOSProviders = map[string]struct{}{
	"linux":   {},
	"macos":   {},
	"windows": {},
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

	include, includeWarnings := sanitizeKeys(opts.Include)
	exclude, excludeWarnings := sanitizeKeys(opts.Exclude)
	warnings := append(includeWarnings, excludeWarnings...)

	targetPaths, err := detectTargetPaths(s.CWD)
	if err != nil {
		return CommandResult{}, err
	}
	if len(targetPaths) == 0 {
		targetPaths = []string{s.CWD}
	}

	if len(targetPaths) == 1 && targetPaths[0] == s.CWD {
		targetResult, detectErr := s.detectTarget(ctx, targetPaths[0], s.Manager, include, exclude, opts.DryRun)
		if detectErr != nil {
			return CommandResult{}, detectErr
		}

		return CommandResult{
			Command:                "detect",
			CWD:                    s.CWD,
			DetectedProviders:      targetResult.DetectedProviders,
			IncludedProviders:      include,
			ExcludedProviders:      exclude,
			FinalProviders:         targetResult.FinalProviders,
			UnsupportedKeyWarnings: warnings,
			RemoteProviderWarnings: remoteWarnings,
			DetectionResults:       targetResult.DetectionResults,
			FileAction:             targetResult.FileAction,
			TemplateProviderCount:  targetResult.TemplateProviderCount,
		}, nil
	}

	targetResults := make([]TargetResult, 0, len(targetPaths))
	detected := makeSet(nil)
	final := makeSet(nil)
	templateProviderCount := 0
	fileActions := make([]gitignore.FileAction, 0, len(targetPaths))
	for _, targetPath := range targetPaths {
		manager := gitignore.NewManager(targetPath)
		targetResult, detectErr := s.detectTarget(ctx, targetPath, manager, include, exclude, opts.DryRun)
		if detectErr != nil {
			return CommandResult{}, detectErr
		}
		for _, key := range targetResult.DetectedProviders {
			detected[key] = struct{}{}
		}
		for _, key := range targetResult.FinalProviders {
			final[key] = struct{}{}
		}
		templateProviderCount += targetResult.TemplateProviderCount
		fileActions = append(fileActions, targetResult.FileAction)
		targetResults = append(targetResults, targetResult)
	}

	return CommandResult{
		Command:                "detect",
		CWD:                    s.CWD,
		Targets:                targetResults,
		DetectedProviders:      mapKeysSorted(detected),
		IncludedProviders:      include,
		ExcludedProviders:      exclude,
		FinalProviders:         mapKeysSorted(final),
		UnsupportedKeyWarnings: warnings,
		RemoteProviderWarnings: remoteWarnings,
		FileAction:             aggregateFileAction(fileActions),
		TemplateProviderCount:  templateProviderCount,
	}, nil
}

func (s *Service) detectTarget(ctx context.Context, targetPath string, manager *gitignore.Manager, include []string, exclude []string, dryRun bool) (TargetResult, error) {
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

	existingManagedProviders, err := manager.ReadManagedProviders()
	if err != nil {
		return TargetResult{}, err
	}

	final := makeSet(mapKeysSorted(detected))
	for _, key := range existingManagedProviders {
		if shouldCarryForwardDetectedManagedProvider(key) {
			final[key] = struct{}{}
		}
	}
	for _, key := range include {
		final[key] = struct{}{}
	}
	for _, key := range exclude {
		delete(final, key)
	}
	finalProviders := mapKeysSorted(final)
	if len(finalProviders) == 0 {
		return TargetResult{}, fmt.Errorf("no providers selected after include/exclude")
	}

	template, err := s.Client.FetchTemplate(ctx, finalProviders)
	if err != nil {
		return TargetResult{}, err
	}
	block := gitignore.BuildManagedBlock(finalProviders, template.Content)
	action, err := manager.UpsertManagedBlock(block, dryRun)
	if err != nil {
		return TargetResult{}, err
	}

	relPath, relErr := filepath.Rel(s.CWD, targetPath)
	if relErr != nil {
		relPath = targetPath
	}

	return TargetResult{
		Path:                  relPath,
		DetectedProviders:     mapKeysSorted(detected),
		FinalProviders:        finalProviders,
		DetectionResults:      detections,
		FileAction:            action,
		TemplateProviderCount: len(template.Providers),
	}, nil
}

func detectTargetPaths(cwd string) ([]string, error) {
	packagesDir := filepath.Join(cwd, "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read packages directory: %w", err)
	}

	targets := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		targets = append(targets, filepath.Join(packagesDir, entry.Name()))
	}
	sort.Strings(targets)
	return targets, nil
}

func aggregateFileAction(actions []gitignore.FileAction) gitignore.FileAction {
	if len(actions) == 0 {
		return ""
	}

	allCreated := true
	for _, action := range actions {
		if action == gitignore.FileActionDryRun {
			return gitignore.FileActionDryRun
		}
		if action != gitignore.FileActionCreated {
			allCreated = false
		}
	}
	if allCreated {
		return gitignore.FileActionCreated
	}
	return gitignore.FileActionUpdated
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
	for _, key := range provider.RemoteSupportedKeys() {
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

func shouldCarryForwardDetectedManagedProvider(key string) bool {
	_, ok := detectCarryForwardOSProviders[key]
	return ok
}
