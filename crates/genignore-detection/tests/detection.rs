//! Detection engine tests: one-level scan scope, ignored dirs, hidden
//! signals, rule matcher paths + line evidence, symlink safety, size cap.

use std::fs;
use std::path::{Path, PathBuf};

use genignore_detection::catalog::{self, Rule, RuleType};
use genignore_detection::signalfile::{self, ReadError};
use genignore_detection::{
    globmatch::path_match, one_level_search_dirs, registry, scan_target, ScanCtx,
};

fn tempdir(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "genignore-det-{}-{}-{:?}",
        name,
        std::process::id(),
        std::thread::current().id()
    ));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).unwrap();
    dir
}

fn git_init(dir: &Path) {
    let ok = std::process::Command::new("git")
        .args(["init", "-q"])
        .current_dir(dir)
        .status()
        .unwrap()
        .success();
    assert!(ok, "git init failed");
}

fn detectors() -> std::collections::BTreeMap<String, genignore_detection::Detector> {
    registry(|_| true).unwrap()
}

#[test]
fn glob_match_go_semantics() {
    assert!(path_match("*.kt", "main.kt").unwrap());
    assert!(!path_match("*.kt", "dir/main.kt").unwrap());
    assert!(path_match("a?c", "abc").unwrap());
    assert!(!path_match("a?c", "a/c").unwrap());
    assert!(path_match("[a-z]x", "bx").unwrap());
    assert!(!path_match("[a-z]x", "Bx").unwrap());
    assert!(path_match("[^a-z]x", "Bx").unwrap());
    assert!(path_match("*", "anything").unwrap());
    assert!(path_match("*", "").unwrap());
    assert!(!path_match("*", "a/b").unwrap());
    assert!(path_match("\\*", "*").unwrap());
    assert!(path_match("a\\[b", "a[b").unwrap());
    // Go path.Match has no `!` class negation: [!a] is the set {!,a}.
    assert!(!path_match("[!a]", "b").unwrap());
    assert!(path_match("[!a]", "!").unwrap());
    assert!(path_match("[^a]", "b").unwrap());
    assert!(path_match("a/*/b", "a/x/b").unwrap());
    assert!(!path_match("a/*/b", "a/x/y/b").unwrap());
}

#[test]
fn scan_is_one_level_only() {
    let dir = tempdir("onelevel");
    fs::create_dir_all(dir.join("nested/deep")).unwrap();
    // signal only below the one-level boundary -> no match
    fs::write(dir.join("nested/deep/package.json"), "{}").unwrap();
    let dirs = one_level_search_dirs(&dir);
    assert!(dirs.contains(&dir.join("nested")));
    assert!(!dirs.contains(&dir.join("nested/deep")));

    let det = detectors();
    let ctx = ScanCtx::new(&dir);
    let (matched, _) = scan_target(&ctx, &det);
    assert!(!matched.contains("node"));

    // one level down -> matches
    fs::write(dir.join("nested/package.json"), "{}").unwrap();
    let ctx = ScanCtx::new(&dir);
    let (matched, _) = scan_target(&ctx, &det);
    assert!(matched.contains("node"));

    // root level -> matches, evidence at root path
    fs::remove_file(dir.join("nested/package.json")).unwrap();
    fs::remove_dir_all(dir.join("nested")).unwrap();
    fs::write(dir.join("package.json"), "{}").unwrap();
    let ctx = ScanCtx::new(&dir);
    let (matched, results) = scan_target(&ctx, &det);
    assert!(matched.contains("node"));
    let node = results.iter().find(|r| r.key == "node").unwrap();
    assert_eq!(node.reason, "found node project file");
}

