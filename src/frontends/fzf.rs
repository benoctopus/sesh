use crate::error::Result;
use crate::frontends::traits::{Picker, PickerItem, PickerOptions};
use async_trait::async_trait;
use std::process::{Command, Stdio};
use tracing::debug;

pub struct FzfPicker;

impl FzfPicker {
    pub fn new() -> Self {
        Self
    }
}

#[async_trait]
impl Picker for FzfPicker {
    fn name(&self) -> &'static str {
        "fzf"
    }
    
    fn is_available(&self) -> bool {
        Command::new("fzf")
            .arg("--version")
            .output()
            .is_ok()
    }
    
    async fn pick(
        &self,
        items: Vec<PickerItem>,
        options: PickerOptions,
    ) -> Result<Vec<String>> {
        debug!(item_count = items.len(), "Running fzf picker");
        
        // Format items for fzf (display\tvalue)
        let input: String = items
            .iter()
            .map(|item| format!("{}\t{}", item.display, item.value))
            .collect::<Vec<_>>()
            .join("\n");
        
        tokio::task::spawn_blocking(move || {
            let mut cmd = Command::new("fzf");
            
            if options.multi_select {
                cmd.arg("--multi");
            }
            
            if let Some(prompt) = &options.prompt {
                cmd.arg("--prompt").arg(prompt);
            }
            
            if let Some(header) = &options.header {
                cmd.arg("--header").arg(header);
            }
            
            // Use tab separator and extract second field (value)
            cmd.arg("--delimiter").arg("\t");
            cmd.arg("--with-nth").arg("1"); // Show only display (first field)
            
            let mut child = cmd
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::null())
                .spawn()
                .map_err(|e| crate::error::Error::GitError {
                    message: format!("Failed to spawn fzf: {}", e),
                })?;
            
            // Write input to fzf
            use std::io::Write;
            if let Some(mut stdin) = child.stdin.take() {
                stdin.write_all(input.as_bytes())
                    .map_err(|e| crate::error::Error::GitError {
                        message: format!("Failed to write to fzf: {}", e),
                    })?;
            }
            
            let output = child.wait_with_output()
                .map_err(|e| crate::error::Error::GitError {
                    message: format!("Failed to wait for fzf: {}", e),
                })?;
            
            if !output.status.success() {
                // User cancelled
                return Ok(Vec::new());
            }
            
            // Parse output - extract values (second field after tab)
            let selected: Vec<String> = String::from_utf8_lossy(&output.stdout)
                .lines()
                .filter_map(|line| {
                    // Find the matching item and return its value
                    items.iter()
                        .find(|item| item.display == line.trim())
                        .map(|item| item.value.clone())
                })
                .collect();
            
            Ok(selected)
        })
        .await
        .map_err(|e| crate::error::Error::GitError {
            message: format!("Fzf picker task failed: {}", e),
        })?
    }
}

