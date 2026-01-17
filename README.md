# wt

Lightweight Git worktree manager with an interactive TUI.

```
wt
```

That's it. Run `wt` and start managing worktrees.

## Features

- **Zero config** - just run `wt` inside any git repo
- **Fuzzy finder** - fzf-style interactive picker for switching worktrees
- **Multi-repo support** - tracks all your repositories in one place
- **Smart defaults** - worktrees go to `../{repo}.worktrees/{branch}`
- **Shell integration** - `cd` into worktrees seamlessly

## Installation

```bash
go install github.com/roveo/wt@latest
```

Add shell integration to your rc file:

```bash
# bash
eval "$(wt init bash)"

# zsh
eval "$(wt init zsh)"

# fish
wt init fish | source
```

## Usage

### Interactive mode

```bash
wt              # Open fuzzy finder to switch worktrees
```

**Keybindings:**
- `enter` - switch to selected worktree
- `tab` - create new worktree from selected repo
- `ctrl-d` - delete selected worktree
- `esc` - quit

### Commands

```bash
wt add <branch>     # Add a new worktree
wt remove [path]    # Remove a worktree
wt list             # List all tracked worktrees
```

## How it works

`wt` maintains a SQLite database tracking your repositories and worktrees.
When you run `wt`:

1. **Sync** - indexes current repo (if in one), syncs all tracked repos
2. **Display** - shows interactive picker with your worktrees

Current repo's worktrees appear first in the list.

## Worktree location

Worktrees are created at `../{repo}.worktrees/{branch}`:

```
~/projects/
  myapp/                    # main repo
  myapp.worktrees/
    feature-auth/           # wt add feature/auth
    fix-bug-123/            # wt add fix/bug-123
```

## Configuration

### Global config

`~/.config/wt/config.toml`:

```toml
# Default worktree directory pattern (supports {repo_name} placeholder)
worktrees_dir = "../{repo_name}.worktrees"

[tmux]
# "disabled" - just cd (default)
# "window" - create/switch to a tmux window per worktree
mode = "disabled"

# Optional: dedicated tmux session for all worktrees
# If set, wt will always use/create this session
# If not in tmux, outputs "tmux attach -t <session>" for shell to eval
session = ""
```

#### tmux integration

When `mode = "window"`:
- **In tmux**: Creates a new window named `{repo}:{branch}` or switches to it if it already exists
- **Not in tmux**: Falls back to regular cd behavior

When `session` is set (e.g., `session = "wt"`):
- **In tmux**: Switches to the dedicated session, then creates/switches window
- **Not in tmux**: Creates the session if needed, creates the window, outputs `tmux attach -t wt`

Example for a dedicated worktree session:
```toml
[tmux]
mode = "window"
session = "wt"
```

### Per-project config

`.wt.toml` in your repo root:

```toml
# Override worktree location for this project
worktrees_dir = "../myproject.worktrees"

# Command to run after cd-ing into a worktree (e.g. open editor)
on_enter = "nvim"

# Setup command(s) to run after creating a new worktree
setup = "npm install"

# Or multiple commands
setup = ["npm install", "npm run build"]
```

## License

MIT
