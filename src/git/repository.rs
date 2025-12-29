use crate::error::{Error, Result};
use git2::{Repository, FetchOptions};
use std::path::Path;
use tracing::{debug, info};

/// Clone a repository to the given path
pub async fn clone(url: &str, path: &Path) -> Result<Repository> {
    debug!(url = %url, path = %path.display(), "Cloning repository");
    
    let url = url.to_string();
    let path = path.to_path_buf();
    tokio::task::spawn_blocking(move || {
        Repository::clone(&url, &path)
            .map_err(|e| Error::GitError {
                message: format!("Failed to clone {}: {}", url, e),
            })
    })
    .await
    .map_err(|e| Error::GitError {
        message: format!("Clone task failed: {}", e),
    })?
}

/// Open an existing repository
pub fn open(path: &Path) -> Result<Repository> {
    Repository::open(path).map_err(|e| Error::GitError {
        message: format!("Failed to open repository at {}: {}", path.display(), e),
    })
}

/// Get the default branch of a repository
pub fn get_default_branch(repo: &Repository) -> Result<String> {
    // Try to get the default branch from the remote
    let remote = repo.find_remote("origin")
        .or_else(|_| repo.remotes().and_then(|remotes| {
            if remotes.len() > 0 {
                repo.find_remote(remotes.get(0).unwrap())
            } else {
                Err(git2::Error::from_str("No remotes found"))
            }
        }))
        .map_err(|e| Error::GitError {
            message: format!("Failed to find remote: {}", e),
        })?;
    
    let refspec = format!("{}/HEAD", remote.name().unwrap());
    let ref_ = repo.find_reference(&refspec)
        .or_else(|_| repo.find_reference("refs/heads/main"))
        .or_else(|_| repo.find_reference("refs/heads/master"))
        .map_err(|e| Error::GitError {
            message: format!("Failed to find default branch: {}", e),
        })?;
    
    let branch = ref_.shorthand()
        .ok_or_else(|| Error::GitError {
            message: "Could not determine default branch".to_string(),
        })?;
    
    Ok(branch.to_string())
}

/// Get the remote URL for a repository
pub fn get_remote_url(path: &Path) -> Result<String> {
    let repo = open(path)?;
    let remote = repo.find_remote("origin")
        .or_else(|_| {
            let remotes = repo.remotes()?;
            if remotes.len() > 0 {
                repo.find_remote(remotes.get(0).unwrap())
            } else {
                Err(git2::Error::from_str("No remotes found"))
            }
        })
        .map_err(|e| Error::GitError {
            message: format!("Failed to find remote: {}", e),
        })?;
    
    remote.url()
        .ok_or_else(|| Error::GitError {
            message: "Remote has no URL".to_string(),
        })
        .map(|s| s.to_string())
}

/// Fetch from remote
pub async fn fetch(_repo: &Repository) -> Result<()> {
    info!("Fetching from remote");
    // TODO: Implement actual fetch logic
    // git2::Repository is not Send, so we need to rethink this
    Ok(())
}
