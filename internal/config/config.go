package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/rotisserie/eris"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	WorkspaceDir   string `yaml:"workspace_dir"`
	SessionBackend string `yaml:"session_backend"` // "tmux", "zellij", "screen", "auto"
	StartupCommand string `yaml:"startup_command"` // Command to run on session creation
	FuzzyFinder    string `yaml:"fuzzy_finder"`    // "fzf", "peco", "auto"
}

// configFile represents the YAML config file structure
type configFile struct {
	Version        string `yaml:"version"`
	WorkspaceDir   string `yaml:"workspace_dir"`
	SessionBackend string `yaml:"session_backend"`
	StartupCommand string `yaml:"startup_command"`
	FuzzyFinder    string `yaml:"fuzzy_finder"`
}

const (
	// CurrentConfigVersion is the current version of the config file format
	CurrentConfigVersion = "1"
)

// ProjectConfig holds project-specific configuration
type ProjectConfig struct {
	StartupCommand string `yaml:"startup_command"`
}

// GetConfigDir returns the OS-specific config directory for sesh
func GetConfigDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", eris.Wrap(err, "failed to get user home directory")
		}
		baseDir = filepath.Join(home, "Library", "Application Support")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", eris.New("APPDATA environment variable not set")
		}
		baseDir = appData
	default: // linux and others
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			baseDir = xdgConfigHome
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", eris.Wrap(err, "failed to get user home directory")
			}
			baseDir = filepath.Join(home, ".config")
		}
	}

	return filepath.Join(baseDir, "sesh"), nil
}

// GetWorkspaceDir returns the workspace directory with configuration hierarchy
func GetWorkspaceDir() (string, error) {
	// 1. Environment variable (highest priority)
	if envDir := os.Getenv("SESH_WORKSPACE"); envDir != "" {
		return expandHome(envDir)
	}

	// 2. Config file
	config, err := loadConfigFile()
	if err == nil && config.WorkspaceDir != "" {
		return expandHome(config.WorkspaceDir)
	}

	// 3. Default (lowest priority)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get user home directory")
	}

	return filepath.Join(home, ".sesh"), nil
}

// GetSessionBackend returns the session backend with configuration hierarchy
func GetSessionBackend() (string, error) {
	// 1. Environment variable (highest priority)
	if envBackend := os.Getenv("SESH_SESSION_BACKEND"); envBackend != "" {
		return envBackend, nil
	}

	// 2. Config file
	config, err := loadConfigFile()
	if err == nil && config.SessionBackend != "" {
		return config.SessionBackend, nil
	}

	// 3. Auto-detect (default)
	return "auto", nil
}

// GetFuzzyFinder returns the fuzzy finder with configuration hierarchy
func GetFuzzyFinder() (string, error) {
	// 1. Environment variable (highest priority)
	if envFinder := os.Getenv("SESH_FUZZY_FINDER"); envFinder != "" {
		return envFinder, nil
	}

	// 2. Config file
	config, err := loadConfigFile()
	if err == nil && config.FuzzyFinder != "" {
		return config.FuzzyFinder, nil
	}

	// 3. Auto-detect (default)
	return "auto", nil
}

// GetDBPath returns the full path to the SQLite database
func GetDBPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get config directory")
	}

	return filepath.Join(configDir, "sesh.db"), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return eris.Wrap(err, "failed to get config directory")
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return eris.Wrapf(err, "failed to create config directory: %s", configDir)
	}

	return nil
}

// EnsureWorkspaceDir creates the workspace directory if it doesn't exist
func EnsureWorkspaceDir() error {
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return eris.Wrap(err, "failed to get workspace directory")
	}

	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return eris.Wrapf(err, "failed to create workspace directory: %s", workspaceDir)
	}

	return nil
}

// LoadConfig loads the full configuration with all settings resolved
func LoadConfig() (*Config, error) {
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get workspace directory")
	}

	sessionBackend, err := GetSessionBackend()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get session backend")
	}

	startupCommand, err := GetStartupCommand("")
	if err != nil {
		return nil, eris.Wrap(err, "failed to get startup command")
	}

	fuzzyFinder, err := GetFuzzyFinder()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get fuzzy finder")
	}

	return &Config{
		WorkspaceDir:   workspaceDir,
		SessionBackend: sessionBackend,
		StartupCommand: startupCommand,
		FuzzyFinder:    fuzzyFinder,
	}, nil
}

