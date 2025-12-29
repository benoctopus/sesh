use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::SessionManager;
use crate::backends::traits::{BackendKind, create_backend};

pub async fn run() -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    
    let backend_kind = BackendKind::from_str(&config.session.backend)
        .unwrap_or(BackendKind::Tmux);
    let backend = create_backend(backend_kind);
    let session_manager = SessionManager::new(store, backend);
    
    session_manager.pop().await?;
    
    Ok(())
}

