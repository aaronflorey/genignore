package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/api"
	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

const noisyToptalTemplate = `# Created by https://www.toptal.com/developers/gitignore/api/go,macos
# Edit at https://www.toptal.com/developers/gitignore?templates=go,macos

### Go ###
# If you prefer the allow list template instead of the deny list, see community template:
# https://github.com/github/gitignore/blob/main/community/Golang/Go.AllowList.gitignore
#
# Binaries for programs and plugins
*.exe

### macOS ###
.DS_Store

# End of https://www.toptal.com/developers/gitignore/api/go,macos
`

type fakeAPI struct {
	available      []string
	template       string
	availableErr   error
	templateErr    error
	availableCalls int
	requests       [][]string
}

func (f *fakeAPI) AvailableProviders(_ context.Context) ([]string, error) {
	f.availableCalls++
	if f.availableErr != nil {
		return nil, f.availableErr
	}
	return f.available, nil
}

func (f *fakeAPI) FetchTemplate(_ context.Context, providers []string) (api.TemplateResponse, error) {
	if f.templateErr != nil {
		return api.TemplateResponse{}, f.templateErr
	}
	if f.availableErr != nil {
		for _, key := range providers {
			if !customtemplate.HasProvider(key) {
				return api.TemplateResponse{}, f.availableErr
			}
		}
	}
	f.requests = append(f.requests, append([]string{}, providers...))
	return api.TemplateResponse{Providers: providers, Content: f.template, AvailableProviders: f.available}, nil
}

func matchedDetector(key string) provider.Detector {
	return provider.DetectorFunc(func(_ context.Context, _ string) provider.Result {
		return provider.Result{Key: key, Matched: true, Reason: "matched"}
	})
}

func detectorResult(result provider.Result) provider.Detector {
	return provider.DetectorFunc(func(_ context.Context, _ string) provider.Result {
		return result
	})
}

func TestDetectPreservesManagedOSProvidersAcrossCrossRuntimeRuns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"macos"}, ".DS_Store\n")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: manager,
		Detectors: map[string]provider.Detector{
			"linux": matchedDetector("linux"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"linux", "macos"}) {
		t.Fatalf("expected detect to preserve managed OS provider, got %v", res.FinalProviders)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one template request, got %d", len(client.requests))
	}
	if !reflect.DeepEqual(client.requests[0], []string{"linux", "macos"}) {
		t.Fatalf("expected sorted template request providers, got %v", client.requests[0])
	}
}

func TestDetectResetDropsManagedNonOSProviders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"python"}, "venv/\n")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: manager,
		Detectors: map[string]provider.Detector{
			"linux": matchedDetector("linux"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"linux"}) {
		t.Fatalf("expected detect reset to drop non-OS managed providers, got %v", res.FinalProviders)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one template request, got %d", len(client.requests))
	}
	if !reflect.DeepEqual(client.requests[0], []string{"linux"}) {
		t.Fatalf("expected sorted template request providers, got %v", client.requests[0])
	}
}

func TestDetectExcludeRemovesPreservedManagedOSProvider(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"macos"}, ".DS_Store\n")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: manager,
		Detectors: map[string]provider.Detector{
			"linux": matchedDetector("linux"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{Exclude: []string{"macos"}})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"linux"}) {
		t.Fatalf("expected exclude to remove preserved OS provider, got %v", res.FinalProviders)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one template request, got %d", len(client.requests))
	}
	if !reflect.DeepEqual(client.requests[0], []string{"linux"}) {
		t.Fatalf("expected sorted template request providers, got %v", client.requests[0])
	}
}

func TestAddAppendsOnlyMissingProviders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"node"}, "node_modules/")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: manager}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"node", "go"}})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !reflect.DeepEqual(res.AddedProviders, []string{"go"}) {
		t.Fatalf("expected go only added, got %v", res.AddedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go", "node"}) {
		t.Fatalf("expected alphabetical final set, got %v", res.FinalProviders)
	}
	if client.availableCalls != 0 {
		t.Fatalf("expected add to reuse template catalog data, got %d separate calls", client.availableCalls)
	}
}

func TestUnsupportedWarnings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir), Detectors: map[string]provider.Detector{"node": matchedDetector("node")}}

	res, err := svc.Detect(context.Background(), DetectOptions{Include: []string{"unknown"}})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(res.UnsupportedKeyWarnings) == 0 {
		t.Fatalf("expected unsupported warning")
	}
	if !reflect.DeepEqual(res.UnsupportedKeyWarnings, []string{"unsupported provider key: unknown"}) {
		t.Fatalf("unexpected unsupported warnings: %v", res.UnsupportedKeyWarnings)
	}
}

func TestAddMixedSupportedUnsupportedKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir)}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"go", "unknown", "node", "bad"}})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go", "node"}) {
		t.Fatalf("expected valid providers only, got %v", res.FinalProviders)
	}
	if !reflect.DeepEqual(res.UnsupportedKeyWarnings, []string{"unsupported provider key: bad", "unsupported provider key: unknown"}) {
		t.Fatalf("unexpected unsupported warnings: %v", res.UnsupportedKeyWarnings)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one template request, got %d", len(client.requests))
	}
	if !reflect.DeepEqual(client.requests[0], []string{"go", "node"}) {
		t.Fatalf("expected sorted template request providers, got %v", client.requests[0])
	}
}

func TestRemoteProviderDriftWarning(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: []string{"node"}, template: "node_modules/\n"}
	svc := &Service{
		CWD:       dir,
		Client:    client,
		Manager:   gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{"node": matchedDetector("node")},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(res.RemoteProviderWarnings) == 0 {
		t.Fatalf("expected remote provider drift warning")
	}
	if client.availableCalls != 0 {
		t.Fatalf("expected detect to reuse template catalog data, got %d separate calls", client.availableCalls)
	}
	for _, warning := range []string{
		"supported provider missing remotely: angular",
		"supported provider missing remotely: go",
	} {
		if !containsString(res.RemoteProviderWarnings, warning) {
			t.Fatalf("expected warning %q in %v", warning, res.RemoteProviderWarnings)
		}
	}
	for _, warning := range []string{
		"supported provider missing remotely: ai-agents",
		"supported provider missing remotely: wrangler",
	} {
		if containsString(res.RemoteProviderWarnings, warning) {
			t.Fatalf("did not expect embedded exception warning %q in %v", warning, res.RemoteProviderWarnings)
		}
	}
}

func TestDetectAcceptsEmbeddedExceptionProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: []string{"node"}, template: "generated\n"}
	svc := &Service{
		CWD:       dir,
		Client:    client,
		Manager:   gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{"node": matchedDetector("node")},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{Include: []string{"wrangler"}})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.IncludedProviders, []string{"wrangler"}) {
		t.Fatalf("unexpected included providers: %v", res.IncludedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"node", "wrangler"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if len(res.UnsupportedKeyWarnings) != 0 {
		t.Fatalf("unexpected unsupported warnings: %v", res.UnsupportedKeyWarnings)
	}
}

func TestDetectCustomOnlySucceedsWithoutRemoteCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{availableErr: errors.New("catalog boom"), template: "generated\n"}
	svc := &Service{
		CWD:       dir,
		Client:    client,
		Manager:   gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{"node": matchedDetector("node")},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{Include: []string{"wrangler"}, Exclude: []string{"node"}})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"wrangler"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if client.availableCalls != 0 {
		t.Fatalf("expected detect to skip remote catalog lookup, got %d calls", client.availableCalls)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"wrangler"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
	if len(res.RemoteProviderWarnings) != 0 {
		t.Fatalf("unexpected remote warnings: %v", res.RemoteProviderWarnings)
	}
	if res.FileAction != gitignore.FileActionCreated {
		t.Fatalf("unexpected file action: %s", res.FileAction)
	}
}

func TestAddCustomOnlySucceedsWithoutRemoteCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{availableErr: errors.New("catalog boom"), template: "generated\n"}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir)}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"ai-agents"}})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"ai-agents"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if client.availableCalls != 0 {
		t.Fatalf("expected add to skip remote catalog lookup, got %d calls", client.availableCalls)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"ai-agents"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
	if len(res.RemoteProviderWarnings) != 0 {
		t.Fatalf("unexpected remote warnings: %v", res.RemoteProviderWarnings)
	}
}

func TestAPIFailureHardFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := &Service{CWD: dir, Client: &fakeAPI{availableErr: errors.New("boom")}, Manager: gitignore.NewManager(dir)}

	if _, err := svc.Add(context.Background(), AddOptions{Keys: []string{"node"}}); err == nil {
		t.Fatalf("expected API failure")
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected no file write on list API failure")
	}
}

func TestTemplateAPIFailureHardFailsWithoutWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{
		available:   provider.SupportedKeys,
		templateErr: errors.New("template boom"),
	}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir)}

	if _, err := svc.Add(context.Background(), AddOptions{Keys: []string{"node"}}); err == nil {
		t.Fatalf("expected template API failure")
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected no file write on template API failure")
	}
}