// GetStartupCommand returns the startup command with configuration hierarchy
// Priority: per-project config > global config > empty string
func GetStartupCommand(projectPath string) (string, error) {
	// 1. Check per-project config (highest priority)
	if projectPath != "" {
		projectConfig, err := LoadProjectConfig(projectPath)
		if err == nil && projectConfig.StartupCommand != "" {
			return projectConfig.StartupCommand, nil
		}
	}

	// 2. Check global config
	config, err := loadConfigFile()
	if err == nil && config.StartupCommand != "" {
		return config.StartupCommand, nil
	}

	// 3. Default to empty (no startup command)
	return "", nil
}

// LoadProjectConfig loads project-specific configuration from .sesh.yaml in the project directory
func LoadProjectConfig(projectPath string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectPath, ".sesh.yaml")

	// If config file doesn't exist, return empty config (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &ProjectConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read project config file: %s", configPath)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, eris.Wrapf(err, "failed to parse project config file: %s", configPath)
	}

	return &config, nil
}

// loadConfigFile loads the config file from disk (internal helper)
func loadConfigFile() (*configFile, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get config directory")
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// If config file doesn't exist, return empty config (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &configFile{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read config file: %s", configPath)
	}

	var config configFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, eris.Wrapf(err, "failed to parse config file: %s", configPath)
	}

	return &config, nil
}

// expandHome expands ~ to the user's home directory in a path
func expandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get user home directory")
	}

	if len(path) == 1 {
		return home, nil
	}

	if path[1] == '/' || path[1] == filepath.Separator {
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get config directory")
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return eris.Wrap(err, "failed to get config path")
	}

	// Ensure config directory exists
	if err := EnsureConfigDir(); err != nil {
		return eris.Wrap(err, "failed to ensure config directory")
	}

	// Convert to configFile structure with version
	cf := configFile{
		Version:        CurrentConfigVersion,
		WorkspaceDir:   config.WorkspaceDir,
		SessionBackend: config.SessionBackend,
		StartupCommand: config.StartupCommand,
		FuzzyFinder:    config.FuzzyFinder,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&cf)
	if err != nil {
		return eris.Wrap(err, "failed to marshal config to YAML")
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return eris.Wrapf(err, "failed to write config file: %s", configPath)
	}

	return nil
}

// ValidateConfig validates the configuration settings
func ValidateConfig(config *configFile) error {
	// Validate fuzzy finder
	if config.FuzzyFinder != "" && config.FuzzyFinder != "auto" {
		validFinders := []string{"fzf", "peco"}
		valid := false
		for _, finder := range validFinders {
			if config.FuzzyFinder == finder {
				valid = true
				break
			}
		}
		if !valid {
			return eris.Errorf("invalid fuzzy_finder: %s (must be one of: auto, fzf, peco)", config.FuzzyFinder)
		}
	}

	// Validate session backend
	if config.SessionBackend != "" && config.SessionBackend != "auto" {
		validBackends := []string{"tmux", "zellij", "screen"}
		valid := false
		for _, backend := range validBackends {
			if config.SessionBackend == backend {
				valid = true
				break
			}
		}
		if !valid {
			return eris.Errorf("invalid session_backend: %s (must be one of: auto, tmux, zellij, screen)", config.SessionBackend)
		}
	}

	// Validate workspace directory (if provided, it should be expandable)
	if config.WorkspaceDir != "" {
		_, err := expandHome(config.WorkspaceDir)
		if err != nil {
			return eris.Wrap(err, "invalid workspace_dir")
		}
	}

	return nil
}

// ValidateConfigFile validates a config file at the given path
func ValidateConfigFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return eris.Wrapf(err, "failed to read config file: %s", configPath)
	}

	var config configFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return eris.Wrapf(err, "failed to parse config file: %s", configPath)
	}

	// Validate version (for future compatibility)
	if config.Version != "" && config.Version != CurrentConfigVersion {
		// In the future, we can handle migrations here
		// For now, we'll just warn but not fail
	}

	return ValidateConfig(&config)
}
