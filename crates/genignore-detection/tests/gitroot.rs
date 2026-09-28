//! Git working-tree root gate: the intentional contract change — detect/add/
//! resolve/doctor only run when cwd is the canonical top-level of a working
//! tree.

use std::fs;
use std::path::{Path, PathBuf};

use genignore_detection::gitroot::require_worktree_root;

fn tempdir(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "genignore-gitroot-{}-{}-{:?}",
        name,
        std::process::id(),
        std::thread::current().id()
    ));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).unwrap();
    dir
}

fn git(dir: &Path, args: &[&str]) {
    let ok = std::process::Command::new("git")
        .args(args)
        .current_dir(dir)
        .env("GIT_CONFIG_NOSYSTEM", "1")
        .env("HOME", dir) // isolate from user gitconfig (e.g. init.defaultBranch)
        .status()
        .unwrap()
        .success();
    assert!(ok, "git {:?} failed", args);
}

#[test]
fn rejects_non_repository() {
    let dir = tempdir("nongit");
    let err = require_worktree_root("detect", &dir).unwrap_err();
    assert!(err.to_string().contains("must run from the top-level"));
}

#[test]
fn accepts_repository_root() {
    let dir = tempdir("repo");
    git(&dir, &["init", "-q"]);
    require_worktree_root("detect", &dir).unwrap();
}

#[test]
fn rejects_repository_subdirectory() {
    let dir = tempdir("subdir");
    git(&dir, &["init", "-q"]);
    let sub = dir.join("sub/dir");
    fs::create_dir_all(&sub).unwrap();
    let err = require_worktree_root("add", &sub).unwrap_err();
    assert!(err.to_string().contains("must run from the top-level"));
}

#[test]
fn rejects_bare_repository() {
    let dir = tempdir("bare");
    git(&dir, &["init", "--bare", "-q"]);
    let err = require_worktree_root("resolve", &dir).unwrap_err();
    assert!(err.to_string().contains("must run from the top-level"));
}

#[test]
fn accepts_linked_worktree_root() {
    let dir = tempdir("mainrepo");
    git(&dir, &["init", "-q"]);
    git(&dir, &["config", "user.email", "t@t"]);
    git(&dir, &["config", "user.name", "t"]);
    fs::write(dir.join("f"), "x").unwrap();
    git(&dir, &["add", "."]);
    git(&dir, &["commit", "-qm", "init"]);
    let wt = tempdir("linkedwt");
    let _ = fs::remove_dir_all(&wt);
    git(&dir, &["worktree", "add", wt.to_str().unwrap(), "-q"]);
    require_worktree_root("doctor", &wt).unwrap();
    // but a subdir of the linked worktree is still rejected
    let sub = wt.join("sub");
    fs::create_dir_all(&sub).unwrap();
    assert!(require_worktree_root("doctor", &sub).is_err());
}

#[test]
fn symlinked_cwd_path_passes() {
    let dir = tempdir("linkcwd");
    git(&dir, &["init", "-q"]);
    let link = dir.parent().unwrap().join(format!(
        "genignore-linkcwd-{}-{:?}",
        std::process::id(),
        std::thread::current().id()
    ));
    std::os::unix::fs::symlink(&dir, &link).unwrap();
    require_worktree_root("detect", &link).unwrap();
    let _ = fs::remove_file(&link);
}
