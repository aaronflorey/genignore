package app

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/aaronflorey/genignore/internal/api"
	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

type APIClient interface {
	AvailableProviders(ctx context.Context) ([]string, error)
	FetchTemplate(ctx context.Context, providers []string) (api.TemplateResponse, error)
	InspectRuntime(providers []string) api.RuntimeDiagnostics
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

type resolvedSelection struct {
	DetectedProviders      []string
	IncludedProviders      []string
	ExcludedProviders      []string
	FinalProviders         []string
	UnsupportedKeyWarnings []string
	DetectionResults       []provider.Result
}

type DetectOptions struct {
	Include []string
	Exclude []string
	DryRun  bool
	Diff    bool
	Verbose bool
}

type AddOptions struct {
	Keys    []string
	DryRun  bool
	Diff    bool
	Verbose bool
}

func NewService(cwd string, cfg Config) *Service {
	return &Service{
		CWD:       cwd,
		Config:    cfg,
		Client:    api.NewClientWithOptions(api.Options{Offline: cfg.Runtime.Offline, UpstreamCommit: cfg.Runtime.UpstreamCommit}),
		Manager:   gitignore.NewManager(cwd),
		Detectors: provider.Registry(),
	}
}

func (s *Service) Resolve(ctx context.Context, opts ResolveOptions) (ResolveResult, error) {
	selection, err := s.resolveSelection(ctx, opts.Include, opts.Exclude)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		Command:                "resolve",
		CWD:                    s.CWD,
		DetectedProviders:      selection.DetectedProviders,
		IncludedProviders:      selection.IncludedProviders,
		ExcludedProviders:      selection.ExcludedProviders,
		FinalProviders:         selection.FinalProviders,
		UnsupportedKeyWarnings: selection.UnsupportedKeyWarnings,
		DetectionResults:       selection.DetectionResults,
	}, nil
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (CommandResult, error) {
	selection, err := s.resolveSelection(ctx, opts.Include, opts.Exclude)
	previewOnly := opts.Diff
	if err != nil {
		return CommandResult{}, err
	}

	template, err := s.Client.FetchTemplate(ctx, selection.FinalProviders)
	if err != nil {
		return CommandResult{}, err
	}
	block := gitignore.BuildManagedBlockWithMetadata(selection.FinalProviders, managedBlockMetadata(selection.FinalProviders, s.Config.Runtime.UpstreamCommit), template.Content, s.Config.Defaults.IgnoreRules)
	action, diff, err := applyManagedBlock(s.Manager, block, opts.DryRun, opts.Diff)
	if err != nil {
		return CommandResult{}, err
	}

	return CommandResult{
		Command:                "detect",
		CWD:                    s.CWD,
		DetectedProviders:      selection.DetectedProviders,
		IncludedProviders:      selection.IncludedProviders,
		ExcludedProviders:      selection.ExcludedProviders,
		FinalProviders:         selection.FinalProviders,
		UnsupportedKeyWarnings: selection.UnsupportedKeyWarnings,
		RuntimeWarnings:        runtimeWarnings(s.Config.Runtime.Offline, selection.FinalProviders),
		RemoteProviderWarnings: remoteWarningsFromTemplate(template),
		DetectionResults:       selection.DetectionResults,
		FileAction:             action,
		PreviewOnly:            previewOnly,
		Diff:                   diff,
		TemplateProviderCount:  len(template.Providers),
	}, nil
}

func (s *Service) resolveSelection(ctx context.Context, includeInput []string, excludeInput []string) (resolvedSelection, error) {
	if len(includeInput) == 0 {
		includeInput = s.Config.Defaults.Providers
	}

	include, includeWarnings := sanitizeKeys(includeInput)
	exclude, excludeWarnings := sanitizeKeys(excludeInput)
	warnings := append(includeWarnings, excludeWarnings...)

	targetResult, err := s.scanTarget(ctx, s.CWD)
	if err != nil {
		return resolvedSelection{}, err
	}

	finalProviders, err := s.detectFinalProviders(targetResult.DetectedProviders, include, exclude)
	if err != nil {
		return resolvedSelection{}, err
	}

	return resolvedSelection{
		DetectedProviders:      targetResult.DetectedProviders,
		IncludedProviders:      include,
		ExcludedProviders:      exclude,
		FinalProviders:         finalProviders,
		UnsupportedKeyWarnings: warnings,
		DetectionResults:       targetResult.DetectionResults,
	}, nil
}

func (s *Service) scanTarget(ctx context.Context, targetPath string) (TargetResult, error) {
	ctx = provider.ContextWithInputs(ctx, targetPath)
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
	block := gitignore.BuildManagedBlockWithMetadata(finalProviders, managedBlockMetadata(finalProviders, s.Config.Runtime.UpstreamCommit), template.Content, s.Config.Defaults.IgnoreRules)
	action, diff, err := applyManagedBlock(s.Manager, block, opts.DryRun, opts.Diff)
	if err != nil {
		return CommandResult{}, err
	}

	sort.Strings(added)
	previewOnly := opts.Diff
	return CommandResult{
		Command:                "add",
		CWD:                    s.CWD,
		AddedProviders:         added,
		FinalProviders:         finalProviders,
		UnsupportedKeyWarnings: warnings,
		RuntimeWarnings:        runtimeWarnings(s.Config.Runtime.Offline, finalProviders),
		RemoteProviderWarnings: remoteWarningsFromTemplate(template),
		FileAction:             action,
		PreviewOnly:            previewOnly,
		Diff:                   diff,
		TemplateProviderCount:  len(template.Providers),
	}, nil
}

func (s *Service) Doctor(ctx context.Context, opts DoctorOptions) (DoctorResult, error) {
	selection, err := s.resolveSelection(ctx, opts.Include, opts.Exclude)
	if err != nil {
		return DoctorResult{}, err
	}
	runtimeInfo := s.Client.InspectRuntime(selection.FinalProviders)

	return DoctorResult{
		Command:                "doctor",
		CWD:                    s.CWD,
		DetectedProviders:      selection.DetectedProviders,
		IncludedProviders:      selection.IncludedProviders,
		ExcludedProviders:      selection.ExcludedProviders,
		FinalProviders:         selection.FinalProviders,
		UnsupportedKeyWarnings: selection.UnsupportedKeyWarnings,
		RuntimeWarnings:        runtimeWarnings(s.Config.Runtime.Offline, selection.FinalProviders),
		Detections:             doctorDetections(selection.DetectionResults),
		Runtime:                doctorRuntime(runtimeInfo),
		Provenance:             managedBlockMetadata(selection.FinalProviders, s.Config.Runtime.UpstreamCommit),
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

func applyManagedBlock(manager *gitignore.Manager, block string, dryRun bool, previewOnly bool) (gitignore.FileAction, string, error) {
	if previewOnly {
		preview, err := manager.PreviewManagedBlock(block)
		if err != nil {
			return "", "", err
		}
		return preview.Action, preview.Diff, nil
	}
	if dryRun {
		action, err := manager.UpsertManagedBlock(block, true)
		return action, "", err
	}
	preview, err := manager.PreviewManagedBlock(block)
	if err != nil {
		return "", "", err
	}
	if preview.Action == gitignore.FileActionNoOp {
		return gitignore.FileActionNoOp, preview.Diff, nil
	}
	action, err := manager.UpsertManagedBlock(block, false)
	if err != nil {
		return "", "", err
	}
	return action, preview.Diff, nil
}

func managedBlockMetadata(providers []string, upstreamCommit string) []string {
	remoteProviders := make([]string, 0, len(providers))
	embeddedProviders := make([]string, 0, len(providers))
	for _, key := range providers {
		if customtemplate.HasProvider(key) {
			embeddedProviders = append(embeddedProviders, key)
			continue
		}
		remoteProviders = append(remoteProviders, key)
	}

	parts := make([]string, 0, 2)
	if len(remoteProviders) > 0 {
		commit := strings.TrimSpace(upstreamCommit)
		if commit == "" {
			commit = api.DefaultUpstreamCommit
		}
		parts = append(parts, fmt.Sprintf("github/gitignore@%s [%s]", commit, strings.Join(remoteProviders, ",")))
	}
	if len(embeddedProviders) > 0 {
		parts = append(parts, fmt.Sprintf("embedded [%s]", strings.Join(embeddedProviders, ",")))
	}
	if len(parts) == 0 {
		return nil
	}
	return []string{"# Provenance: " + strings.Join(parts, "; ")}
}

func doctorDetections(results []provider.Result) []DoctorDetection {
	detections := make([]DoctorDetection, 0, len(results))
	for _, result := range results {
		detections = append(detections, DoctorDetection{
			Key:      result.Key,
			Matched:  result.Matched,
			Origin:   detectionOrigin(result),
			Reason:   result.Reason,
			Evidence: result.Evidence,
			Error:    result.Error,
		})
	}
	return detections
}

func doctorRuntime(runtimeInfo api.RuntimeDiagnostics) DoctorRuntime {
	cacheEntries := make([]DoctorCacheEntry, 0, len(runtimeInfo.CacheEntries))
	for _, entry := range runtimeInfo.CacheEntries {
		cacheEntries = append(cacheEntries, DoctorCacheEntry{Provider: entry.Provider, State: entry.State, Detail: entry.Detail})
	}
	return DoctorRuntime{
		UpstreamCommit:    runtimeInfo.UpstreamCommit,
		Offline:           runtimeInfo.Offline,
		RemoteProviders:   runtimeInfo.RemoteProviders,
		EmbeddedProviders: runtimeInfo.EmbeddedProviders,
		CacheEntries:      cacheEntries,
		Decisions:         runtimeInfo.Decisions,
	}
}

func detectionOrigin(result provider.Result) string {
	switch {
	case strings.Contains(result.Reason, "runtime OS"):
		return "host"
	case strings.Contains(result.Reason, "installed application") || strings.Contains(result.Reason, "application not found"):
		return "host"
	case strings.Contains(result.Reason, "jetbrains install"):
		return "repository+host"
	default:
		return "repository"
	}
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
