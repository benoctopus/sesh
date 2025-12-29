use crate::backends::traits::{BackendKind, SessionBackend};
use crate::error::{Error, Result};
use std::path::Path;
use std::process::Command;
use tracing::{debug, info};

pub struct EditorBackend {
    command: String,
    kind: BackendKind,
}

impl EditorBackend {
    pub fn new(command: &str) -> Self {
        let kind = match command {
            "code" => BackendKind::Code,
            "cursor" => BackendKind::Cursor,
            _ => BackendKind::Code,
        };
        
        Self {
            command: command.to_string(),
            kind,
        }
    }
    
    fn check_available(&self) -> Result<()> {
        let output = Command::new(&self.command)
            .arg("--version")
            .output()
            .map_err(|_| Error::BackendUnavailable {
                backend: self.command.clone(),
                install_hint: match self.kind {
                    BackendKind::Code => "Install VS Code and run 'Install code command in PATH'".to_string(),
                    BackendKind::Cursor => "Install Cursor from https://cursor.sh/".to_string(),
                    _ => "Install the editor".to_string(),
                },
            })?;
        
        if !output.status.success() {
            return Err(Error::BackendUnavailable {
                backend: self.command.clone(),
                install_hint: match self.kind {
                    BackendKind::Code => "Install VS Code and run 'Install code command in PATH'".to_string(),
                    BackendKind::Cursor => "Install Cursor from https://cursor.sh/".to_string(),
                    _ => "Install the editor".to_string(),
                },
            });
        }
        
        Ok(())
    }
}

#[async_trait::async_trait]
impl SessionBackend for EditorBackend {
    fn kind(&self) -> BackendKind {
        self.kind
    }
    
    async fn create(&self, name: &str, working_dir: &Path) -> Result<()> {
        debug!(session = %name, path = %working_dir.display(), "Creating editor workspace");
        
        // For editors, "create" means opening the workspace
        // We use the session name as a workspace identifier
        self.attach(name).await
    }
    
    async fn attach(&self, name: &str) -> Result<()> {
        info!(session = %name, "Opening editor workspace");
        
        // For editors, we need to get the path from the session name
        // This will be handled by the session manager
        // For now, we'll just check availability
        self.check_available()?;
        
        // The actual opening will be done by the session manager
        // which has access to the worktree path
        Ok(())
    }
    
    async fn switch(&self, name: &str) -> Result<()> {
        debug!(session = %name, "Switching to editor workspace");
        
        // For editors, switching means opening a new window/workspace
        // Similar to attach
        self.attach(name).await
    }
    
    async fn delete(&self, _name: &str) -> Result<()> {
        // Editors don't have a concept of "deleting" a workspace
        // The workspace file/directory can be deleted, but that's handled elsewhere
        debug!("Editor workspaces are not 'deleted', they just stop being used");
        Ok(())
    }
    
    async fn exists(&self, _name: &str) -> Result<bool> {
        // For editors, we can't really check if a "session" exists
        // We just check if the editor is available
        self.check_available().map(|_| true)
    }
    
    async fn list_active(&self) -> Result<Vec<String>> {
        // Editors don't have a simple way to list active workspaces
        // This would require parsing editor state files
        // For now, return empty list
        Ok(Vec::new())
    }
    
    fn is_inside_session(&self) -> bool {
        // Check if we're inside an editor process
        // This is tricky - we could check environment variables
        // For now, return false as editors are typically launched separately
        false
    }
    
    fn current_session(&self) -> Option<String> {
        // Can't determine current editor workspace from outside
        None
    }
}

/// Open a directory in the editor
pub async fn open_workspace(command: &str, path: &Path) -> Result<()> {
    tokio::task::spawn_blocking({
        let command = command.to_string();
        let path = path.to_path_buf();
        move || {
            Command::new(&command)
                .arg(&path)
                .spawn()
                .map_err(|e| Error::BackendUnavailable {
                    backend: command.clone(),
                    install_hint: format!("Failed to open workspace: {}", e),
                })?;
            
            Ok(())
        }
    })
    .await
    .map_err(|e| Error::GitError {
        message: format!("Open workspace task failed: {}", e),
    })?
}

