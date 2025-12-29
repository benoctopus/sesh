use crate::error::{Error, Result};
use git2::{Repository, Worktree, WorktreeAddOptions};
use std::path::Path;
use tracing::{debug, info};

/// Create a new worktree
pub async fn create_worktree(
    repo: &Repository,
    branch: &str,
    path: &Path,
) -> Result<Worktree> {
    debug!(branch = %branch, path = %path.display(), "Creating worktree");
    
    tokio::task::spawn_blocking({
        let repo = repo.clone();
        let branch = branch.to_string();
        let path = path.to_path_buf();
        move || {
            // First, ensure the branch exists locally or fetch it
            let branch_ref = format!("refs/heads/{}", branch);
            if repo.find_reference(&branch_ref).is_err() {
                // Try to create from remote
                let remote_branch = format!("origin/{}", branch);
                if repo.find_reference(&format!("refs/remotes/{}", remote_branch)).is_ok() {
                    let (object, reference) = repo.revparse_ext(&remote_branch)
                        .map_err(|e| Error::GitError {
                            message: format!("Failed to find remote branch {}: {}", branch, e),
                        })?;
                    
                    repo.branch(&branch, &object, false)
                        .map_err(|e| Error::GitError {
                            message: format!("Failed to create local branch {}: {}", branch, e),
                        })?;
                } else {
                    return Err(Error::BranchNotFound {
                        branch: branch.clone(),
                    });
                }
            }
            
            let mut opts = WorktreeAddOptions::new();
            opts.reference(Some(&repo.find_reference(&branch_ref)?));
            
            repo.worktree(&branch, &path, Some(&mut opts))
                .map_err(|e| Error::GitError {
                    message: format!("Failed to create worktree at {}: {}", path.display(), e),
                })
        }
    })
    .await
    .map_err(|e| Error::GitError {
        message: format!("Worktree creation task failed: {}", e),
    })?
}

/// Remove a worktree
pub async fn remove_worktree(repo_path: &Path, worktree_path: &Path) -> Result<()> {
    debug!(worktree_path = %worktree_path.display(), "Removing worktree");
    
    tokio::task::spawn_blocking({
        let repo_path = repo_path.to_path_buf();
        let worktree_path = worktree_path.to_path_buf();
        move || {
            let repo = Repository::open(&repo_path)?;
            
            // Find the worktree
            let worktrees = repo.worktrees()
                .map_err(|e| Error::GitError {
                    message: format!("Failed to list worktrees: {}", e),
                })?;
            
            for wt_name in worktrees {
                let wt = repo.find_worktree(&wt_name)
                    .map_err(|e| Error::GitError {
                        message: format!("Failed to find worktree {}: {}", wt_name, e),
                    })?;
                
                if wt.path() == worktree_path {
                    repo.worktree_remove(&wt_name, true)
                        .map_err(|e| Error::GitError {
                            message: format!("Failed to remove worktree {}: {}", wt_name, e),
                        })?;
                    return Ok(());
                }
            }
            
            // If not found in git worktrees, just remove the directory
            if worktree_path.exists() {
                std::fs::remove_dir_all(&worktree_path)?;
            }
            
            Ok::<(), Error>(())
        }
    })
    .await
    .map_err(|e| Error::GitError {
        message: format!("Worktree removal task failed: {}", e),
    })?
}

