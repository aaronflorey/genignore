//! Git working-tree gate: detection-dependent commands only run when the
//! current directory is exactly the top-level of a Git working tree.

use std::fmt;
use std::path::Path;
use std::process::Command;

#[derive(Debug)]
pub struct GitRootError(pub String);

impl fmt::Display for GitRootError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for GitRootError {}

/// Require `cwd` to be the canonical top-level of a Git working tree.
/// `command` names the calling command for the user-facing error message.
/// Rejects non-repositories, repository subdirectories, and bare
/// repositories — all before any scan or write.
pub fn require_worktree_root(command: &str, cwd: &Path) -> Result<(), GitRootError> {
    let not_root = |why: &str| -> GitRootError {
        GitRootError(format!(
            "{} must run from the top-level of a Git working tree ({})",
            command, why
        ))
    };

    let output = Command::new("git")
        .args(["rev-parse", "--show-toplevel"])
        .current_dir(cwd)
        .output()
        .map_err(|err| GitRootError(format!("failed to inspect git working tree: {}", err)))?;

    if !output.status.success() {
        return Err(not_root("not a git repository or not a working tree"));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let toplevel = stdout.trim();
    if toplevel.is_empty() {
        return Err(not_root("not a git repository or not a working tree"));
    }

    let canonical_top = std::fs::canonicalize(toplevel)
        .map_err(|err| GitRootError(format!("failed to resolve git top-level: {}", err)))?;
    let canonical_cwd = std::fs::canonicalize(cwd)
        .map_err(|err| GitRootError(format!("failed to resolve current directory: {}", err)))?;

    if canonical_top != canonical_cwd {
        return Err(not_root("not at the git repository root"));
    }
    Ok(())
}
