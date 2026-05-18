package provider

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	gitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

var ideInstallCandidatesByKey = map[string][]string{
	"phpstorm": {
		"/Applications/PhpStorm.app",
		"/opt/phpstorm",
		"/opt/PhpStorm",
	},
	"jetbrains": {
		"/Applications/IntelliJ IDEA.app",
		"/Applications/PhpStorm.app",
		"/Applications/PyCharm.app",
		"/Applications/WebStorm.app",
		"/opt/jetbrains",
		"/opt/JetBrains",
		"/var/lib/flatpak/app/com.jetbrains.IntelliJ-IDEA-Community",
		"/var/lib/flatpak/app/com.jetbrains.IntelliJ-IDEA-Ultimate",
		"/var/lib/flatpak/app/com.jetbrains.PhpStorm",
	},
	"intellij": {
		"/Applications/IntelliJ IDEA.app",
		"/opt/intellij-idea",
		"/opt/idea",
	},
	"pycharm": {
		"/Applications/PyCharm.app",
		"/opt/pycharm",
		"/opt/PyCharm",
	},
	"webstorm": {
		"/Applications/WebStorm.app",
		"/opt/webstorm",
		"/opt/WebStorm",
	},
	"goland": {
		"/Applications/GoLand.app",
		"/opt/goland",
		"/opt/GoLand",
	},
	"rubymine": {
		"/Applications/RubyMine.app",
		"/opt/rubymine",
		"/opt/RubyMine",
	},
	"rider": {
		"/Applications/Rider.app",
		"/opt/rider",
		"/opt/Rider",
	},
	"clion": {
		"/Applications/CLion.app",
		"/opt/clion",
		"/opt/CLion",
	},
	"appcode": {
		"/Applications/AppCode.app",
		"/opt/appcode",
		"/opt/AppCode",
	},
	"androidstudio": {
		"/Applications/Android Studio.app",
		"/opt/android-studio",
		"/opt/AndroidStudio",
	},
}

type DetectorFunc func(ctx context.Context, cwd string) Result

type detectorInputs struct {
	cwd        string
	searchDirs []string
}

type detectorInputsContextKey struct{}

func (f DetectorFunc) Detect(ctx context.Context, cwd string) Result {
	return f(ctx, cwd)
}

func Registry() map[string]Detector {
	return map[string]Detector{
		"composer":         fileExistsDetector("composer", "composer.json", "found composer.json"),
		"node":             nodeDetector(),
		"go":               fileExistsDetector("go", "go.mod", "found go.mod"),
		"terraform":        anyGlobDetector("terraform", "found terraform file", "*.tf", "*.tfvars", ".terraform.lock.hcl"),
		"rust":             fileExistsDetector("rust", "Cargo.toml", "found Cargo.toml"),
		"java":             anyFileDetector("java", []string{"pom.xml", "build.gradle", "build.gradle.kts"}, "found java project file"),
		"kotlin":           anySignalDetector("kotlin", signalDetector{reason: "found kotlin project file", match: anySignalMatch(anyFileSignal("build.gradle.kts", "settings.gradle.kts"), anyGlobSignal("found kotlin source file", "*.kt"))}),
		"dotnetcore":       anyGlobDetector("dotnetcore", "found dotnet project file", "*.sln", "*.csproj"),
		"csharp":           anySignalDetector("csharp", signalDetector{reason: "found csharp project file", match: anySignalMatch(anyGlobSignal("found csharp solution/project file", "*.sln", "*.csproj"), anyGlobSignal("found csharp source file", "*.cs"))}),
		"dart":             fileExistsDetector("dart", "pubspec.yaml", "found pubspec.yaml"),
		"flutter":          flutterDetector(),
		"swift":            anySignalDetector("swift", signalDetector{reason: "found swift project file", match: anySignalMatch(fileSignal("Package.swift"), anyGlobSignal("found swift source file", "*.swift"))}),
		"xcode":            anyGlobDetector("xcode", "found xcode project file", "*.xcodeproj", "*.xcworkspace"),
		"android":          anyFileDetector("android", []string{"AndroidManifest.xml", filepath.Join("app", "src", "main", "AndroidManifest.xml")}, "found android manifest"),
		"ruby":             fileExistsDetector("ruby", "Gemfile", "found Gemfile"),
		"maven":            fileExistsDetector("maven", "pom.xml", "found pom.xml"),
		"rails":            anyFileDetector("rails", []string{filepath.Join("bin", "rails"), filepath.Join("config", "application.rb")}, "found rails project file"),
		"jekyll":           fileExistsDetector("jekyll", "_config.yml", "found _config.yml"),
		"symfony":          anyFileDetector("symfony", []string{filepath.Join("bin", "console"), filepath.Join("config", "bundles.php"), "symfony.lock"}, "found symfony project file"),
		"laravel":          laravelDetector(),
		"nextjs":           anyFileDetector("nextjs", []string{"next.config.js", "next.config.mjs", "next.config.ts"}, "found next config"),
		"nuxtjs":           anyFileDetector("nuxtjs", []string{"nuxt.config.js", "nuxt.config.mjs", "nuxt.config.ts"}, "found nuxt config"),
		"python":           anyFileDetector("python", []string{"pyproject.toml", "requirements.txt", "setup.py"}, "found python project file"),
		"vue":              vueDetector(),
		"react":            reactDetector(),
		"macos":            osDetector("macos", "darwin"),
		"linux":            osDetector("linux", "linux"),
		"windows":          osDetector("windows", "windows"),
		"visualstudiocode": vscodeProjectDetector(),
		"phpstorm":         ideWithJetBrainsLanguageInferenceDetector("phpstorm", "composer.json"),
		"jetbrains":        jetbrainsProjectDetector(),
		"intellij":         ideDetector("intellij"),
		"pycharm":          ideWithJetBrainsSignalDetector("pycharm", signalDetector{reason: "python project file", match: anyFileSignal("pyproject.toml", "requirements.txt", "setup.py")}),
		"webstorm":         ideWithJetBrainsLanguageInferenceDetector("webstorm", "package.json"),
		"goland":           ideWithJetBrainsLanguageInferenceDetector("goland", "go.mod"),
		"rubymine":         ideWithJetBrainsLanguageInferenceDetector("rubymine", "Gemfile"),
		"rider":            ideWithJetBrainsSignalDetector("rider", signalDetector{reason: ".sln/.csproj", match: anyGlobSignal(".sln/.csproj", "*.sln", "*.csproj")}),
		"clion":            ideWithJetBrainsLanguageInferenceDetector("clion", "CMakeLists.txt"),
		"appcode":          ideDetector("appcode"),
		"androidstudio":    ideDetector("androidstudio"),
	}
}

