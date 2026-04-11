package provider

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
}

type DetectorFunc func(ctx context.Context, cwd string) Result

func (f DetectorFunc) Detect(ctx context.Context, cwd string) Result {
	return f(ctx, cwd)
}

func Registry() map[string]Detector {
	return map[string]Detector{
		"composer":         fileExistsDetector("composer", "composer.json", "found composer.json"),
		"node":             fileExistsDetector("node", "package.json", "found package.json"),
		"go":               fileExistsDetector("go", "go.mod", "found go.mod"),
		"laravel":          laravelDetector(),
		"nextjs":           anyFileDetector("nextjs", []string{"next.config.js", "next.config.mjs", "next.config.ts"}, "found next config"),
		"nuxtjs":           anyFileDetector("nuxtjs", []string{"nuxt.config.js", "nuxt.config.mjs", "nuxt.config.ts"}, "found nuxt config"),
		"python":           anyFileDetector("python", []string{"pyproject.toml", "requirements.txt", "setup.py"}, "found python project file"),
		"vue":              vueDetector(),
		"react":            reactDetector(),
		"macos":            osDetector("macos", "darwin"),
		"linux":            osDetector("linux", "linux"),
		"windows":          osDetector("windows", "windows"),
		"visualstudiocode": pathDetector("visualstudiocode", []string{"code"}, "found VS Code binary in PATH"),
		"phpstorm":         ideWithJetBrainsLanguageInferenceDetector("phpstorm", "composer.json"),
		"jetbrains":        ideDetector("jetbrains"),
		"intellij":         ideDetector("intellij"),
		"pycharm":          ideDetector("pycharm"),
		"webstorm":         ideDetector("webstorm"),
		"goland":           ideWithJetBrainsLanguageInferenceDetector("goland", "go.mod"),
		"rubymine":         ideDetector("rubymine"),
		"rider":            ideDetector("rider"),
		"clion":            ideDetector("clion"),
		"appcode":          ideDetector("appcode"),
	}
}

func ideDetector(key string) Detector {
	return appDetector(key, ideInstallCandidatesForKey(key))
}

func ideWithJetBrainsLanguageInferenceDetector(key, signalFile string) Detector {
	base := ideDetector(key)
	return DetectorFunc(func(ctx context.Context, cwd string) Result {
		result := base.Detect(ctx, cwd)
		if result.Matched || result.Reason != "application not found" {
			return result
		}

		signalPath := filepath.Join(cwd, signalFile)
		if _, err := os.Stat(signalPath); err != nil {
			return result
		}

		jetbrains := ideDetector("jetbrains").Detect(ctx, cwd)
		if !jetbrains.Matched {
			return result
		}

		return Result{Key: key, Matched: true, Reason: "inferred from jetbrains install and " + signalFile, Evidence: jetbrains.Evidence}
	})
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

func vueDetector() Detector {
	return DetectorFunc(func(_ context.Context, cwd string) Result {
		for _, file := range []string{"vue.config.js", "vue.config.ts"} {
			path := filepath.Join(cwd, file)
			if _, err := os.Stat(path); err == nil {
				return Result{Key: "vue", Matched: true, Reason: "found vue config", Evidence: path}
			} else if os.IsPermission(err) {
				return Result{Key: "vue", Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}
			}
		}

		packagePath := filepath.Join(cwd, "package.json")
		content, result, ok := readSignalFile("vue", packagePath)
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
	return DetectorFunc(func(_ context.Context, cwd string) Result {
		path := filepath.Join(cwd, fileName)
		if _, err := os.Stat(path); err == nil {
			return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
		} else if os.IsPermission(err) {
			return Result{Key: key, Matched: false, Reason: "permission denied", Error: err.Error()}
		}
		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func anyFileDetector(key string, files []string, reason string) Detector {
	return DetectorFunc(func(_ context.Context, cwd string) Result {
		for _, file := range files {
			path := filepath.Join(cwd, file)
			if _, err := os.Stat(path); err == nil {
				return Result{Key: key, Matched: true, Reason: reason, Evidence: path}
			} else if os.IsPermission(err) {
				return Result{Key: key, Matched: false, Reason: "permission denied", Error: err.Error()}
			}
		}
		return Result{Key: key, Matched: false, Reason: "signal not found"}
	})
}

func laravelDetector() Detector {
	return DetectorFunc(func(_ context.Context, cwd string) Result {
		artisan := filepath.Join(cwd, "artisan")
		if _, err := os.Stat(artisan); err == nil {
			return Result{Key: "laravel", Matched: true, Reason: "found artisan file", Evidence: artisan}
		}
		composer := filepath.Join(cwd, "composer.json")
		content, result, ok := readSignalFile("laravel", composer)
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
	return DetectorFunc(func(_ context.Context, cwd string) Result {
		packagePath := filepath.Join(cwd, "package.json")
		content, result, ok := readSignalFile("react", packagePath)
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

func readSignalFile(key string, path string) ([]byte, Result, bool) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, Result{}, true
	}
	if os.IsNotExist(err) {
		return nil, Result{Key: key, Matched: false, Reason: "signal not found"}, false
	}
	if os.IsPermission(err) {
		return nil, Result{Key: key, Matched: false, Reason: "permission denied", Evidence: path, Error: err.Error()}, false
	}
	return nil, Result{Key: key, Matched: false, Reason: "failed to read signal file", Evidence: path, Error: err.Error()}, false
}
