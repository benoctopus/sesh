use crate::error::Result;
use crate::frontends::traits::{Picker, PickerItem, PickerOptions};
use async_trait::async_trait;
use skim::prelude::*;
use std::io::Cursor;
use std::process::Command;
use tracing::debug;

pub struct SkimPicker;

impl SkimPicker {
    pub fn new() -> Self {
        Self
    }
}

#[async_trait]
impl Picker for SkimPicker {
    fn name(&self) -> &'static str {
        "skim"
    }
    
    fn is_available(&self) -> bool {
        // Check if skim binary is available
        Command::new("sk")
            .arg("--version")
            .output()
            .is_ok()
    }
    
    async fn pick(
        &self,
        items: Vec<PickerItem>,
        options: PickerOptions,
    ) -> Result<Vec<String>> {
        debug!(item_count = items.len(), "Running skim picker");
        
        // Format items for skim (display\tvalue)
        let input: String = items
            .iter()
            .map(|item| format!("{}\t{}", item.display, item.value))
            .collect::<Vec<_>>()
            .join("\n");
        
        let mut skim_options = SkimOptionsBuilder::default()
            .height("50%".to_string())
            .multi(options.multi_select)
            .build()
            .map_err(|e| crate::error::Error::GitError {
                message: format!("Failed to build skim options: {}", e),
            })?;
        
        if let Some(prompt) = &options.prompt {
            skim_options.prompt = prompt.clone().into();
        }
        
        if let Some(header) = &options.header {
            skim_options.header = Some(header.clone());
        }
        
        let item_reader = SkimItemReader::default();
        let items = item_reader.of_bufread(Cursor::new(input));
        
        let selected_items = Skim::run_with(&skim_options, Some(items))
            .map(|output| {
                output
                    .selected_items
                    .iter()
                    .map(|item| {
                        // Extract value (after tab)
                        let text = item.output().to_string();
                        text.split('\t')
                            .nth(1)
                            .unwrap_or(&text)
                            .to_string()
                    })
                    .collect()
            })
            .unwrap_or_default();
        
        Ok(selected_items)
    }
}

