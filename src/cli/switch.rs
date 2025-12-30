use crate::backends::traits::{create_backend, BackendKind};
use crate::config;
use crate::error::Result;
use crate::frontends::traits::detect_picker;
use crate::git;
use crate::managers::{ProjectManager, SessionManager, WorktreeManager};
use crate::store::Store;
use clap::Args;
use tracing::info;

#[derive(Args)]
pub struct SwitchArgs {
    /// Branch name to switch to
    pub branch: Option<String>,

    /// Project name (required if branch is specified)
    #[arg(short = 'p', long)]
    pub project: Option<String>,

    /// Custom path for the worktree
    #[arg(long)]
    pub path: Option<String>,

    /// Command to run after switching to session
    #[arg(short = 'c', long)]
    pub command: Option<String>,

    /// Select from open pull requests
    #[arg(long)]
    pub pr: bool,

    /// Create session without attaching to it
    #[arg(short = 'd', long)]
    pub detach: bool,

    /// Show local git branches (not just existing worktrees)
    #[arg(long)]
    pub local: bool,

    /// Show all branches including remote (current behavior)
    #[arg(long)]
    pub remote: bool,
}

pub async fn run(args: SwitchArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;

    let project_manager = ProjectManager::new(store.clone(), config.clone());
    let worktree_manager = WorktreeManager::new(store.clone(), config.clone());

    // Determine project
    let project = if let Some(project_name) = args.project {
        crate::store::queries::find_project_by_name(store.pool(), &project_name).await?
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
        // Determine which branches to show based on flags
        let branches = if args.remote {
            // Show all branches (local + remote)
            let repo = git::open(&project.clone_path)?;
            git::list_all_branches(&repo)?
        } else if args.local {
            // Show only local git branches
            let repo = git::open(&project.clone_path)?;
            git::list_local_branches(&repo)?
        } else {
            // Default: show only existing worktrees
            let worktrees = worktree_manager.list(project.id).await?;
            worktrees.into_iter().map(|w| w.branch).collect()
        };

        if branches.is_empty() {
            let message = if args.remote || args.local {
                "No branches found".to_string()
            } else {
                "No worktrees found for this project. Use --local to see local branches or --remote to see all branches.".to_string()
            };
            return Err(crate::error::Error::GitError { message });
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
    let worktree = worktree_manager
        .switch_branch(project.id, &branch, path)
        .await?;

    // Create backend and session manager
    let backend_kind = BackendKind::from_str(&config.session.backend).unwrap_or(BackendKind::Tmux);
    let backend = create_backend(backend_kind);
    let session_manager = SessionManager::new(store, backend);

    // Switch to session (with optional detach)
    session_manager
        .switch_to_with_detach(worktree.id, args.detach)
        .await?;

    info!(project = %project.name, branch = %branch, "Switched to branch");

    Ok(())
}
