//! Command services: resolve/detect/add/doctor, provider selection,
//! provenance metadata, and doctor runtime diagnostics — ported from
//! `internal/app/service.go`. All detection-dependent entry points enforce
//! the Git worktree-root gate first.

use std::collections::{BTreeMap, BTreeSet, HashSet};
use std::path::PathBuf;
use std::sync::OnceLock;

use genignore_detection::{
    registry, require_worktree_root, scan_target, DetectionResult, Detector, ScanCtx,
};

use crate::catalog;
use crate::config::Config;
use crate::manager::{self, FileAction, Manager};
use crate::types::{
    CatalogResult, CommandResult, DoctorDetection, DoctorResult, DoctorRuntime, ResolveResult,
};

pub struct Service {
    pub cwd: PathBuf,
    pub config: Config,
    pub manager: Manager,
    pub detectors: BTreeMap<String, Detector>,
    supported: HashSet<String>,
}

impl Service {
    pub fn new(cwd: PathBuf, config: Config) -> Result<Service, String> {
        let supported: HashSet<String> = catalog::supported_keys()
            .map_err(|err| format!("initialize provider registry: {}", err))?
            .into_iter()
            .collect();
        let detectors = registry(|key| supported.contains(key))?;
        Ok(Service {
            manager: Manager::new(&cwd),
            cwd,
            config,
            detectors,
            supported,
        })
    }

    fn git_gate(&self, command: &str) -> Result<(), String> {
        require_worktree_root(command, &self.cwd).map_err(|e| e.to_string())
    }

    pub fn resolve(&self, include: &[String], exclude: &[String]) -> Result<ResolveResult, String> {
        self.git_gate("resolve")?;
        let selection = self.resolve_selection(include, exclude)?;
        Ok(ResolveResult {
            command: "resolve".to_string(),
            cwd: self.cwd.to_string_lossy().to_string(),
            detected_providers: selection.detected_providers,
            included_providers: selection.included_providers,
            excluded_providers: selection.excluded_providers,
            final_providers: selection.final_providers,
            unsupported_key_warnings: selection.warnings,
            detection_results: selection.detection_results,
        })
    }

    pub fn detect(
        &self,
        include: &[String],
        exclude: &[String],
        dry_run: bool,
        diff: bool,
    ) -> Result<CommandResult, String> {
        self.git_gate("detect")?;
        let selection = self.resolve_selection(include, exclude)?;
        let (tpl_providers, content) = catalog::fetch_template(&selection.final_providers)?;
        let block = manager::build_managed_block(
            &selection.final_providers,
            &managed_block_metadata(&selection.final_providers),
            &content,
            std::slice::from_ref(&self.config.defaults.ignore_rules),
        );
        let (action, diff_text) = apply_managed_block(&self.manager, &block, dry_run, diff)?;
        Ok(CommandResult {
            command: "detect".to_string(),
            cwd: self.cwd.to_string_lossy().to_string(),
            detected_providers: selection.detected_providers,
            included_providers: selection.included_providers,
            excluded_providers: selection.excluded_providers,
            added_providers: Vec::new(),
            final_providers: selection.final_providers,
            unsupported_key_warnings: selection.warnings,
            detection_results: selection.detection_results,
            file_action: action.as_str().to_string(),
            preview_only: diff,
            diff: diff_text,
            template_provider_count: tpl_providers.len(),
        })
    }