#[test]
fn glob_only_signals_match_csproj() {
    // Exercises filepath_glob's path handling: the pattern is split and the
    // matcher runs on filenames only, so a joined absolute path (which would
    // contain '\' separators on Windows) is never globbed as a single name.
    let dir = tempdir("csproj");
    fs::write(dir.join("App.csproj"), "<Project />").unwrap();
    let det = detectors();
    let ctx = ScanCtx::new(&dir);
    let (matched, results) = scan_target(&ctx, &det);
    assert!(matched.contains("csharp"));
    let csharp = results.iter().find(|r| r.key == "csharp").unwrap();
    assert_eq!(csharp.reason, "found csharp solution/project file");
    assert!(matched.contains("dotnetcore"));

    // A .cs file in a first-level subdirectory is a valid glob match too.
    fs::remove_file(dir.join("App.csproj")).unwrap();
    fs::create_dir_all(dir.join("src")).unwrap();
    fs::write(dir.join("src/Main.cs"), "class Main {}").unwrap();
    let ctx = ScanCtx::new(&dir);
    let (matched, results) = scan_target(&ctx, &det);
    assert!(matched.contains("csharp"));
    let csharp = results.iter().find(|r| r.key == "csharp").unwrap();
    assert_eq!(csharp.reason, "found csharp source file");
}

#[test]
fn upstream_build_engine_signals_report_matching_evidence() {
    let cases = [
        ("gradle", "build.gradle"),
        ("gradle", "build.gradle.kts"),
        ("gradle", "settings.gradle"),
        ("gradle", "settings.gradle.kts"),
        ("cmake", "CMakeLists.txt"),
        ("godot", "project.godot"),
        ("unity", "ProjectSettings/ProjectVersion.txt"),
    ];

    for (provider, signal) in cases {
        let dir = tempdir(&format!("{provider}-{signal}"));
        let signal_path = dir.join(signal);
        fs::create_dir_all(signal_path.parent().unwrap()).unwrap();
        fs::write(&signal_path, "signal").unwrap();

        let (matched, results) = scan_target(&ScanCtx::new(&dir), &detectors());
        assert!(matched.contains(provider), "{provider} missed {signal}");
        let result = results
            .iter()
            .find(|result| result.key == provider)
            .unwrap();
        assert!(result.matched, "{provider} did not match {signal}");
        assert_eq!(result.evidence, signal_path.to_string_lossy());
    }
}

#[test]
fn upstream_javascript_runtime_and_framework_signals_report_evidence() {
    let cases = [
        ("angular", "angular.json", None),
        ("deno", "deno.json", None),
        ("deno", "deno.jsonc", None),
        ("bun", "bun.lock", Some("node")),
        ("bun", "bun.lockb", Some("node")),
        ("nestjs", "nest-cli.json", None),
    ];

    for (provider, signal, also_matches) in cases {
        let dir = tempdir(&format!("{provider}-{signal}"));
        let signal_path = dir.join(signal);
        fs::write(&signal_path, "{}").unwrap();

        let (matched, results) = scan_target(&ScanCtx::new(&dir), &detectors());
        assert!(matched.contains(provider), "{provider} missed {signal}");
        let result = results
            .iter()
            .find(|result| result.key == provider)
            .unwrap();
        assert!(result.matched, "{provider} did not match {signal}");
        assert_eq!(result.evidence, signal_path.to_string_lossy());
        if let Some(additional_provider) = also_matches {
            assert!(
                matched.contains(additional_provider),
                "{signal} should also match {additional_provider}"
            );
        }
    }
}

#[test]
fn scan_skips_gitignored_subdirs() {
    let dir = tempdir("ignored");
    git_init(&dir);
    fs::write(dir.join(".gitignore"), "ignored/\n").unwrap();
    fs::create_dir_all(dir.join("ignored")).unwrap();
    fs::write(dir.join("ignored/composer.json"), "{}").unwrap();
    fs::create_dir_all(dir.join("kept")).unwrap();
    fs::write(dir.join("kept/go.mod"), "module x\n").unwrap();

    let dirs = one_level_search_dirs(&dir);
    assert!(dirs.contains(&dir.join("kept")));
    assert!(!dirs.contains(&dir.join("ignored")));

    let ctx = ScanCtx::new(&dir);
    let (matched, _) = scan_target(&ctx, &detectors());
    assert!(matched.contains("go"));
    assert!(!matched.contains("composer"));
}

#[test]
fn scan_reads_hidden_signal_files() {
    let dir = tempdir("hidden");
    git_init(&dir);
    fs::write(dir.join(".terraform.lock.hcl"), "x").unwrap();
    let ctx = ScanCtx::new(&dir);
    let (matched, _) = scan_target(&ctx, &detectors());
    assert!(matched.contains("terraform"));
}