func TestDetectTemplateFailureLeavesExistingGitignoreUnchanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := gitignore.BuildManagedBlock([]string{"python"}, "venv/\n") + "# user rule\n.env\n"
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	client := &fakeAPI{
		available:   provider.SupportedKeys,
		templateErr: errors.New("template boom"),
	}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node": matchedDetector("node"),
		},
	}

	if _, err := svc.Detect(context.Background(), DetectOptions{}); err == nil {
		t.Fatalf("expected template API failure")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if string(content) != seed {
		t.Fatalf("expected existing file to remain unchanged on failure")
	}
}

func TestDryRunDoesNotWriteFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir), Detectors: map[string]provider.Detector{"node": matchedDetector("node")}}

	res, err := svc.Detect(context.Background(), DetectOptions{DryRun: true})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if res.FileAction != gitignore.FileActionDryRun {
		t.Fatalf("expected dry-run action")
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected .gitignore to not be written")
	}
}

func TestDetectDryRunLeavesExistingGitignoreUnchanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"python"}, "venv/\n") + "# user rule\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: manager,
		Detectors: map[string]provider.Detector{
			"node": matchedDetector("node"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{DryRun: true})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if res.FileAction != gitignore.FileActionDryRun {
		t.Fatalf("expected dry-run action, got %s", res.FileAction)
	}
	if res.TemplateProviderCount != 1 {
		t.Fatalf("unexpected template provider count: %d", res.TemplateProviderCount)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if string(content) != seed {
		t.Fatalf("expected dry-run to leave .gitignore unchanged\nwant:\n%s\n got:\n%s", seed, string(content))
	}
}

func TestAddDryRunLeavesExistingGitignoreUnchanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"node"}, "node_modules/\n") + "# user rule\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\nbin/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: manager}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"go"}, DryRun: true})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if res.FileAction != gitignore.FileActionDryRun {
		t.Fatalf("expected dry-run action, got %s", res.FileAction)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go", "node"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if res.TemplateProviderCount != 2 {
		t.Fatalf("unexpected template provider count: %d", res.TemplateProviderCount)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"go", "node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if string(content) != seed {
		t.Fatalf("expected dry-run to leave .gitignore unchanged\nwant:\n%s\n got:\n%s", seed, string(content))
	}
}

func TestDetectAppliesIncludeExcludeWithSortedSelections(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"python": matchedDetector("python"),
			"node":   matchedDetector("node"),
			"go":     matchedDetector("go"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{
		Include: []string{"angular", "node", "bad"},
		Exclude: []string{"python", "angular", "unknown"},
	})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	if !reflect.DeepEqual(res.DetectedProviders, []string{"go", "node", "python"}) {
		t.Fatalf("unexpected detected providers: %v", res.DetectedProviders)
	}
	if !reflect.DeepEqual(res.IncludedProviders, []string{"angular", "node"}) {
		t.Fatalf("unexpected included providers: %v", res.IncludedProviders)
	}
	if !reflect.DeepEqual(res.ExcludedProviders, []string{"angular", "python"}) {
		t.Fatalf("unexpected excluded providers: %v", res.ExcludedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go", "node"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if !reflect.DeepEqual(res.UnsupportedKeyWarnings, []string{"unsupported provider key: bad", "unsupported provider key: unknown"}) {
		t.Fatalf("unexpected warnings: %v", res.UnsupportedKeyWarnings)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"go", "node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestDetectUsesConfigDefaultProvidersWhenCLIIncludeMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Config:  Config{Defaults: ConfigDefaults{Providers: []string{"angular"}}},
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node": matchedDetector("node"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.IncludedProviders, []string{"angular"}) {
		t.Fatalf("unexpected included providers: %v", res.IncludedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"angular", "node"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"angular", "node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestDetectCLIIncludeOverridesConfigDefaultProviders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Config:  Config{Defaults: ConfigDefaults{Providers: []string{"python"}}},
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node": matchedDetector("node"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{Include: []string{"angular"}})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.IncludedProviders, []string{"angular"}) {
		t.Fatalf("unexpected included providers: %v", res.IncludedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"angular", "node"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"angular", "node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestDetectDropsUnsupportedMatchedProvidersFromFinalSelection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node":  matchedDetector("node"),
			"react": matchedDetector("react"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.DetectedProviders, []string{"node"}) {
		t.Fatalf("unexpected detected providers: %v", res.DetectedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"node"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"node"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestAddMergesConfigDefaultProviders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Config:  Config{Defaults: ConfigDefaults{Providers: []string{"python"}}},
		Client:  client,
		Manager: gitignore.NewManager(dir),
	}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"node"}})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !reflect.DeepEqual(res.AddedProviders, []string{"node", "python"}) {
		t.Fatalf("unexpected added providers: %v", res.AddedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"node", "python"}) {
		t.Fatalf("unexpected final providers: %v", res.FinalProviders)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"node", "python"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestDetectAppendsConfigIgnoreRulesToManagedBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{
		CWD:     dir,
		Config:  Config{Defaults: ConfigDefaults{IgnoreRules: []string{".direnv/", "coverage.out"}}},
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node": matchedDetector("node"),
		},
	}

	_, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	value := string(content)
	for _, fragment := range []string{"node_modules/", ".direnv/", "coverage.out", ".env\n.env.*\n!.env.example\n!.env.ci"} {
		if !strings.Contains(value, fragment) {
			t.Fatalf("missing %q in managed block\n%s", fragment, value)
		}
	}
}

func TestDetectFailsBeforeTemplateFetchWhenSelectionIsEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:       dir,
		Client:    client,
		Manager:   gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{},
	}

	_, err := svc.Detect(context.Background(), DetectOptions{})
	if err == nil || err.Error() != "no providers selected after include/exclude" {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("expected no template requests, got %v", client.requests)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no file write, stat err=%v", statErr)
	}
}

