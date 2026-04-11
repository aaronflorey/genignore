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

func TestIDEDetectorsMatchConfiguredMacAndLinuxInstallCandidates(t *testing.T) {
	for _, tc := range []struct {
		name       string
		installRel string
	}{
		{name: "macOS-style app bundle", installRel: "Applications/PhpStorm.app"},
		{name: "linux-style install dir", installRel: "opt/phpstorm"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			installPath := filepath.Join(root, filepath.FromSlash(tc.installRel))
			if err := os.MkdirAll(installPath, 0o755); err != nil {
				t.Fatalf("mkdir install path: %v", err)
			}

			original := ideInstallCandidatesByKey
			ideInstallCandidatesByKey = map[string][]string{
				"phpstorm": {filepath.Join(root, "Applications", "PhpStorm.app"), filepath.Join(root, "opt", "phpstorm")},
			}
			t.Cleanup(func() {
				ideInstallCandidatesByKey = original
			})

			result := Registry()["phpstorm"].Detect(context.Background(), root)
			expected := Result{Key: "phpstorm", Matched: true, Reason: "detected installed application", Evidence: installPath}
			if result != expected {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestJetBrainsLanguageInferencePrefersPhpStormForPHPProjects(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "IDEA-U")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}
	composerPath := filepath.Join(root, "composer.json")
	if err := os.WriteFile(composerPath, []byte(`{"name":"example/project"}`), 0o644); err != nil {
		t.Fatalf("write composer.json: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"phpstorm":  {filepath.Join(root, "missing", "PhpStorm.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["phpstorm"].Detect(context.Background(), root)
	expected := Result{Key: "phpstorm", Matched: true, Reason: "inferred from jetbrains install and composer.json", Evidence: jetbrainsPath}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJetBrainsLanguageInferencePrefersGoLandForGoProjects(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "GoLand")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}
	goModPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module example.com/genignore\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"goland":    {filepath.Join(root, "missing", "GoLand.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["goland"].Detect(context.Background(), root)
	expected := Result{Key: "goland", Matched: true, Reason: "inferred from jetbrains install and go.mod", Evidence: jetbrainsPath}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJetBrainsLanguageInferenceDoesNotMatchWithoutLanguageSignal(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "IDEA-U")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"phpstorm":  {filepath.Join(root, "missing", "PhpStorm.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["phpstorm"].Detect(context.Background(), root)
	expected := Result{Key: "phpstorm", Matched: false, Reason: "application not found"}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJetBrainsLanguageInferencePrefersPyCharmForPythonProjects(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "PyCharm")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}
	pyProjectPath := filepath.Join(root, "pyproject.toml")
	if err := os.WriteFile(pyProjectPath, []byte("[project]\nname='demo'\n"), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"pycharm":   {filepath.Join(root, "missing", "PyCharm.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["pycharm"].Detect(context.Background(), root)
	expected := Result{Key: "pycharm", Matched: true, Reason: "inferred from jetbrains install and pyproject.toml", Evidence: jetbrainsPath}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJetBrainsLanguageInferencePrefersWebStormForJavaScriptProjects(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "WebStorm")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}
	packageJSONPath := filepath.Join(root, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"webstorm":  {filepath.Join(root, "missing", "WebStorm.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["webstorm"].Detect(context.Background(), root)
	expected := Result{Key: "webstorm", Matched: true, Reason: "inferred from jetbrains install and package.json", Evidence: jetbrainsPath}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJetBrainsLanguageInferencePrefersLanguageSpecificIDEs(t *testing.T) {
	t.Run("Ruby infers RubyMine", func(t *testing.T) {
		root := t.TempDir()
		jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "RubyMine")
		if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
			t.Fatalf("mkdir jetbrains path: %v", err)
		}
		signalPath := filepath.Join(root, "Gemfile")
		if err := os.WriteFile(signalPath, []byte("source 'https://rubygems.org'\n"), 0o644); err != nil {
			t.Fatalf("write Gemfile: %v", err)
		}

		original := ideInstallCandidatesByKey
		ideInstallCandidatesByKey = map[string][]string{
			"rubymine":  {filepath.Join(root, "missing", "RubyMine.app")},
			"jetbrains": {jetbrainsPath},
		}
		t.Cleanup(func() {
			ideInstallCandidatesByKey = original
		})

		result := Registry()["rubymine"].Detect(context.Background(), root)
		expected := Result{Key: "rubymine", Matched: true, Reason: "inferred from jetbrains install and Gemfile", Evidence: jetbrainsPath}
		if result != expected {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("DotNet infers Rider", func(t *testing.T) {
		root := t.TempDir()
		jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "Rider")
		if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
			t.Fatalf("mkdir jetbrains path: %v", err)
		}
		signalPath := filepath.Join(root, "Demo.sln")
		if err := os.WriteFile(signalPath, []byte("\n"), 0o644); err != nil {
			t.Fatalf("write solution file: %v", err)
		}

		original := ideInstallCandidatesByKey
		ideInstallCandidatesByKey = map[string][]string{
			"rider":     {filepath.Join(root, "missing", "Rider.app")},
			"jetbrains": {jetbrainsPath},
		}
		t.Cleanup(func() {
			ideInstallCandidatesByKey = original
		})

		result := Registry()["rider"].Detect(context.Background(), root)
		expected := Result{Key: "rider", Matched: true, Reason: "inferred from jetbrains install and .sln/.csproj", Evidence: jetbrainsPath}
		if result != expected {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("C and C++ infers CLion", func(t *testing.T) {
		root := t.TempDir()
		jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "CLion")
		if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
			t.Fatalf("mkdir jetbrains path: %v", err)
		}
		signalPath := filepath.Join(root, "CMakeLists.txt")
		if err := os.WriteFile(signalPath, []byte("cmake_minimum_required(VERSION 3.20)\n"), 0o644); err != nil {
			t.Fatalf("write CMakeLists.txt: %v", err)
		}

		original := ideInstallCandidatesByKey
		ideInstallCandidatesByKey = map[string][]string{
			"clion":     {filepath.Join(root, "missing", "CLion.app")},
			"jetbrains": {jetbrainsPath},
		}
		t.Cleanup(func() {
			ideInstallCandidatesByKey = original
		})

		result := Registry()["clion"].Detect(context.Background(), root)
		expected := Result{Key: "clion", Matched: true, Reason: "inferred from jetbrains install and CMakeLists.txt", Evidence: jetbrainsPath}
		if result != expected {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}

func TestJetBrainsLanguageInferenceDoesNotMatchWithoutMatchingSignalForNewIDEPaths(t *testing.T) {
	root := t.TempDir()
	jetbrainsPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "WebStorm")
	if err := os.MkdirAll(jetbrainsPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains path: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"webstorm":  {filepath.Join(root, "missing", "WebStorm.app")},
		"jetbrains": {jetbrainsPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	result := Registry()["webstorm"].Detect(context.Background(), root)
	expected := Result{Key: "webstorm", Matched: false, Reason: "application not found"}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRegistryIncludesAutoDetectableSupportedJetBrainsIDEs(t *testing.T) {
	t.Parallel()

	registry := Registry()
	for _, key := range []string{"androidstudio", "appcode", "clion", "goland", "intellij", "jetbrains", "phpstorm", "pycharm", "rider", "rubymine", "webstorm"} {
		if !IsSupported(key) {
			t.Fatalf("expected %q to be supported", key)
		}
		detector, ok := registry[key]
		if !ok {
			t.Fatalf("expected registry detector for %q", key)
		}
		if detector == nil {
			t.Fatalf("detector for %q is nil", key)
		}
		if len(ideInstallCandidatesForKey(key)) == 0 {
			t.Fatalf("expected non-empty install candidates for %q", key)
		}
	}
}
