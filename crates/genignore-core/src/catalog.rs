//! Embedded provider catalog: the github/gitignore snapshot table generated
//! by build.rs plus the checked-in custom templates.

use std::collections::HashSet;
use std::sync::OnceLock;

include!(concat!(env!("OUT_DIR"), "/templates.rs"));

pub const DEFAULT_UPSTREAM_COMMIT: &str = "dcc0fc7bc2b5ba480cf117ad1be31bafceeaff46";

const CUSTOM_TEMPLATES: &[(&str, &str)] = &[
    (
        "ai-agents",
        include_str!("../../../internal/customtemplate/templates/ai-agents.gitignore"),
    ),
    (
        "wrangler",
        include_str!("../../../internal/customtemplate/templates/wrangler.gitignore"),
    ),
];

fn remote_keys() -> &'static [&'static str] {
    static KEYS: OnceLock<Vec<&'static str>> = OnceLock::new();
    KEYS.get_or_init(|| TEMPLATES.iter().map(|(k, _)| *k).collect())
}

/// Remote (upstream snapshot) provider keys, sorted.
pub fn remote_supported_keys() -> Vec<String> {
    remote_keys().iter().map(|s| s.to_string()).collect()
}

pub fn has_remote_provider(key: &str) -> bool {
    remote_keys().binary_search(&key).is_ok()
}

pub fn has_custom_provider(key: &str) -> bool {
    CUSTOM_TEMPLATES.iter().any(|(k, _)| *k == key)
}

pub fn custom_provider_keys() -> Vec<String> {
    let mut keys: Vec<String> = CUSTOM_TEMPLATES
        .iter()
        .map(|(k, _)| k.to_string())
        .collect();
    keys.sort();
    keys
}

pub fn custom_content(key: &str) -> Option<String> {
    CUSTOM_TEMPLATES
        .iter()
        .find(|(k, _)| *k == key)
        .map(|(_, v)| v.trim_matches('\n').to_string())
}

pub fn remote_content(key: &str) -> Option<String> {
    TEMPLATES
        .iter()
        .find(|(k, _)| *k == key)
        .map(|(_, v)| v.to_string())
}

/// Content for a provider key — custom templates take precedence over remote
/// (they can never collide in practice; a collision is an init error).
pub fn content_for_provider(key: &str) -> Result<String, String> {
    if let Some(content) = custom_content(key) {
        return Ok(content);
    }
    remote_content(key).ok_or_else(|| format!("embedded upstream template not found: {}", key))
}

/// All supported keys (remote + custom), sorted. Mirrors the Go init-time
/// collision check between the two catalogs.
pub fn supported_keys() -> Result<Vec<String>, String> {
    let remote = remote_keys();
    let remote_set: HashSet<&str> = remote.iter().copied().collect();
    let mut keys: Vec<String> = remote.iter().map(|s| s.to_string()).collect();
    for (key, _) in CUSTOM_TEMPLATES {
        if remote_set.contains(*key) {
            return Err(format!(
                "embedded custom template key collides with remote provider key: {}",
                key
            ));
        }
        keys.push((*key).to_string());
    }
    keys.sort();
    keys.dedup();
    Ok(keys)
}

pub fn is_supported(keys: &HashSet<String>, key: &str) -> bool {
    keys.contains(key)
}

/// Equivalent of api.Client.FetchTemplate: join per-provider content with a
/// blank line, skipping whitespace-only bodies, in requested order.
pub fn fetch_template(providers: &[String]) -> Result<(Vec<String>, String), String> {
    if providers.is_empty() {
        return Err("providers must not be empty".to_string());
    }
    let mut parts = Vec::new();
    for key in providers {
        let content = content_for_provider(key)?;
        if content.trim().is_empty() {
            continue;
        }
        parts.push(content);
    }
    Ok((providers.to_vec(), parts.join("\n\n")))
}

/// list/search over the supported key set.
pub fn search_providers(term: &str, supported: &[String]) -> Vec<String> {
    let needle = term.to_lowercase();
    let mut filtered: Vec<String> = supported
        .iter()
        .filter(|k| k.to_lowercase().contains(&needle))
        .cloned()
        .collect();
    filtered.sort();
    filtered
}
