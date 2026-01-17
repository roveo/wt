package git

import (
	"bufio"
	"os/exec"
	"strings"
)

// WorktreeInfo contains information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	IsMain bool
}

// ListWorktrees returns all worktrees for the repository at the given path
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWorktreeList(string(output))
}

func parseWorktreeList(output string) ([]WorktreeInfo, error) {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			// Start of a new worktree entry
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}

		case strings.HasPrefix(line, "branch "):
			// Branch reference (e.g., "branch refs/heads/main")
			branch := strings.TrimPrefix(line, "branch ")
			branch = strings.TrimPrefix(branch, "refs/heads/")
			current.Branch = branch

		case line == "bare":
			// Bare repository, skip
			current.Path = ""

		case strings.HasPrefix(line, "detached"):
			// Detached HEAD
			current.Branch = "(detached)"

		case line == "":
			// Empty line marks end of entry
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
		}
	}

	// Don't forget the last entry
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	// The first worktree in git's output is always the main worktree
	if len(worktrees) > 0 {
		worktrees[0].IsMain = true
	}

	return worktrees, scanner.Err()
}

// AddWorktree creates a new worktree for the given branch
func AddWorktree(repoPath, branch, targetPath string) error {
	// First, try to create from an existing branch
	cmd := exec.Command("git", "worktree", "add", targetPath, branch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// If that fails, try creating a new branch from the remote
		cmd = exec.Command("git", "worktree", "add", "-b", branch, targetPath, "origin/"+branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			// Last resort: create new branch from current HEAD
			cmd = exec.Command("git", "worktree", "add", "-b", branch, targetPath)
			cmd.Dir = repoPath
			return cmd.Run()
		}
	}
	return nil
}

// RemoveWorktree removes a worktree
func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = repoPath
	return cmd.Run()
}

// RemoveWorktreeForce forcefully removes a worktree
func RemoveWorktreeForce(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = repoPath
	return cmd.Run()
}

// PruneWorktrees removes stale worktree entries
func PruneWorktrees(repoPath string) error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = repoPath
	return cmd.Run()
}
