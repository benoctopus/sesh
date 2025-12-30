use crate::config;
use crate::error::Result;
use crate::managers::{ProjectManager, WorktreeManager};
use crate::store::Store;
use clap::Args;

#[derive(Args)]
pub struct ListArgs {
    /// List projects
    #[arg(long)]
    pub projects: bool,

    /// List worktrees
    #[arg(long)]
    pub worktrees: bool,

    /// List sessions
    #[arg(long)]
    pub sessions: bool,
    
    /// Show open pull requests
    #[arg(long)]
    pub pr: bool,
    
    /// Output in JSON format
    #[arg(long)]
    pub json: bool,
    
    /// Output session names only (for piping)
    #[arg(long)]
    pub plain: bool,
    
    /// Filter to sessions for current project
    #[arg(long)]
    pub current_project: bool,
    
    /// Show only running sessions
    #[arg(long)]
    pub running: bool,
    
    /// Show all sessions (running and stopped)
    #[arg(long)]
    pub all: bool,
}

pub async fn run(args: ListArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;

    let project_manager = ProjectManager::new(store.clone(), config.clone());
    let worktree_manager = WorktreeManager::new(store.clone(), config);

    // Default: list all if no specific flag
    let list_all = !args.projects && !args.worktrees && !args.sessions;

    if list_all || args.projects {
        let projects = project_manager.list().await?;
        println!("Projects:");
        for project in projects {
            println!("  {} ({})", project.display_name, project.name);
        }
    }

    if list_all || args.worktrees {
        let projects = project_manager.list().await?;
        println!("\nWorktrees:");
        for project in projects {
            let worktrees = worktree_manager.list(project.id).await?;
            for worktree in worktrees {
                println!(
                    "  {}:{} ({})",
                    project.display_name,
                    worktree.branch,
                    worktree.path.display()
                );
            }
        }
    }

    if list_all || args.sessions {
        // List sessions from database
        let rows = sqlx::query(
            "SELECT session_name, backend FROM sessions ORDER BY last_attached_at DESC",
        )
        .fetch_all(store.pool())
        .await?;

        println!("\nSessions:");
        use sqlx::Row;
        for row in rows {
            let session_name: String = row.get::<String, _>("session_name");
            let backend: String = row.get::<String, _>("backend");
            println!("  {} ({})", session_name, backend);
        }
    }

    Ok(())
}
