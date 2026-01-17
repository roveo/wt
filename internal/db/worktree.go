package db

import (
	"database/sql"
	"time"
)

// Worktree represents a git worktree tracked by wt
type Worktree struct {
	ID        int64
	RepoID    int64
	Path      string
	Branch    string
	IsMain    bool
	CreatedAt time.Time
	DeletedAt *time.Time

	// Joined fields (not stored in DB)
	RepoName string
	RepoPath string
}

// UpsertWorktree creates or updates a worktree
func UpsertWorktree(db *sql.DB, wt *Worktree) error {
	query := `
		INSERT INTO worktrees (repo_id, path, branch, is_main)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			repo_id = excluded.repo_id,
			branch = excluded.branch,
			is_main = excluded.is_main,
			deleted_at = NULL
		RETURNING id, created_at
	`
	return db.QueryRow(query, wt.RepoID, wt.Path, wt.Branch, wt.IsMain).
		Scan(&wt.ID, &wt.CreatedAt)
}

// GetWorktreeByPath retrieves a worktree by its path
func GetWorktreeByPath(db *sql.DB, path string) (*Worktree, error) {
	query := `
		SELECT w.id, w.repo_id, w.path, w.branch, w.is_main, w.created_at, w.deleted_at,
		       r.name, r.path
		FROM worktrees w
		JOIN repos r ON w.repo_id = r.id
		WHERE w.path = ? AND w.deleted_at IS NULL
	`
	wt := &Worktree{}
	err := db.QueryRow(query, path).Scan(
		&wt.ID, &wt.RepoID, &wt.Path, &wt.Branch, &wt.IsMain,
		&wt.CreatedAt, &wt.DeletedAt, &wt.RepoName, &wt.RepoPath,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return wt, nil
}

// ListWorktreesByRepo retrieves all non-deleted worktrees for a repository
func ListWorktreesByRepo(db *sql.DB, repoID int64) ([]*Worktree, error) {
	query := `
		SELECT w.id, w.repo_id, w.path, w.branch, w.is_main, w.created_at, w.deleted_at,
		       r.name, r.path
		FROM worktrees w
		JOIN repos r ON w.repo_id = r.id
		WHERE w.repo_id = ? AND w.deleted_at IS NULL
		ORDER BY w.is_main DESC, w.branch
	`
	return queryWorktrees(db, query, repoID)
}

// ListAllWorktrees retrieves all non-deleted worktrees across all repos
func ListAllWorktrees(db *sql.DB) ([]*Worktree, error) {
	query := `
		SELECT w.id, w.repo_id, w.path, w.branch, w.is_main, w.created_at, w.deleted_at,
		       r.name, r.path
		FROM worktrees w
		JOIN repos r ON w.repo_id = r.id
		WHERE w.deleted_at IS NULL AND r.deleted_at IS NULL
		ORDER BY r.name, w.is_main DESC, w.branch
	`
	return queryWorktrees(db, query)
}

// ListAllWorktreesWithRepoFirst retrieves all worktrees, with the specified repo's worktrees first
func ListAllWorktreesWithRepoFirst(db *sql.DB, currentRepoPath string) ([]*Worktree, error) {
	query := `
		SELECT w.id, w.repo_id, w.path, w.branch, w.is_main, w.created_at, w.deleted_at,
		       r.name, r.path
		FROM worktrees w
		JOIN repos r ON w.repo_id = r.id
		WHERE w.deleted_at IS NULL AND r.deleted_at IS NULL
		ORDER BY 
			CASE WHEN r.path = ? THEN 0 ELSE 1 END,
			r.name, 
			w.is_main DESC, 
			w.branch
	`
	return queryWorktrees(db, query, currentRepoPath)
}

func queryWorktrees(db *sql.DB, query string, args ...any) ([]*Worktree, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var worktrees []*Worktree
	for rows.Next() {
		wt := &Worktree{}
		err := rows.Scan(
			&wt.ID, &wt.RepoID, &wt.Path, &wt.Branch, &wt.IsMain,
			&wt.CreatedAt, &wt.DeletedAt, &wt.RepoName, &wt.RepoPath,
		)
		if err != nil {
			return nil, err
		}
		worktrees = append(worktrees, wt)
	}
	return worktrees, rows.Err()
}

// SoftDeleteWorktree marks a worktree as deleted
func SoftDeleteWorktree(db *sql.DB, id int64) error {
	query := `UPDATE worktrees SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

// SoftDeleteWorktreeByPath marks a worktree as deleted by its path
func SoftDeleteWorktreeByPath(db *sql.DB, path string) error {
	query := `UPDATE worktrees SET deleted_at = CURRENT_TIMESTAMP WHERE path = ?`
	_, err := db.Exec(query, path)
	return err
}

// SoftDeleteMissingWorktrees marks worktrees as deleted if they're not in the provided list of paths
func SoftDeleteMissingWorktrees(db *sql.DB, repoID int64, existingPaths []string) error {
	if len(existingPaths) == 0 {
		// Mark all worktrees for this repo as deleted
		query := `UPDATE worktrees SET deleted_at = CURRENT_TIMESTAMP WHERE repo_id = ? AND deleted_at IS NULL`
		_, err := db.Exec(query, repoID)
		return err
	}

	// Build a query with placeholders for the existing paths
	query := `UPDATE worktrees SET deleted_at = CURRENT_TIMESTAMP WHERE repo_id = ? AND deleted_at IS NULL AND path NOT IN (`
	args := make([]any, 0, len(existingPaths)+1)
	args = append(args, repoID)
	for i, p := range existingPaths {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args = append(args, p)
	}
	query += ")"

	_, err := db.Exec(query, args...)
	return err
}
