package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/aaronflorey/genignore/internal/gitignore"
)

func TestCompiledBinaryDetectDryRunJSON(t *testing.T) {
	t.Parallel()

	binaryPath := buildCompiledBinary(t)
	repoDir := copyRepoFixture(t, "node-app")
	homeDir := prepareOfflineHome(t, map[string]string{"node": "node_modules/\n"})

	result := runCompiledBinary(t, binaryPath, repoDir, homeDir, "detect", "--dry-run", "--exclude", "linux,macos,windows", "--json")
	if result.exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%s", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %s", result.stderr)
	}

	var payload CommandResult
	if err := json.Unmarshal([]byte(result.stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v\nstdout=%s", err, result.stdout)
	}
	if payload.Command != "detect" {
		t.Fatalf("unexpected command: %+v", payload)
	}
	wantDetected := []string{"node"}
	if osProvider := runtimeProviderKey(); osProvider != "" {
		wantDetected = append(wantDetected, osProvider)
		slices.Sort(wantDetected)
	}
	if !reflect.DeepEqual(payload.DetectedProviders, wantDetected) {
		t.Fatalf("unexpected detected providers: %v", payload.DetectedProviders)
	}
	if !reflect.DeepEqual(payload.FinalProviders, []string{"node"}) {
		t.Fatalf("unexpected final providers: %v", payload.FinalProviders)
	}
	if !reflect.DeepEqual(payload.ExcludedProviders, []string{"linux", "macos", "windows"}) {
		t.Fatalf("unexpected excluded providers: %v", payload.ExcludedProviders)
	}
	if payload.FileAction != gitignore.FileActionDryRun {
		t.Fatalf("unexpected file action: %s", payload.FileAction)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid writing .gitignore")
	}
}

func TestCompiledBinaryAddUpdatesManagedBlock(t *testing.T) {
	t.Parallel()

	binaryPath := buildCompiledBinary(t)
	repoDir := copyRepoFixture(t, "managed-node")
	homeDir := prepareOfflineHome(t, map[string]string{
		"go":   "bin/\n",
		"node": "node_modules/\n",
	})

	result := runCompiledBinary(t, binaryPath, repoDir, homeDir, "add", "go")
	if result.exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%s", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %s", result.stderr)
	}
	for _, fragment := range []string{"Command: add", "Added: go", "Final: go, node", "File: updated"} {
		if !bytes.Contains([]byte(result.stdout), []byte(fragment)) {
			t.Fatalf("missing %q in stdout: %s", fragment, result.stdout)
		}
	}

	content, err := os.ReadFile(filepath.Join(repoDir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	expected := gitignore.BuildManagedBlock([]string{"go", "node"}, "bin/\n\nnode_modules/\n") + "# user rule\ncoverage.out\n"
	if string(content) != expected {
		t.Fatalf("unexpected .gitignore contents\nwant:\n%s\n got:\n%s", expected, string(content))
	}
}

func TestCompiledBinaryListJSON(t *testing.T) {
	t.Parallel()

	binaryPath := buildCompiledBinary(t)
	repoDir := copyRepoFixture(t, "node-app")
	homeDir := prepareOfflineHome(t, nil)

	result := runCompiledBinary(t, binaryPath, repoDir, homeDir, "list", "--json")
	if result.exitCode != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%s", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %s", result.stderr)
	}

	var payload CatalogResult
	if err := json.Unmarshal([]byte(result.stdout), &payload); err != nil {
		t.Fatalf("invalid json output: %v\nstdout=%s", err, result.stdout)
	}
	if payload.Command != "list" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if !slices.IsSorted(payload.Providers) {
		t.Fatalf("expected sorted providers, got %v", payload.Providers)
	}
	for _, key := range []string{"go", "node", "wrangler"} {
		if !slices.Contains(payload.Providers, key) {
			t.Fatalf("expected provider %q in %v", key, payload.Providers)
		}
	}
}

type compiledRunResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func buildCompiledBinary(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "genignore")
	cmd := exec.Command("go", "build", "-o", path, ".")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build compiled binary: %v\n%s", err, string(output))
	}
	return path
}

func copyRepoFixture(t *testing.T, name string) string {
	t.Helper()

	dst := t.TempDir()
	src := filepath.Join(repoRoot(t), "testdata", "repos", name)
	copyDir(t, src, dst)
	return dst
}

func prepareOfflineHome(t *testing.T, templates map[string]string) string {
	t.Helper()

	homeDir := t.TempDir()
	configPath := filepath.Join(homeDir, ".config", "genignore", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("[runtime]\noffline = true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cacheHome := filepath.Join(homeDir, ".cache")
	for key, content := range templates {
		path := filepath.Join(cacheHome, "genignore", "github-gitignore", "templates", key+".gitignore")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write cache template: %v", err)
		}
	}
	if osProvider := runtimeProviderKey(); osProvider != "" {
		if _, ok := templates[osProvider]; !ok {
			path := filepath.Join(cacheHome, "genignore", "github-gitignore", "templates", osProvider+".gitignore")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir cache dir: %v", err)
			}
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				t.Fatalf("write os cache template: %v", err)
			}
		}
	}

	return homeDir
}

func runtimeProviderKey() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux", "windows":
		return runtime.GOOS
	default:
		return ""
	}
}

func runCompiledBinary(t *testing.T, binaryPath string, cwd string, homeDir string, args ...string) compiledRunResult {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"HOME="+homeDir,
		"XDG_CACHE_HOME="+filepath.Join(homeDir, ".cache"),
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := compiledRunResult{stdout: stdout.String(), stderr: stderr.String()}
	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run compiled binary: %v", err)
	}
	result.exitCode = exitErr.ExitCode()
	return result
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func copyDir(t *testing.T, src string, dst string) {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read dir %s: %v", src, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", dstPath, err)
			}
			copyDir(t, srcPath, dstPath)
			continue
		}

		content, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read fixture %s: %v", srcPath, err)
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat fixture %s: %v", srcPath, err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			t.Fatalf("mkdir parent %s: %v", dstPath, err)
		}
		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			t.Fatalf("write fixture %s: %v", dstPath, err)
		}
	}
}
