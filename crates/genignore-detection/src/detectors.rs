//! Provider detector registry: the hardcoded detectors plus the rule-catalog
//! backed detectors, ported 1:1 from `internal/provider/detectors.go`.

use std::collections::BTreeMap;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use crate::catalog::{load_rule_catalog, Entry, RuleType};
use crate::gitignore_match::load_root_matcher;
use crate::signalfile::{
    glob_in_root, is_not_found, is_permission, is_safe_skip, read_signal_file,
};
use crate::DetectionResult;

/// Detector inputs computed once per scan target: the cwd plus its one-level
/// (non-gitignored) subdirectory list.
pub struct ScanCtx {
    pub cwd: PathBuf,
    pub search_dirs: Vec<PathBuf>,
}

impl ScanCtx {
    pub fn new(cwd: &Path) -> ScanCtx {
        ScanCtx {
            cwd: cwd.to_path_buf(),
            search_dirs: one_level_search_dirs(cwd),
        }
    }
}

pub type Detector = Box<dyn Fn(&ScanCtx) -> DetectionResult>;

fn result(key: &str, matched: bool, reason: &str, evidence: &str, error: &str) -> DetectionResult {
    DetectionResult {
        key: key.to_string(),
        matched,
        reason: reason.to_string(),
        evidence: evidence.to_string(),
        error: error.to_string(),
    }
}

/// One-level search scope: the cwd plus its immediate subdirectories that the
/// root gitignore matcher does not ignore.
pub fn one_level_search_dirs(cwd: &Path) -> Vec<PathBuf> {
    let mut dirs = vec![cwd.to_path_buf()];
    let entries = match fs::read_dir(cwd) {
        Ok(e) => e,
        Err(_) => return dirs,
    };

    let matcher = load_root_matcher(cwd);

    let mut names: Vec<String> = Vec::new();
    for entry in entries.flatten() {
        let is_dir = entry.file_type().map(|t| t.is_dir()).unwrap_or(false);
        if !is_dir {
            continue;
        }
        names.push(entry.file_name().to_string_lossy().to_string());
    }
    names.sort();

    for name in names {
        if let Some(m) = &matcher {
            if m.is_ignored(std::slice::from_ref(&name), true) {
                continue;
            }
        }
        dirs.push(cwd.join(&name));
    }
    dirs
}

fn search_dirs(ctx: &ScanCtx) -> &[PathBuf] {
    &ctx.search_dirs
}

// ---- signal helpers ----

type SignalMatch = Box<dyn Fn(&ScanCtx) -> Option<String>>;

fn file_signal(file_name: &'static str) -> SignalMatch {
    Box::new(move |ctx| {
        for dir in search_dirs(ctx) {
            if fs::metadata(dir.join(file_name)).is_ok() {
                return Some(file_name.to_string());
            }
        }
        None
    })
}

fn any_file_signal(file_names: &'static [&'static str]) -> SignalMatch {
    Box::new(move |ctx| {
        for file_name in file_names {
            for dir in search_dirs(ctx) {
                if fs::metadata(dir.join(file_name)).is_ok() {
                    return Some((*file_name).to_string());
                }
            }
        }
        None
    })
}

/// Port of Go's `filepath.Glob` (unconfined OS-level glob): non-meta patterns
/// stat via lstat, meta patterns read directories recursively. Returns the
/// matched paths (Go returns them in sorted order).
fn filepath_glob(dir: &Path, pattern: &str) -> Vec<String> {
    glob_os(&dir.join(pattern).to_string_lossy())
}

