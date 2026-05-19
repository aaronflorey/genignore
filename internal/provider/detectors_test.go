package provider

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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

func TestFileExistsDetectorMatchesOneLevelSubdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nested := filepath.Join(dir, "service")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested directory: %v", err)
	}

	path := filepath.Join(nested, "go.mod")
	if err := os.WriteFile(path, []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	result := fileExistsDetector("go", "go.mod", "found go.mod").Detect(context.Background(), dir)
	if result != (Result{Key: "go", Matched: true, Reason: "found go.mod", Evidence: path}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFileExistsDetectorSkipsIgnoredOneLevelSubdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nested := filepath.Join(dir, "service")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("service/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	path := filepath.Join(nested, "go.mod")
	if err := os.WriteFile(path, []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	result := fileExistsDetector("go", "go.mod", "found go.mod").Detect(context.Background(), dir)
	if result != (Result{Key: "go", Matched: false, Reason: "signal not found"}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFileExistsDetectorRespectsNegatedOneLevelSubdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nested := filepath.Join(dir, "service")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*\n!service/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	path := filepath.Join(nested, "go.mod")
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

func TestVSCodeDetectorRequiresWorkspaceMetadata(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, "code")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	detector := Registry()["visualstudiocode"]

	missing := detector.Detect(context.Background(), root)
	if missing != (Result{Key: "visualstudiocode", Matched: false, Reason: "signal not found"}) {
		t.Fatalf("unexpected missing result: %+v", missing)
	}

	workspacePath := filepath.Join(root, ".vscode")
	if err := os.Mkdir(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir .vscode: %v", err)
	}

	matched := detector.Detect(context.Background(), root)
	expected := Result{Key: "visualstudiocode", Matched: true, Reason: "found VS Code workspace metadata", Evidence: workspacePath}
	if matched != expected {
		t.Fatalf("unexpected matched result: %+v", matched)
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

func TestVueDetectorMatchesConfigInOneLevelSubdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "frontend", "vue.config.ts")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir frontend directory: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write vue config: %v", err)
	}

	result := vueDetector().Detect(context.Background(), dir)
	expected := Result{Key: "vue", Matched: true, Reason: "found vue config", Evidence: configPath}
	if result != expected {
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

func TestJetBrainsDetectorRequiresProjectMetadata(t *testing.T) {
	root := t.TempDir()
	installPath := filepath.Join(root, "opt", "jetbrains-toolbox", "apps", "IDEA-U")
	if err := os.MkdirAll(installPath, 0o755); err != nil {
		t.Fatalf("mkdir jetbrains install: %v", err)
	}

	original := ideInstallCandidatesByKey
	ideInstallCandidatesByKey = map[string][]string{
		"jetbrains": {installPath},
	}
	t.Cleanup(func() {
		ideInstallCandidatesByKey = original
	})

	detector := Registry()["jetbrains"]
	missing := detector.Detect(context.Background(), root)
	if missing != (Result{Key: "jetbrains", Matched: false, Reason: "signal not found"}) {
		t.Fatalf("unexpected missing result: %+v", missing)
	}

	metadataPath := filepath.Join(root, ".idea")
	if err := os.Mkdir(metadataPath, 0o755); err != nil {
		t.Fatalf("mkdir .idea: %v", err)
	}

	matched := detector.Detect(context.Background(), root)
	expected := Result{Key: "jetbrains", Matched: true, Reason: "found JetBrains project metadata", Evidence: metadataPath}
	if matched != expected {
		t.Fatalf("unexpected matched result: %+v", matched)
	}
}

func TestContextWithInputsReusesSearchDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ctx := ContextWithInputs(context.Background(), root)
	initial := searchDirsFor(ctx, root)

	lateDir := filepath.Join(root, "late")
	if err := os.Mkdir(lateDir, 0o755); err != nil {
		t.Fatalf("mkdir late directory: %v", err)
	}

	cached := searchDirsFor(ctx, root)
	if !slices.Equal(initial, cached) {
		t.Fatalf("expected cached search dirs to stay stable, got %v want %v", cached, initial)
	}
	if slices.Contains(cached, lateDir) {
		t.Fatalf("expected cached search dirs to exclude late directory: %v", cached)
	}

	fresh := searchDirsFor(context.Background(), root)
	if !slices.Contains(fresh, lateDir) {
		t.Fatalf("expected fresh search dirs to include late directory: %v", fresh)
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

func TestRegistryIncludesAutoDetectableJetBrainsIDEs(t *testing.T) {
	t.Parallel()

	registry := Registry()
	for _, key := range []string{"androidstudio", "appcode", "clion", "goland", "intellij", "jetbrains", "phpstorm", "pycharm", "rider", "rubymine", "webstorm"} {
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

	if !IsSupported("jetbrains") {
		t.Fatalf("expected %q to remain supported", "jetbrains")
	}
	for _, key := range []string{"androidstudio", "appcode", "clion", "goland", "intellij", "phpstorm", "pycharm", "rider", "rubymine", "webstorm"} {
		if IsSupported(key) {
			t.Fatalf("expected %q to be unsupported under the GitHub-backed contract", key)
		}
	}
}

func TestRequestedLanguageDetectorsMatchCommonProjectSignals(t *testing.T) {
	t.Parallel()

	registry := Registry()

	for _, tc := range []struct {
		name       string
		key        string
		relPath    string
		content    string
		expectOnly bool
	}{
		{name: "terraform from tf file", key: "terraform", relPath: "main.tf", content: "terraform {}\n"},
		{name: "rust from cargo file", key: "rust", relPath: "Cargo.toml", content: "[package]\nname = \"demo\"\n"},
		{name: "java from gradle file", key: "java", relPath: "build.gradle", content: "plugins { id 'java' }\n"},
		{name: "kotlin from kt source", key: "kotlin", relPath: "main.kt", content: "fun main() {}\n"},
		{name: "dotnetcore from solution", key: "dotnetcore", relPath: "Demo.sln", content: "\n"},
		{name: "csharp from cs source", key: "csharp", relPath: "Program.cs", content: "class Program {}\n"},
		{name: "dart from pubspec", key: "dart", relPath: "pubspec.yaml", content: "name: demo\n"},
		{name: "swift from package file", key: "swift", relPath: "Package.swift", content: "// swift-tools-version:5.10\n"},
		{name: "xcode from xcodeproj", key: "xcode", relPath: "Demo.xcodeproj", expectOnly: true},
		{name: "android from nested manifest", key: "android", relPath: filepath.Join("app", "src", "main", "AndroidManifest.xml"), content: "<manifest/>\n"},
		{name: "ruby from gemfile", key: "ruby", relPath: "Gemfile", content: "source 'https://rubygems.org'\n"},
		{name: "maven from pom", key: "maven", relPath: "pom.xml", content: "<project/>\n"},
		{name: "rails from config application", key: "rails", relPath: filepath.Join("config", "application.rb"), content: "module Demo\nend\n"},
		{name: "jekyll from config", key: "jekyll", relPath: "_config.yml", content: "title: demo\n"},
		{name: "symfony from lock", key: "symfony", relPath: "symfony.lock", content: "{}\n"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.relPath)
			if tc.expectOnly {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatalf("mkdir signal path: %v", err)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir signal parent: %v", err)
				}
				if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
					t.Fatalf("write signal file: %v", err)
				}
			}

			detector, ok := registry[tc.key]
			if !ok {
				t.Fatalf("missing detector for %q", tc.key)
			}

			result := detector.Detect(context.Background(), dir)
			if !result.Matched || result.Key != tc.key {
				t.Fatalf("expected matched result for %q, got %+v", tc.key, result)
			}
		})
	}
}

func TestNodeDetectorMatchesBunLockfiles(t *testing.T) {
	t.Parallel()

	detector, ok := Registry()["node"]
	if !ok {
		t.Fatalf("missing node detector")
	}

	for _, tc := range []struct {
		name    string
		relPath string
	}{
		{name: "matches bun.lock in root", relPath: "bun.lock"},
		{name: "matches bun.lockb in one-level subdirectory", relPath: filepath.Join("web", "bun.lockb")},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.relPath)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir signal parent: %v", err)
			}
			if err := os.WriteFile(path, []byte("\n"), 0o644); err != nil {
				t.Fatalf("write signal file: %v", err)
			}

			result := detector.Detect(context.Background(), dir)
			expected := Result{Key: "node", Matched: true, Reason: "found node project file", Evidence: path}
			if result != expected {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestNodeDetectorDoesNotMatchDeeperThanOneLevel(t *testing.T) {
	t.Parallel()

	detector, ok := Registry()["node"]
	if !ok {
		t.Fatalf("missing node detector")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "apps", "web", "bun.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir signal parent: %v", err)
	}
	if err := os.WriteFile(path, []byte("\n"), 0o644); err != nil {
		t.Fatalf("write signal file: %v", err)
	}

	result := detector.Detect(context.Background(), dir)
	expected := Result{Key: "node", Matched: false, Reason: "signal not found"}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReactDetectorMatchesPackageJSONInOneLevelSubdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packagePath := filepath.Join(dir, "frontend", "package.json")
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		t.Fatalf("mkdir frontend directory: %v", err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"dependencies":{"react":"^19.0.0"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	result := reactDetector().Detect(context.Background(), dir)
	expected := Result{Key: "react", Matched: true, Reason: "package.json dependency includes react", Evidence: packagePath}
	if result != expected {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFlutterDetectorRequiresFlutterSignalInPubspec(t *testing.T) {
	t.Parallel()

	registry := Registry()
	detector, ok := registry["flutter"]
	if !ok {
		t.Fatalf("missing detector for flutter")
	}

	t.Run("matches when flutter is declared", func(t *testing.T) {
		dir := t.TempDir()
		pubspec := filepath.Join(dir, "pubspec.yaml")
		if err := os.WriteFile(pubspec, []byte("name: demo\nflutter:\n  uses-material-design: true\n"), 0o644); err != nil {
			t.Fatalf("write pubspec: %v", err)
		}

		result := detector.Detect(context.Background(), dir)
		expected := Result{Key: "flutter", Matched: true, Reason: "pubspec.yaml references flutter", Evidence: pubspec}
		if result != expected {
			t.Fatalf("unexpected flutter match result: %+v", result)
		}
	})

	t.Run("does not match plain dart pubspec", func(t *testing.T) {
		dir := t.TempDir()
		pubspec := filepath.Join(dir, "pubspec.yaml")
		if err := os.WriteFile(pubspec, []byte("name: demo\nenvironment:\n  sdk: ^3.0.0\n"), 0o644); err != nil {
			t.Fatalf("write pubspec: %v", err)
		}

		result := detector.Detect(context.Background(), dir)
		expected := Result{Key: "flutter", Matched: false, Reason: "signal not found"}
		if result != expected {
			t.Fatalf("unexpected flutter miss result: %+v", result)
		}
	})
}

func TestRegistryIncludesRequestedLanguageDetectors(t *testing.T) {
	t.Parallel()

	registry := Registry()
	for _, key := range []string{"terraform", "rust", "java", "kotlin", "dotnetcore", "csharp", "dart", "flutter", "swift", "xcode", "android", "ruby", "maven", "rails", "jekyll", "symfony"} {
		detector, ok := registry[key]
		if !ok {
			t.Fatalf("expected registry detector for %q", key)
		}
		if detector == nil {
			t.Fatalf("detector for %q is nil", key)
		}
	}

	for _, key := range []string{"terraform", "rust", "java", "kotlin", "dart", "flutter", "swift", "xcode", "android", "ruby", "maven", "rails", "jekyll", "symfony"} {
		if !IsSupported(key) {
			t.Fatalf("expected %q to be supported", key)
		}
	}
	for _, key := range []string{"dotnetcore", "csharp"} {
		if IsSupported(key) {
			t.Fatalf("expected %q to be unsupported under the GitHub-backed contract", key)
		}
	}
}

func TestRegistryMatchesCuratedRepositoryFixtures(t *testing.T) {
	t.Parallel()

	registry := Registry()
	tests := []struct {
		name    string
		fixture string
		keys    []string
		want    []string
	}{
		{
			name:    "next-vscode-app",
			fixture: "next-vscode-app",
			keys:    []string{"jetbrains", "nextjs", "node", "react", "visualstudiocode"},
			want:    []string{"jetbrains", "nextjs", "node", "react", "visualstudiocode"},
		},
		{
			name:    "laravel-jetbrains-app",
			fixture: "laravel-jetbrains-app",
			keys:    []string{"composer", "jetbrains", "laravel"},
			want:    []string{"composer", "jetbrains", "laravel"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			root := fixtureRepoPath(t, tt.fixture)
			ctx := ContextWithInputs(context.Background(), root)

			matched := make([]string, 0, len(tt.keys))
			for _, key := range tt.keys {
				result := registry[key].Detect(ctx, root)
				if result.Matched {
					matched = append(matched, key)
				}
			}

			if !slices.Equal(matched, tt.want) {
				t.Fatalf("unexpected matched detectors for %s: got %v want %v", tt.fixture, matched, tt.want)
			}
		})
	}
}

func fixtureRepoPath(t *testing.T, name string) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	return filepath.Join(filepath.Clean(filepath.Join(wd, "..", "..")), "testdata", "repos", name)
}
