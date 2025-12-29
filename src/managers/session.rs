use crate::backends::traits::SessionBackend;
use crate::error::{Error, Result};
use crate::store::{Store, CreateSession};
use crate::store::models::{Worktree, Session};
use crate::managers::project::ProjectManager;
use tracing::{debug, info, warn, instrument};

pub struct SessionManager {
    store: Store,
    backend: Box<dyn SessionBackend>,
}

impl SessionManager {
    pub fn new(store: Store, backend: Box<dyn SessionBackend>) -> Self {
        Self { store, backend }
    }
    
    /// Create or attach to a session for a worktree
    #[instrument(skip(self), fields(worktree_id))]
    pub async fn switch_to(&self, worktree_id: i64) -> Result<()> {
        let worktree = crate::store::queries::get_worktree(self.store.pool(), worktree_id).await?;
        
        // Validate worktree path first
        if !worktree.path.exists() {
            return Err(Error::WorktreeStale {
                path: worktree.path.clone(),
            });
        }
        
        debug!("Getting or creating session for worktree");
        
        // Check if session already exists in database
        let session = match crate::store::queries::get_session_for_worktree(
            self.store.pool(),
            worktree_id,
        )
        .await {
            Ok(s) => {
                // Session record exists - verify it's actually running
                if !self.backend.exists(&s.session_name).await? {
                    // Session died - recreate it
                    warn!(session = %s.session_name, "Recreating session (previous session no longer exists)");
                    self.backend.create(&s.session_name, &worktree.path).await?;
                }
                s
            }
            Err(_) => {
                // No session record - create fresh
                self.create_session_for_worktree(&worktree).await?
            }
        };
        
        // Record in history before switching
        crate::store::queries::add_to_history(self.store.pool(), session.id).await?;
        
        // Update last_accessed_at
        crate::store::queries::touch_worktree(self.store.pool(), worktree_id).await?;
        crate::store::queries::touch_session(self.store.pool(), session.id).await?;
        
        // Attach or switch depending on context
        if self.backend.is_inside_session() {
            debug!("Switching to session");
            self.backend.switch(&session.session_name).await?;
        } else {
            info!(session = %session.session_name, "Attaching to session");
            match self.backend.kind() {
                crate::backends::traits::BackendKind::Code | 
                crate::backends::traits::BackendKind::Cursor => {
                    // For editors, open the workspace
                    crate::backends::editor::open_workspace(
                        self.backend.kind().as_str(),
                        &worktree.path,
                    ).await?;
                }
                _ => {
                    self.backend.attach(&session.session_name).await?;
                }
            }
        }
        
        Ok(())
    }
    
    async fn create_session_for_worktree(&self, worktree: &Worktree) -> Result<Session> {
        let project = crate::store::queries::get_project(
            self.store.pool(),
            worktree.project_id,
        )
        .await?;
        
        let name = ProjectManager::generate_session_name(&project.name, &worktree.branch);
        
        debug!(session = %name, "Creating new session");
        
        self.backend.create(&name, &worktree.path).await?;
        
        let session = crate::store::queries::create_session(
            self.store.pool(),
            CreateSession {
                worktree_id: worktree.id,
                session_name: name.clone(),
                backend: self.backend.kind().as_str().to_string(),
            },
        )
        .await?;
        
        Ok(session)
    }
    
    /// Pop to previous session from history
    #[instrument(skip(self))]
    pub async fn pop(&self) -> Result<()> {
        let current = self.backend.current_session();
        let previous = crate::store::queries::get_previous_session(
            self.store.pool(),
            current.as_deref(),
        )
        .await?;
        
        info!(session = %previous.session_name, "Popping to previous session");
        
        self.backend.switch(&previous.session_name).await?;
        
        Ok(())
    }
    
    pub async fn get_for_worktree(&self, worktree_id: i64) -> Result<Option<Session>> {
        match crate::store::queries::get_session_for_worktree(
            self.store.pool(),
            worktree_id,
        )
        .await {
            Ok(s) => Ok(Some(s)),
            Err(_) => Ok(None),
        }
    }
}

