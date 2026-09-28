//! genignore-core: embedded catalogs, config, managed-block engine, and
//! command services shared by the CLI.

pub mod catalog;
pub mod config;
pub mod manager;
pub mod service;
pub mod types;

pub use config::{load as load_config, Config};
pub use manager::{FileAction, Manager};
pub use service::{catalog_result, Service};
pub use types::{
    CatalogResult, CommandResult, DoctorDetection, DoctorResult, DoctorRuntime, ResolveResult,
};
