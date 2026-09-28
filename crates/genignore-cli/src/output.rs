//! Human-readable and JSON output, byte-faithful to `internal/app/cli.go`.

use genignore_core::types::{
    CatalogResult, CommandResult, DoctorDetection, DoctorResult, ResolveResult,
};
use genignore_detection::DetectionResult;

/// Serialize like Go `json.MarshalIndent(v, "", "  ")`: two-space indent plus
/// HTML escaping of `<`, `>`, `&` (and U+2028/9) inside strings.
pub fn to_json<T: serde::Serialize>(value: &T) -> String {
    let raw = serde_json::to_string_pretty(value).unwrap_or_default();
    escape_html_json(&raw)
}

fn escape_html_json(input: &str) -> String {
    let mut out = String::with_capacity(input.len());
    let mut in_string = false;
    let mut escaped = false;
    for ch in input.chars() {
        if in_string {
            match ch {
                '\\' if !escaped => {
                    escaped = true;
                    out.push(ch);
                    continue;
                }
                '"' if !escaped => in_string = false,
                '<' if !escaped => {
                    out.push_str("\\u003c");
                    continue;
                }
                '>' if !escaped => {
                    out.push_str("\\u003e");
                    continue;
                }
                '&' if !escaped => {
                    out.push_str("\\u0026");
                    continue;
                }
                '\u{2028}' if !escaped => {
                    out.push_str("\\u2028");
                    continue;
                }
                '\u{2029}' if !escaped => {
                    out.push_str("\\u2029");
                    continue;
                }
                _ => {}
            }
            escaped = false;
        } else if ch == '"' {
            in_string = true;
        }
        out.push(ch);
    }
    out
}

fn format_provider_list(providers: &[String]) -> String {
    let mut sorted = providers.to_vec();
    sorted.sort();
    sorted.join(", ")
}

fn format_detection_result(result: &DetectionResult) -> String {
    let status = if result.matched {
        "matched"
    } else if !result.error.is_empty() {
        "error"
    } else {
        "skipped"
    };
    let mut parts = vec![
        result.key.clone(),
        status.to_string(),
        result.reason.clone(),
    ];
    if !result.evidence.is_empty() {
        parts.push(result.evidence.clone());
    }
    if !result.error.is_empty() {
        parts.push(result.error.clone());
    }
    parts.join(" | ")
}

fn format_doctor_detection(result: &DoctorDetection) -> String {
    let status = if result.matched {
        "matched"
    } else if !result.error.is_empty() {
        "error"
    } else {
        "skipped"
    };
    let mut parts = vec![
        result.key.clone(),
        result.origin.clone(),
        status.to_string(),
        result.reason.clone(),
    ];
    if !result.evidence.is_empty() {
        parts.push(result.evidence.clone());
    }
    if !result.error.is_empty() {
        parts.push(result.error.clone());
    }
    parts.join(" | ")
}

pub fn print_catalog_result(result: &CatalogResult, json_output: bool) {
    if json_output {
        println!("{}", to_json(result));
        return;
    }
    println!("Command: {}", result.command);
    if !result.query.is_empty() {
        println!("Query: {}", result.query);
    }
    println!("Providers:");
    for key in &result.providers {
        println!("{}", key);
    }
}

pub fn print_resolve_result(result: &ResolveResult, json_output: bool, verbose: bool) {
    if json_output {
        println!("{}", to_json(result));
        return;
    }
    println!("Command: {}", result.command);
    if !result.detected_providers.is_empty() {
        println!(
            "Detected: {}",
            format_provider_list(&result.detected_providers)
        );
    }
    if !result.included_providers.is_empty() {
        println!(
            "Included: {}",
            format_provider_list(&result.included_providers)
        );
    }
    if !result.excluded_providers.is_empty() {
        println!(
            "Excluded: {}",
            format_provider_list(&result.excluded_providers)
        );
    }
    println!("Final: {}", format_provider_list(&result.final_providers));
    for warning in &result.unsupported_key_warnings {
        println!("Warning: {}", warning);
    }
    if verbose {
        for detection in &result.detection_results {
            if !detection.matched && detection.error.is_empty() {
                continue;
            }
            println!("Detection: {}", format_detection_result(detection));
        }
    }
}

pub fn print_result(result: &CommandResult, json_output: bool, verbose: bool) {
    if json_output {
        println!("{}", to_json(result));
        return;
    }
    println!("Command: {}", result.command);
    if !result.detected_providers.is_empty() {
        println!(
            "Detected: {}",
            format_provider_list(&result.detected_providers)
        );
    }
    if !result.added_providers.is_empty() {
        println!("Added: {}", format_provider_list(&result.added_providers));
    }
    if !result.included_providers.is_empty() {
        println!(
            "Included: {}",
            format_provider_list(&result.included_providers)
        );
    }
    if !result.excluded_providers.is_empty() {
        println!(
            "Excluded: {}",
            format_provider_list(&result.excluded_providers)
        );
    }
    println!("Final: {}", format_provider_list(&result.final_providers));
    for warning in &result.unsupported_key_warnings {
        println!("Warning: {}", warning);
    }
    if verbose {
        for detection in &result.detection_results {
            if !detection.matched && detection.error.is_empty() {
                continue;
            }
            println!("Detection: {}", format_detection_result(detection));
        }
    }
    if !result.file_action.is_empty() {
        println!("File: {}", result.file_action);
    }
    if result.preview_only {
        println!("Preview: diff-only (no file written)");
    }
    if result.preview_only && !result.diff.is_empty() {
        println!("Diff:\n{}", result.diff);
    }
}

pub fn print_doctor_result(result: &DoctorResult, json_output: bool) {
    if json_output {
        println!("{}", to_json(result));
        return;
    }
    println!("Command: {}", result.command);
    if !result.detected_providers.is_empty() {
        println!(
            "Detected: {}",
            format_provider_list(&result.detected_providers)
        );
    }
    if !result.included_providers.is_empty() {
        println!(
            "Included: {}",
            format_provider_list(&result.included_providers)
        );
    }
    if !result.excluded_providers.is_empty() {
        println!(
            "Excluded: {}",
            format_provider_list(&result.excluded_providers)
        );
    }
    println!("Final: {}", format_provider_list(&result.final_providers));
    for warning in &result.unsupported_key_warnings {
        println!("Warning: {}", warning);
    }
    for detection in &result.detections {
        if !detection.matched && detection.error.is_empty() {
            continue;
        }
        println!("Detection: {}", format_doctor_detection(detection));
    }
    println!(
        "Embedded catalog providers: {}",
        result.runtime.embedded_provider_count
    );
    println!(
        "Selected providers: {}",
        format_provider_list(&result.runtime.selected_providers)
    );
    println!(
        "Rule catalog: {} ({} providers)",
        result.runtime.rule_catalog_status, result.runtime.rule_catalog_provider_count
    );
    if !result.runtime.retained_custom_providers.is_empty() {
        println!(
            "Retained custom providers: {}",
            format_provider_list(&result.runtime.retained_custom_providers)
        );
    }
    for decision in &result.runtime.decisions {
        println!("Decision: {}", decision);
    }
    for line in &result.provenance {
        println!(
            "Provenance: {}",
            line.strip_prefix("# Provenance: ").unwrap_or(line)
        );
    }
}
