package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

type stubCommandService struct {
	resolveResult ResolveResult
	resolveErr    error
	detectResult  CommandResult
	detectErr     error
	addResult     CommandResult
	addErr        error
	doctorResult  DoctorResult
	doctorErr     error
}

type stubCatalogClient struct {
	providers []string
	err       error
}

func (s stubCommandService) Resolve(_ context.Context, _ ResolveOptions) (ResolveResult, error) {
	return s.resolveResult, s.resolveErr
}

func (s stubCommandService) Detect(_ context.Context, _ DetectOptions) (CommandResult, error) {
	return s.detectResult, s.detectErr
}

func TestResolveCommandOutputShowsReadOnlyProviders(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{resolveResult: ResolveResult{
			Command:                "resolve",
			DetectedProviders:      []string{"go", "node"},
			IncludedProviders:      []string{"react"},
			ExcludedProviders:      []string{"python"},
			FinalProviders:         []string{"go", "node", "react"},
			UnsupportedKeyWarnings: []string{"unsupported provider key: bad"},
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"resolve", "--include", "react", "--exclude", "python"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, fragment := range []string{"Command: resolve", "Detected: go, node", "Included: react", "Excluded: python", "Final: go, node, react", "Warning: unsupported provider key: bad"} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("missing %q in stdout: %s", fragment, stdout)
		}
	}
	if strings.Contains(stdout, "File:") {
		t.Fatalf("resolve output should stay read-only: %s", stdout)
	}
	if strings.Contains(stdout, "Detection:") {
		t.Fatalf("resolve output should omit verbose evidence by default: %s", stdout)
	}
}

func (s stubCommandService) Add(_ context.Context, _ AddOptions) (CommandResult, error) {
	return s.addResult, s.addErr
}

func (s stubCommandService) Doctor(_ context.Context, _ DoctorOptions) (DoctorResult, error) {
	return s.doctorResult, s.doctorErr
}

func (s stubCatalogClient) AvailableProviders(_ context.Context) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]string(nil), s.providers...), nil
}

func TestListCommand(t *testing.T) {
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		return stubCatalogClient{providers: []string{"go", "macos", "node"}}
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	_, stdout, stderr := captureRunOutput(t, []string{"list"})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected command, providers label, and provider lines, got: %v", lines)
	}
	if lines[0] != "Command: list" {
		t.Fatalf("unexpected command line: %q", lines[0])
	}
	if lines[1] != "Providers:" {
		t.Fatalf("unexpected providers label: %q", lines[1])
	}
	providers := lines[2:]
	if !slices.IsSorted(providers) {
		t.Fatalf("expected sorted providers, got %v", providers)
	}
}

func TestSearchCommand(t *testing.T) {
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		return stubCatalogClient{providers: []string{"go", "goland", "macos", "node"}}
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	_, stdout, stderr := captureRunOutput(t, []string{"search", "go"})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected command/query/providers label/provider lines, got: %v", lines)
	}
	if lines[0] != "Command: search" {
		t.Fatalf("unexpected command line: %q", lines[0])
	}
	if lines[1] != "Query: go" {
		t.Fatalf("unexpected query line: %q", lines[1])
	}
	if lines[2] != "Providers:" {
		t.Fatalf("unexpected providers label: %q", lines[2])
	}
	providers := lines[3:]
	if !slices.IsSorted(providers) {
		t.Fatalf("expected sorted search results, got %v", providers)
	}
	for _, key := range providers {
		if !strings.Contains(strings.ToLower(key), "go") {
			t.Fatalf("provider %q does not match query", key)
		}
	}
}

