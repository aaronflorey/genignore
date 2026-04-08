package provider

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileExistsDetectorMatchesPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	result := fileExistsDetector("go", "go.mod", "found go.mod").Detect(context.Background(), dir)
	if result != (Result{Key: "go", Matched: true, Reason: "found go.mod", Evidence: path}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOSDetectorProvidesStableEvidence(t *testing.T) {
	t.Parallel()

	matched := osDetector("runtime", runtime.GOOS).Detect(context.Background(), t.TempDir())
	if matched != (Result{Key: "runtime", Matched: true, Reason: "matched runtime OS", Evidence: runtime.GOOS}) {
		t.Fatalf("unexpected matched result: %+v", matched)
	}

	mismatched := osDetector("runtime", runtime.GOOS+"-other").Detect(context.Background(), t.TempDir())
	if mismatched != (Result{Key: "runtime", Matched: false, Reason: "runtime OS mismatch", Evidence: runtime.GOOS}) {
		t.Fatalf("unexpected mismatched result: %+v", mismatched)
	}
}

func TestPathDetectorMatchesAndMisses(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, "code")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	t.Setenv("PATH", binDir)

	matched := pathDetector("visualstudiocode", []string{"code"}, "found VS Code binary in PATH").Detect(context.Background(), t.TempDir())
	if matched != (Result{Key: "visualstudiocode", Matched: true, Reason: "found VS Code binary in PATH", Evidence: binaryPath}) {
		t.Fatalf("unexpected matched result: %+v", matched)
	}

	missing := pathDetector("visualstudiocode", []string{"missing-binary"}, "found VS Code binary in PATH").Detect(context.Background(), t.TempDir())
	if missing != (Result{Key: "visualstudiocode", Matched: false, Reason: "binary not found in PATH"}) {
		t.Fatalf("unexpected missing result: %+v", missing)
	}
}

func TestAppDetectorMatchesInstalledPath(t *testing.T) {
	t.Parallel()

	appPath := filepath.Join(t.TempDir(), "PhpStorm.app")
	if err := os.Mkdir(appPath, 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}

	matched := appDetector("phpstorm", []string{appPath}).Detect(context.Background(), t.TempDir())
	if matched != (Result{Key: "phpstorm", Matched: true, Reason: "detected installed application", Evidence: appPath}) {
		t.Fatalf("unexpected matched result: %+v", matched)
	}
}

func TestAppDetectorReportsStructuredInspectErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blockingPath := filepath.Join(dir, "Applications")
	if err := os.WriteFile(blockingPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking path: %v", err)
	}

	appPath := filepath.Join(blockingPath, "PhpStorm.app")
	result := appDetector("phpstorm", []string{appPath}).Detect(context.Background(), t.TempDir())
	if result.Key != "phpstorm" || result.Matched || result.Reason != "failed to inspect application path" || result.Evidence != appPath || result.Error == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReactAndLaravelDetectorsReportStructuredErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packagePath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(packagePath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	composerPath := filepath.Join(dir, "composer.json")
	if err := os.Mkdir(composerPath, 0o755); err != nil {
		t.Fatalf("mkdir composer.json: %v", err)
	}

	react := reactDetector().Detect(context.Background(), dir)
	if react.Key != "react" || react.Matched || react.Reason != "invalid package.json" || react.Evidence != packagePath || react.Error == "" {
		t.Fatalf("unexpected react result: %+v", react)
	}

	laravel := laravelDetector().Detect(context.Background(), dir)
	if laravel.Key != "laravel" || laravel.Matched || laravel.Reason != "failed to read signal file" || laravel.Evidence != composerPath || laravel.Error == "" {
		t.Fatalf("unexpected laravel result: %+v", laravel)
	}
}

func TestVueDetectorDoesNotMatchGenericViteConfigWithoutVueSignal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vite.config.ts"), []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write vite config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"vite":"^7.0.0"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	result := vueDetector().Detect(context.Background(), dir)
	if result != (Result{Key: "vue", Matched: false, Reason: "signal not found"}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVueAndReactDetectorsMatchOnlyRealPackageSignals(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packagePath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(packagePath, []byte(`{"dependencies":{"vue":"^3.0.0"},"devDependencies":{"react":"^19.0.0"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	vue := vueDetector().Detect(context.Background(), dir)
	if vue != (Result{Key: "vue", Matched: true, Reason: "package.json dependency includes vue", Evidence: packagePath}) {
		t.Fatalf("unexpected vue result: %+v", vue)
	}

	react := reactDetector().Detect(context.Background(), dir)
	if react != (Result{Key: "react", Matched: true, Reason: "package.json devDependency includes react", Evidence: packagePath}) {
		t.Fatalf("unexpected react result: %+v", react)
	}
}