func TestDetectReturnsDetectionResultsSortedByProviderKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"node":  detectorResult(provider.Result{Key: "node", Matched: true, Reason: "found package.json", Evidence: filepath.Join(dir, "package.json")}),
			"go":    detectorResult(provider.Result{Key: "go", Matched: true, Reason: "found go.mod", Evidence: filepath.Join(dir, "go.mod")}),
			"react": detectorResult(provider.Result{Key: "react", Matched: false, Reason: "invalid package.json", Evidence: filepath.Join(dir, "package.json"), Error: "unexpected EOF"}),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	keys := make([]string, 0, len(res.DetectionResults))
	for _, detection := range res.DetectionResults {
		keys = append(keys, detection.Key)
	}
	if !reflect.DeepEqual(keys, []string{"go", "node", "react"}) {
		t.Fatalf("unexpected detection result order: %v", keys)
	}
}

func TestDetectWritesCleanManagedBlockAndPreservesUserLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"# user-owned rule",
		gitignore.StartMarker,
		"# old block",
		gitignore.EndMarker,
		".planning",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: noisyToptalTemplate}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"go": matchedDetector("go"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if res.FileAction != gitignore.FileActionUpdated {
		t.Fatalf("unexpected file action: %s", res.FileAction)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	value := string(content)
	if strings.Contains(value, "# Created by https://www.toptal.com/developers/gitignore/api/go,macos") {
		t.Fatalf("expected detect output to strip Toptal provenance comments\n%s", value)
	}
	if !strings.Contains(value, "# user-owned rule") || !strings.Contains(value, ".planning") {
		t.Fatalf("expected unmanaged user content preserved\n%s", value)
	}
}

func TestAddWritesCleanManagedBlockAndIsStableOnRerun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"go"}, "### Go ###\n*.test\n") + "# local rule\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: noisyToptalTemplate}
	svc := &Service{CWD: dir, Client: client, Manager: manager}

	first, err := svc.Add(context.Background(), AddOptions{Keys: []string{"macos"}})
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if first.FileAction != gitignore.FileActionUpdated {
		t.Fatalf("unexpected first add action: %s", first.FileAction)
	}
	firstContent, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read first .gitignore failed: %v", err)
	}

	second, err := svc.Add(context.Background(), AddOptions{Keys: []string{"macos"}})
	if err != nil {
		t.Fatalf("second add failed: %v", err)
	}
	if second.FileAction != gitignore.FileActionUpdated {
		t.Fatalf("unexpected second add action: %s", second.FileAction)
	}
	secondContent, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read second .gitignore failed: %v", err)
	}

	if strings.Contains(string(secondContent), "# End of https://www.toptal.com/developers/gitignore/api/go,macos") {
		t.Fatalf("expected add output to strip Toptal footer comment\n%s", string(secondContent))
	}
	if !strings.Contains(string(secondContent), ".env\n.env.*\n!.env.example\n!.env.ci\n# END genignore\n") {
		t.Fatalf("expected normalized env rules in deterministic order\n%s", string(secondContent))
	}
	if strings.Contains(string(secondContent), "# local rule\n.env\n") {
		t.Fatalf("expected unmanaged duplicate env line to be removed\n%s", string(secondContent))
	}
	if string(firstContent) != string(secondContent) {
		t.Fatalf("expected rerun to remain byte-stable\nfirst:\n%s\nsecond:\n%s", string(firstContent), string(secondContent))
	}
	if !strings.HasSuffix(string(secondContent), "# local rule\n") {
		t.Fatalf("expected unmanaged trailing lines preserved\n%s", string(secondContent))
	}
}

