use crate::error::Result;
use async_trait::async_trait;
use std::path::Path;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BackendKind {
    Tmux,
    Code,
    Cursor,
}

impl BackendKind {
    pub fn from_str(s: &str) -> Option<Self> {
        match s {
            "tmux" => Some(BackendKind::Tmux),
            "code" => Some(BackendKind::Code),
            "cursor" => Some(BackendKind::Cursor),
            _ => None,
        }
    }
    
    pub fn as_str(&self) -> &'static str {
        match self {
            BackendKind::Tmux => "tmux",
            BackendKind::Code => "code",
            BackendKind::Cursor => "cursor",
        }
    }
}

#[async_trait]
pub trait SessionBackend: Send + Sync {
    /// Get the backend kind
    fn kind(&self) -> BackendKind;
    
    /// Create a new session at the given working directory
    async fn create(&self, name: &str, working_dir: &Path) -> Result<()>;
    
    /// Attach to an existing session (replaces current process for tmux)
    async fn attach(&self, name: &str) -> Result<()>;
    
    /// Switch to a session (when already inside a session)
    async fn switch(&self, name: &str) -> Result<()>;
    
    /// Delete/kill a session
    async fn delete(&self, name: &str) -> Result<()>;
    
    /// Check if a session exists and is active
    async fn exists(&self, name: &str) -> Result<bool>;
    
    /// List all active sessions managed by this backend
    async fn list_active(&self) -> Result<Vec<String>>;
    
    /// Check if we're currently inside a session of this backend
    fn is_inside_session(&self) -> bool;
    
    /// Get the current session name if inside one
    fn current_session(&self) -> Option<String>;
}

/// Factory function to create backend from config
pub fn create_backend(kind: BackendKind) -> Box<dyn SessionBackend> {
    match kind {
        BackendKind::Tmux => Box::new(crate::backends::tmux::TmuxBackend::new()),
        BackendKind::Code => Box::new(crate::backends::editor::EditorBackend::new("code")),
        BackendKind::Cursor => Box::new(crate::backends::editor::EditorBackend::new("cursor")),
    }
}

