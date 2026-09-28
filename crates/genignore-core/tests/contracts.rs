//! Contract tests against `testdata/contracts/` — the same goldens the Go
//! suite asserts, produced with identical restricted detector sets and the
//! same normalization (cwd -> "<fixture>", evidence relativized).

use std::fs;
use std::path::{Path, PathBuf};

use genignore_core::manager;
use genignore_core::service::Service;

fn copy_dir(src: &Path, dst: &Path) {
    fs::create_dir_all(dst).unwrap();
    for entry in fs::read_dir(src).unwrap().flatten() {
        let name = entry.file_name();
        let to = dst.join(&name);
        let ft = entry.file_type().unwrap();
        if ft.is_dir() {
            copy_dir(&entry.path(), &to);
        } else if ft.is_symlink() {
            let target = fs::read_link(entry.path()).unwrap();
            std::os::unix::fs::symlink(&target, &to).unwrap();
        } else {
            fs::copy(entry.path(), &to).unwrap();
        }
    }
}

fn fixture(name: &str) -> PathBuf {
    let src = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../testdata/repos")
        .join(name);
    let dst = std::env::temp_dir().join(format!(
        "genignore-contract-{}-{}-{:?}",
        name,
        std::process::id(),
        std::thread::current().id()
    ));
    let _ = fs::remove_dir_all(&dst);
    copy_dir(&src, &dst);
    git_init(&dst);
    dst
}

fn git_init(dir: &Path) {
    let status = std::process::Command::new("git")
        .args(["init", "-q"])
        .current_dir(dir)
        .status()
        .expect("git init");
    assert!(status.success());
}

fn contract(name: &str) -> String {
    let path = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../testdata/contracts")
        .join(name);
    fs::read_to_string(&path).expect("read contract")
}

fn restricted_service(dir: &Path, keys: &[&str]) -> Service {
    let mut svc = Service::new(dir.to_path_buf(), Default::default()).unwrap();
    let all = std::mem::take(&mut svc.detectors);
    svc.detectors = all
        .into_iter()
        .filter(|(k, _)| keys.contains(&k.as_str()))
        .collect();
    svc
}

fn normalize_evidence(root: &Path, evidence: &str) -> String {
    if evidence.is_empty() {
        return String::new();
    }
    let ev = Path::new(evidence);
    match ev.strip_prefix(root) {
        Ok(rel) => rel.to_string_lossy().replace('\\', "/"),
        Err(_) => evidence.replace('\\', "/"),
    }
}

#[test]
fn detect_diff_next_vscode_app_contract() {
    let dir = fixture("next-vscode-app");
    let svc = restricted_service(
        &dir,
        &["jetbrains", "nextjs", "node", "react", "visualstudiocode"],
    );

    let mut res = svc.detect(&[], &[], false, true).unwrap();
    assert!(
        !dir.join(".gitignore").exists(),
        "diff must not write .gitignore"
    );

    res.cwd = "<fixture>".to_string();
    for d in &mut res.detection_results {
        d.evidence = normalize_evidence(&dir, &d.evidence);
    }
    let got = serde_json::to_string_pretty(&res).unwrap() + "\n";
    assert_eq!(got, contract("detect_diff_next_vscode_app.json"));
}

#[test]
fn resolve_next_vscode_app_contract() {
    let dir = fixture("next-vscode-app");
    let svc = restricted_service(
        &dir,
        &["jetbrains", "nextjs", "node", "react", "visualstudiocode"],
    );

    let mut res = svc.resolve(&[], &[]).unwrap();
    assert!(!dir.join(".gitignore").exists(), "resolve must not write");

    res.cwd = "<fixture>".to_string();
    for d in &mut res.detection_results {
        d.evidence = normalize_evidence(&dir, &d.evidence);
    }
    let got = serde_json::to_string_pretty(&res).unwrap() + "\n";
    assert_eq!(got, contract("resolve_next_vscode_app.json"));
}

#[test]
fn doctor_laravel_jetbrains_app_contract() {
    let dir = fixture("laravel-jetbrains-app");
    let svc = restricted_service(&dir, &["composer", "jetbrains", "laravel"]);

    let mut res = svc.doctor(&[], &[]).unwrap();
    res.cwd = "<fixture>".to_string();
    for d in &mut res.detections {
        d.evidence = normalize_evidence(&dir, &d.evidence);
    }
    let got = serde_json::to_string_pretty(&res).unwrap() + "\n";
    assert_eq!(got, contract("doctor_laravel_jetbrains_app.json"));
}

#[test]
fn detect_managed_block_next_vscode_app_contract() {
    let run = || {
        let dir = fixture("next-vscode-app");
        let svc = restricted_service(
            &dir,
            &["jetbrains", "nextjs", "node", "react", "visualstudiocode"],
        );
        let res = svc.detect(&[], &[], false, false).unwrap();
        (res, fs::read_to_string(dir.join(".gitignore")).unwrap())
    };
    let (_, first_block) = run();
    let (_, second_block) = run();
    assert_eq!(
        first_block, second_block,
        "managed block changed across runs"
    );
    assert_eq!(
        first_block,
        contract("managed_block_next_vscode_app.gitignore")
    );
    assert!(
        !first_block.to_lowercase().contains("remote"),
        "managed block contains obsolete wording"
    );
}

#[test]
fn add_managed_block_contract_preserves_user_lines() {
    // AC-009/AC-010 equivalent of the Go suite: manual content outside
    // markers survives verbatim and the regenerated block matches the golden.
    let dir = fixture("next-vscode-app");
    let path = dir.join(".gitignore");
    fs::write(
        &path,
        "# user-owned rule\n# BEGIN genignore\n# old block\n# END genignore\n.planning\n",
    )
    .unwrap();

    let svc = Service::new(dir.clone(), Default::default()).unwrap();
    let res = svc
        .add(
            &[
                "jetbrains".into(),
                "nextjs".into(),
                "node".into(),
                "visualstudiocode".into(),
            ],
            false,
            false,
        )
        .unwrap();
    assert_eq!(res.file_action, "updated");

    let content = fs::read_to_string(&path).unwrap();
    let start = content.find(manager::START_MARKER).unwrap();
    let end = content.find(manager::END_MARKER).unwrap() + manager::END_MARKER.len() + 1;
    let managed_block = &content[start..end];
    assert_eq!(
        managed_block,
        contract("managed_block_next_vscode_app.gitignore")
    );
    assert_eq!(
        content,
        format!("# user-owned rule\n{}.planning\n", managed_block)
    );
}
