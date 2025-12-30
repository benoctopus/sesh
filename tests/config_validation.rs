use sesh::config::Config;

#[test]
fn test_valid_config() {
    let config = Config::default();
    assert!(config.validate().is_ok());
}

#[test]
fn test_invalid_backend() {
    let mut config = Config::default();
    config.session.backend = "invalid_backend".to_string();
    assert!(config.validate().is_err());
}

#[test]
fn test_invalid_finder() {
    let mut config = Config::default();
    config.picker.finder = "invalid_finder".to_string();
    assert!(config.validate().is_err());
}

#[test]
fn test_valid_tmux_backend() {
    let mut config = Config::default();
    config.session.backend = "tmux".to_string();
    assert!(config.validate().is_ok());
}

#[test]
fn test_valid_code_backends() {
    let backends = vec![
        "code",
        "code:open",
        "code:workspace",
        "code:replace",
    ];
    
    for backend in backends {
        let mut config = Config::default();
        config.session.backend = backend.to_string();
        assert!(
            config.validate().is_ok(),
            "Backend '{}' should be valid",
            backend
        );
    }
}

#[test]
fn test_valid_cursor_backends() {
    let backends = vec![
        "cursor",
        "cursor:open",
        "cursor:workspace",
        "cursor:replace",
    ];
    
    for backend in backends {
        let mut config = Config::default();
        config.session.backend = backend.to_string();
        assert!(
            config.validate().is_ok(),
            "Backend '{}' should be valid",
            backend
        );
    }
}

#[test]
fn test_valid_finders() {
    let finders = vec!["auto", "fzf", "skim"];
    
    for finder in finders {
        let mut config = Config::default();
        config.picker.finder = finder.to_string();
        assert!(
            config.validate().is_ok(),
            "Finder '{}' should be valid",
            finder
        );
    }
}

#[test]
fn test_tilde_expansion_in_paths() {
    let mut config = Config::default();
    config.workspace.projects_dir = "~/projects".to_string();
    config.workspace.worktrees_dir = "~/worktrees".to_string();
    
    // Should validate without error
    assert!(config.validate().is_ok());
    
    // Should expand tilde
    let projects_path = config.projects_dir();
    let worktrees_path = config.worktrees_dir();
    
    assert!(!projects_path.to_string_lossy().contains('~'));
    assert!(!worktrees_path.to_string_lossy().contains('~'));
}

