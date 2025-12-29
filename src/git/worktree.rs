use crate::error::{Error, Result};
use git2::{Repository, WorktreeAddOptions};
use std::path::Path;
use tracing::debug;

/// Create a new worktree
pub async fn create_worktree(repo_path: &Path, branch: &str, path: &Path) -> Result<()> {
    debug!(branch = %branch, path = %path.display(), "Creating worktree");

    let repo_path = repo_path.to_path_buf();
    let branch = branch.to_string();
    let path = path.to_path_buf();
    tokio::task::spawn_blocking(move || {
        let repo = Repository::open(&repo_path).map_err(|e| Error::GitError {
            message: format!("Failed to open repository: {}", e),
        })?;

        // First, ensure the branch exists locally or fetch it
        let branch_ref = format!("refs/heads/{}", branch);
        if repo.find_reference(&branch_ref).is_err() {
            // Try to create from remote
            let remote_branch = format!("origin/{}", branch);
            if repo
                .find_reference(&format!("refs/remotes/{}", remote_branch))
                .is_ok()
            {
                let (object, _reference) =
                    repo.revparse_ext(&remote_branch)
                        .map_err(|e| Error::GitError {
                            message: format!("Failed to find remote branch {}: {}", branch, e),
                        })?;

                repo.branch(
                    &branch,
                    &object.peel_to_commit().map_err(|e| Error::GitError {
                        message: format!("Failed to peel object to commit: {}", e),
                    })?,
                    false,
                )
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
        let branch_ref_obj = repo
            .find_reference(&branch_ref)
            .map_err(|e| Error::GitError {
                message: format!("Failed to find reference: {}", e),
            })?;
        opts.reference(Some(&branch_ref_obj));

        // Create worktree - don't return the Worktree object as it's not Send
        repo.worktree(&branch, &path, Some(&mut opts))
            .map_err(|e| Error::GitError {
                message: format!("Failed to create worktree at {}: {}", path.display(), e),
            })?;

        Ok::<(), Error>(())
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
            let repo = Repository::open(&repo_path).map_err(|e| Error::GitError {
                message: format!("Failed to open repository: {}", e),
            })?;

            // Find the worktree
            let worktrees = repo.worktrees().map_err(|e| Error::GitError {
                message: format!("Failed to list worktrees: {}", e),
            })?;

            let worktree_names: Vec<String> = worktrees
                .into_iter()
                .filter_map(|s| s)
                .map(|s| s.to_string())
                .collect();

            for wt_name in worktree_names {
                let wt = repo.find_worktree(&wt_name).map_err(|e| Error::GitError {
                    message: format!("Failed to find worktree {}: {}", wt_name, e),
                })?;

                if wt.path() == worktree_path {
                    // Remove the worktree directory first, then prune
                    if worktree_path.exists() {
                        std::fs::remove_dir_all(&worktree_path)?;
                    }
                    // Note: git2 0.20 doesn't have worktree_remove, we handle it manually
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
