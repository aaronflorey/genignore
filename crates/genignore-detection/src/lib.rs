//! genignore-detection: provider signal detection over a repository checkout,
//! plus the Git working-tree root gate used by detection-dependent commands.

pub mod catalog;
pub mod detectors;
pub mod gitignore_match;
pub mod gitroot;
pub mod globmatch;
pub mod signalfile;

use serde::Serialize;

pub use catalog::{load_rule_catalog, Entry, Rule, RuleType};
pub use detectors::{one_level_search_dirs, registry, scan_target, Detector, ScanCtx};
pub use gitroot::{require_worktree_root, GitRootError};

/// One detector's outcome — JSON shape matches `provider.Result` in Go.
#[derive(Debug, Clone, Serialize)]
pub struct DetectionResult {
    pub key: String,
    pub matched: bool,
    pub reason: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub evidence: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub error: String,
}
