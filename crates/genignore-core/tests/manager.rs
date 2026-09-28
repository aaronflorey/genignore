//! Managed-block engine tests: marker parsing, merge byte-preservation,
//! normalization, preview/dry-run/no-op/diff — parity with
//! `internal/gitignore` semantics.

use std::fs;
use std::path::PathBuf;

use genignore_core::manager::{build_managed_block, parse_managed_providers, FileAction, Manager};

fn tempdir(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "genignore-mgr-{}-{}-{:?}",
        name,
        std::process::id(),
        std::thread::current().id()
    ));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).unwrap();
    dir
}

fn block(providers: &[&str], template: &str) -> String {
    let keys: Vec<String> = providers.iter().map(|s| s.to_string()).collect();
    build_managed_block(&keys, &[], template, &[])
}

const TPL: &str = "*.log\nbuild/\n";

#[test]
fn creates_block_in_empty_file() {
    let dir = tempdir("create");
    let mgr = Manager::new(&dir);
    let action = mgr
        .upsert_managed_block(&block(&["node"], TPL), false)
        .unwrap();
    assert_eq!(action, FileAction::Created);
    let content = fs::read_to_string(dir.join(".gitignore")).unwrap();
    assert!(content.starts_with("# BEGIN genignore\n"));
    assert!(content.contains("# Providers: node\n"));
    assert!(content.contains("*.log\nbuild/\n"));
    assert!(content.ends_with("# END genignore\n"));
}

#[test]
fn merges_block_before_existing_content() {
    let dir = tempdir("merge");
    fs::write(dir.join(".gitignore"), "# mine\n.custom\n").unwrap();
    let mgr = Manager::new(&dir);
    let action = mgr
        .upsert_managed_block(&block(&["node"], TPL), false)
        .unwrap();
    assert_eq!(action, FileAction::Updated);
    let content = fs::read_to_string(dir.join(".gitignore")).unwrap();
    assert!(content.starts_with("# BEGIN genignore"));
    assert!(content.ends_with("# mine\n.custom\n"));
}

#[test]
fn no_markers_missing_final_newline_preserved() {
    let dir = tempdir("noeol");
    fs::write(dir.join(".gitignore"), "tail-rule").unwrap();
    let mgr = Manager::new(&dir);
    mgr.upsert_managed_block(&block(&["node"], TPL), false)
        .unwrap();
    let content = fs::read_to_string(dir.join(".gitignore")).unwrap();
    assert!(content.ends_with("\ntail-rule"));
    assert!(!content.ends_with('\n'));
}

#[test]
fn crlf_bytes_preserved_outside_markers() {
    let dir = tempdir("crlf");
    let existing = "line1\r\n# BEGIN genignore\nold\n# END genignore\r\nline2\r\n";
    fs::write(dir.join(".gitignore"), existing).unwrap();
    let mgr = Manager::new(&dir);
    mgr.upsert_managed_block(&block(&["go"], TPL), false)
        .unwrap();
    let content = fs::read_to_string(dir.join(".gitignore")).unwrap();
    assert!(content.starts_with("line1\r\n"));
    assert!(content.ends_with("line2\r\n"));
}

#[test]
fn malformed_markers_error_and_untouched() {
    let dir = tempdir("badmark");
    let bad = "# BEGIN genignore\nx\n# BEGIN genignore\n# END genignore\n";
    fs::write(dir.join(".gitignore"), bad).unwrap();
    let mgr = Manager::new(&dir);
    let err = mgr
        .upsert_managed_block(&block(&["go"], TPL), false)
        .unwrap_err();
    assert!(err.contains("malformed managed markers"));
    assert!(err.contains("\"# BEGIN genignore\""));
    assert_eq!(fs::read_to_string(dir.join(".gitignore")).unwrap(), bad);
}

#[test]
fn reversed_markers_error() {
    let dir = tempdir("reversed");
    fs::write(
        dir.join(".gitignore"),
        "# END genignore\nmid\n# BEGIN genignore\n",
    )
    .unwrap();
    let mgr = Manager::new(&dir);
    let err = mgr
        .upsert_managed_block(&block(&["go"], TPL), false)
        .unwrap_err();
    assert!(err.contains("malformed managed markers"));
}