fn glob_os(pattern: &str) -> Vec<String> {
    if crate::globmatch::path_match(pattern, "").is_err() {
        return Vec::new();
    }
    if !pattern.bytes().any(|b| matches!(b, b'*' | b'?' | b'[')) {
        return match fs::symlink_metadata(pattern) {
            Ok(_) => vec![pattern.to_string()],
            Err(_) => Vec::new(),
        };
    }
    let (dir_part, file_part) = match pattern.rfind('/') {
        Some(idx) => (&pattern[..idx], &pattern[idx + 1..]),
        None => (".", pattern),
    };
    let dir_part = if dir_part.is_empty() { "/" } else { dir_part };
    if dir_part == pattern {
        return Vec::new();
    }
    let dirs = if dir_part.bytes().any(|b| matches!(b, b'*' | b'?' | b'[')) {
        glob_os(dir_part)
    } else {
        vec![dir_part.to_string()]
    };
    let mut matches = Vec::new();
    for d in dirs {
        let entries = match fs::read_dir(&d) {
            Ok(e) => e,
            Err(_) => continue,
        };
        let mut names: Vec<String> = entries
            .flatten()
            .map(|e| e.file_name().to_string_lossy().to_string())
            .collect();
        names.sort();
        for name in names {
            if crate::globmatch::path_match(file_part, &name).unwrap_or(false) {
                let joined = if d == "." {
                    name.clone()
                } else {
                    format!("{}/{}", d.trim_end_matches('/'), name)
                };
                matches.push(joined);
            }
        }
    }
    matches
}

fn any_glob_signal(reason: &'static str, patterns: &'static [&'static str]) -> SignalMatch {
    Box::new(move |ctx| {
        for dir in search_dirs(ctx) {
            for pattern in patterns {
                if !filepath_glob(dir, pattern).is_empty() {
                    return Some(reason.to_string());
                }
            }
        }
        None
    })
}

fn any_signal_match(signals: Vec<SignalMatch>) -> SignalMatch {
    Box::new(move |ctx| {
        for signal in &signals {
            if let Some(m) = signal(ctx) {
                return Some(m);
            }
        }
        None
    })
}

fn any_signal_detector(key: &'static str, reason: &'static str, signal: SignalMatch) -> Detector {
    Box::new(move |ctx| {
        if let Some(matched) = signal(ctx) {
            let reason = if matched.is_empty() { reason } else { &matched };
            return result(key, true, reason, &ctx.cwd.to_string_lossy(), "");
        }
        result(key, false, "signal not found", "", "")
    })
}

fn any_glob_detector(
    key: &'static str,
    reason: &'static str,
    patterns: &'static [&'static str],
) -> Detector {
    any_signal_detector(key, reason, any_glob_signal(reason, patterns))
}

fn any_file_detector(
    key: &'static str,
    files: &'static [&'static str],
    reason: &'static str,
) -> Detector {
    Box::new(move |ctx| {
        for dir in search_dirs(ctx) {
            for file in files {
                let path = dir.join(file);
                match fs::metadata(&path) {
                    Ok(_) => return result(key, true, reason, &path.to_string_lossy(), ""),
                    Err(e) if e.kind() == io::ErrorKind::PermissionDenied => {
                        return result(key, false, "permission denied", "", &e.to_string())
                    }
                    Err(_) => {}
                }
            }
        }
        result(key, false, "signal not found", "", "")
    })
}

// ---- readSignalFile ----

fn read_signal_any(
    ctx: &ScanCtx,
    key: &str,
    file_name: &str,
) -> Result<Option<(Vec<u8>, PathBuf)>, DetectionResult> {
    for dir in search_dirs(ctx) {
        let path = dir.join(file_name);
        match read_signal_file(dir, file_name) {
            Ok(content) => return Ok(Some((content, path))),
            Err(err) => {
                if is_not_found(&err) || is_safe_skip(&err) {
                    continue;
                }
                if is_permission(&err) {
                    return Err(result(
                        key,
                        false,
                        "permission denied",
                        &path.to_string_lossy(),
                        &err.to_string(),
                    ));
                }
                return Err(result(
                    key,
                    false,
                    "failed to read signal file",
                    &path.to_string_lossy(),
                    &err.to_string(),
                ));
            }
        }
    }
    Ok(None)
}

// ---- package.json signal detectors ----

