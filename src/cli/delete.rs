use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::{ProjectManager, WorktreeManager};
use tracing::info;

#[derive(Args)]
pub struct DeleteArgs {
    /// Project name to delete
    #[arg(long)]
    pub project: Option<String>,
    
    /// Worktree ID to delete
    #[arg(long)]
    pub worktree: Option<i64>,
}

pub async fn run(args: DeleteArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let project_manager = ProjectManager::new(store.clone(), config.clone());
    let worktree_manager = WorktreeManager::new(store, config);
    
    if let Some(project_name) = args.project {
        let project = project_manager.get_by_name(&project_name).await?;
        project_manager.delete(project.id).await?;
        info!(project = %project_name, "Project deleted");
        println!("Deleted project: {}", project_name);
    } else if let Some(worktree_id) = args.worktree {
        worktree_manager.delete(worktree_id).await?;
        info!(worktree_id, "Worktree deleted");
        println!("Deleted worktree: {}", worktree_id);
    } else {
        eprintln!("Must specify either --project or --worktree");
    }
    
    Ok(())
}

