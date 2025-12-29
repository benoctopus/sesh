use sesh::cli;
use sesh::logging;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize logging before anything else
    let verbose = std::env::var("SESH_LOG").is_ok();
    logging::init(verbose)?;
    
    // Parse CLI and run command
    cli::run().await
}

