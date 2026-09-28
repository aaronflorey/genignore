//! Machine-level config at `$HOME/.config/genignore/config.toml`, strict TOML
//! decode (unknown fields are rejected, matching the Go contract).

use std::fs;
use std::io;
use std::path::PathBuf;

use serde::Deserialize;

pub const CONFIG_RELATIVE_PATH: &str = ".config/genignore/config.toml";

#[derive(Debug, Default, Clone)]
pub struct Config {
    pub defaults: ConfigDefaults,
}

#[derive(Debug, Default, Clone)]
pub struct ConfigDefaults {
    pub providers: Vec<String>,
    pub ignore_rules: Vec<String>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawConfig {
    #[serde(default)]
    defaults: RawDefaults,
}

#[derive(Deserialize, Default)]
#[serde(deny_unknown_fields)]
struct RawDefaults {
    #[serde(default)]
    providers: Vec<String>,
    #[serde(default)]
    ignore_rules: Vec<String>,
}

fn config_path() -> Result<PathBuf, String> {
    let home = std::env::var_os("HOME")
        .or_else(|| std::env::var_os("USERPROFILE"))
        .map(PathBuf::from)
        .ok_or_else(|| "resolve config home: environment variable not set".to_string())?;
    Ok(home.join(CONFIG_RELATIVE_PATH))
}

pub fn load() -> Result<Config, String> {
    let path = config_path()?;
    match fs::metadata(&path) {
        Ok(_) => {}
        Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(Config::default()),
        Err(err) => return Err(format!("stat config file {}: {}", path.display(), err)),
    }
    let content = fs::read_to_string(&path)
        .map_err(|err| format!("read config file {}: {}", path.display(), err))?;
    let raw: RawConfig = toml::from_str(&content)
        .map_err(|err| format!("invalid config file {}: {}", path.display(), err))?;
    Ok(Config {
        defaults: ConfigDefaults {
            providers: raw.defaults.providers,
            ignore_rules: raw.defaults.ignore_rules,
        },
    })
}
