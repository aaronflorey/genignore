package app

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

func TestJSONDetectCommandContract(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:                "detect",
			CWD:                    "/tmp/project",
			DetectedProviders:      []string{"go", "node"},
			IncludedProviders:      []string{"react"},
			ExcludedProviders:      []string{"python"},
			FinalProviders:         []string{"go", "node", "react"},
			UnsupportedKeyWarnings: []string{"unsupported provider key: bad"},
			RuntimeWarnings:        []string{"runtime.offline is enabled; remote templates were loaded from the local cache without a live GitHub refresh"},
			RemoteProviderWarnings: []string{"supported provider missing remotely: android"},
			DetectionResults: []provider.Result{
				{Key: "go", Matched: true, Reason: "found go.mod", Evidence: "/tmp/project/go.mod"},
				{Key: "node", Matched: true, Reason: "found package.json", Evidence: "/tmp/project/package.json"},
			},
			FileAction:            gitignore.FileActionDryRun,
			PreviewOnly:           true,
			Diff:                  "--- .gitignore\n+++ .gitignore\n@@ managed-block @@\n-# old\n+# new",
			TemplateProviderCount: 3,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload CommandResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if payload.Command != "detect" || payload.CWD != "/tmp/project" {
		t.Fatalf("unexpected command metadata: %+v", payload)
	}
	if !slices.Equal(payload.DetectedProviders, []string{"go", "node"}) {
		t.Fatalf("unexpected detected providers: %v", payload.DetectedProviders)
	}
	if !slices.Equal(payload.IncludedProviders, []string{"react"}) {
		t.Fatalf("unexpected included providers: %v", payload.IncludedProviders)
	}
	if !slices.Equal(payload.ExcludedProviders, []string{"python"}) {
		t.Fatalf("unexpected excluded providers: %v", payload.ExcludedProviders)
	}
	if !slices.Equal(payload.FinalProviders, []string{"go", "node", "react"}) {
		t.Fatalf("unexpected final providers: %v", payload.FinalProviders)
	}
	if !slices.Equal(payload.UnsupportedKeyWarnings, []string{"unsupported provider key: bad"}) {
		t.Fatalf("unexpected unsupported warnings: %v", payload.UnsupportedKeyWarnings)
	}
	if !slices.Equal(payload.RuntimeWarnings, []string{"runtime.offline is enabled; remote templates were loaded from the local cache without a live GitHub refresh"}) {
		t.Fatalf("unexpected runtime warnings: %v", payload.RuntimeWarnings)
	}
	if !slices.Equal(payload.RemoteProviderWarnings, []string{"supported provider missing remotely: android"}) {
		t.Fatalf("unexpected remote warnings: %v", payload.RemoteProviderWarnings)
	}
	if len(payload.DetectionResults) != 2 || payload.DetectionResults[0].Key != "go" || payload.DetectionResults[1].Key != "node" {
		t.Fatalf("unexpected detection results: %+v", payload.DetectionResults)
	}
	if payload.FileAction != gitignore.FileActionDryRun {
		t.Fatalf("unexpected file action: %s", payload.FileAction)
	}
	if !payload.PreviewOnly || payload.Diff == "" {
		t.Fatalf("expected preview metadata in payload: %+v", payload)
	}
	if payload.TemplateProviderCount != 3 {
		t.Fatalf("unexpected template provider count: %d", payload.TemplateProviderCount)
	}
	if strings.Contains(stdout, "Command:") {
		t.Fatalf("json payload contains human-readable labels: %s", stdout)
	}
}

