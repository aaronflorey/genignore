package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aaronflorey/gitignore-gen/internal/api"
	"github.com/aaronflorey/gitignore-gen/internal/gitignore"
	"github.com/aaronflorey/gitignore-gen/internal/provider"
)

type fakeAPI struct {
	available    []string
	template     string
	availableErr error
	templateErr  error
	requests     [][]string
}

func (f *fakeAPI) AvailableProviders(_ context.Context) ([]string, error) {
	if f.availableErr != nil {
		return nil, f.availableErr
	}
	return f.available, nil
}

func (f *fakeAPI) FetchTemplate(_ context.Context, providers []string) (api.TemplateResponse, error) {
	if f.templateErr != nil {
		return api.TemplateResponse{}, f.templateErr
	}
	f.requests = append(f.requests, append([]string{}, providers...))
	return api.TemplateResponse{Providers: providers, Content: f.template}, nil
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

func TestDetectResetsManagedSet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := gitignore.NewManager(dir)
	seed := gitignore.BuildManagedBlock([]string{"python"}, "venv/")
	if _, err := manager.UpsertManagedBlock(seed, false); err != nil {
		t.Fatalf("seed failed: %v", err)
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

	res, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !reflect.DeepEqual(res.FinalProviders, []string{"node"}) {
		t.Fatalf("expected detect reset to node only, got %v", res.FinalProviders)
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
	if !reflect.DeepEqual(res.RemoteProviderWarnings[:3], []string{
		"supported provider missing remotely: android",
		"supported provider missing remotely: androidstudio",
		"supported provider missing remotely: angular",
	}) {
		t.Fatalf("unexpected warning prefix: %v", res.RemoteProviderWarnings[:3])
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
		Include: []string{"react", "node", "bad"},
		Exclude: []string{"python", "react", "unknown"},
	})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	if !reflect.DeepEqual(res.DetectedProviders, []string{"go", "node", "python"}) {
		t.Fatalf("unexpected detected providers: %v", res.DetectedProviders)
	}
	if !reflect.DeepEqual(res.IncludedProviders, []string{"node", "react"}) {
		t.Fatalf("unexpected included providers: %v", res.IncludedProviders)
	}
	if !reflect.DeepEqual(res.ExcludedProviders, []string{"python", "react"}) {
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
