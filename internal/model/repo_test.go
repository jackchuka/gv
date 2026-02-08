// internal/model/repo_test.go
package model

import (
	"testing"
)

func TestRepository_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		repo     Repository
		expected string
	}{
		{
			name:     "uses Name if set",
			repo:     Repository{Name: "my-project", Path: "/home/user/code/my-project"},
			expected: "my-project",
		},
		{
			name:     "derives from path if Name empty",
			repo:     Repository{Path: "/home/user/code/my-project"},
			expected: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.DisplayName()
			if got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRepoStatus_IsDirty(t *testing.T) {
	tests := []struct {
		name     string
		status   RepoStatus
		expected bool
	}{
		{
			name:     "clean repo",
			status:   RepoStatus{},
			expected: false,
		},
		{
			name:     "has staged files",
			status:   RepoStatus{Staged: 1},
			expected: true,
		},
		{
			name:     "has modified files",
			status:   RepoStatus{Modified: 2},
			expected: true,
		},
		{
			name:     "has untracked files",
			status:   RepoStatus{Untracked: 1},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsDirty()
			if got != tt.expected {
				t.Errorf("IsDirty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepoStatus_HasSpecialState(t *testing.T) {
	tests := []struct {
		name     string
		status   RepoStatus
		expected bool
	}{
		{
			name:     "normal state",
			status:   RepoStatus{},
			expected: false,
		},
		{
			name:     "merge in progress",
			status:   RepoStatus{MergeHead: true},
			expected: true,
		},
		{
			name:     "rebase in progress",
			status:   RepoStatus{RebaseHead: true},
			expected: true,
		},
		{
			name:     "cherry-pick in progress",
			status:   RepoStatus{CherryPick: true},
			expected: true,
		},
		{
			name:     "revert in progress",
			status:   RepoStatus{Reverting: true},
			expected: true,
		},
		{
			name:     "bisect in progress",
			status:   RepoStatus{Bisecting: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.HasSpecialState()
			if got != tt.expected {
				t.Errorf("HasSpecialState() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDiffStats_TotalDiffVolume(t *testing.T) {
	tests := []struct {
		name     string
		stats    DiffStats
		expected int
	}{
		{
			name:     "zero values",
			stats:    DiffStats{},
			expected: 0,
		},
		{
			name:     "only additions",
			stats:    DiffStats{TotalAdded: 50},
			expected: 50,
		},
		{
			name:     "only deletions",
			stats:    DiffStats{TotalDeleted: 30},
			expected: 30,
		},
		{
			name:     "both additions and deletions",
			stats:    DiffStats{TotalAdded: 100, TotalDeleted: 40},
			expected: 140,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.TotalDiffVolume()
			if got != tt.expected {
				t.Errorf("TotalDiffVolume() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestDiffStats_TopChurnFiles(t *testing.T) {
	tests := []struct {
		name      string
		churn     map[string]int
		n         int
		wantLen   int
		wantFirst string // expected first entry path (highest churn)
		wantLast  string // expected last entry path
	}{
		{
			name:    "empty map",
			churn:   map[string]int{},
			n:       5,
			wantLen: 0,
		},
		{
			name:      "n=0 returns all entries",
			churn:     map[string]int{"a.go": 3, "b.go": 1},
			n:         0,
			wantLen:   2,
			wantFirst: "a.go",
			wantLast:  "b.go",
		},
		{
			name:      "n limits results",
			churn:     map[string]int{"a.go": 5, "b.go": 3, "c.go": 1},
			n:         2,
			wantLen:   2,
			wantFirst: "a.go",
			wantLast:  "b.go",
		},
		{
			name:      "n larger than entries returns all",
			churn:     map[string]int{"a.go": 2},
			n:         10,
			wantLen:   1,
			wantFirst: "a.go",
			wantLast:  "a.go",
		},
		{
			name:      "ties sorted by path ascending",
			churn:     map[string]int{"z.go": 5, "a.go": 5, "m.go": 5},
			n:         3,
			wantLen:   3,
			wantFirst: "a.go",
			wantLast:  "z.go",
		},
		{
			name:      "mixed counts and tie-breaking",
			churn:     map[string]int{"x.go": 2, "a.go": 10, "b.go": 2},
			n:         3,
			wantLen:   3,
			wantFirst: "a.go",
			wantLast:  "x.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &DiffStats{FileChurn: tt.churn}
			got := ds.TopChurnFiles(tt.n)
			if len(got) != tt.wantLen {
				t.Fatalf("TopChurnFiles(%d) returned %d entries, want %d", tt.n, len(got), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if got[0].Path != tt.wantFirst {
					t.Errorf("first entry = %q, want %q", got[0].Path, tt.wantFirst)
				}
				if got[len(got)-1].Path != tt.wantLast {
					t.Errorf("last entry = %q, want %q", got[len(got)-1].Path, tt.wantLast)
				}
			}
		})
	}
}
