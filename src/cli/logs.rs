use clap::Args;
use crate::error::Result;
use crate::logging;
use std::fs;
use std::io::{BufRead, BufReader};

#[derive(Args)]
pub struct LogsArgs {
    /// Follow log file (like tail -f)
    #[arg(short, long)]
    pub follow: bool,
    
    /// Show logs from a specific date (YYYY-MM-DD)
    #[arg(long)]
    pub date: Option<String>,
    
    /// Number of lines to show
    #[arg(short, long, default_value = "100")]
    pub lines: usize,
}

pub async fn run(args: LogsArgs) -> Result<()> {
    let log_dir = logging::log_dir()?;
    
    let log_file = if let Some(date) = &args.date {
        log_dir.join(format!("sesh.log.{}", date))
    } else {
        log_dir.join("sesh.log")
    };
    
    if !log_file.exists() {
        eprintln!("Log file not found: {}", log_file.display());
        return Ok(());
    }
    
    if args.follow {
        // For following, we'd need a more sophisticated implementation
        // For now, just show recent lines
        eprintln!("Follow mode not yet implemented, showing recent lines");
    }
    
    let file = fs::File::open(&log_file)?;
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<std::result::Result<_, _>>()?;
    
    let start = if lines.len() > args.lines {
        lines.len() - args.lines
    } else {
        0
    };
    
    for line in &lines[start..] {
        println!("{}", line);
    }
    
    Ok(())
}

