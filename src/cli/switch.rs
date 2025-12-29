use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::{ProjectManager, WorktreeManager, SessionManager};
use crate::backends::traits::{BackendKind, create_backend};
use crate::frontends::traits::detect_picker;
use crate::git;
use tracing::info;

#[derive(Args)]
pub struct SwitchArgs {
    /// Branch name to switch to
    pub branch: Option<String>,
    
    /// Project name (required if branch is specified)
    #[arg(long)]
    pub project: Option<String>,
    
    /// Custom path for the worktree
    #[arg(long)]
    pub path: Option<String>,
}

pub async fn run(args: SwitchArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let project_manager = ProjectManager::new(store.clone(), config.clone());
    let worktree_manager = WorktreeManager::new(store.clone(), config.clone());
    
    // Determine project
    let project = if let Some(project_name) = args.project {
        project_manager.get_by_name(&project_name).await?
    } else {
        // List projects and let user pick
        let projects = project_manager.list().await?;
        if projects.is_empty() {
            eprintln!("No projects found. Clone a repository first with: sesh clone <url>");
            return Ok(());
        }
        
        if projects.len() == 1 {
            projects.into_iter().next().unwrap()
        } else {
            // Use picker to select project
            let picker = detect_picker()?;
            let items: Vec<_> = projects
                .iter()
                .map(|p| crate::frontends::traits::PickerItem {
                    display: format!("{} ({})", p.display_name, p.name),
                    value: p.name.clone(),
                })
                .collect();
            
            let selected = picker.pick(items, Default::default()).await?;
            if selected.is_empty() {
                return Ok(());
            }
            
            project_manager.get_by_name(&selected[0]).await?
        }
    };
    
    // Determine branch
    let branch = if let Some(branch) = args.branch {
        branch
    } else {
        // List branches and let user pick
        let repo = git::open(&project.clone_path)?;
        let branches = git::list_all_branches(&repo)?;
        
        if branches.is_empty() {
            return Err(crate::error::Error::GitError {
                message: "No branches found".to_string(),
            });
        }
        
        let picker = detect_picker()?;
        let items: Vec<_> = branches
            .iter()
            .map(|b| crate::frontends::traits::PickerItem {
                display: b.clone(),
                value: b.clone(),
            })
            .collect();
        
        let selected = picker.pick(items, Default::default()).await?;
        if selected.is_empty() {
            return Ok(());
        }
        
        selected[0].clone()
    };
    
    // Ensure worktree exists
    let path = args.path.map(|p| p.into());
    let worktree = worktree_manager.switch_branch(project.id, &branch, path).await?;
    
    // Create backend and session manager
    let backend_kind = BackendKind::from_str(&config.session.backend)
        .unwrap_or(BackendKind::Tmux);
    let backend = create_backend(backend_kind);
    let session_manager = SessionManager::new(store, backend);
    
    // Switch to session
    session_manager.switch_to(worktree.id).await?;
    
    info!(project = %project.name, branch = %branch, "Switched to branch");
    
    Ok(())
}

