use crate::config;
use crate::error::{Error, Result};
use anyhow::Context;
use sha2::{Digest, Sha256};
use std::io::{self, Write};
use std::process::Command;

/// Get the user's preferred editor from environment variables
/// Priority: VISUAL > EDITOR > vi
fn get_editor() -> String {
    std::env::var("VISUAL")
        .or_else(|_| std::env::var("EDITOR"))
        .unwrap_or_else(|_| "vi".to_string())
}

/// Compute SHA256 hash of a file
fn hash_file(path: &std::path::Path) -> anyhow::Result<String> {
    let content = std::fs::read(path)
        .with_context(|| format!("Failed to read file for hashing: {}", path.display()))?;

    let mut hasher = Sha256::new();
    hasher.update(&content);
    let result = hasher.finalize();

    Ok(hex::encode(result))
}

/// Prompt user for a choice when validation fails
fn prompt_validation_failure() -> anyhow::Result<String> {
    println!("\nThe config file has errors. What would you like to do?");
    println!("  1. Edit again to fix errors");
    println!("  2. Discard changes (config will remain invalid until fixed)");
    println!("  3. Keep invalid config anyway (not recommended)");
    print!("\nChoice (1-3): ");
    io::stdout().flush()?;

    let mut input = String::new();
    io::stdin().read_line(&mut input)?;

    Ok(input.trim().to_string())
}

/// Run the edit command
pub async fn run() -> Result<()> {
    // Ensure config directory exists
    config::ensure_config_dir().map_err(|e| Error::ConfigError(e.to_string()))?;

    let config_path = config::config_path().map_err(|e| Error::ConfigError(e.to_string()))?;

    // Create config with defaults if it doesn't exist
    if !config_path.exists() {
        config::save_default().map_err(|e| Error::ConfigError(e.to_string()))?;
        println!("Created default config at: {}", config_path.display());
    }

    // Compute hash before editing
    let hash_before =
        hash_file(&config_path).context("Failed to hash config file before editing")?;

    // Open in editor
    let editor = get_editor();
    let status = Command::new(&editor)
        .arg(&config_path)
        .status()
        .with_context(|| format!("Failed to run editor: {}", editor))?;

    if !status.success() {
        return Err(Error::ConfigError(format!(
            "Editor exited with error status: {}",
            status
        )));
    }

    // Compute hash after editing
    let hash_after = hash_file(&config_path).context("Failed to hash config file after editing")?;

    // Check if file was modified
    if hash_before == hash_after {
        println!("No changes made to config");
        return Ok(());
    }

    // Validate the config
    loop {
        match config::load() {
            Ok(cfg) => {
                // Config parsed successfully, now validate values
                if let Err(e) = cfg.validate() {
                    eprintln!("\nConfig validation failed: {}", e);

                    // Check if we're in an interactive terminal
                    if !std::io::IsTerminal::is_terminal(&std::io::stdin()) {
                        return Err(Error::ConfigError(format!(
                            "Config validation failed in non-interactive mode. Please fix manually: {}",
                            config_path.display()
                        )));
                    }

                    let choice = prompt_validation_failure()?;

                    match choice.as_str() {
                        "1" => {
                            // Edit again
                            let status = Command::new(&editor)
                                .arg(&config_path)
                                .status()
                                .with_context(|| format!("Failed to run editor: {}", editor))?;

                            if !status.success() {
                                return Err(Error::ConfigError(format!(
                                    "Editor exited with error status: {}",
                                    status
                                )));
                            }
                            // Continue the loop to validate again
                            continue;
                        }
                        "2" => {
                            // Discard changes - user needs to manually fix
                            println!(
                                "\nPlease manually fix the config file or delete it to start over."
                            );
                            return Err(Error::ConfigError("Config validation failed".to_string()));
                        }
                        "3" => {
                            // Save anyway
                            println!(
                                "\nWarning: Config file contains errors. This may cause issues."
                            );
                            return Ok(());
                        }
                        _ => {
                            return Err(Error::ConfigError("Invalid choice".to_string()));
                        }
                    }
                } else {
                    // Validation succeeded
                    break;
                }
            }
            Err(e) => {
                eprintln!("\nConfig parsing failed: {}", e);

                // Check if we're in an interactive terminal
                if !std::io::IsTerminal::is_terminal(&std::io::stdin()) {
                    return Err(Error::ConfigError(format!(
                        "Config parsing failed in non-interactive mode. Please fix manually: {}",
                        config_path.display()
                    )));
                }

                let choice = prompt_validation_failure()?;

                match choice.as_str() {
                    "1" => {
                        // Edit again
                        let status = Command::new(&editor)
                            .arg(&config_path)
                            .status()
                            .with_context(|| format!("Failed to run editor: {}", editor))?;

                        if !status.success() {
                            return Err(Error::ConfigError(format!(
                                "Editor exited with error status: {}",
                                status
                            )));
                        }
                        // Continue the loop to validate again
                        continue;
                    }
                    "2" => {
                        // Discard changes - user needs to manually fix
                        println!(
                            "\nPlease manually fix the config file or delete it to start over."
                        );
                        return Err(Error::ConfigError("Config parsing failed".to_string()));
                    }
                    "3" => {
                        // Save anyway
                        println!("\nWarning: Config file contains errors. This may cause issues.");
                        return Ok(());
                    }
                    _ => {
                        return Err(Error::ConfigError("Invalid choice".to_string()));
                    }
                }
            }
        }
    }

    println!(
        "Config saved and validated successfully: {}",
        config_path.display()
    );
    Ok(())
}
