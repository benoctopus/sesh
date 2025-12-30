use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::{ProjectManager, SessionManager};
use crate::backends::traits::{BackendKind, create_backend};
use tracing::info;

#[derive(Args)]
pub struct CloneArgs {
    /// Repository URL to clone
    pub url: String,
    
    /// Custom path for the clone (default: ~/.sesh/projects/{project_name})
    #[arg(long)]
    pub path: Option<String>,
    
    /// Create session without attaching to it
    #[arg(short = 'd', long)]
    pub detach: bool,
}

pub async fn run(args: CloneArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    let manager = ProjectManager::new(store.clone(), config.clone());
    
    let path = args.path.map(|p| p.into());
    let project = manager.clone(&args.url, path).await?;
    
    info!(project = %project.name, "Project cloned successfully");
    println!("Cloned {} to {}", project.display_name, project.clone_path.display());
    
    // Create session for the default worktree
    let worktrees = crate::store::queries::list_worktrees(store.pool(), project.id).await?;
    if let Some(primary_worktree) = worktrees.iter().find(|w| w.is_primary) {
        let backend_kind = BackendKind::from_str(&config.session.backend)
            .unwrap_or(BackendKind::Tmux);
        let backend = create_backend(backend_kind);
        let session_manager = SessionManager::new(store, backend);
        
        // Switch to session (with optional detach)
        session_manager.switch_to_with_detach(primary_worktree.id, args.detach).await?;
    }
    
    Ok(())
}