    pub fn add(&self, keys: &[String], dry_run: bool, diff: bool) -> Result<CommandResult, String> {
        self.git_gate("add")?;
        let mut input_keys = self.config.defaults.providers.clone();
        input_keys.extend_from_slice(keys);
        let (sanitized_keys, warnings) = self.sanitize_keys(&input_keys);

        let existing = self
            .manager
            .read_managed_providers()?
            .unwrap_or_default()
            .into_iter()
            .filter(|key| self.supported.contains(key))
            .collect::<Vec<String>>();

        let mut final_set: BTreeSet<String> = existing.iter().cloned().collect();
        let mut added = Vec::new();
        for key in &sanitized_keys {
            if !final_set.contains(key) {
                added.push(key.clone());
            }
            final_set.insert(key.clone());
        }
        added.sort();

        let final_providers: Vec<String> = final_set.into_iter().collect();
        if final_providers.is_empty() {
            return Err("no providers available to add".to_string());
        }

        let (tpl_providers, content) = catalog::fetch_template(&final_providers)?;
        let block = manager::build_managed_block(
            &final_providers,
            &managed_block_metadata(&final_providers),
            &content,
            std::slice::from_ref(&self.config.defaults.ignore_rules),
        );
        let (action, diff_text) = apply_managed_block(&self.manager, &block, dry_run, diff)?;
        Ok(CommandResult {
            command: "add".to_string(),
            cwd: self.cwd.to_string_lossy().to_string(),
            detected_providers: Vec::new(),
            included_providers: Vec::new(),
            excluded_providers: Vec::new(),
            added_providers: added,
            final_providers,
            unsupported_key_warnings: warnings,
            detection_results: Vec::new(),
            file_action: action.as_str().to_string(),
            preview_only: diff,
            diff: diff_text,
            template_provider_count: tpl_providers.len(),
        })
    }

    pub fn doctor(&self, include: &[String], exclude: &[String]) -> Result<DoctorResult, String> {
        self.git_gate("doctor")?;
        let selection = self.resolve_selection(include, exclude)?;
        let detections = selection
            .detection_results
            .iter()
            .map(|r| DoctorDetection {
                key: r.key.clone(),
                matched: r.matched,
                origin: detection_origin(r),
                reason: r.reason.clone(),
                evidence: r.evidence.clone(),
                error: r.error.clone(),
            })
            .collect();
        Ok(DoctorResult {
            command: "doctor".to_string(),
            cwd: self.cwd.to_string_lossy().to_string(),
            detected_providers: selection.detected_providers,
            included_providers: selection.included_providers,
            excluded_providers: selection.excluded_providers,
            final_providers: selection.final_providers.clone(),
            unsupported_key_warnings: selection.warnings,
            detections,
            runtime: self.doctor_runtime(&selection.final_providers),
            provenance: managed_block_metadata(&selection.final_providers),
        })
    }

    fn resolve_selection(
        &self,
        include_input: &[String],
        exclude_input: &[String],
    ) -> Result<ResolvedSelection, String> {
        let default_include;
        let include_input = if include_input.is_empty() {
            default_include = self.config.defaults.providers.clone();
            &default_include
        } else {
            include_input
        };

        let (include, include_warnings) = self.sanitize_keys(include_input);
        let (exclude, exclude_warnings) = self.sanitize_keys(exclude_input);
        let mut warnings = include_warnings;
        warnings.extend(exclude_warnings);
        warnings.sort();

        let ctx = ScanCtx::new(&self.cwd);
        let (detected, detection_results) = scan_target(&ctx, &self.detectors);
        let detected_providers: Vec<String> = detected
            .into_iter()
            .filter(|key| self.supported.contains(key))
            .collect();

        let mut final_set: BTreeSet<String> = detected_providers.iter().cloned().collect();
        for key in &include {
            final_set.insert(key.clone());
        }
        for key in &exclude {
            final_set.remove(key);
        }
        let final_providers: Vec<String> = final_set.into_iter().collect();
        if final_providers.is_empty() {
            return Err("no providers selected after include/exclude".to_string());
        }

        Ok(ResolvedSelection {
            detected_providers,
            included_providers: include,
            excluded_providers: exclude,
            final_providers,
            warnings,
            detection_results,
        })
    }

    fn sanitize_keys(&self, keys: &[String]) -> (Vec<String>, Vec<String>) {
        let mut set = BTreeSet::new();
        let mut warnings = Vec::new();
        for key in keys {
            if key.is_empty() {
                continue;
            }
            if !self.supported.contains(key) {
                warnings.push(format!("unsupported provider key: {}", key));
                continue;
            }
            set.insert(key.clone());
        }
        warnings.sort();
        (set.into_iter().collect(), warnings)
    }

