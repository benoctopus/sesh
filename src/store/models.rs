use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Project {
    pub id: i64,
    pub name: String,
    pub display_name: String,
    pub remote_url: String,
    pub clone_path: PathBuf,
    pub default_branch: String,
    pub created_at: String,
    pub last_fetched_at: Option<String>,
}

#[derive(Debug, Clone)]
pub struct CreateProject {
    pub name: String,
    pub display_name: String,
    pub remote_url: String,
    pub clone_path: PathBuf,
    pub default_branch: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Worktree {
    pub id: i64,
    pub project_id: i64,
    pub branch: String,
    pub path: PathBuf,
    pub is_primary: bool,
    pub created_at: String,
    pub last_accessed_at: String,
}

#[derive(Debug, Clone)]
pub struct CreateWorktree {
    pub project_id: i64,
    pub branch: String,
    pub path: PathBuf,
    pub is_primary: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub id: i64,
    pub worktree_id: i64,
    pub session_name: String,
    pub backend: String,
    pub created_at: String,
    pub last_attached_at: String,
}

#[derive(Debug, Clone)]
pub struct CreateSession {
    pub worktree_id: i64,
    pub session_name: String,
    pub backend: String,
}

