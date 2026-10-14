<div align="center">

# gv

**One dashboard for every git repo on your machine.**

A terminal UI that discovers your repositories and worktrees,<br>
then shows which are dirty, unpushed, behind, or mid-merge as they change.

[![Test](https://github.com/jackchuka/gv/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/gv/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/jackchuka/gv?sort=semver)](https://github.com/jackchuka/gv/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<img src="assets/demo.gif" alt="gv showing five repositories, filtering by state, and picking up a new file in the background" width="800">

[Installation](#installation) · [Quick Start](#quick-start) · [Usage](#usage) · [Configuration](#configuration) · [Keybindings](#keybindings)

</div>

---

If you keep dozens of clones and worktrees around, it is easy to lose track of the one with uncommitted work, the branch you never pushed, or the merge you left half-finished. Running `git status` in each directory does not scale.

**gv** (Git Vision) scans the directories you point it at, finds every repository and linked worktree, and keeps a live table of their state. Filter down to what needs attention, open the repo in your editor or a shell, and move on.

## Features

- **Auto-discovery** — finds repositories and linked worktrees under your scan paths, skipping `node_modules`, `vendor`, build output, and similar directories
- **Live status** — branch, staged/modified/untracked counts, and ahead/behind against the upstream
- **Diff insights** — lines added and removed, net delta, and the most-changed files per repo
- **Activity sparklines** — commits per day over the last 7 days, per repo and in total
- **Worktree aware** — linked worktrees are grouped under their main repository
- **In-progress operations** — flags merges, rebases, cherry-picks, reverts, and bisects
- **Background polling** — refreshes changed repos automatically on a configurable interval
- **Quick actions** — fetch, open in `$EDITOR`, open a shell, reveal in the file manager, or copy the path
- **Vim-style navigation** — `j`/`k`, half-page scrolling, and `/` to filter

## Installation

### Homebrew

```bash
brew install jackchuka/tap/gv
```

### Go

```bash
go install github.com/jackchuka/gv@latest
```

### Binary

Download a prebuilt binary for Linux, macOS, or Windows from the [releases page](https://github.com/jackchuka/gv/releases).

### From Source

```bash
git clone https://github.com/jackchuka/gv.git
cd gv
make install
```

`git` must be on your `PATH` at runtime.

## Quick Start

1. **Create a config** with the interactive wizard. Enter the directories where you keep your repos:

   ```bash
   gv init
   ```

   It writes `~/.config/gv/config.yaml` and asks before overwriting an existing one.

2. **Launch the dashboard:**

   ```bash
   gv
   ```

To try it without a config, pass the paths directly:

```bash
gv --scan ~/code --scan ~/work
```

## Usage

```bash
gv                          # launch the dashboard
gv init                     # interactive config setup
gv --scan ~/code            # scan these paths instead of the configured ones
gv --config ./gv.yaml       # use a different config file (also applies to init)
gv --version                # print the version
```

| Flag              | Description                                                                                              |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| `--config`        | Config file path (default `$XDG_CONFIG_HOME/gv/config.yaml`, falling back to `~/.config/gv/config.yaml`) |
| `-s`, `--scan`    | Paths to scan; overrides `scan_paths`. Repeat the flag or separate with commas; `~` is expanded          |
| `-v`, `--version` | Print the version and exit                                                                               |

## Configuration

```yaml
scan_paths:
  - ~/code
  - ~/work

ignore_patterns:
  - "**/node_modules/**"
  - "**/vendor/**"
  - "**/.venv/**"

max_depth: 10 # how deep to look for repos below each scan path
poll_interval: 5s # how often to check repos for changes
auto_refresh: true # turn background polling on or off
```

| Key               | Default   | Description                                     |
| ----------------- | --------- | ----------------------------------------------- |
| `scan_paths`      | none      | Directories to search; `~` is expanded          |
| `ignore_patterns` | see below | Directories to skip, as `**/<name>/**` patterns |
| `max_depth`       | `10`      | Maximum directory depth below each scan path    |
| `poll_interval`   | `5s`      | Polling interval, as a Go duration (`2s`, `1m`) |
| `auto_refresh`    | `true`    | Poll for changes in the background              |

Setting `ignore_patterns` replaces the defaults: `node_modules`, `vendor`, `.cache`, `.npm`, `.pnpm`, `__pycache__`, `.venv`, `venv`, `.tox`, `target`, `build`, and `dist`.

## Keybindings

Press `?` inside gv to see this list. If it does not fit the terminal, scroll it with `j`/`k`; any other key closes it.

### Navigation

| Key          | Action                                         |
| ------------ | ---------------------------------------------- |
| `j` / `↓`    | Move down                                      |
| `k` / `↑`    | Move up                                        |
| `g` / `Home` | Jump to top                                    |
| `G` / `End`  | Jump to bottom                                 |
| `Ctrl+d`     | Half page down                                 |
| `Ctrl+u`     | Half page up                                   |
| `/`          | Filter repos by name or path (`Enter` to keep) |
| `Esc`        | Clear filter                                   |
| `q`          | Quit                                           |

### Actions

| Key | Action                                                     |
| --- | ---------------------------------------------------------- |
| `r` | Reload selected repo                                       |
| `f` | Fetch selected repo                                        |
| `F` | Fetch all repos                                            |
| `e` | Open in `$EDITOR` (default `vim`)                          |
| `:` | Open `$SHELL` in the repo directory; exit to return        |
| `o` | Open in the file manager (Finder, Explorer, or `xdg-open`) |
| `y` | Copy the repo path to the clipboard                        |

### Views & Sorting

| Key | Action                                                  |
| --- | ------------------------------------------------------- |
| `1` | All repos                                               |
| `2` | Dirty repos only                                        |
| `3` | Repos ahead of their upstream                           |
| `4` | Repos mid-merge, rebase, cherry-pick, revert, or bisect |
| `5` | Toggle sort by diff volume                              |
| `6` | Toggle sort by file churn                               |
| `d` | Toggle the detail panel                                 |
| `?` | Toggle help                                             |

## How It Works

gv walks each scan path looking for `.git` directories and `.git` files (linked worktrees), and reads `.git/worktrees` to find worktrees that live elsewhere. For each repo it runs `git status --porcelain=v2` alongside supplementary commands (`stash`, `log`, `diff --numstat`) to build a snapshot. Scan paths are resolved through symlinks, so a repo reachable by more than one path (for example `/tmp` and `/private/tmp` on macOS) is listed once. While the dashboard is open, it polls each repo and only reloads the ones whose status output has changed.

## Development

```bash
make test   # go test ./...
make lint   # golangci-lint run
make build
```

## License

[MIT](LICENSE)
