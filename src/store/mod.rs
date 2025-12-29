pub mod models;
pub mod queries;

use crate::config;
use crate::error::{Error, Result};
use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};
use std::path::PathBuf;
use std::sync::Arc;

#[derive(Clone)]
pub struct Store {
    pool: Arc<SqlitePool>,
}

impl Store {
    pub async fn open() -> Result<Self> {
        let db_path = db_path()?;
        
        // Ensure parent directory exists
        if let Some(parent) = db_path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        let db_exists = db_path.exists();
        
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect(&format!("sqlite:{}", db_path.display()))
            .await
            .map_err(|e| {
                if db_exists {
                    Error::DatabaseCorrupted {
                        path: db_path.clone(),
                        suggestion: format!(
                            "Try: mv {} {}.bak && sesh list",
                            db_path.display(),
                            db_path.display()
                        ),
                    }
                } else {
                    Error::DatabaseOpen {
                        path: db_path,
                        source: e,
                    }
                }
            })?;
        
        // Run migrations
        sqlx::migrate!("./migrations")
            .run(&pool)
            .await?;
        
        Ok(Self { pool: Arc::new(pool) })
    }
    
    pub fn pool(&self) -> &SqlitePool {
        &self.pool
    }
}

fn db_path() -> Result<PathBuf> {
    Ok(config::ensure_config_dir()?.join("sesh.db"))
}

// Re-export query functions and models for convenience
pub use queries::*;
pub use models::{CreateProject, CreateWorktree, CreateSession};

