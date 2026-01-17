package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// IsInsideRepo checks if the given path is inside a git repository
func IsInsideRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// GetRepoRoot returns the root directory of the current worktree
// Note: When inside a worktree, this returns the worktree's root, not the main repo
func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetMainRepoPath returns the path to the main repository
// This works correctly even when called from inside a worktree
func GetMainRepoPath(path string) (string, error) {
	// Get the common git dir (points to main repo's .git)
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	gitDir := strings.TrimSpace(string(output))

	// The main repo path is the parent of .git directory
	// Handle both absolute and relative paths
	if !filepath.IsAbs(gitDir) {
		absPath, err := filepath.Abs(filepath.Join(path, gitDir))
		if err != nil {
			return "", err
		}
		gitDir = absPath
	}

	// Remove trailing .git to get repo root
	mainRepoPath := filepath.Dir(gitDir)
	return mainRepoPath, nil
}

// GetRepoName returns the name of the repository (directory name)
func GetRepoName(repoPath string) string {
	return filepath.Base(repoPath)
}

// GetDefaultWorktreesDir returns the default worktrees directory for a repo
// Convention: ../{repo-name}.worktrees/
func GetDefaultWorktreesDir(repoPath string) string {
	name := GetRepoName(repoPath)
	parent := filepath.Dir(repoPath)
	return filepath.Join(parent, name+".worktrees")
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ListRemoteBranches returns a list of remote branches
func ListRemoteBranches(path string) ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove origin/ prefix if present
		if strings.HasPrefix(line, "origin/") {
			line = strings.TrimPrefix(line, "origin/")
		}
		// Skip HEAD pointer
		if line == "HEAD" || strings.Contains(line, "->") {
			continue
		}
		branches = append(branches, line)
	}
	return branches, nil
}

// ListLocalBranches returns a list of local branches
func ListLocalBranches(path string) ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// Fetch fetches from the remote
func Fetch(path string) error {
	cmd := exec.Command("git", "fetch", "--all", "--prune")
	cmd.Dir = path
	return cmd.Run()
}
