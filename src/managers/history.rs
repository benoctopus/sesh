use crate::store::Store;
use crate::error::Result;
use crate::store::models::Session;

pub struct HistoryManager {
    store: Store,
}

impl HistoryManager {
    pub fn new(store: Store) -> Self {
        Self { store }
    }
    
    /// Get the previous session from history
    pub async fn get_previous(&self, current_session: Option<&str>) -> Result<Session> {
        crate::store::queries::get_previous_session(
            self.store.pool(),
            current_session,
        )
        .await
    }
    
    /// Add a session to history
    pub async fn add(&self, session_id: i64) -> Result<()> {
        crate::store::queries::add_to_history(self.store.pool(), session_id).await
    }
}