    fn doctor_runtime(&self, providers: &[String]) -> DoctorRuntime {
        let mut selected = providers.to_vec();
        selected.sort();
        let retained: Vec<String> = selected
            .iter()
            .filter(|key| catalog::has_custom_provider(key))
            .cloned()
            .collect();

        let rule_entries = rule_catalog_count();
        let (rule_catalog_status, rule_catalog_provider_count) = match rule_entries {
            Ok(count) => ("loaded".to_string(), count),
            Err(err) => (err, 0),
        };

        let embedded_provider_count = catalog::remote_supported_keys().len();
        let mut decisions = vec![format!(
            "provider support is validated against the embedded github/gitignore catalog snapshot ({} providers)",
            embedded_provider_count
        )];
        if rule_catalog_status == "loaded" {
            decisions.push(format!(
                "repository detection is backed by the embedded JSON rule catalog ({} providers)",
                rule_catalog_provider_count
            ));
        } else {
            decisions.push(
                "repository detection rules are unavailable because the embedded JSON rule catalog failed to load"
                    .to_string(),
            );
        }
        if !retained.is_empty() {
            decisions.push(format!(
                "retained embedded custom providers: {}",
                retained.join(", ")
            ));
        }

        DoctorRuntime {
            embedded_provider_count,
            selected_providers: selected,
            rule_catalog_status,
            rule_catalog_provider_count,
            retained_custom_providers: retained,
            decisions,
        }
    }
}

fn rule_catalog_count() -> Result<usize, String> {
    static COUNT: OnceLock<Result<usize, String>> = OnceLock::new();
    COUNT
        .get_or_init(|| {
            let supported: HashSet<String> = catalog::supported_keys()
                .map_err(|e| e.clone())?
                .into_iter()
                .collect();
            genignore_detection::load_rule_catalog(|key| supported.contains(key))
                .map(|entries| entries.len())
                .map_err(|e| e.to_string())
        })
        .clone()
}

struct ResolvedSelection {
    detected_providers: Vec<String>,
    included_providers: Vec<String>,
    excluded_providers: Vec<String>,
    final_providers: Vec<String>,
    warnings: Vec<String>,
    detection_results: Vec<DetectionResult>,
}

fn managed_block_metadata(providers: &[String]) -> Vec<String> {
    let mut remote = Vec::new();
    let mut embedded = Vec::new();
    for key in providers {
        if catalog::has_custom_provider(key) {
            embedded.push(key.clone());
        } else {
            remote.push(key.clone());
        }
    }
    remote.sort();
    embedded.sort();

    let mut parts = Vec::new();
    if !remote.is_empty() {
        parts.push(format!(
            "github/gitignore@{} [{}]",
            catalog::DEFAULT_UPSTREAM_COMMIT,
            remote.join(",")
        ));
    }
    if !embedded.is_empty() {
        parts.push(format!("embedded [{}]", embedded.join(",")));
    }
    if parts.is_empty() {
        return Vec::new();
    }
    vec![format!("# Provenance: {}", parts.join("; "))]
}

fn detection_origin(result: &DetectionResult) -> String {
    let reason = &result.reason;
    if reason.contains("runtime OS")
        || reason.contains("installed application")
        || reason.contains("application not found")
    {
        "host".to_string()
    } else if reason.contains("jetbrains install") {
        "repository+host".to_string()
    } else {
        "repository".to_string()
    }
}

fn apply_managed_block(
    manager: &Manager,
    block: &str,
    dry_run: bool,
    preview_only: bool,
) -> Result<(FileAction, String), String> {
    if preview_only {
        let preview = manager.preview_managed_block(block)?;
        return Ok((preview.action, preview.diff));
    }
    if dry_run {
        let action = manager.upsert_managed_block(block, true)?;
        return Ok((action, String::new()));
    }
    let preview = manager.preview_managed_block(block)?;
    if preview.action == FileAction::NoOp {
        return Ok((FileAction::NoOp, preview.diff));
    }
    let action = manager.upsert_managed_block(block, false)?;
    Ok((action, preview.diff))
}

/// list/search catalog command (no Git gate — works anywhere).
pub fn catalog_result(command: &str, query: &str) -> Result<CatalogResult, String> {
    let supported = catalog::supported_keys()
        .map_err(|err| format!("initialize provider registry: {}", err))?;
    let providers = if command == "search" {
        catalog::search_providers(query, &supported)
    } else {
        supported
    };
    Ok(CatalogResult {
        command: command.to_string(),
        query: query.to_string(),
        providers,
    })
}
