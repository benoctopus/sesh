use sesh::store::Store;
use tempfile::TempDir;

#[tokio::test]
async fn test_store_creation() {
    let temp_dir = TempDir::new().unwrap();
    let db_path = temp_dir.path().join("test.db");
    
    // This test verifies the store can be created and migrations run
    // We'll need to modify Store::open to accept a path for testing
    // For now, this is a placeholder
}

