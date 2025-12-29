use crate::config::Config;
use crate::error::{Error, Result};
use crate::git;
use crate::store::{Store, CreateWorktree};
use crate::store::models::{Project, Worktree};
use std::path::PathBuf;
use tracing::{debug, info, warn, instrument};

pub struct WorktreeManager {
    store: Store,
    config: Config,
}

impl WorktreeManager {
    pub fn new(store: Store, config: Config) -> Self {
        Self { store, config }
    }
    
    /// Ensure a worktree exists for a project and branch
    #[instrument(skip(self), fields(project_id, branch = %branch))]
    pub async fn ensure_worktree(
        &self,
        project_id: i64,
        branch: &str,
        path: Option<PathBuf>,
    ) -> Result<Worktree> {
        debug!("Ensuring worktree exists");
        
        // Check if worktree already exists
        if let Some(worktree) = crate::store::queries::get_worktree_by_project_branch(
            self.store.pool(),
            project_id,
            branch,
        )
        .await? {
            debug!(worktree_id = worktree.id, "Worktree already exists");
            return Ok(worktree);
        }
        
        // Get project
        let project = crate::store::queries::get_project(self.store.pool(), project_id).await?;
        
        // Determine worktree path
        let worktree_path = path.unwrap_or_else(|| {
            let project_name = &project.name;
            self.config.worktrees_dir()
                .join(project_name)
                .join(branch.replace('/', "-"))
        });
        
        // Check if path already exists
        if worktree_path.exists() {
            return Err(Error::WorktreeExists {
                branch: branch.to_string(),
            });
        }
        
        info!(branch = %branch, path = %worktree_path.display(), "Creating worktree");
        
        // Open the repository
        let repo = git::open(&project.clone_path)?;
        
        // Create the worktree
        git::create_worktree(&repo, branch, &worktree_path).await?;
        
        // Register in database
        let worktree = crate::store::queries::create_worktree(
            self.store.pool(),
            CreateWorktree {
                project_id,
                branch: branch.to_string(),
                path: worktree_path,
                is_primary: false,
            },
        )
        .await?;
        
        Ok(worktree)
    }
    
    pub async fn list(&self, project_id: i64) -> Result<Vec<Worktree>> {
        crate::store::queries::list_worktrees(self.store.pool(), project_id).await
    }
    
    pub async fn get(&self, id: i64) -> Result<Worktree> {
        crate::store::queries::get_worktree(self.store.pool(), id).await
    }
    
    pub async fn delete(&self, worktree_id: i64) -> Result<()> {
        let worktree = self.get(worktree_id).await?;
        let project = crate::store::queries::get_project(
            self.store.pool(),
            worktree.project_id,
        )
        .await?;
        
        if worktree.is_primary {
            return Err(Error::GitError {
                message: "Cannot delete primary worktree. Delete the project instead.".to_string(),
            });
        }
        
        // Remove from filesystem
        git::remove_worktree(&project.clone_path, &worktree.path).await?;
        
        // Database cascade will clean up sessions
        // We need to manually delete the worktree record
        sqlx::query!("DELETE FROM worktrees WHERE id = ?1")
            .bind(worktree_id)
            .execute(self.store.pool())
            .await?;
        
        Ok(())
    }
    
    /// Check if a branch still exists (locally or remotely)
    pub async fn validate_branch(
        &self,
        project: &Project,
        branch: &str,
    ) -> Result<BranchStatus> {
        let repo = git::open(&project.clone_path)?;
        
        // Check local first
        if git::branch_exists_local(&repo, branch)? {
            // Check if it has a remote tracking branch
            match git::get_upstream(&repo, branch) {
                Ok(upstream) => {
                    // Verify upstream still exists
                    match git::remote_branch_exists(&repo, &upstream).await {
                        Ok(true) => Ok(BranchStatus::Tracked),
                        Ok(false) => Ok(BranchStatus::LocalOnly {
                            warning: Some("Remote branch was deleted".to_string()),
                        }),
                        Err(_) => Ok(BranchStatus::LocalOnly {
                            warning: Some("Could not verify remote (network issue?)".to_string()),
                        }),
                    }
                }
                Err(_) => Ok(BranchStatus::LocalOnly { warning: None }),
            }
        } else {
            Ok(BranchStatus::NotFound)
        }
    }
    
    /// Switch to branch with graceful handling of missing remote
    pub async fn switch_branch(
        &self,
        project_id: i64,
        branch: &str,
        path: Option<PathBuf>,
    ) -> Result<Worktree> {
        let project = crate::store::queries::get_project(self.store.pool(), project_id).await?;
        
        let status = self.validate_branch(&project, branch).await?;
        
        match status {
            BranchStatus::Tracked => {
                // Normal case - proceed
                self.ensure_worktree(project_id, branch, path).await
            }
            BranchStatus::LocalOnly { warning } => {
                // Warn but proceed
                if let Some(msg) = warning {
                    warn!("{}", msg);
                }
                self.ensure_worktree(project_id, branch, path).await
            }
            BranchStatus::NotFound => {
                // Offer to create
                Err(Error::BranchNotFound {
                    branch: branch.to_string(),
                })
            }
        }
    }
}

#[derive(Debug, Clone)]
pub enum BranchStatus {
    Tracked,
    LocalOnly {
        warning: Option<String>,
    },
    NotFound,
}

