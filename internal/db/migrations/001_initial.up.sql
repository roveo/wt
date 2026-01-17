CREATE TABLE repos (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    worktrees_dir TEXT NOT NULL,
    last_synced_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE worktrees (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    path TEXT UNIQUE NOT NULL,
    branch TEXT NOT NULL,
    is_main BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_worktrees_repo_id ON worktrees(repo_id);
CREATE INDEX idx_worktrees_deleted_at ON worktrees(deleted_at);
CREATE INDEX idx_repos_deleted_at ON repos(deleted_at);
