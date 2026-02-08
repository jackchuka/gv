// internal/model/repo.go
package model

import (
	"path/filepath"
	"sort"
	"time"
)

type Repository struct {
	Path         string      // Absolute path to repo root
	Name         string      // Display name (derived from path or config)
	IsWorktree   bool        // True if this is a linked worktree
	MainWorktree string      // If IsWorktree, path to main repo
	Status       *RepoStatus // Current status (nil if not yet scanned)
	Diff         *DiffStats  // Line-level diff and activity data (nil if not loaded)
	LastScanned  time.Time   // When status was last refreshed
}

func (r *Repository) DisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	return filepath.Base(r.Path)
}

type RepoStatus struct {
	Branch       string // Current branch name (empty if detached)
	DetachedHead bool   // True if HEAD is detached
	CommitHash   string // Short hash of HEAD

	// Working tree state
	Staged    int // Number of staged files
	Modified  int // Number of modified files
	Untracked int // Number of untracked files

	// Remote state
	Remote string // Tracking remote (e.g., "origin/main")
	Owner  string // Owner/org from remote URL (e.g., "jackchuka")
	Ahead  int    // Commits ahead of remote
	Behind int    // Commits behind remote

	// Special states
	Stashes    int  // Number of stashes
	MergeHead  bool // Merge in progress
	RebaseHead bool // Rebase in progress
	CherryPick bool // Cherry-pick in progress
	Reverting  bool // Revert in progress
	Bisecting  bool // Bisect in progress

	// Timestamps
	LastCommit   time.Time // Time of last commit
	LastModified time.Time // Last working tree modification

	// Custom command outputs
	Aliases map[string]string // alias name -> output
}

func (s *RepoStatus) IsDirty() bool {
	return s.Staged > 0 || s.Modified > 0 || s.Untracked > 0
}

func (s *RepoStatus) HasSpecialState() bool {
	return s.MergeHead || s.RebaseHead || s.CherryPick || s.Reverting || s.Bisecting
}

type FileDiffStat struct {
	Path    string // Relative file path
	Added   int    // Lines added
	Deleted int    // Lines deleted
	Binary  bool   // True if binary file
}

type FileChurnEntry struct {
	Path  string
	Count int
}

type DiffStats struct {
	// Unstaged diff (working tree vs index)
	UnstagedFiles   []FileDiffStat
	UnstagedAdded   int
	UnstagedDeleted int

	// Staged diff (index vs HEAD)
	StagedFiles   []FileDiffStat
	StagedAdded   int
	StagedDeleted int

	// Aggregate totals
	TotalAdded   int // UnstagedAdded + StagedAdded
	TotalDeleted int // UnstagedDeleted + StagedDeleted
	NetDelta     int // TotalAdded - TotalDeleted

	// Commit activity (7-day history, index 0 = 6 days ago, index 6 = today)
	DailyCommits [7]int

	// File churn (7-day history, key = file path, value = number of commits)
	FileChurn map[string]int

	// Timestamp
	CollectedAt time.Time
}

func (d *DiffStats) TotalDiffVolume() int {
	return d.TotalAdded + d.TotalDeleted
}

func (d *DiffStats) TopChurnFiles(n int) []FileChurnEntry {
	entries := make([]FileChurnEntry, 0, len(d.FileChurn))
	for path, count := range d.FileChurn {
		entries = append(entries, FileChurnEntry{Path: path, Count: count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Path < entries[j].Path
	})
	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}
	return entries
}
