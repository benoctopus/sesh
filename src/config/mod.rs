pub mod schema;

pub use schema::Config;

use anyhow::{Context, Result};
use std::path::PathBuf;

pub fn load() -> Result<Config> {
    let config_path = config_path()?;

    if config_path.exists() {
        let content = std::fs::read_to_string(&config_path)
            .with_context(|| format!("Failed to read config from {}", config_path.display()))?;
        toml::from_str(&content)
            .with_context(|| format!("Failed to parse config from {}", config_path.display()))
    } else {
        // Return default config
        Ok(Config::default())
    }
}

pub fn config_path() -> Result<PathBuf> {
    Ok(config_dir()?.join("config.toml"))
}

pub fn ensure_config_dir() -> Result<PathBuf> {
    let config_dir = config_dir()?;
    std::fs::create_dir_all(&config_dir).with_context(|| {
        format!(
            "Failed to create config directory: {}",
            config_dir.display()
        )
    })?;
    Ok(config_dir)
}

fn config_dir() -> Result<PathBuf> {
    // Check for override from environment (set by --config-dir flag)
    if let Ok(override_dir) = std::env::var("SESH_CONFIG_DIR_OVERRIDE") {
        return Ok(PathBuf::from(override_dir));
    }

    Ok(dirs::config_dir()
        .ok_or_else(|| anyhow::anyhow!("Could not determine config directory"))?
        .join("sesh"))
}

/// Save configuration to disk
pub fn save(config: &Config) -> Result<()> {
    let config_path = config_path()?;
    ensure_config_dir()?;
    
    let content = toml::to_string_pretty(config)
        .context("Failed to serialize config to TOML")?;
    
    std::fs::write(&config_path, content)
        .with_context(|| format!("Failed to write config to {}", config_path.display()))?;
    
    Ok(())
}

/// Create config file with default values if it doesn't exist
pub fn save_default() -> Result<()> {
    let config_path = config_path()?;
    
    if !config_path.exists() {
        let default_config = Config::default();
        save(&default_config)?;
    }
    
    Ok(())
}
