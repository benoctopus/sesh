use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::ProjectManager;
use std::io::{self, Write};
use tracing::info;

#[derive(Args)]
pub struct CleanArgs {
    /// Remove stale entries (paths don't exist)
    #[arg(long)]
    pub stale: bool,
    
    /// Remove orphaned sessions
    #[arg(long)]
    pub orphaned: bool,
    
    /// Project name to clean
    #[arg(short = 'p', long)]
    pub project: Option<String>,
    
    /// Skip confirmation prompts
    #[arg(short = 'f', long)]
    pub force: bool,
    
    /// Delete local worktrees for branches that have been deleted on the remote
    #[arg(long)]
    pub remote_deleted: bool,
}

fn confirm_cleanup(message: &str, force: bool) -> Result<bool> {
    if force {
        return Ok(true);
    }
    
    // Check if we're in an interactive terminal
    let is_tty = std::io::IsTerminal::is_terminal(&std::io::stdin());
    if !is_tty {
        return Err(crate::error::Error::GitError {
            message: "--force flag required for cleanup in noninteractive mode".to_string(),
        });
    }
    
    print!("{}", message);
    io::stdout().flush()?;
    
    let mut input = String::new();
    io::stdin().read_line(&mut input)?;
    
    let response = input.trim().to_lowercase();
    Ok(response == "yes" || response == "y")
}

pub async fn run(args: CleanArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let project_manager = ProjectManager::new(store.clone(), config);
    
    if args.stale {
        let projects = project_manager.list_validated().await?;
        let stale_projects: Vec<_> = projects
            .iter()
            .filter(|(_, status)| matches!(status, crate::managers::project::EntityStatus::Stale))
            .collect();
        
        if !stale_projects.is_empty() {
            let message = format!(
                "Found {} stale project(s). Delete these projects? (yes/no): ",
                stale_projects.len()
            );
            
            if !confirm_cleanup(&message, args.force)? {
                println!("Cleanup cancelled.");
                return Ok(());
            }
        }
        
        let mut removed = 0;
        for (project, _) in stale_projects {
            info!(project = %project.name, "Removing stale project");
            let _ = project_manager.delete(project.id).await;
            removed += 1;
        }
        
        println!("Removed {} stale entries", removed);
    }
    
    if args.orphaned {
        // Remove sessions where worktree doesn't exist
        let rows = sqlx::query(
            r#"
            SELECT s.id FROM sessions s
            LEFT JOIN worktrees w ON s.worktree_id = w.id
            WHERE w.id IS NULL
            "#,
        )
        .fetch_all(store.pool())
        .await?;
        
        if !rows.is_empty() {
            let message = format!(
                "Found {} orphaned session(s). Delete these sessions? (yes/no): ",
                rows.len()
            );
            
            if !confirm_cleanup(&message, args.force)? {
                println!("Cleanup cancelled.");
                return Ok(());
            }
        }
        
        let mut count = 0;
        use sqlx::Row;
        for row in rows {
            let id: i64 = row.get::<i64, _>("id");
            sqlx::query("DELETE FROM sessions WHERE id = ?1")
                .bind(id)
                .execute(store.pool())
                .await?;
            count += 1;
        }
        
        println!("Removed {} orphaned sessions", count);
    }
    
    if args.remote_deleted {
        // TODO: Implement remote-deleted branch cleanup
        eprintln!("--remote-deleted flag is not yet implemented");
    }
    
    if !args.stale && !args.orphaned && !args.remote_deleted {
        eprintln!("Must specify --stale, --orphaned, or --remote-deleted");
    }
    
    Ok(())
}

