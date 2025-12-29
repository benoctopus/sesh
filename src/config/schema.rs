use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(default = "default_workspace")]
    pub workspace: WorkspaceConfig,
    
    #[serde(default = "default_session")]
    pub session: SessionConfig,
    
    #[serde(default = "default_picker")]
    pub picker: PickerConfig,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            workspace: default_workspace(),
            session: default_session(),
            picker: default_picker(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkspaceConfig {
    /// Base directory for cloned projects
    #[serde(default = "default_projects_dir")]
    pub projects_dir: String,
    
    /// Base directory for worktrees
    #[serde(default = "default_worktrees_dir")]
    pub worktrees_dir: String,
}

fn default_workspace() -> WorkspaceConfig {
    WorkspaceConfig {
        projects_dir: default_projects_dir(),
        worktrees_dir: default_worktrees_dir(),
    }
}

fn default_projects_dir() -> String {
    "~/.sesh/projects".to_string()
}

fn default_worktrees_dir() -> String {
    "~/.sesh/worktrees".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionConfig {
    /// Backend: "tmux", "code", "cursor"
    #[serde(default = "default_backend")]
    pub backend: String,
    
    /// Startup command to run in new sessions
    #[serde(default)]
    pub startup_command: Option<String>,
}

fn default_session() -> SessionConfig {
    SessionConfig {
        backend: default_backend(),
        startup_command: None,
    }
}

fn default_backend() -> String {
    "tmux".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PickerConfig {
    /// Fuzzy finder: "auto", "fzf", "skim"
    #[serde(default = "default_finder")]
    pub finder: String,
}

fn default_picker() -> PickerConfig {
    PickerConfig {
        finder: default_finder(),
    }
}

fn default_finder() -> String {
    "auto".to_string()
}

impl Config {
    /// Expand tilde in paths and return absolute PathBuf
    pub fn projects_dir(&self) -> PathBuf {
        expand_tilde(&self.workspace.projects_dir)
    }
    
    pub fn worktrees_dir(&self) -> PathBuf {
        expand_tilde(&self.workspace.worktrees_dir)
    }
}

fn expand_tilde(path: &str) -> PathBuf {
    if path.starts_with("~/") {
        if let Some(home) = dirs::home_dir() {
            return home.join(&path[2..]);
        }
    }
    PathBuf::from(path)
}

