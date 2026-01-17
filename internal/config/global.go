package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the global wt configuration
type Config struct {
	// WorktreesDir is the directory pattern for storing worktrees.
	// Defaults to "../{repo_name}.worktrees" (sibling to main repo).
	WorktreesDir string `toml:"worktrees_dir"`

	Tmux TmuxConfig `toml:"tmux"`
}

// TmuxConfig holds tmux-related settings
type TmuxConfig struct {
	// Mode controls tmux integration behavior.
	// "disabled" - no tmux integration, just cd (default)
	// "window" - create/switch to a tmux window for the worktree
	Mode string `toml:"mode"`

	// Session is the tmux session name to use.
	// Empty means use current session (if in tmux) or no tmux (if not in tmux).
	// If set, wt will always use/create this dedicated session.
	Session string `toml:"session"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		WorktreesDir: "../{repo_name}.worktrees",
		Tmux: TmuxConfig{
			Mode:    "disabled",
			Session: "",
		},
	}
}

// Load reads the global config from ~/.config/wt/config.toml.
// If the file doesn't exist, returns default config.
func Load() (Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return DefaultConfig(), err
	}
	return LoadFrom(path)
}

// LoadFrom reads config from the specified path.
// If the file doesn't exist, returns default config.
func LoadFrom(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// DefaultPath returns the default config file path
func DefaultPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "wt", "config.toml"), nil
}
