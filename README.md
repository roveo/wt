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

## License

MIT
