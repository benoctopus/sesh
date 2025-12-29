use crate::error::{Error, Result};
use git2::{BranchType, Repository};

/// Check if a branch exists locally
pub fn branch_exists_local(repo: &Repository, branch: &str) -> Result<bool> {
    let branch_ref = format!("refs/heads/{}", branch);
    match repo.find_reference(&branch_ref) {
        Ok(_) => Ok(true),
        Err(_) => Ok(false),
    }
}

/// Get the upstream branch for a local branch
pub fn get_upstream(repo: &Repository, branch: &str) -> Result<String> {
    let branch_ref = repo
        .find_branch(branch, BranchType::Local)
        .map_err(|e| Error::GitError {
            message: format!("Branch {} not found: {}", branch, e),
        })?;

    let upstream = branch_ref.upstream().map_err(|e| Error::GitError {
        message: format!("No upstream for branch {}: {}", branch, e),
    })?;

    upstream
        .name()
        .map_err(|_| Error::GitError {
            message: "Upstream branch name is invalid UTF-8".to_string(),
        })?
        .ok_or_else(|| Error::GitError {
            message: "Upstream branch has no name".to_string(),
        })
        .map(|s| s.to_string())
}

/// Check if a remote branch exists
pub fn remote_branch_exists(repo: &Repository, upstream: &str) -> Result<bool> {
    // Parse upstream (e.g., "origin/main")
    let parts: Vec<&str> = upstream.split('/').collect();
    if parts.len() != 2 {
        return Ok(false);
    }

    let remote_name = parts[0];
    let branch_name = parts[1];

    let remote_ref = format!("refs/remotes/{}/{}", remote_name, branch_name);
    match repo.find_reference(&remote_ref) {
        Ok(_) => Ok(true),
        Err(_) => Ok(false),
    }
}

/// List all local branches
pub fn list_local_branches(repo: &Repository) -> Result<Vec<String>> {
    let branches = repo
        .branches(Some(BranchType::Local))
        .map_err(|e| Error::GitError {
            message: format!("Failed to list branches: {}", e),
        })?;

    let mut branch_names = Vec::new();
    for branch in branches {
        let (branch, _) = branch.map_err(|e| Error::GitError {
            message: format!("Failed to process branch: {}", e),
        })?;

        if let Ok(Some(name)) = branch.name() {
            branch_names.push(name.to_string());
        }
    }

    Ok(branch_names)
}

/// List all remote branches
pub fn list_remote_branches(repo: &Repository) -> Result<Vec<String>> {
    let branches = repo
        .branches(Some(BranchType::Remote))
        .map_err(|e| Error::GitError {
            message: format!("Failed to list remote branches: {}", e),
        })?;

    let mut branch_names = Vec::new();
    for branch in branches {
        let (branch, _) = branch.map_err(|e| Error::GitError {
            message: format!("Failed to process branch: {}", e),
        })?;

        if let Ok(Some(name)) = branch.name() {
            // Remove "origin/" prefix
            let name = name
                .strip_prefix("origin/")
                .or_else(|| name.strip_prefix("remotes/origin/"))
                .unwrap_or(name);
            branch_names.push(name.to_string());
        }
    }

    Ok(branch_names)
}

/// List all branches (local and remote, deduplicated)
pub fn list_all_branches(repo: &Repository) -> Result<Vec<String>> {
    let mut branches = std::collections::HashSet::new();

    // Add local branches
    for branch in list_local_branches(repo)? {
        branches.insert(branch);
    }

    // Add remote branches
    for branch in list_remote_branches(repo)? {
        branches.insert(branch);
    }

    let mut result: Vec<String> = branches.into_iter().collect();
    result.sort();
    Ok(result)
}
