use anyhow::Result;
use tracing_subscriber::{fmt, prelude::*, EnvFilter};
use tracing_appender::rolling::{RollingFileAppender, Rotation};
use std::fs;

/// Initialize logging subsystem
/// - File logging: Always enabled, writes to ~/.config/sesh/logs/
/// - Terminal logging: Only with SESH_LOG env var or --verbose flag
pub fn init(verbose: bool) -> Result<()> {
    let log_dir = dirs::config_dir()
        .ok_or_else(|| anyhow::anyhow!("Could not determine config directory"))?
        .join("sesh")
        .join("logs");
    
    // Ensure log directory exists
    fs::create_dir_all(&log_dir)?;
    
    // Rolling file appender - new file daily, keep 7 days
    let file_appender = RollingFileAppender::new(
        Rotation::DAILY,
        &log_dir,
        "sesh.log",
    );
    
    // File layer - always debug level
    let file_layer = fmt::layer()
        .with_writer(file_appender)
        .with_ansi(false)
        .with_target(true)
        .with_thread_ids(true)
        .with_filter(EnvFilter::new("debug"));
    
    // Terminal layer - only if verbose or SESH_LOG is set
    let terminal_layer = if verbose || std::env::var("SESH_LOG").is_ok() {
        Some(fmt::layer()
            .with_writer(std::io::stderr)
            .with_target(false)
            .with_filter(EnvFilter::from_env("SESH_LOG")))
    } else {
        None
    };
    
    let registry = tracing_subscriber::registry()
        .with(file_layer);
    
    if let Some(terminal) = terminal_layer {
        registry.with(terminal).init();
    } else {
        registry.init();
    }
    
    Ok(())
}

/// Get the log directory path
pub fn log_dir() -> Result<std::path::PathBuf> {
    Ok(dirs::config_dir()
        .ok_or_else(|| anyhow::anyhow!("Could not determine config directory"))?
        .join("sesh")
        .join("logs"))
}

