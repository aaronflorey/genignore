//! Strict loader for the embedded JSON rule catalog. Ports
//! `internal/rulecatalog` validation semantics; the supported-provider set is
//! supplied by the caller so this crate stays independent of core.

use std::collections::HashSet;

use serde::Deserialize;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum RuleType {
    FilePath,
    FileContentLine,
}

impl RuleType {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::FilePath => "file_path",
            Self::FileContentLine => "file_content_line",
        }
    }
}

#[derive(Debug, Clone)]
pub struct Rule {
    pub rule_type: RuleType,
    pub path: String,
    pub contains: String,
}

#[derive(Debug, Clone)]
pub struct Entry {
    pub provider: String,
    pub matches: Vec<Rule>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawCatalog {
    #[serde(rename = "$schema", default)]
    #[allow(dead_code)]
    schema: Option<String>,
    providers: Option<Vec<RawEntry>>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawEntry {
    provider: Option<String>,
    #[serde(rename = "match")]
    matches: Option<Vec<RawRule>>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawRule {
    #[serde(rename = "type")]
    rule_type: Option<String>,
    path: Option<String>,
    contains: Option<String>,
}

/// Load and validate the embedded `rules.json` rule catalog against the
/// caller-supplied supported-provider set.
pub fn load_rule_catalog<F>(supported: F) -> Result<Vec<Entry>, String>
where
    F: Fn(&str) -> bool,
{
    const RULES_JSON: &str = include_str!("../../../internal/rulecatalog/rules.json");
    let raw = RULES_JSON;
    load_rule_bytes(raw, &supported)
        .map_err(|err| format!("decode rule catalog \"rules.json\": {}", err))
}

/// Strict loader for rule-catalog JSON text (exposed for tests; production
/// loading goes through `load_rule_catalog`).
pub fn load_rule_bytes(raw: &str, supported: &dyn Fn(&str) -> bool) -> Result<Vec<Entry>, String> {
    let mut de = serde_json::Deserializer::from_str(raw);
    let catalog: RawCatalog = match serde::de::Deserialize::deserialize(&mut de) {
        Ok(c) => c,
        Err(err) => return Err(err.to_string()),
    };
    // Second decode: a single JSON object only.
    if de.end().is_err() {
        return Err("expected a single JSON object".to_string());
    }

    let providers = catalog.providers.unwrap_or_default();
    if providers.is_empty() {
        return Err("providers must not be empty".to_string());
    }

    let mut entries = Vec::with_capacity(providers.len());
    let mut seen_providers: HashSet<String> = HashSet::new();
    for (entry_index, raw_entry) in providers.into_iter().enumerate() {
        let provider = raw_entry.provider.unwrap_or_default().trim().to_lowercase();
        if provider.is_empty() {
            return Err(format!("provider {} must not be empty", entry_index + 1));
        }
        if !supported(&provider) {
            return Err(format!(
                "provider {:?} is not available in the embedded upstream or custom template catalogs",
                provider
            ));
        }
        let raw_rules = raw_entry.matches.unwrap_or_default();
        if raw_rules.is_empty() {
            return Err(format!(
                "provider {:?} must define at least one match rule",
                provider
            ));
        }
        if !seen_providers.insert(provider.clone()) {
            return Err(format!("duplicate provider {:?}", provider));
        }

        let mut rules = Vec::with_capacity(raw_rules.len());
        let mut seen_rules: HashSet<String> = HashSet::new();
        for (rule_index, raw_rule) in raw_rules.into_iter().enumerate() {
            let rule = normalize_rule(&raw_rule).map_err(|err| {
                format!("provider {:?} rule {}: {}", provider, rule_index + 1, err)
            })?;
            let key = format!(
                "{}\0{}\0{}",
                rule.rule_type.as_str(),
                rule.path,
                rule.contains
            );
            if !seen_rules.insert(key.clone()) {
                return Err(format!(
                    "provider {:?} contains duplicate rule {:?}",
                    provider, key
                ));
            }
            rules.push(rule);
        }

        rules.sort_by(|a, b| {
            a.path
                .cmp(&b.path)
                .then_with(|| a.rule_type.as_str().cmp(b.rule_type.as_str()))
                .then_with(|| a.contains.cmp(&b.contains))
        });
        entries.push(Entry {
            provider,
            matches: rules,
        });
    }

    entries.sort_by(|a, b| a.provider.cmp(&b.provider));
    Ok(entries)
}

fn normalize_rule(raw: &RawRule) -> Result<Rule, String> {
    let path = raw.path.clone().unwrap_or_default();
    validate_rule_path(&path)?;

    match raw.rule_type.as_deref().unwrap_or("") {
        "file_path" => {
            if raw.contains.is_some() {
                return Err("file_path rules must not set contains".to_string());
            }
            Ok(Rule {
                rule_type: RuleType::FilePath,
                path,
                contains: String::new(),
            })
        }
        "file_content_line" => {
            let contains = raw.contains.clone().unwrap_or_default();
            if contains.trim().is_empty() {
                return Err(
                    "file_content_line rules must define a non-empty contains value".to_string(),
                );
            }
            Ok(Rule {
                rule_type: RuleType::FileContentLine,
                path,
                contains,
            })
        }
        other => {
            if other.trim().is_empty() {
                return Err("type must not be empty".to_string());
            }
            Err(format!("unsupported rule type {:?}", other))
        }
    }
}

fn validate_rule_path(rule_path: &str) -> Result<(), String> {
    if rule_path.trim().is_empty() {
        return Err("path must not be empty".to_string());
    }

    let normalized = rule_path.replace('\\', "/");
    let cleaned = clean_path(&normalized);

    if normalized.starts_with('/') || normalized.starts_with("//") {
        return Err("path must be project-relative".to_string());
    }
    if rule_path.len() >= 2 && rule_path.as_bytes()[1] == b':' {
        return Err("path must be project-relative".to_string());
    }
    for candidate in [&normalized, &cleaned] {
        for component in candidate.split('/') {
            if component == ".." {
                return Err("path must not contain parent traversal".to_string());
            }
        }
    }
    Ok(())
}

fn clean_path(p: &str) -> String {
    let mut out: Vec<&str> = Vec::new();
    for seg in p.split('/') {
        match seg {
            "" | "." => {}
            ".." => {
                if out.last().is_some_and(|last| *last != "..") {
                    out.pop();
                } else {
                    out.push("..");
                }
            }
            _ => out.push(seg),
        }
    }
    if out.is_empty() {
        ".".to_string()
    } else {
        out.join("/")
    }
}
