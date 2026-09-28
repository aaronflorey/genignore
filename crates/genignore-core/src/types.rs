//! JSON wire types — field names and order mirror `internal/app/types.go`
//! exactly so `--json` output stays byte-identical (modulo HTML escaping,
//! applied by the CLI layer like Go's encoding/json).

use genignore_detection::DetectionResult;
use serde::Serialize;

fn is_false(b: &bool) -> bool {
    !*b
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CommandResult {
    pub command: String,
    pub cwd: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detected_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub included_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub excluded_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub added_providers: Vec<String>,
    pub final_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub unsupported_key_warnings: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detection_results: Vec<DetectionResult>,
    pub file_action: String,
    #[serde(skip_serializing_if = "is_false")]
    pub preview_only: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub diff: String,
    pub template_provider_count: usize,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ResolveResult {
    pub command: String,
    pub cwd: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detected_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub included_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub excluded_providers: Vec<String>,
    pub final_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub unsupported_key_warnings: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detection_results: Vec<DetectionResult>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DoctorResult {
    pub command: String,
    pub cwd: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detected_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub included_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub excluded_providers: Vec<String>,
    pub final_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub unsupported_key_warnings: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub detections: Vec<DoctorDetection>,
    pub runtime: DoctorRuntime,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub provenance: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct DoctorDetection {
    pub key: String,
    pub matched: bool,
    pub origin: String,
    pub reason: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub evidence: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DoctorRuntime {
    pub embedded_provider_count: usize,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub selected_providers: Vec<String>,
    pub rule_catalog_status: String,
    pub rule_catalog_provider_count: usize,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub retained_custom_providers: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub decisions: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct CatalogResult {
    pub command: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub query: String,
    pub providers: Vec<String>,
}
