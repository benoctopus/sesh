use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::backends::traits::{BackendKind, create_backend};

pub async fn run() -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let backend_kind = BackendKind::from_str(&config.session.backend)
        .unwrap_or(BackendKind::Tmux);
    let backend = create_backend(backend_kind);
    
    if backend.is_inside_session() {
        if let Some(session_name) = backend.current_session() {
            println!("Current session: {}", session_name);
            
            // Get session details from database
            if let Some(session) = crate::store::queries::get_session_by_name(
                store.pool(),
                &session_name,
            )
            .await? {
                let worktree = crate::store::queries::get_worktree(
                    store.pool(),
                    session.worktree_id,
                )
                .await?;
                let project = crate::store::queries::get_project(
                    store.pool(),
                    worktree.project_id,
                )
                .await?;
                
                println!("  Project: {}", project.display_name);
                println!("  Branch: {}", worktree.branch);
                println!("  Path: {}", worktree.path.display());
            }
        } else {
            println!("Inside session but name unknown");
        }
    } else {
        println!("Not inside a session");
    }
    
    Ok(())
}