#[test]
fn identical_block_reports_noop_and_skips_write() {
    let dir = tempdir("noop");
    let mgr = Manager::new(&dir);
    let b = block(&["node"], TPL);
    mgr.upsert_managed_block(&b, false).unwrap();
    let before = fs::metadata(dir.join(".gitignore"))
        .unwrap()
        .modified()
        .unwrap();
    let action = mgr.upsert_managed_block(&b, false).unwrap();
    assert_eq!(action, FileAction::NoOp);
    let after = fs::metadata(dir.join(".gitignore"))
        .unwrap()
        .modified()
        .unwrap();
    assert_eq!(before, after);
}

#[test]
fn dry_run_always_reports_dry_run_and_never_writes() {
    let dir = tempdir("dry");
    let mgr = Manager::new(&dir);
    let b = block(&["node"], TPL);
    assert_eq!(
        mgr.upsert_managed_block(&b, true).unwrap(),
        FileAction::DryRun
    );
    assert!(!dir.join(".gitignore").exists());
    // even when the content would be identical
    mgr.upsert_managed_block(&b, false).unwrap();
    assert_eq!(
        mgr.upsert_managed_block(&b, true).unwrap(),
        FileAction::DryRun
    );
}

#[test]
fn preview_returns_diff_without_writing() {
    let dir = tempdir("preview");
    fs::write(
        dir.join(".gitignore"),
        "# BEGIN genignore\n# old\n# END genignore\n",
    )
    .unwrap();
    let mgr = Manager::new(&dir);
    let preview = mgr.preview_managed_block(&block(&["go"], TPL)).unwrap();
    assert_eq!(preview.action, FileAction::Updated);
    assert!(preview
        .diff
        .starts_with("--- .gitignore\n+++ .gitignore\n@@ managed-block @@\n"));
    assert!(preview.diff.contains("-# old"));
    assert!(preview.diff.contains("+# BEGIN genignore"));
    assert!(preview.diff.contains("+# Providers: go"));
    // file untouched
    assert_eq!(
        fs::read_to_string(dir.join(".gitignore")).unwrap(),
        "# BEGIN genignore\n# old\n# END genignore\n"
    );
}

#[test]
fn parse_managed_providers_roundtrip() {
    let content = "# BEGIN genignore\n# Providers: a,b\n# END genignore\n";
    assert_eq!(
        parse_managed_providers(content),
        Some(vec!["a".to_string(), "b".to_string()])
    );
    assert_eq!(parse_managed_providers("no markers"), None);
    assert_eq!(
        parse_managed_providers("# END genignore\n# BEGIN genignore\n"),
        None
    );
    // malformed markers -> None (merge step surfaces the error later)
    assert_eq!(
        parse_managed_providers("# BEGIN genignore\n# BEGIN genignore\n# END genignore\n"),
        None
    );
}

#[test]
fn block_normalization() {
    // env rules always injected, sorted extras deduped, toptal comments stripped
    let tpl = "# Created by https://www.toptal.com/developers/gitignore/api/node\n*.log\n.env.local\n# comment\n!.env.example\n";
    let extra = vec![
        ".env.prod".to_string(),
        "*.log".to_string(),
        "custom/".to_string(),
    ];
    let b = build_managed_block(&["node".to_string()], &[], tpl, &[extra]);
    assert!(!b.contains("toptal.com"));
    // required env rules present in order
    let env_pos = |pat: &str| b.find(pat).unwrap();
    assert!(env_pos("\n.env\n") < env_pos("\n.env.*\n"));
    assert!(env_pos("\n.env.*\n") < env_pos("\n!.env.example\n"));
    assert!(env_pos("\n!.env.example\n") < env_pos("\n!.env.ci\n"));
    // conflicting duplicates of required rules dropped
    assert_eq!(b.matches("!.env.example").count(), 1);
    // template-provided env rules canonicalized into the env section
    assert!(b.contains("\n.env.local\n"));
    assert!(b.contains("\n.env.prod\n"));
    // custom non-env extra rule appended
    assert!(b.contains("custom/"));
    // *.log kept once (template copy), config dup dropped
    assert_eq!(b.matches("*.log").count(), 1);
}

#[test]
fn provenance_metadata_line() {
    let meta = vec!["# Provenance: github/gitignore@abc [node]".to_string()];
    let b = build_managed_block(&["node".to_string()], &meta, TPL, &[]);
    assert!(b.contains("# Provenance: github/gitignore@abc [node]\n"));
    // blank metadata lines skipped
    let b2 = build_managed_block(&["node".to_string()], &[String::new()], TPL, &[]);
    assert!(!b2.contains("Provenance"));
}