func TestJSONDoctorCommandContract(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{doctorResult: DoctorResult{
			Command:           "doctor",
			CWD:               "/tmp/project",
			DetectedProviders: []string{"go", "node"},
			FinalProviders:    []string{"go", "node"},
			Detections:        []DoctorDetection{{Key: "go", Matched: true, Origin: "repository", Reason: "found go.mod", Evidence: "/tmp/project/go.mod"}},
			Runtime: DoctorRuntime{
				UpstreamCommit:  "abc123",
				Offline:         true,
				RemoteProviders: []string{"go", "node"},
				CacheEntries:    []DoctorCacheEntry{{Provider: "go", State: "fresh"}},
				Decisions:       []string{"runtime.offline is enabled, so remote templates must come from the local cache without a live GitHub refresh"},
			},
			Provenance: []string{"# Provenance: github/gitignore@abc123 [go,node]"},
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"doctor", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload DoctorResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if payload.Command != "doctor" || payload.CWD != "/tmp/project" {
		t.Fatalf("unexpected doctor metadata: %+v", payload)
	}
	if len(payload.Detections) != 1 || payload.Detections[0].Origin != "repository" {
		t.Fatalf("unexpected detections: %+v", payload.Detections)
	}
	if payload.Runtime.UpstreamCommit != "abc123" || !payload.Runtime.Offline {
		t.Fatalf("unexpected runtime payload: %+v", payload.Runtime)
	}
	if len(payload.Provenance) != 1 || !strings.Contains(payload.Provenance[0], "github/gitignore@abc123") {
		t.Fatalf("unexpected provenance payload: %+v", payload.Provenance)
	}
	if strings.Contains(stdout, "Command:") {
		t.Fatalf("json payload contains human-readable labels: %s", stdout)
	}
}

func TestJSONAddCommandContractOmitsDetectOnlyFields(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{addResult: CommandResult{
			Command:                "add",
			CWD:                    "/tmp/project",
			AddedProviders:         []string{"go"},
			FinalProviders:         []string{"go", "node"},
			UnsupportedKeyWarnings: []string{"unsupported provider key: bad", "unsupported provider key: unknown"},
			RuntimeWarnings:        []string{"runtime.offline is enabled; remote templates were loaded from the local cache without a live GitHub refresh"},
			RemoteProviderWarnings: []string{"supported provider missing remotely: android", "supported provider missing remotely: angular"},
			FileAction:             gitignore.FileActionUpdated,
			TemplateProviderCount:  2,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"add", "go", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	for _, field := range []string{"detectedProviders", "includedProviders", "excludedProviders", "detectionResults"} {
		if _, ok := payload[field]; ok {
			t.Fatalf("unexpected detect-only field %q in add payload: %s", field, stdout)
		}
	}

	var result CommandResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid add payload: %v", err)
	}
	if result.Command != "add" || result.CWD != "/tmp/project" {
		t.Fatalf("unexpected command metadata: %+v", result)
	}
	if !slices.Equal(result.AddedProviders, []string{"go"}) {
		t.Fatalf("unexpected added providers: %v", result.AddedProviders)
	}
	if !slices.Equal(result.FinalProviders, []string{"go", "node"}) {
		t.Fatalf("unexpected final providers: %v", result.FinalProviders)
	}
	if !slices.Equal(result.UnsupportedKeyWarnings, []string{"unsupported provider key: bad", "unsupported provider key: unknown"}) {
		t.Fatalf("unexpected unsupported warnings: %v", result.UnsupportedKeyWarnings)
	}
	if !slices.Equal(result.RuntimeWarnings, []string{"runtime.offline is enabled; remote templates were loaded from the local cache without a live GitHub refresh"}) {
		t.Fatalf("unexpected runtime warnings: %v", result.RuntimeWarnings)
	}
	if !slices.Equal(result.RemoteProviderWarnings, []string{"supported provider missing remotely: android", "supported provider missing remotely: angular"}) {
		t.Fatalf("unexpected remote warnings: %v", result.RemoteProviderWarnings)
	}
	if result.FileAction != gitignore.FileActionUpdated || result.TemplateProviderCount != 2 {
		t.Fatalf("unexpected add payload result: %+v", result)
	}
	if strings.Contains(stdout, "Command:") {
		t.Fatalf("json payload contains human-readable labels: %s", stdout)
	}
}

func TestJSONDetectNoOpFileActionContract(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:               "detect",
			CWD:                   "/tmp/project",
			FinalProviders:        []string{"go"},
			FileAction:            gitignore.FileActionNoOp,
			TemplateProviderCount: 1,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload CommandResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if payload.FileAction != gitignore.FileActionNoOp {
		t.Fatalf("unexpected no-op file action: %s", payload.FileAction)
	}
}

func TestJSONListCatalogContract(t *testing.T) {
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		return stubCatalogClient{providers: []string{"go", "macos", "node"}}
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"list", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload CatalogResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid list json output: %v", err)
	}
	if payload.Command != "list" {
		t.Fatalf("unexpected list command: %+v", payload)
	}
	if !slices.IsSorted(payload.Providers) {
		t.Fatalf("expected sorted providers, got %v", payload.Providers)
	}
	if strings.Contains(stdout, "Providers:") {
		t.Fatalf("json payload contains human-readable labels: %s", stdout)
	}
}

func TestJSONSearchCatalogContract(t *testing.T) {
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		return stubCatalogClient{providers: []string{"go", "goland", "macos", "node"}}
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"search", "go", "--json"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload CatalogResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid search json output: %v", err)
	}
	if payload.Command != "search" || payload.Query != "go" {
		t.Fatalf("unexpected search payload metadata: %+v", payload)
	}
	if !slices.IsSorted(payload.Providers) {
		t.Fatalf("expected sorted providers, got %v", payload.Providers)
	}
	if strings.Contains(stdout, "Providers:") {
		t.Fatalf("json payload contains human-readable labels: %s", stdout)
	}
}