func ideDetector(key string) Detector {
	return appDetector(key, ideInstallCandidatesForKey(key))
}

func jetbrainsInstallDetector() Detector {
	return appDetector("jetbrains", ideInstallCandidatesForKey("jetbrains"))
}

func ideWithJetBrainsLanguageInferenceDetector(key, signalFile string) Detector {
	return ideWithJetBrainsSignalDetector(key, signalDetector{reason: signalFile, match: fileSignal(signalFile)})
}

func ideWithJetBrainsSignalDetector(key string, signal signalDetector) Detector {
	base := ideDetector(key)
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		result := base.Detect(ctx, cwd)
		if result.Matched || result.Reason != "application not found" {
			return result
		}

		matchedSignal, ok := signal.match(ctx, cwd)
		if !ok {
			return result
		}

		jetbrains := jetbrainsInstallDetector().Detect(ctx, cwd)
		if !jetbrains.Matched {
			return result
		}

		reason := signal.reason
		if matchedSignal != "" {
			reason = matchedSignal
		}

		return Result{Key: key, Matched: true, Reason: "inferred from jetbrains install and " + reason, Evidence: jetbrains.Evidence}
	})
}

type signalDetector struct {
	reason string
	match  func(ctx context.Context, cwd string) (string, bool)
}

func fileSignal(fileName string) func(ctx context.Context, cwd string) (string, bool) {
	return func(ctx context.Context, cwd string) (string, bool) {
		for _, dir := range searchDirsFor(ctx, cwd) {
			path := filepath.Join(dir, fileName)
			if _, err := os.Stat(path); err == nil {
				return fileName, true
			}
		}
		return "", false
	}
}

func anyFileSignal(fileNames ...string) func(ctx context.Context, cwd string) (string, bool) {
	return func(ctx context.Context, cwd string) (string, bool) {
		for _, fileName := range fileNames {
			if matched, ok := fileSignal(fileName)(ctx, cwd); ok {
				return matched, true
			}
		}
		return "", false
	}
}

func anyGlobSignal(reason string, patterns ...string) func(ctx context.Context, cwd string) (string, bool) {
	return func(ctx context.Context, cwd string) (string, bool) {
		for _, dir := range searchDirsFor(ctx, cwd) {
			for _, pattern := range patterns {
				matches, err := filepath.Glob(filepath.Join(dir, pattern))
				if err != nil {
					continue
				}
				if len(matches) > 0 {
					return reason, true
				}
			}
		}
		return "", false
	}
}