fn package_json_detector(key: &'static str, dep_key: &'static str) -> Detector {
    Box::new(move |ctx| {
        let (content, package_path) = match read_signal_any(ctx, key, "package.json") {
            Ok(Some(v)) => v,
            Ok(None) => return result(key, false, "signal not found", "", ""),
            Err(err) => return err,
        };
        let text = match String::from_utf8(content) {
            Ok(t) => t,
            Err(err) => {
                return result(
                    key,
                    false,
                    "invalid package.json",
                    &package_path.to_string_lossy(),
                    &err.to_string(),
                )
            }
        };
        let json: serde_json::Value = match serde_json::from_str(&text) {
            Ok(v) => v,
            Err(err) => {
                return result(
                    key,
                    false,
                    "invalid package.json",
                    &package_path.to_string_lossy(),
                    &err.to_string(),
                )
            }
        };
        let has_dep = |section: &str| json.get(section).and_then(|d| d.get(dep_key)).is_some();
        if has_dep("dependencies") {
            return result(
                key,
                true,
                &format!("package.json dependency includes {}", dep_key),
                &package_path.to_string_lossy(),
                "",
            );
        }
        if has_dep("devDependencies") {
            return result(
                key,
                true,
                &format!("package.json devDependency includes {}", dep_key),
                &package_path.to_string_lossy(),
                "",
            );
        }
        result(key, false, "signal not found", "", "")
    })
}

fn vue_detector() -> Detector {
    Box::new(|ctx| {
        for dir in search_dirs(ctx) {
            for file in ["vue.config.js", "vue.config.ts"] {
                let path = dir.join(file);
                match fs::metadata(&path) {
                    Ok(_) => {
                        return result("vue", true, "found vue config", &path.to_string_lossy(), "")
                    }
                    Err(e) if e.kind() == io::ErrorKind::PermissionDenied => {
                        return result(
                            "vue",
                            false,
                            "permission denied",
                            &path.to_string_lossy(),
                            &e.to_string(),
                        )
                    }
                    Err(_) => {}
                }
            }
        }
        package_json_detector("vue", "vue")(ctx)
    })
}

fn flutter_detector() -> Detector {
    Box::new(|ctx| {
        let (content, pubspec_path) = match read_signal_any(ctx, "flutter", "pubspec.yaml") {
            Ok(Some(v)) => v,
            Ok(None) => return result("flutter", false, "signal not found", "", ""),
            Err(err) => return err,
        };
        let text = String::from_utf8_lossy(&content);
        if text.contains("flutter:") || text.contains("sdk: flutter") {
            return result(
                "flutter",
                true,
                "pubspec.yaml references flutter",
                &pubspec_path.to_string_lossy(),
                "",
            );
        }
        result("flutter", false, "signal not found", "", "")
    })
}

// ---- host detectors ----

fn goos() -> &'static str {
    match std::env::consts::OS {
        "macos" => "darwin",
        other => other,
    }
}

fn os_detector(key: &'static str, expected: &'static str) -> Detector {
    Box::new(move |_| {
        if goos() == expected {
            result(key, true, "matched runtime OS", goos(), "")
        } else {
            result(key, false, "runtime OS mismatch", goos(), "")
        }
    })
}

