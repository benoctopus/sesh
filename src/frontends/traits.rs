use crate::error::Result;
use async_trait::async_trait;

#[derive(Debug, Clone)]
pub struct PickerItem {
    pub display: String,    // What the user sees
    pub value: String,      // What gets returned on selection
}

#[derive(Debug, Clone, Default)]
pub struct PickerOptions {
    pub prompt: Option<String>,
    pub header: Option<String>,
    pub preview_command: Option<String>,
    pub multi_select: bool,
}

#[async_trait]
pub trait Picker: Send + Sync {
    /// Get the picker name (for config/detection)
    fn name(&self) -> &'static str;
    
    /// Check if this picker is available on the system
    fn is_available(&self) -> bool;
    
    /// Run the picker with a list of items
    async fn pick(
        &self,
        items: Vec<PickerItem>,
        options: PickerOptions,
    ) -> Result<Vec<String>>;
}

/// Detect and create the best available picker
pub fn detect_picker() -> Result<Box<dyn Picker>> {
    // Try skim first (Rust-native), then fzf
    if crate::frontends::skim::SkimPicker::new().is_available() {
        return Ok(Box::new(crate::frontends::skim::SkimPicker::new()));
    }
    if crate::frontends::fzf::FzfPicker::new().is_available() {
        return Ok(Box::new(crate::frontends::fzf::FzfPicker::new()));
    }
    Err(crate::error::Error::NoPickerAvailable)
}

