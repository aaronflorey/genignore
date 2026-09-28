//! Generates the embedded upstream template catalog table by walking the
//! checked-in `internal/templatecatalog/github-gitignore` submodule. The build
//! fails when the snapshot is missing — the binary must never rely on network
//! fetching for template content.

use std::collections::{BTreeMap, BTreeSet};
use std::env;
use std::fs;
use std::path::{Path, PathBuf};

fn main() {
    let manifest_dir = env::var("CARGO_MANIFEST_DIR").unwrap();
    let root = Path::new(&manifest_dir)
        .join("../../internal/templatecatalog/github-gitignore")
        .canonicalize()
        .expect("embedded github/gitignore snapshot missing: run `git submodule update --init --recursive`");
    println!("cargo:rerun-if-changed={}", root.display());

    let mut template_paths = Vec::new();
    collect(&root, "", &mut template_paths);
    template_paths.sort();

    let root_keys: BTreeSet<String> = template_paths
        .iter()
        .filter(|p| !p.contains('/'))
        .map(|p| p.trim_end_matches(".gitignore").to_lowercase())
        .collect();

    let mut providers: BTreeMap<String, (String, String)> = BTreeMap::new();
    for path in &template_paths {
        let parts: Vec<&str> = path.split('/').collect();
        let base = parts.last().unwrap().trim_end_matches(".gitignore");
        let key = if parts.len() == 3 {
            format!("{}/{}", parts[1], base).to_lowercase()
        } else if parts.len() == 2 {
            let lower = base.to_lowercase();
            if root_keys.contains(&lower) {
                format!("{}/{}", parts[0], base).to_lowercase()
            } else {
                lower
            }
        } else {
            base.to_lowercase()
        };
        if providers.contains_key(&key) {
            panic!(
                "duplicate embedded upstream template provider {:?} for {:?}",
                key, path
            );
        }
        let raw = fs::read_to_string(root.join(path))
            .unwrap_or_else(|e| panic!("read embedded upstream template {:?}: {}", path, e));
        let content = raw.trim_matches('\n').to_string();
        providers.insert(key, (path.clone(), content));
    }
    if providers.is_empty() {
        panic!("no embedded upstream templates found");
    }

    let mut out = String::from("pub static TEMPLATES: &[(&str, &str)] = &[\n");
    for (key, (_path, content)) in &providers {
        out.push_str(&format!("    ({:?}, {:?}),\n", key, content));
    }
    out.push_str("];\n");

    let dest = PathBuf::from(env::var("OUT_DIR").unwrap()).join("templates.rs");
    fs::write(&dest, out).unwrap();
}

fn collect(dir: &Path, prefix: &str, paths: &mut Vec<String>) {
    let mut entries: Vec<_> = fs::read_dir(dir)
        .unwrap_or_else(|e| panic!("read embedded catalog dir {:?}: {}", dir, e))
        .flatten()
        .collect();
    entries.sort_by_key(|e| e.file_name());
    for entry in entries {
        let name = entry.file_name().to_string_lossy().to_string();
        let rel = if prefix.is_empty() {
            name.clone()
        } else {
            format!("{}/{}", prefix, name)
        };
        let ft = entry.file_type().unwrap();
        if ft.is_dir() {
            collect(&entry.path(), &rel, paths);
            continue;
        }
        if !ft.is_file() {
            continue;
        }
        if !name.ends_with(".gitignore") {
            continue;
        }
        let parts: Vec<&str> = rel.split('/').collect();
        let eligible = parts.len() == 1
            || (parts.len() == 2 && parts[0] == "Global")
            || (parts.len() == 3 && parts[0] == "community");
        if eligible {
            paths.push(rel);
        }
    }
}
