use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::ProjectManager;
use tracing::info;

#[derive(Args)]
pub struct CleanArgs {
    /// Remove stale entries (paths don't exist)
    #[arg(long)]
    pub stale: bool,
    
    /// Remove orphaned sessions
    #[arg(long)]
    pub orphaned: bool,
}

pub async fn run(args: CleanArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let project_manager = ProjectManager::new(store.clone(), config);
    
    if args.stale {
        let projects = project_manager.list_validated().await?;
        let mut removed = 0;
        
        for (project, status) in projects {
            if matches!(status, crate::managers::project::EntityStatus::Stale) {
                info!(project = %project.name, "Removing stale project");
                let _ = project_manager.delete(project.id).await;
                removed += 1;
            }
        }
        
        println!("Removed {} stale entries", removed);
    }
    
    if args.orphaned {
        // Remove sessions where worktree doesn't exist
        let sessions = sqlx::query!(
            r#"
            SELECT s.* FROM sessions s
            LEFT JOIN worktrees w ON s.worktree_id = w.id
            WHERE w.id IS NULL
            "#
        )
        .fetch_all(store.pool())
        .await?;
        
        for session in sessions {
            sqlx::query!("DELETE FROM sessions WHERE id = ?1", session.id)
                .execute(store.pool())
                .await?;
        }
        
        println!("Removed {} orphaned sessions", sessions.len());
    }
    
    if !args.stale && !args.orphaned {
        eprintln!("Must specify --stale or --orphaned");
    }
    
    Ok(())
}