fn app_detector(key: &'static str, paths: &'static [&'static str]) -> Detector {
    Box::new(move |_| {
        for path in paths {
            match fs::metadata(path) {
                Ok(_) => return result(key, true, "detected installed application", path, ""),
                Err(e) if e.kind() == io::ErrorKind::PermissionDenied => {
                    return result(key, false, "permission denied", path, &e.to_string())
                }
                Err(e) if e.kind() != io::ErrorKind::NotFound => {
                    return result(
                        key,
                        false,
                        "failed to inspect application path",
                        path,
                        &e.to_string(),
                    )
                }
                Err(_) => {}
            }
        }
        result(key, false, "application not found", "", "")
    })
}

fn ide_detector(key: &'static str) -> Detector {
    app_detector(key, ide_install_candidates(key))
}

fn jetbrains_install_detector() -> Detector {
    app_detector("jetbrains", ide_install_candidates("jetbrains"))
}

fn ide_with_jetbrains_language_inference(key: &'static str, signal_file: &'static str) -> Detector {
    ide_with_jetbrains_signal(key, "unused", file_signal(signal_file))
}

fn ide_with_jetbrains_signal(
    key: &'static str,
    reason: &'static str,
    signal: SignalMatch,
) -> Detector {
    let base = ide_detector(key);
    Box::new(move |ctx| {
        let res = base(ctx);
        if res.matched || res.reason != "application not found" {
            return res;
        }
        let Some(matched_signal) = signal(ctx) else {
            return res;
        };
        let jetbrains = jetbrains_install_detector()(ctx);
        if !jetbrains.matched {
            return res;
        }
        let reason_str = if matched_signal.is_empty() {
            reason.to_string()
        } else {
            matched_signal
        };
        result(
            key,
            true,
            &format!("inferred from jetbrains install and {}", reason_str),
            &jetbrains.evidence,
            "",
        )
    })
}

// ---- rule-catalog backed detectors ----

fn match_rule(root_dir: &Path, rule: &crate::catalog::Rule) -> Result<RuleMatch, io::Error> {
    match rule.rule_type {
        RuleType::FilePath => {
            let matches = glob_in_root(root_dir, &rule.path)?;
            if matches.is_empty() {
                return Ok(RuleMatch::none());
            }
            Ok(RuleMatch {
                matched: true,
                path: matches[0].clone(),
                line: 0,
            })
        }
        RuleType::FileContentLine => match read_signal_file(root_dir, &rule.path) {
            Ok(content) => {
                let line = contains_line(&content, &rule.contains);
                match line {
                    Some(line) => Ok(RuleMatch {
                        matched: true,
                        path: rule.path.clone(),
                        line,
                    }),
                    None => Ok(RuleMatch::none()),
                }
            }
            Err(err) => {
                if is_not_found(&err) || is_safe_skip(&err) {
                    return Ok(RuleMatch::none());
                }
                Err(io::Error::other(err.to_string()))
            }
        },
    }
}

pub struct RuleMatch {
    pub matched: bool,
    pub path: String,
    pub line: usize,
}

impl RuleMatch {
    fn none() -> Self {
        RuleMatch {
            matched: false,
            path: String::new(),
            line: 0,
        }
    }
}

fn contains_line(content: &[u8], contains: &str) -> Option<usize> {
    for (idx, line) in content.split(|b| *b == b'\n').enumerate() {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if String::from_utf8_lossy(line).contains(contains) {
            return Some(idx + 1);
        }
    }
    None
}

fn rule_match_reason(provider: &str, rule_type: RuleType, path: &str, contains: &str) -> String {
    match provider {
        "composer" => return "found composer.json".to_string(),
        "android" => return "found android manifest".to_string(),
        "dart" => return "found pubspec.yaml".to_string(),
        "go" => return "found go.mod".to_string(),
        "java" => return "found java project file".to_string(),
        "jekyll" => return "found _config.yml".to_string(),
        "jetbrains" => return "found JetBrains project metadata".to_string(),
        "laravel" => {
            return if rule_type == RuleType::FileContentLine {
                "composer.json references laravel/framework".to_string()
            } else {
                "found artisan file".to_string()
            }
        }
        "maven" => return "found pom.xml".to_string(),
        "nextjs" => return "found next config".to_string(),
        "node" => return "found node project file".to_string(),
        "nuxtjs" => return "found nuxt config".to_string(),
        "python" => return "found python project file".to_string(),
        "rails" => return "found rails project file".to_string(),
        "ruby" => return "found Gemfile".to_string(),
        "rust" => return "found Cargo.toml".to_string(),
        "symfony" => return "found symfony project file".to_string(),
        "swift" => {
            return if path == "Package.swift" {
                "found swift project file".to_string()
            } else {
                "found swift source file".to_string()
            }
        }
        "terraform" => return "found terraform file".to_string(),
        "visualstudiocode" => return "found VS Code workspace metadata".to_string(),
        "xcode" => return "found xcode project file".to_string(),
        _ => {}
    }
    match rule_type {
        RuleType::FilePath => "matched file path rule".to_string(),
        RuleType::FileContentLine => {
            format!("matched content line rule containing {:?}", contains)
        }
    }
}

fn rule_entry_detector(entry: &Entry) -> Detector {
    let provider = entry.provider.clone();
    let rules: Vec<crate::catalog::Rule> = entry.matches.clone();
    Box::new(move |ctx| {
        for dir in search_dirs(ctx) {
            if fs::metadata(dir).is_err() {
                match fs::metadata(dir).err() {
                    Some(e) if e.kind() == io::ErrorKind::PermissionDenied => {
                        return result(
                            &provider,
                            false,
                            "permission denied",
                            &dir.to_string_lossy(),
                            &e.to_string(),
                        )
                    }
                    Some(e) => {
                        return result(
                            &provider,
                            false,
                            "failed to inspect detection root",
                            &dir.to_string_lossy(),
                            &e.to_string(),
                        )
                    }
                    None => {}
                }
            }
            for rule in &rules {
                match match_rule(dir, rule) {
                    Err(err) => {
                        return result(
                            &provider,
                            false,
                            "failed to evaluate detection rule",
                            &dir.join(&rule.path).to_string_lossy(),
                            &err.to_string(),
                        )
                    }
                    Ok(m) if m.matched => {
                        let mut evidence = dir.join(&m.path).to_string_lossy().to_string();
                        if m.line > 0 {
                            evidence = format!("{}:{}", evidence, m.line);
                        }
                        return result(
                            &provider,
                            true,
                            &rule_match_reason(&provider, rule.rule_type, &m.path, &rule.contains),
                            &evidence,
                            "",
                        );
                    }
                    _ => {}
                }
            }
        }
        result(&provider, false, "signal not found", "", "")
    })
}

// ---- IDE install candidates ----

fn ide_install_candidates(key: &str) -> &'static [&'static str] {
    match key {
        "phpstorm" => &[
            "/Applications/PhpStorm.app",
            "/opt/phpstorm",
            "/opt/PhpStorm",
        ],
        "jetbrains" => &[
            "/Applications/IntelliJ IDEA.app",
            "/Applications/PhpStorm.app",
            "/Applications/PyCharm.app",
            "/Applications/WebStorm.app",
            "/opt/jetbrains",
            "/opt/JetBrains",
            "/var/lib/flatpak/app/com.jetbrains.IntelliJ-IDEA-Community",
            "/var/lib/flatpak/app/com.jetbrains.IntelliJ-IDEA-Ultimate",
            "/var/lib/flatpak/app/com.jetbrains.PhpStorm",
        ],
        "intellij" => &[
            "/Applications/IntelliJ IDEA.app",
            "/opt/intellij-idea",
            "/opt/idea",
        ],
        "pycharm" => &["/Applications/PyCharm.app", "/opt/pycharm", "/opt/PyCharm"],
        "webstorm" => &[
            "/Applications/WebStorm.app",
            "/opt/webstorm",
            "/opt/WebStorm",
        ],
        "goland" => &["/Applications/GoLand.app", "/opt/goland", "/opt/GoLand"],
        "rubymine" => &[
            "/Applications/RubyMine.app",
            "/opt/rubymine",
            "/opt/RubyMine",
        ],
        "rider" => &["/Applications/Rider.app", "/opt/rider", "/opt/Rider"],
        "clion" => &["/Applications/CLion.app", "/opt/clion", "/opt/CLion"],
        "appcode" => &["/Applications/AppCode.app", "/opt/appcode", "/opt/AppCode"],
        "androidstudio" => &[
            "/Applications/Android Studio.app",
            "/opt/android-studio",
            "/opt/AndroidStudio",
        ],
        _ => &[],
    }
}