func anySignalMatch(signals ...func(ctx context.Context, cwd string) (string, bool)) func(ctx context.Context, cwd string) (string, bool) {
	return func(ctx context.Context, cwd string) (string, bool) {
		for _, signal := range signals {
			if matched, ok := signal(ctx, cwd); ok {
				return matched, true
			}
		}
		return "", false
	}
}

func anySignalDetector(key string, signal signalDetector) Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		if matched, ok := signal.match(ctx, cwd); ok {
			reason := signal.reason
			if matched != "" {
				reason = matched
			}
			return Result{Key: key, Matched: true, Reason: reason, Evidence: cwd}
		}
		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func anyGlobDetector(key, reason string, patterns ...string) Detector {
	return anySignalDetector(key, signalDetector{reason: reason, match: anyGlobSignal(reason, patterns...)})
}

func ideInstallCandidatesForKey(key string) []string {
	candidates, ok := ideInstallCandidatesByKey[key]
	if !ok {
		return nil
	}
	copyOfCandidates := make([]string, len(candidates))
	copy(copyOfCandidates, candidates)
	return copyOfCandidates
}

func vscodeProjectDetector() Detector {
	return workspaceMetadataDetector("visualstudiocode", []string{".vscode"}, []string{"*.code-workspace"}, "found VS Code workspace metadata")
}

func jetbrainsProjectDetector() Detector {
	return workspaceMetadataDetector("jetbrains", []string{".idea"}, []string{"*.iml"}, "found JetBrains project metadata")
}

func workspaceMetadataDetector(key string, dirNames []string, patterns []string, reason string) Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		for _, dir := range searchDirsFor(ctx, cwd) {
			for _, name := range dirNames {
				path := filepath.Join(dir, name)
				info, err := os.Stat(path)
				if err == nil {
					if info.IsDir() {
						return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
					}
					continue
				}
				if os.IsPermission(err) {
					return Result{Key: key, Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}
				}
			}

			for _, pattern := range patterns {
				matches, err := filepath.Glob(filepath.Join(dir, pattern))
				if err != nil {
					continue
				}
				if len(matches) > 0 {
					return Result{Key: key, Matched: true, Reason: reason, Evidence: matches[0]}
				}
			}
		}

		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func vueDetector() Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		for _, dir := range searchDirsFor(ctx, cwd) {
			for _, file := range []string{"vue.config.js", "vue.config.ts"} {
				path := filepath.Join(dir, file)
				if _, err := os.Stat(path); err == nil {
					return Result{Key: "vue", Matched: true, Reason: "found vue config", Evidence: path}
				} else if os.IsPermission(err) {
					return Result{Key: "vue", Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}
				}
			}
		}

		content, packagePath, result, ok := readSignalFile(ctx, "vue", cwd, "package.json")
		if !ok {
			return result
		}

		var packageJSON struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(content, &packageJSON); err != nil {
			return Result{Key: "vue", Matched: false, Reason: "invalid package.json", Evidence: packagePath, Error: err.Error()}
		}
		if _, ok := packageJSON.Dependencies["vue"]; ok {
			return Result{Key: "vue", Matched: true, Reason: "package.json dependency includes vue", Evidence: packagePath}
		}
		if _, ok := packageJSON.DevDependencies["vue"]; ok {
			return Result{Key: "vue", Matched: true, Reason: "package.json devDependency includes vue", Evidence: packagePath}
		}
		return Result{Key: "vue", Matched: false, Reason: "signal not found"}
	})
}

func fileExistsDetector(key, fileName, reason string) Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		for _, dir := range searchDirsFor(ctx, cwd) {
			path := filepath.Join(dir, fileName)
			if _, err := os.Stat(path); err == nil {
				return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
			} else if os.IsPermission(err) {
				return Result{Key: key, Matched: false, Reason: "permission denied", Error: err.Error()}
			}
		}
		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func anyFileDetector(key string, files []string, reason string) Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		for _, dir := range searchDirsFor(ctx, cwd) {
			for _, file := range files {
				path := filepath.Join(dir, file)
				if _, err := os.Stat(path); err == nil {
					return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
				} else if os.IsPermission(err) {
					return Result{Key: key, Matched: false, Reason: "permission denied", Error: err.Error()}
				}
			}
		}
		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func nodeDetector() Detector {
	return anyFileDetector("node", []string{"package.json", "bun.lock", "bun.lockb"}, "found node project file")
}

func laravelDetector() Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		for _, dir := range searchDirsFor(ctx, cwd) {
			artisan := filepath.Join(dir, "artisan")
			if _, err := os.Stat(artisan); err == nil {
				return Result{Key: "laravel", Matched: true, Reason: "found artisan file", Evidence: artisan}
			}
		}
		content, composer, result, ok := readSignalFile(ctx, "laravel", cwd, "composer.json")
		if !ok {
			return result
		}
		if strings.Contains(string(content), "laravel/framework") {
			return Result{Key: "laravel", Matched: true, Reason: "composer.json references laravel/framework", Evidence: composer}
		}
		return Result{Key: "laravel", Matched: false, Reason: "signal not found"}
	})
}