#[test]
fn file_path_rule_glob_and_evidence() {
    let dir = tempdir("filepath");
    fs::create_dir_all(dir.join("src")).unwrap();
    fs::write(dir.join("src/lib.rs"), "fn main(){}").unwrap();
    let m = signalfile::glob_in_root(&dir, "src/*.rs").unwrap();
    assert_eq!(m, vec!["src/lib.rs".to_string()]);

    assert_eq!(
        signalfile::glob_in_root(&dir, "src/lib.rs").unwrap(),
        vec!["src/lib.rs"]
    );
    assert!(signalfile::glob_in_root(&dir, "src/none.rs")
        .unwrap()
        .is_empty());
    // escaping/absolute patterns resolve to "no match", like fs.Glob's
    // swallowed Stat errors against a confined root
    assert!(signalfile::glob_in_root(&dir, "../x").unwrap().is_empty());
    assert!(signalfile::glob_in_root(&dir, "/abs").unwrap().is_empty());
}

#[test]
fn content_line_rule_reports_path_line() {
    let dir = tempdir("contentline");
    fs::write(
        dir.join("composer.json"),
        "{\r\n  \"require\": {\"laravel/framework\": \"*\"}\r\n}\r\n",
    )
    .unwrap();
    let det = detectors();
    let ctx = ScanCtx::new(&dir);
    let (_m, results) = scan_target(&ctx, &det);
    let laravel = results.iter().find(|r| r.key == "laravel").unwrap();
    assert!(laravel.matched);
    assert_eq!(laravel.reason, "composer.json references laravel/framework");
    assert_eq!(
        laravel.evidence,
        format!("{}:2", dir.join("composer.json").display())
    );
}