/// Build the detector registry: hardcoded detectors, then rule-catalog entries
/// override any colliding keys (matching Go). `supported` is the supported
/// provider-key set used to validate the embedded catalog.
pub fn registry<F>(supported: F) -> Result<BTreeMap<String, Detector>, String>
where
    F: Fn(&str) -> bool,
{
    let entries = load_rule_catalog(&supported)?;

    let mut registry: BTreeMap<String, Detector> = BTreeMap::new();

    let mut insert = |key: &'static str, d: Detector| {
        registry.insert(key.to_string(), d);
    };

    insert(
        "terraform",
        any_glob_detector(
            "terraform",
            "found terraform file",
            &["*.tf", "*.tfvars", ".terraform.lock.hcl"],
        ),
    );
    insert(
        "kotlin",
        any_signal_detector(
            "kotlin",
            "found kotlin project file",
            any_signal_match(vec![
                any_file_signal(&["build.gradle.kts", "settings.gradle.kts"]),
                any_glob_signal("found kotlin source file", &["*.kt"]),
            ]),
        ),
    );
    insert(
        "dotnetcore",
        any_glob_detector(
            "dotnetcore",
            "found dotnet project file",
            &["*.sln", "*.csproj"],
        ),
    );
    insert(
        "csharp",
        any_signal_detector(
            "csharp",
            "found csharp project file",
            any_signal_match(vec![
                any_glob_signal("found csharp solution/project file", &["*.sln", "*.csproj"]),
                any_glob_signal("found csharp source file", &["*.cs"]),
            ]),
        ),
    );
    insert("flutter", flutter_detector());
    insert(
        "xcode",
        any_glob_detector(
            "xcode",
            "found xcode project file",
            &["*.xcodeproj", "*.xcworkspace"],
        ),
    );
    insert(
        "nuxtjs",
        any_file_detector(
            "nuxtjs",
            &["nuxt.config.js", "nuxt.config.mjs", "nuxt.config.ts"],
            "found nuxt config",
        ),
    );
    insert("vue", vue_detector());
    insert("react", package_json_detector("react", "react"));
    insert("macos", os_detector("macos", "darwin"));
    insert("linux", os_detector("linux", "linux"));
    insert("windows", os_detector("windows", "windows"));
    insert(
        "phpstorm",
        ide_with_jetbrains_language_inference("phpstorm", "composer.json"),
    );
    insert("intellij", ide_detector("intellij"));
    insert(
        "pycharm",
        ide_with_jetbrains_signal(
            "pycharm",
            "python project file",
            any_file_signal(&["pyproject.toml", "requirements.txt", "setup.py"]),
        ),
    );
    insert(
        "webstorm",
        ide_with_jetbrains_language_inference("webstorm", "package.json"),
    );
    insert(
        "goland",
        ide_with_jetbrains_language_inference("goland", "go.mod"),
    );
    insert(
        "rubymine",
        ide_with_jetbrains_language_inference("rubymine", "Gemfile"),
    );
    insert(
        "rider",
        ide_with_jetbrains_signal(
            "rider",
            ".sln/.csproj",
            any_glob_signal(".sln/.csproj", &["*.sln", "*.csproj"]),
        ),
    );
    insert(
        "clion",
        ide_with_jetbrains_language_inference("clion", "CMakeLists.txt"),
    );
    insert("appcode", ide_detector("appcode"));
    insert("androidstudio", ide_detector("androidstudio"));

    for entry in &entries {
        registry.insert(entry.provider.clone(), rule_entry_detector(entry));
    }

    Ok(registry)
}

/// Run all detectors in sorted key order over the scan target.
pub fn scan_target(
    ctx: &ScanCtx,
    detectors: &BTreeMap<String, Detector>,
) -> (std::collections::BTreeSet<String>, Vec<DetectionResult>) {
    let mut detected = std::collections::BTreeSet::new();
    let mut results = Vec::with_capacity(detectors.len());
    for (key, detector) in detectors {
        let mut result = detector(ctx);
        if result.key.is_empty() {
            result.key = key.clone();
        }
        if result.matched {
            detected.insert(result.key.clone());
        }
        results.push(result);
    }
    results.sort_by(|a, b| a.key.cmp(&b.key));
    (detected, results)
}
