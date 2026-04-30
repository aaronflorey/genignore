package app

import (
	"path/filepath"
	"strings"
	"testing"

	"os"

	"github.com/aaronflorey/genignore/internal/gitignore"
)

func TestLoadConfigMissingFileReturnsEmptyConfig(t *testing.T) {
	home := t.TempDir()
	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = oldUserHomeDir })

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if len(cfg.Defaults.Providers) != 0 || len(cfg.Defaults.IgnoreRules) != 0 {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestLoadConfigValidFile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, configRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	content := strings.Join([]string{
		"[defaults]",
		"providers = [\"go\", \"node\"]",
		"ignore_rules = [\".direnv/\", \"coverage.out\"]",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = oldUserHomeDir })

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if want := []string{"go", "node"}; strings.Join(cfg.Defaults.Providers, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected providers: %v", cfg.Defaults.Providers)
	}
	if want := []string{".direnv/", "coverage.out"}; strings.Join(cfg.Defaults.IgnoreRules, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected ignore rules: %v", cfg.Defaults.IgnoreRules)
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, configRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("[defaults]\nproviders = \"go\"\n"), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = oldUserHomeDir })

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected config load error")
	}
	if !strings.Contains(err.Error(), "invalid config file "+path) {
		t.Fatalf("expected config path in error, got %v", err)
	}
}

func TestRunInvalidConfigReturnsError(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, configRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("[defaults]\nproviders = \"go\"\n"), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	oldFactory := newCommandService
	newCommandService = func(string, Config) commandService {
		t.Fatal("service should not be created when config is invalid")
		return nil
	}
	t.Cleanup(func() { newCommandService = oldFactory })
	oldCatalogClient := newCatalogClient
	newCatalogClient = func() providerCatalog {
		t.Fatal("catalog client should not be created when config is invalid")
		return nil
	}
	t.Cleanup(func() { newCatalogClient = oldCatalogClient })

	exitCode, stdout, stderr := captureRunOutputWithHome(t, []string{"list"}, home)
	if exitCode != 1 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: invalid config file "+path) {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stderr, "cannot decode TOML string into struct field app.ConfigDefaults.Providers of type []string") {
		t.Fatalf("expected validation detail in stderr: %s", stderr)
	}
}

func TestRunLoadsConfigAndPassesItToService(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, configRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	content := strings.Join([]string{
		"[defaults]",
		"providers = [\"wrangler\"]",
		"ignore_rules = [\".direnv/\"]",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	oldFactory := newCommandService
	newCommandService = func(_ string, cfg Config) commandService {
		if got, want := cfg.Defaults.Providers, []string{"wrangler"}; strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("unexpected providers: %v", got)
		}
		if got, want := cfg.Defaults.IgnoreRules, []string{".direnv/"}; strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("unexpected ignore rules: %v", got)
		}
		return stubCommandService{detectResult: CommandResult{Command: "detect", FinalProviders: []string{"wrangler"}, FileAction: gitignore.FileActionDryRun}}
	}
	t.Cleanup(func() { newCommandService = oldFactory })

	exitCode, _, stderr := captureRunOutputWithHome(t, []string{"detect", "--dry-run"}, home)
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}