func TestSearchCommandJSON(t *testing.T) {
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		return stubCatalogClient{providers: []string{"go", "goland", "macos", "node"}}
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	_, stdout, stderr := captureRunOutput(t, []string{"search", "go", "--json"})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	var payload CatalogResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if payload.Command != "search" || payload.Query != "go" {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if !slices.IsSorted(payload.Providers) {
		t.Fatalf("expected sorted providers in json output, got %v", payload.Providers)
	}
}

func TestDetectDefaultCommandOutputOmitsEvidence(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:           "detect",
			DetectedProviders: []string{"node", "react"},
			FinalProviders:    []string{"node", "react"},
			DetectionResults: []provider.Result{
				{Key: "node", Matched: true, Reason: "found package.json", Evidence: "/tmp/package.json"},
				{Key: "laravel", Matched: false, Reason: "failed to read signal file", Evidence: "/tmp/composer.json", Error: "read error"},
			},
			FileAction: gitignore.FileActionDryRun,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect", "--dry-run"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "Detected: node, react") || !strings.Contains(stdout, "Final: node, react") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "File: dry-run") {
		t.Fatalf("missing dry-run file action: %s", stdout)
	}
	if strings.Contains(stdout, "Detection:") || strings.Contains(stdout, "/tmp/package.json") || strings.Contains(stdout, "failed to read signal file") {
		t.Fatalf("default output leaked evidence: %s", stdout)
	}
}

func TestAddDefaultCommandOutputShowsWarningsAndFileAction(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{addResult: CommandResult{
			Command:                "add",
			AddedProviders:         []string{"go"},
			FinalProviders:         []string{"go", "node"},
			UnsupportedKeyWarnings: []string{"unsupported provider key: bad", "unsupported provider key: unknown"},
			RemoteProviderWarnings: []string{"supported provider missing remotely: android", "supported provider missing remotely: angular"},
			FileAction:             gitignore.FileActionUpdated,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"add", "go"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, fragment := range []string{
		"Command: add",
		"Added: go",
		"Final: go, node",
		"Warning: unsupported provider key: bad",
		"Warning: unsupported provider key: unknown",
		"Warning: supported provider missing remotely: android",
		"Warning: supported provider missing remotely: angular",
		"File: updated",
	} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("missing %q in stdout: %s", fragment, stdout)
		}
	}
	if strings.Index(stdout, "Warning: unsupported provider key: bad") > strings.Index(stdout, "Warning: unsupported provider key: unknown") {
		t.Fatalf("unsupported warnings out of order: %s", stdout)
	}
	if strings.Index(stdout, "Warning: unsupported provider key: unknown") > strings.Index(stdout, "Warning: supported provider missing remotely: android") {
		t.Fatalf("warning groups out of order: %s", stdout)
	}
	if strings.Index(stdout, "Warning: supported provider missing remotely: android") > strings.Index(stdout, "Warning: supported provider missing remotely: angular") {
		t.Fatalf("remote warnings out of order: %s", stdout)
	}
	if strings.Contains(stdout, "Detected:") || strings.Contains(stdout, "Included:") || strings.Contains(stdout, "Excluded:") {
		t.Fatalf("output included empty sections: %s", stdout)
	}
}

func TestDetectDefaultCommandOutputShowsNoOpFileAction(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:        "detect",
			FinalProviders: []string{"go"},
			FileAction:     gitignore.FileActionNoOp,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "File: no-op") {
		t.Fatalf("missing no-op file action: %s", stdout)
	}
}

func TestDetectDefaultCommandOutputOmitsDiffAfterWrite(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:        "detect",
			FinalProviders: []string{"go"},
			FileAction:     gitignore.FileActionUpdated,
			Diff:           "--- .gitignore\n+++ .gitignore\n@@ managed-block @@\n-# old\n+# new",
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "File: updated") {
		t.Fatalf("missing updated file action: %s", stdout)
	}
	if strings.Contains(stdout, "Diff:") || strings.Contains(stdout, "--- .gitignore") {
		t.Fatalf("default detect output should not show diff after write: %s", stdout)
	}
}

func TestDetectDiffCommandOutputShowsPreviewAndDiff(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:        "detect",
			FinalProviders: []string{"go"},
			FileAction:     gitignore.FileActionUpdated,
			PreviewOnly:    true,
			Diff:           "--- .gitignore\n+++ .gitignore\n@@ managed-block @@\n-# old\n+# new",
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect", "--diff"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, fragment := range []string{"File: updated", "Preview: diff-only (no file written)", "Diff:", "--- .gitignore", "+# new"} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("missing %q in stdout: %s", fragment, stdout)
		}
	}
}

