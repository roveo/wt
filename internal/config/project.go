package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// StringOrSlice holds a value that can be either a string or slice of strings in TOML
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		*s = []string{v}
	case []any:
		*s = make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				(*s)[i] = str
			}
		}
	}
	return nil
}

// ProjectConfig represents per-project .wt.toml configuration
type ProjectConfig struct {
	// WorktreesDir overrides the global worktrees_dir for this project.
	WorktreesDir string `toml:"worktrees_dir"`

	// Setup is a shell command (or list of commands) to run after creating a new worktree.
	Setup StringOrSlice `toml:"setup"`

	// OnEnter is a command to run after cd-ing into the worktree (e.g. "nvim", "code .").
	OnEnter string `toml:"on_enter"`
}

// DefaultProjectConfig returns an empty ProjectConfig
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{}
}

// LoadProject reads the project config from .wt.toml in the given directory.
// If the file doesn't exist, returns empty config.
func LoadProject(repoRoot string) (ProjectConfig, error) {
	path := filepath.Join(repoRoot, ".wt.toml")
	return LoadProjectFrom(path)
}

// LoadProjectFrom reads project config from the specified path.
// If the file doesn't exist, returns empty config.
func LoadProjectFrom(path string) (ProjectConfig, error) {
	cfg := DefaultProjectConfig()

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
