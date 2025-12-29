use crate::config::Config;
use crate::error::{Error, Result};
use crate::git;
use crate::store::{Store, CreateProject, CreateWorktree};
use crate::store::models::Project;
use sha2::{Sha256, Digest};
use std::path::PathBuf;
use tracing::{debug, info, instrument};

pub struct ProjectManager {
    store: Store,
    config: Config,
}

impl ProjectManager {
    pub fn new(store: Store, config: Config) -> Self {
        Self { store, config }
    }
    
    /// Parse project name from URL
    fn parse_project_name(url: &str) -> Result<String> {
        // Extract host/user/repo from URL
        let url = url.trim_end_matches(".git");
        let parts: Vec<&str> = url.split('/').collect();
        
        if parts.len() < 3 {
            return Err(Error::InvalidUrl {
                url: url.to_string(),
            });
        }
        
        // Get last 3 parts (host, user, repo)
        let name = parts[parts.len() - 3..].join("/");
        Ok(name)
    }
    
    /// Extract repo name from project name
    fn extract_repo_name(project_name: &str) -> String {
        project_name
            .rsplit('/')
            .next()
            .unwrap_or(project_name)
            .to_string()
    }
    
    /// Generate a unique session name for a worktree
    pub fn generate_session_name(project_name: &str, branch: &str) -> String {
        let repo_name = Self::extract_repo_name(project_name);
        
        let branch_slug = branch
            .replace('/', "-")
            .chars()
            .take(20)
            .collect::<String>();
        
        let hash_input = format!("{}:{}", project_name, branch);
        let hash = Sha256::digest(hash_input.as_bytes());
        let hash_suffix = hex::encode(&hash[..2]); // 4 hex chars
        
        format!("{}_{}_{}", repo_name, branch_slug, hash_suffix)
    }
    
    #[instrument(skip(self), fields(url = %url))]
    pub async fn clone(
        &self,
        url: &str,
        path: Option<PathBuf>,
    ) -> Result<Project> {
        debug!("Starting clone operation");
        
        let project_name = Self::parse_project_name(url)?;
        let clone_path = path.unwrap_or_else(|| {
            let p = self.config.projects_dir().join(&project_name);
            debug!(path = %p.display(), "Using default clone path");
            p
        });
        
        // Check if path already exists
        if clone_path.exists() {
            return Err(Error::RepositoryExists {
                path: clone_path,
            });
        }
        
        info!(project = %project_name, "Cloning repository");
        
        // Clone the repository
        let repo = git::clone(url, &clone_path).await?;
        let default_branch = git::get_default_branch(&repo)?;
        
        debug!("Clone successful, registering in database");
        
        // Register in database
        let project = crate::store::queries::create_project(
            self.store.pool(),
            CreateProject {
                name: project_name.clone(),
                display_name: Self::extract_repo_name(&project_name),
                remote_url: url.to_string(),
                clone_path: clone_path.clone(),
                default_branch: default_branch.clone(),
            },
        )
        .await?;
        
        // Register the clone as the primary worktree
        crate::store::queries::create_worktree(
            self.store.pool(),
            CreateWorktree {
                project_id: project.id,
                branch: default_branch,
                path: clone_path,
                is_primary: true,
            },
        )
        .await?;
        
        Ok(project)
    }
    
    pub async fn list(&self) -> Result<Vec<Project>> {
        crate::store::queries::list_projects(self.store.pool()).await
    }
    
    pub async fn get(&self, id: i64) -> Result<Project> {
        crate::store::queries::get_project(self.store.pool(), id).await
    }
    
    pub async fn get_by_name(&self, name: &str) -> Result<Project> {
        crate::store::queries::get_project_by_name(self.store.pool(), name).await
    }
    
    #[instrument(skip(self))]
    pub async fn delete(&self, project_id: i64) -> Result<()> {
        let project = self.get(project_id).await?;
        let worktrees = crate::store::queries::list_worktrees(self.store.pool(), project_id).await?;
        
        debug!(worktree_count = worktrees.len(), "Deleting project and worktrees");
        
        // Delete worktrees from filesystem (non-primary first)
        for wt in worktrees.iter().filter(|w| !w.is_primary) {
            git::remove_worktree(&project.clone_path, &wt.path).await?;
        }
        
        // Delete the clone directory
        if project.clone_path.exists() {
            tokio::fs::remove_dir_all(&project.clone_path).await?;
        }
        
        // Database cascade will clean up worktrees/sessions
        crate::store::queries::delete_project(self.store.pool(), project_id).await?;
        
        Ok(())
    }
    
    /// Validate a project's filesystem state
    pub async fn validate(&self, project_id: i64) -> Result<EntityStatus> {
        let project = self.get(project_id).await?;
        
        if !project.clone_path.exists() {
            return Ok(EntityStatus::Stale);
        }
        
        if !project.clone_path.join(".git").exists() {
            return Ok(EntityStatus::Corrupted);
        }
        
        // Verify it's the expected repository
        match git::get_remote_url(&project.clone_path) {
            Ok(url) if url == project.remote_url => Ok(EntityStatus::Valid),
            Ok(_) => Ok(EntityStatus::Corrupted), // Different repo at this path
            Err(_) => Ok(EntityStatus::Corrupted),
        }
    }
    
    /// List projects, annotating with validation status
    pub async fn list_validated(&self) -> Result<Vec<(Project, EntityStatus)>> {
        let projects = self.list().await?;
        let mut results = Vec::with_capacity(projects.len());
        
        for project in projects {
            let status = self.validate(project.id).await?;
            results.push((project, status));
        }
        
        Ok(results)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EntityStatus {
    Valid,
    Stale,      // Path doesn't exist
    Corrupted,  // Path exists but invalid state
    Orphaned,   // Exists but parent relationship broken
}