func TestDoctorCommandOutputShowsDiagnostics(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{doctorResult: DoctorResult{
			Command:           "doctor",
			DetectedProviders: []string{"go", "node"},
			FinalProviders:    []string{"go", "node"},
			Detections: []DoctorDetection{
				{Key: "go", Origin: "repository", Matched: true, Reason: "found go.mod", Evidence: "/tmp/project/go.mod"},
				{Key: "macos", Origin: "host", Matched: true, Reason: "matched runtime OS", Evidence: "darwin"},
			},
			Runtime: DoctorRuntime{
				Offline:         true,
				UpstreamCommit:  "abc123",
				RemoteProviders: []string{"go", "node"},
				CacheEntries:    []DoctorCacheEntry{{Provider: "go", State: "fresh"}},
				Decisions:       []string{"runtime.offline is enabled, so remote templates must come from the local cache without a live GitHub refresh"},
			},
			Provenance: []string{"# Provenance: github/gitignore@abc123 [go,node]"},
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"doctor"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, fragment := range []string{"Command: doctor", "Detected: go, node", "Detection: go | repository | matched | found go.mod | /tmp/project/go.mod", "Detection: macos | host | matched | matched runtime OS | darwin", "Offline: true", "Upstream: abc123", "Cache: go | fresh", "Decision: runtime.offline is enabled", "Provenance: github/gitignore@abc123 [go,node]"} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("missing %q in stdout: %s", fragment, stdout)
		}
	}
}

func TestDetectIgnoresGENIGNOREEnvironmentVariables(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:        "detect",
			FinalProviders: []string{"go"},
			FileAction:     gitignore.FileActionNoOp,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })
	t.Setenv("GENIGNORE_JSON", "1")
	t.Setenv("GENIGNORE_VERBOSE", "1")

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if strings.HasPrefix(stdout, "{") {
		t.Fatalf("expected human-readable output, got json: %s", stdout)
	}
	if !strings.Contains(stdout, "Final: go") || !strings.Contains(stdout, "File: no-op") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if strings.Contains(stdout, "Detection:") {
		t.Fatalf("unexpected verbose output from env vars: %s", stdout)
	}
}

func TestDetectVerboseCommandShowsEvidence(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectResult: CommandResult{
			Command:        "detect",
			FinalProviders: []string{"node"},
			DetectionResults: []provider.Result{
				{Key: "node", Matched: true, Reason: "found package.json", Evidence: "/tmp/package.json"},
				{Key: "react", Matched: false, Reason: "invalid package.json", Evidence: "/tmp/package.json", Error: "unexpected EOF"},
				{Key: "python", Matched: false, Reason: "signal not found"},
			},
			FileAction: gitignore.FileActionDryRun,
		}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect", "--dry-run", "--verbose"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "Detection: node | matched | found package.json | /tmp/package.json") {
		t.Fatalf("missing matched evidence: %s", stdout)
	}
	if !strings.Contains(stdout, "Detection: react | error | invalid package.json | /tmp/package.json | unexpected EOF") {
		t.Fatalf("missing error evidence: %s", stdout)
	}
	if strings.Contains(stdout, "python | skipped") {
		t.Fatalf("verbose output included unmatched noise: %s", stdout)
	}
}

func TestDetectCommandFailureReturnsNonZero(t *testing.T) {
	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		return stubCommandService{detectErr: errors.New("boom")}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, stdout, stderr := captureRunOutput(t, []string{"detect"})
	if exitCode != 1 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "error: boom") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func captureRunOutput(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	return captureRunOutputWithHome(t, args, t.TempDir())
}

func captureRunOutputWithHome(t *testing.T, args []string, home string) (int, string, string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	t.Setenv("HOME", home)

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe failed: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe failed: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	exitCode := Run(args)

	_ = stdoutW.Close()
	_ = stderrW.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	_, _ = io.Copy(&stdoutBuf, stdoutR)
	_, _ = io.Copy(&stderrBuf, stderrR)

	return exitCode, strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String())
}
