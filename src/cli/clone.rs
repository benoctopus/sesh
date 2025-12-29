use clap::Args;
use crate::error::Result;
use crate::config;
use crate::store::Store;
use crate::managers::ProjectManager;
use tracing::info;

#[derive(Args)]
pub struct CloneArgs {
    /// Repository URL to clone
    pub url: String,
    
    /// Custom path for the clone (default: ~/.sesh/projects/{project_name})
    #[arg(long)]
    pub path: Option<String>,
}

pub async fn run(args: CloneArgs) -> Result<()> {
    let config = config::load()?;
    let store = Store::open().await?;
    let manager = ProjectManager::new(store, config);
    
    let path = args.path.map(|p| p.into());
    let project = manager.clone(&args.url, path).await?;
    
    info!(project = %project.name, "Project cloned successfully");
    println!("Cloned {} to {}", project.display_name, project.clone_path.display());
    
    Ok(())
}