#[test]
fn rule_catalog_rejects_unsafe_paths() {
    let load = |path: &str| {
        let raw = format!(
            r#"{{"providers": [{{"provider": "node", "match": [{{"type": "file_path", "path": {path:?}}}]}}]}}"#
        );
        catalog::load_rule_bytes(&raw, &|_| true)
    };
    assert!(load("../escape").unwrap_err().contains("parent traversal"));
    assert!(load("/abs").unwrap_err().contains("project-relative"));
    assert!(load("//host/share")
        .unwrap_err()
        .contains("project-relative"));
    assert!(load("c:\\win").unwrap_err().contains("project-relative"));
    assert!(load("").unwrap_err().contains("must not be empty"));

    // unknown rule type / missing contains / empty providers / dup providers
    let t = |raw: &str| catalog::load_rule_bytes(raw, &|_| true);
    assert!(
        t(r#"{"providers":[{"provider":"node","match":[{"type":"bogus","path":"x"}]}]}"#)
            .unwrap_err()
            .contains("unsupported rule type")
    );
    assert!(t(
        r#"{"providers":[{"provider":"node","match":[{"type":"file_content_line","path":"x"}]}]}"#
    )
    .unwrap_err()
    .contains("non-empty contains"));
    assert!(t(r#"{"providers":[{"provider":"node","match":[{"type":"file_path","path":"x","contains":"y"}]}]}"#)
        .unwrap_err()
        .contains("must not set contains"));
    assert!(t(r#"{"providers":[]}"#)
        .unwrap_err()
        .contains("providers must not be empty"));
    assert!(t(r#"{"providers":[{"provider":"node","match":[{"type":"file_path","path":"x"}]},{"provider":"node","match":[{"type":"file_path","path":"y"}]}]}"#)
        .unwrap_err()
        .contains("duplicate provider"));
    assert!(t(r#"{"providers":[{"provider":"node","match":[{"type":"file_path","path":"x"},{"type":"file_path","path":"x"}]}]}"#)
        .unwrap_err()
        .contains("duplicate rule"));
    let restricted = |raw: &str| catalog::load_rule_bytes(raw, &|k| k == "node");
    assert!(restricted(
        r#"{"providers":[{"provider":"nosuch","match":[{"type":"file_path","path":"x"}]}]}"#
    )
    .unwrap_err()
    .contains("not available"));
    assert!(t(r#"{"providers":[{"provider":"node","match":[]}]}"#)
        .unwrap_err()
        .contains("at least one match rule"));
    // unknown fields rejected (strict decode)
    assert!(t(r#"{"bogus":true,"providers":[]}"#).is_err());
    assert!(t(
        r#"{"providers":[{"provider":"node","bogus":1,"match":[{"type":"file_path","path":"x"}]}]}"#
    )
    .is_err());
}

#[test]
fn symlink_signals_are_rejected() {
    let dir = tempdir("symlink");
    let outside = tempdir("outside");
    fs::write(outside.join("secret.toml"), "x").unwrap();
    std::os::unix::fs::symlink(outside.join("secret.toml"), dir.join("pyproject.toml")).unwrap();

    let err = signalfile::read_signal_file(&dir, "pyproject.toml").unwrap_err();
    assert!(matches!(err, ReadError::Symlink(_)));

    std::os::unix::fs::symlink(&outside, dir.join("linkdir")).unwrap();
    fs::write(outside.join("go.mod"), "module x").unwrap();
    assert!(signalfile::glob_in_root(&dir, "linkdir/*")
        .unwrap()
        .is_empty());
}

#[test]
fn oversized_file_content_line_rule_is_safe_skip() {
    let dir = tempdir("bigline");
    let big = "x".repeat(1024 * 1024 + 5);
    fs::write(dir.join("composer.json"), big.as_bytes()).unwrap();
    let ctx = ScanCtx::new(&dir);
    let (_m, results) = scan_target(&ctx, &detectors());
    let laravel = results.iter().find(|r| r.key == "laravel").unwrap();
    assert!(!laravel.matched && laravel.error.is_empty());
}

#[test]
fn oversized_signal_file_is_safe_skip() {
    let dir = tempdir("big");
    let big = "x".repeat(1024 * 1024 + 5);
    fs::write(dir.join("package.json"), &big).unwrap();
    let err = signalfile::read_signal_file(&dir, "package.json").unwrap_err();
    assert!(matches!(err, ReadError::TooLarge(_)));

    // file_path rules still match (stat, not read); content reads safe-skip.
    let ctx = ScanCtx::new(&dir);
    let (_m, results) = scan_target(&ctx, &detectors());
    let node = results.iter().find(|r| r.key == "node").unwrap();
    assert!(node.matched && node.error.is_empty());
}

#[test]
fn directory_signal_is_safe_skip() {
    let dir = tempdir("dirskip");
    fs::create_dir_all(dir.join("package.json")).unwrap();
    let err = signalfile::read_signal_file(&dir, "package.json").unwrap_err();
    assert!(matches!(err, ReadError::Directory(_)));
}

#[test]
fn rule_sorting_is_deterministic() {
    let mut rules = [
        Rule {
            rule_type: RuleType::FilePath,
            path: "b".into(),
            contains: String::new(),
        },
        Rule {
            rule_type: RuleType::FileContentLine,
            path: "a".into(),
            contains: "x".into(),
        },
        Rule {
            rule_type: RuleType::FilePath,
            path: "a".into(),
            contains: String::new(),
        },
    ];
    rules.sort_by(|a, b| {
        a.path
            .cmp(&b.path)
            .then_with(|| a.rule_type.as_str().cmp(b.rule_type.as_str()))
            .then_with(|| a.contains.cmp(&b.contains))
    });
    assert_eq!(rules[0].path, "a");
    assert_eq!(rules[0].rule_type, RuleType::FileContentLine);
    assert_eq!(rules[1].path, "a");
    assert_eq!(rules[1].rule_type, RuleType::FilePath);
    assert_eq!(rules[2].path, "b");
}

#[test]
fn embedded_rule_catalog_loads() {
    let entries = catalog::load_rule_catalog(|_| true).unwrap();
    assert_eq!(entries.len(), 27);
    let providers: Vec<&str> = entries.iter().map(|e| e.provider.as_str()).collect();
    assert!(providers.contains(&"node"));
    assert!(providers.contains(&"terraform"));
    assert!(providers.contains(&"angular"));
    assert!(providers.contains(&"bun"));
    assert!(providers.contains(&"deno"));
    assert!(providers.contains(&"nestjs"));
    assert!(providers.contains(&"gradle"));
    assert!(providers.contains(&"cmake"));
    assert!(providers.contains(&"godot"));
    assert!(providers.contains(&"unity"));
    let mut sorted = providers.clone();
    sorted.sort();
    assert_eq!(providers, sorted);
}