func TestDetectIgnoresPackagesChildrenAndWritesSingleRootGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packagesDir := filepath.Join(dir, "packages")
	apiDir := filepath.Join(packagesDir, "api")
	webDir := filepath.Join(packagesDir, "web")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("create api package dir failed: %v", err)
	}
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("create web package dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module example.com/api\n"), 0o644); err != nil {
		t.Fatalf("write api go.mod failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write web package.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/root\n"), 0o644); err != nil {
		t.Fatalf("write root go.mod failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: gitignore.NewManager(dir),
		Detectors: map[string]provider.Detector{
			"go": provider.DetectorFunc(func(_ context.Context, cwd string) provider.Result {
				if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
					return provider.Result{Key: "go", Matched: true, Reason: "found go.mod"}
				}
				return provider.Result{Key: "go", Matched: false, Reason: "signal not found"}
			}),
			"node": provider.DetectorFunc(func(_ context.Context, cwd string) provider.Result {
				if _, err := os.Stat(filepath.Join(cwd, "package.json")); err == nil {
					return provider.Result{Key: "node", Matched: true, Reason: "found package.json"}
				}
				return provider.Result{Key: "node", Matched: false, Reason: "signal not found"}
			}),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(res.Targets) != 0 {
		t.Fatalf("expected single-directory detect without target fanout, got %+v", res.Targets)
	}
	if !reflect.DeepEqual(res.DetectedProviders, []string{"go"}) {
		t.Fatalf("expected root-only detected providers, got %v", res.DetectedProviders)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go"}) {
		t.Fatalf("expected root-only final providers, got %v", res.FinalProviders)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected a single template request for the root managed block, got %d", len(client.requests))
	}
	if !reflect.DeepEqual(client.requests[0], []string{"go"}) {
		t.Fatalf("unexpected root request providers: %v", client.requests[0])
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected root .gitignore to be written: %v", err)
	}
	if !strings.Contains(string(content), gitignore.StartMarker) {
		t.Fatalf("expected managed block markers in root .gitignore")
	}
	for _, packagePath := range []string{apiDir, webDir} {
		if _, err := os.Stat(filepath.Join(packagePath, ".gitignore")); !os.IsNotExist(err) {
			t.Fatalf("expected package .gitignore to remain untouched in %s", packagePath)
		}
	}
}

func TestDetectPreservesRootManagedOSProvidersWhenPackagesDirectoryExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "packages", "api"), 0o755); err != nil {
		t.Fatalf("create packages directory failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "packages", "api", "go.mod"), []byte("module example.com/api\n"), 0o644); err != nil {
		t.Fatalf("write api go.mod failed: %v", err)
	}

	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"macos"}, ".DS_Store\n")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "generated\n"}
	svc := &Service{
		CWD:     dir,
		Client:  client,
		Manager: manager,
		Detectors: map[string]provider.Detector{
			"go": matchedDetector("go"),
		},
	}

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"go", "macos"}) {
		t.Fatalf("expected root managed OS provider to be preserved, got %v", res.FinalProviders)
	}
	if len(res.Targets) != 0 {
		t.Fatalf("expected single-directory detect without target fanout, got %+v", res.Targets)
	}
	if len(client.requests) != 1 || !reflect.DeepEqual(client.requests[0], []string{"go", "macos"}) {
		t.Fatalf("unexpected template requests: %v", client.requests)
	}
}

func TestAddRemainsSingleTargetWhenPackagesDirectoryExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "packages", "api"), 0o755); err != nil {
		t.Fatalf("create packages directory failed: %v", err)
	}

	client := &fakeAPI{available: provider.SupportedKeys, template: "node_modules/\n"}
	svc := &Service{CWD: dir, Client: client, Manager: gitignore.NewManager(dir)}

	res, err := svc.Add(context.Background(), AddOptions{Keys: []string{"node"}})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if res.FileAction != gitignore.FileActionCreated {
		t.Fatalf("unexpected root file action: %s", res.FileAction)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatalf("expected root .gitignore to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "packages", "api", ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected add to avoid writing package .gitignore")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
