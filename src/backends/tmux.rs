use crate::backends::traits::{BackendKind, SessionBackend};
use crate::error::{Error, Result};
use std::path::Path;
use std::process::Command;
use tracing::{debug, info, warn};

pub struct TmuxBackend;

impl TmuxBackend {
    pub fn new() -> Self {
        Self
    }
    
    fn check_available(&self) -> Result<()> {
        let output = Command::new("tmux")
            .arg("-V")
            .output()
            .map_err(|_| Error::BackendUnavailable {
                backend: "tmux".to_string(),
                install_hint: "Install with: brew install tmux (macOS) or apt install tmux (Linux)".to_string(),
            })?;
        
        if !output.status.success() {
            return Err(Error::BackendUnavailable {
                backend: "tmux".to_string(),
                install_hint: "Install with: brew install tmux (macOS) or apt install tmux (Linux)".to_string(),
            });
        }
        
        Ok(())
    }
    
    fn run_tmux(&self, args: &[&str]) -> Result<String> {
        self.check_available()?;
        
        let output = Command::new("tmux")
            .args(args)
            .output()
            .map_err(|e| Error::GitError {
                message: format!("Failed to run tmux: {}", e),
            })?;
        
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            return Err(Error::GitError {
                message: format!("tmux command failed: {}", stderr),
            });
        }
        
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    }
}

#[async_trait::async_trait]
impl SessionBackend for TmuxBackend {
    fn kind(&self) -> BackendKind {
        BackendKind::Tmux
    }
    
    async fn create(&self, name: &str, working_dir: &Path) -> Result<()> {
        debug!(session = %name, path = %working_dir.display(), "Creating tmux session");
        
        // Check if session already exists
        if self.exists(name).await? {
            debug!(session = %name, "Session already exists");
            return Ok(());
        }
        
        tokio::task::spawn_blocking({
            let name = name.to_string();
            let working_dir = working_dir.to_path_buf();
            move || {
                // Create new session in detached mode
                let output = Command::new("tmux")
                    .args(&["new-session", "-d", "-s", &name, "-c", &working_dir.to_string_lossy()])
                    .output()
                    .map_err(|e| Error::GitError {
                        message: format!("Failed to create tmux session: {}", e),
                    })?;
                
                if !output.status.success() {
                    let stderr = String::from_utf8_lossy(&output.stderr);
                    return Err(Error::GitError {
                        message: format!("tmux new-session failed: {}", stderr),
                    });
                }
                
                Ok(())
            }
        })
        .await
        .map_err(|e| Error::GitError {
            message: format!("Create session task failed: {}", e),
        })?
    }
    
    async fn attach(&self, name: &str) -> Result<()> {
        info!(session = %name, "Attaching to tmux session");
        
        // Ensure session exists
        if !self.exists(name).await? {
            return Err(Error::SessionNotFound {
                name: name.to_string(),
            });
        }
        
        // For tmux, attach replaces the current process
        // We need to use std::process::Command::exec which never returns
        // This should be called from a blocking context
        let name = name.to_string();
        tokio::task::spawn_blocking(move || {
            use std::os::unix::process::CommandExt;
            let mut cmd = Command::new("tmux");
            cmd.args(&["attach-session", "-t", &name]);
            
            // Replace current process - this never returns
            let err = cmd.exec();
            // This line should never be reached, but rustc needs it
            Err::<(), Error>(Error::GitError {
                message: format!("Failed to attach to tmux session: {}", err),
            })
        })
        .await
        .map_err(|e| Error::GitError {
            message: format!("Attach task failed: {}", e),
        })?
    }
    
    async fn switch(&self, name: &str) -> Result<()> {
        debug!(session = %name, "Switching to tmux session");
        
        if !self.exists(name).await? {
            return Err(Error::SessionNotFound {
                name: name.to_string(),
            });
        }
        
        self.run_tmux(&["switch-client", "-t", name])?;
        Ok(())
    }
    
    async fn delete(&self, name: &str) -> Result<()> {
        debug!(session = %name, "Deleting tmux session");
        
        if !self.exists(name).await? {
            warn!(session = %name, "Session does not exist, skipping delete");
            return Ok(());
        }
        
        self.run_tmux(&["kill-session", "-t", name])?;
        Ok(())
    }
    
    async fn exists(&self, name: &str) -> Result<bool> {
        let output = self.run_tmux(&["has-session", "-t", name]);
        match output {
            Ok(_) => Ok(true),
            Err(_) => Ok(false),
        }
    }
    
    async fn list_active(&self) -> Result<Vec<String>> {
        let output = self.run_tmux(&["list-sessions", "-F", "#{session_name}"])?;
        Ok(output
            .lines()
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .collect())
    }
    
    fn is_inside_session(&self) -> bool {
        std::env::var("TMUX").is_ok()
    }
    
    fn current_session(&self) -> Option<String> {
        if !self.is_inside_session() {
            return None;
        }
        
        // Get current session name
        let output = Command::new("tmux")
            .args(&["display-message", "-p", "#{session_name}"])
            .output()
            .ok()?;
        
        if output.status.success() {
            let name = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !name.is_empty() {
                return Some(name);
            }
        }
        
        None
    }
}

