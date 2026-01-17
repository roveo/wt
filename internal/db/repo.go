package db

import (
	"database/sql"
	"time"
)

// Repo represents a git repository tracked by wt
type Repo struct {
	ID           int64
	Path         string
	Name         string
	WorktreesDir string
	LastSyncedAt *time.Time
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

// UpsertRepo creates or updates a repository
func UpsertRepo(db *sql.DB, repo *Repo) error {
	query := `
		INSERT INTO repos (path, name, worktrees_dir, last_synced_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			worktrees_dir = excluded.worktrees_dir,
			last_synced_at = excluded.last_synced_at,
			deleted_at = NULL
		RETURNING id, created_at
	`
	return db.QueryRow(query, repo.Path, repo.Name, repo.WorktreesDir, repo.LastSyncedAt).
		Scan(&repo.ID, &repo.CreatedAt)
}

// GetRepoByPath retrieves a repository by its path
func GetRepoByPath(db *sql.DB, path string) (*Repo, error) {
	query := `
		SELECT id, path, name, worktrees_dir, last_synced_at, created_at, deleted_at
		FROM repos
		WHERE path = ? AND deleted_at IS NULL
	`
	repo := &Repo{}
	err := db.QueryRow(query, path).Scan(
		&repo.ID, &repo.Path, &repo.Name, &repo.WorktreesDir,
		&repo.LastSyncedAt, &repo.CreatedAt, &repo.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// GetRepoByID retrieves a repository by its ID
func GetRepoByID(db *sql.DB, id int64) (*Repo, error) {
	query := `
		SELECT id, path, name, worktrees_dir, last_synced_at, created_at, deleted_at
		FROM repos
		WHERE id = ? AND deleted_at IS NULL
	`
	repo := &Repo{}
	err := db.QueryRow(query, id).Scan(
		&repo.ID, &repo.Path, &repo.Name, &repo.WorktreesDir,
		&repo.LastSyncedAt, &repo.CreatedAt, &repo.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// ListRepos retrieves all non-deleted repositories
func ListRepos(db *sql.DB) ([]*Repo, error) {
	query := `
		SELECT id, path, name, worktrees_dir, last_synced_at, created_at, deleted_at
		FROM repos
		WHERE deleted_at IS NULL
		ORDER BY name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*Repo
	for rows.Next() {
		repo := &Repo{}
		err := rows.Scan(
			&repo.ID, &repo.Path, &repo.Name, &repo.WorktreesDir,
			&repo.LastSyncedAt, &repo.CreatedAt, &repo.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

// SoftDeleteRepo marks a repository as deleted
func SoftDeleteRepo(db *sql.DB, id int64) error {
	query := `UPDATE repos SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

// UpdateLastSynced updates the last synced timestamp for a repository
func UpdateLastSynced(db *sql.DB, id int64) error {
	query := `UPDATE repos SET last_synced_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}
