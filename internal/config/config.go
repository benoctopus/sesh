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
}

// configFile represents the YAML config file structure
type configFile struct {
	WorkspaceDir   string `yaml:"workspace_dir"`
	SessionBackend string `yaml:"session_backend"`
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

	if err := os.MkdirAll(configDir, 0755); err != nil {
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

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
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

	return &Config{
		WorkspaceDir:   workspaceDir,
		SessionBackend: sessionBackend,
	}, nil
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
