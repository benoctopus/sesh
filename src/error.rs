use std::path::PathBuf;
use thiserror::Error;

pub type Result<T> = std::result::Result<T, Error>;

#[derive(Debug, Error)]
pub enum Error {
    #[error("Project not found: {name}. List available projects with: sesh list --projects")]
    ProjectNotFound { name: String },
    
    #[error("Worktree path no longer exists: {path}. Remove the stale entry with: sesh clean --stale")]
    WorktreeStale { path: PathBuf },
    
    #[error("Branch '{branch}' not found locally or on remote. Create a new branch with: sesh switch --create {branch}")]
    BranchNotFound { branch: String },
    
    #[error("Session backend '{backend}' is not available. {install_hint}. Or change backend in ~/.config/sesh/config.toml")]
    BackendUnavailable { backend: String, install_hint: String },
    
    #[error("No fuzzy finder available for interactive selection. Install skim (recommended): cargo install skim, or install fzf: brew install fzf, or specify the branch directly: sesh switch <branch>")]
    NoPickerAvailable,
    
    #[error("Git operation failed: {message}")]
    GitError { message: String },
    
    #[error("Database error: {0}")]
    DatabaseError(#[from] sqlx::Error),
    
    #[error("Database corrupted: {path}. {suggestion}")]
    DatabaseCorrupted { path: PathBuf, suggestion: String },
    
    #[error("Failed to open database: {path}")]
    DatabaseOpen { path: PathBuf, source: sqlx::Error },
    
    #[error("Migration failed: {0}")]
    MigrationFailed(#[from] sqlx::migrate::MigrateError),
    
    #[error("Configuration error: {0}")]
    ConfigError(String),
    
    #[error("IO error: {0}")]
    IoError(#[from] std::io::Error),
    
    #[error("Invalid path: {path}")]
    InvalidPath { path: PathBuf },
    
    #[error("Session '{name}' not found")]
    SessionNotFound { name: String },
    
    #[error("Worktree '{branch}' already exists for project")]
    WorktreeExists { branch: String },
    
    #[error("No previous session in history")]
    NoPreviousSession,
    
    #[error("Missing dependency: {name}. {install_hint}")]
    MissingDependency { name: String, install_hint: String },
    
    #[error("Invalid URL: {url}")]
    InvalidUrl { url: String },
    
    #[error("Repository already exists at: {path}")]
    RepositoryExists { path: PathBuf },
    
    #[error("Anyhow error: {0}")]
    AnyhowError(#[from] anyhow::Error),
}