func reactDetector() Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		content, packagePath, result, ok := readSignalFile(ctx, "react", cwd, "package.json")
		if !ok {
			return result
		}

		var packageJSON struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(content, &packageJSON); err != nil {
			return Result{Key: "react", Matched: false, Reason: "invalid package.json", Evidence: packagePath, Error: err.Error()}
		}
		if _, ok := packageJSON.Dependencies["react"]; ok {
			return Result{Key: "react", Matched: true, Reason: "package.json dependency includes react", Evidence: packagePath}
		}
		if _, ok := packageJSON.DevDependencies["react"]; ok {
			return Result{Key: "react", Matched: true, Reason: "package.json devDependency includes react", Evidence: packagePath}
		}
		return Result{Key: "react", Matched: false, Reason: "signal not found"}
	})
}

func flutterDetector() Detector {
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		content, pubspecPath, result, ok := readSignalFile(ctx, "flutter", cwd, "pubspec.yaml")
		if !ok {
			return result
		}
		if strings.Contains(string(content), "flutter:") || strings.Contains(string(content), "sdk: flutter") {
			return Result{Key: "flutter", Matched: true, Reason: "pubspec.yaml references flutter", Evidence: pubspecPath}
		}
		return Result{Key: "flutter", Matched: false, Reason: "signal not found"}
	})
}

func osDetector(key, expected string) Detector {
	return DetectorFunc(func(_ context.Context, _ string) Result {
		if runtime.GOOS == expected {
			return Result{Key: key, Matched: true, Reason: "matched runtime OS", Evidence: runtime.GOOS}
		}
		return Result{Key: key, Matched: false, Reason: "runtime OS mismatch", Evidence: runtime.GOOS}
	})
}

func appDetector(key string, paths []string) Detector {
	return DetectorFunc(func(_ context.Context, _ string) Result {
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return Result{Key: key, Matched: true, Reason: "detected installed application", Evidence: path}
			} else if os.IsPermission(err) {
				return Result{Key: key, Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}
			} else if !os.IsNotExist(err) {
				return Result{Key: key, Matched: false, Reason: "failed to inspect application path", Evidence: path, Error: err.Error()}
			}
		}
		return Result{Key: key, Matched: false, Reason: "application not found"}
	})
}

func pathDetector(key string, binaries []string, reason string) Detector {
	return DetectorFunc(func(_ context.Context, _ string) Result {
		for _, binary := range binaries {
			if path, err := exec.LookPath(binary); err == nil {
				return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
			}
		}
		return Result{Key: key, Matched: false, Reason: "binary not found in PATH"}
	})
}

func readSignalFile(ctx context.Context, key, cwd, fileName string) ([]byte, string, Result, bool) {
	for _, dir := range searchDirsFor(ctx, cwd) {
		path := filepath.Join(dir, fileName)
		content, err := os.ReadFile(path)
		if err == nil {
			return content, path, Result{}, true
		}
		if os.IsNotExist(err) {
			continue
		}
		if os.IsPermission(err) {
			return nil, "", Result{Key: key, Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}, false
		}
		return nil, "", Result{Key: key, Matched: false, Reason: "failed to read signal file", Evidence: path, Error: err.Error()}, false
	}

	return nil, "", Result{Key: key, Matched: false, Reason: "signal not found"}, false
}

func ContextWithInputs(ctx context.Context, cwd string) context.Context {
	return context.WithValue(ctx, detectorInputsContextKey{}, detectorInputs{cwd: cwd, searchDirs: oneLevelSearchDirs(cwd)})
}

func searchDirsFor(ctx context.Context, cwd string) []string {
	inputs, ok := ctx.Value(detectorInputsContextKey{}).(detectorInputs)
	if ok && inputs.cwd == cwd {
		return inputs.searchDirs
	}

	return oneLevelSearchDirs(cwd)
}

func oneLevelSearchDirs(cwd string) []string {
	dirs := []string{cwd}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return dirs
	}

	matcher := loadGitignoreMatcher(cwd)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if matcher != nil && matcher.Match([]string{entry.Name()}, true) {
			continue
		}
		dirs = append(dirs, filepath.Join(cwd, entry.Name()))
	}

	return dirs
}

func loadGitignoreMatcher(cwd string) gitignore.Matcher {
	patterns, err := gitignore.ReadPatterns(osfs.New(cwd), nil)
	if err != nil || len(patterns) == 0 {
		return nil
	}

	return gitignore.NewMatcher(patterns)
}
