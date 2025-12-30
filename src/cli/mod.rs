pub mod clean;
pub mod clone;
pub mod delete;
pub mod edit;
pub mod list;
pub mod logs;
pub mod pop;
pub mod status;
pub mod switch;

use anyhow::Result;
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "sesh")]
#[command(about = "A session manager for git worktrees")]
#[command(version)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Enable verbose logging to terminal
    #[arg(long, global = true)]
    pub verbose: bool,

    /// Override config directory (for testing)
    #[arg(long, global = true)]
    pub config_dir: Option<String>,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Clone a repository and create initial session
    #[command(visible_alias = "cl")]
    Clone(clone::CloneArgs),

    /// Switch to a branch (create worktree/session if needed)
    #[command(visible_alias = "sw")]
    Switch(switch::SwitchArgs),

    /// List projects, worktrees, or sessions
    #[command(visible_aliases = ["ls"])]
    List(list::ListArgs),

    /// Delete a project, worktree, or session
    #[command(visible_aliases = ["del", "remove", "rm"])]
    Delete(delete::DeleteArgs),

    /// Clean up stale entries
    Clean(clean::CleanArgs),

    /// Pop to previous session from history
    #[command(visible_aliases = ["p", "back", "last"])]
    Pop,

    /// Show current session status
    Status,

    /// View or follow log files
    Logs(logs::LogsArgs),

    /// Edit the sesh configuration file
    Edit,

    /// Validate state and diagnose issues
    Doctor,

    /// Generate shell completion script
    Completions {
        /// Shell to generate completions for
        #[arg(value_enum)]
        shell: clap_complete::Shell,
    },
}

pub async fn run() -> anyhow::Result<()> {
    let cli = Cli::parse();

    // Set config dir override if provided
    if let Some(config_dir) = &cli.config_dir {
        std::env::set_var("SESH_CONFIG_DIR_OVERRIDE", config_dir);
    }

    match cli.command {
        Commands::Clone(args) => clone::run(args).await.map_err(Into::into),
        Commands::Switch(args) => switch::run(args).await.map_err(Into::into),
        Commands::List(args) => list::run(args).await.map_err(Into::into),
        Commands::Delete(args) => delete::run(args).await.map_err(Into::into),
        Commands::Clean(args) => clean::run(args).await.map_err(Into::into),
        Commands::Pop => pop::run().await.map_err(Into::into),
        Commands::Status => status::run().await.map_err(Into::into),
        Commands::Logs(args) => logs::run(args).await.map_err(Into::into),
        Commands::Edit => edit::run().await.map_err(Into::into),
        Commands::Doctor => {
            eprintln!("Doctor command not yet implemented");
            Ok(())
        }
        Commands::Completions { shell } => generate_completions(shell),
    }
}

/// Generate shell completions
pub fn generate_completions(shell: clap_complete::Shell) -> Result<()> {
    use clap::CommandFactory;
    use std::io;
    let mut cmd = Cli::command();
    clap_complete::generate(shell, &mut cmd, "sesh", &mut io::stdout());
    Ok(())
}
