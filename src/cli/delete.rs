use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::{ProjectManager, WorktreeManager};
use std::io::{self, Write};
use tracing::info;

#[derive(Args)]
pub struct DeleteArgs {
    /// Project name to delete
    #[arg(short = 'p', long)]
    pub project: Option<String>,
    
    /// Worktree ID to delete
    #[arg(long)]
    pub worktree: Option<i64>,
    
    /// Skip confirmation prompt
    #[arg(short = 'f', long)]
    pub force: bool,
    
    /// Delete entire project (requires --project)
    #[arg(long)]
    pub all: bool,
}

fn confirm_deletion(message: &str, force: bool) -> Result<bool> {
    if force {
        return Ok(true);
    }
    
    // Check if we're in an interactive terminal
    let is_tty = std::io::IsTerminal::is_terminal(&std::io::stdin());
    if !is_tty {
        return Err(crate::error::Error::GitError {
            message: "--force flag required for deletion in noninteractive mode".to_string(),
        });
    }
    
    print!("{}", message);
    io::stdout().flush()?;
    
    let mut input = String::new();
    io::stdin().read_line(&mut input)?;
    
    let response = input.trim().to_lowercase();
    Ok(response == "yes" || response == "y")
}

pub async fn run(args: DeleteArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let project_manager = ProjectManager::new(store.clone(), config.clone());
    let worktree_manager = WorktreeManager::new(store.clone(), config);
    
    if let Some(project_name) = args.project {
        let project = project_manager.get_by_name(&project_name).await?;
        
        if args.all {
            // Delete entire project
            let worktrees = crate::store::queries::list_worktrees(store.pool(), project.id).await?;
            let message = format!(
                "This will delete project '{}' with {} worktree(s) and all associated sessions.\nProject path: {}\nAre you sure? (yes/no): ",
                project_name,
                worktrees.len(),
                project.clone_path.display()
            );
            
            if !confirm_deletion(&message, args.force)? {
                println!("Deletion cancelled.");
                return Ok(());
            }
        }
        
        project_manager.delete(project.id).await?;
        info!(project = %project_name, "Project deleted");
        println!("Deleted project: {}", project_name);
    } else if let Some(worktree_id) = args.worktree {
        let worktree = crate::store::queries::get_worktree(store.pool(), worktree_id).await?;
        let message = format!(
            "This will delete worktree for branch '{}' and its associated session.\nWorktree path: {}\nAre you sure? (yes/no): ",
            worktree.branch,
            worktree.path.display()
        );
        
        if !confirm_deletion(&message, args.force)? {
            println!("Deletion cancelled.");
            return Ok(());
        }
        
        worktree_manager.delete(worktree_id).await?;
        info!(worktree_id, "Worktree deleted");
        println!("Deleted worktree: {}", worktree_id);
    } else {
        eprintln!("Must specify either --project or --worktree");
    }
    
    Ok(())
}

