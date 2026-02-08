// internal/status/git_test.go
package status

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jackchuka/gv/internal/model"
)

func TestGitReader_GetStatus(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temp git repo
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify the file
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := NewGitReader()
	status, err := reader.GetStatus(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.Branch != "master" && status.Branch != "main" {
		t.Errorf("Branch = %q, want master or main", status.Branch)
	}

	if status.Modified != 1 {
		t.Errorf("Modified = %d, want 1", status.Modified)
	}
}

func TestGitReader_DetectsSpecialStates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")

	// Create MERGE_HEAD to simulate merge in progress
	mergeHead := filepath.Join(tmpDir, ".git", "MERGE_HEAD")
	if err := os.WriteFile(mergeHead, []byte("abc123"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := NewGitReader()
	status, err := reader.GetStatus(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.MergeHead {
		t.Error("MergeHead should be true")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"whitespace only", "   \n  \t  ", 0},
		{"single line", "stash@{0}: WIP on main", 1},
		{"two lines", "line1\nline2", 2},
		{"trailing newline", "line1\nline2\n", 2},
		{"multiple with blanks trimmed", "  a\nb\nc  ", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.input)
			if got != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseOwnerFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH colon format",
			url:      "git@github.com:jackchuka/gv.git",
			expected: "jackchuka",
		},
		{
			name:     "SSH URL format",
			url:      "ssh://git@github.com/jackchuka/gv.git",
			expected: "jackchuka",
		},
		{
			name:     "HTTPS format",
			url:      "https://github.com/jackchuka/gv.git",
			expected: "jackchuka",
		},
		{
			name:     "HTTP format",
			url:      "http://github.com/jackchuka/gv.git",
			expected: "jackchuka",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "no path segments",
			url:      "https://github.com",
			expected: "",
		},
		{
			name:     "SSH colon with nested path",
			url:      "git@gitlab.com:group/subgroup/repo.git",
			expected: "group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOwnerFromURL(tt.url)
			if got != tt.expected {
				t.Errorf("parseOwnerFromURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckSpecialStates(t *testing.T) {
	reader := NewGitReader()

	t.Run("detects MERGE_HEAD", func(t *testing.T) {
		tmp := t.TempDir()
		gitDir := filepath.Join(tmp, ".git")
		mustMkdirAll(t, gitDir)
		mustWriteFile(t, filepath.Join(gitDir, "MERGE_HEAD"), []byte("abc"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.MergeHead {
			t.Error("expected MergeHead = true")
		}
	})

	t.Run("detects CHERRY_PICK_HEAD", func(t *testing.T) {
		tmp := t.TempDir()
		gitDir := filepath.Join(tmp, ".git")
		mustMkdirAll(t, gitDir)
		mustWriteFile(t, filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("abc"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.CherryPick {
			t.Error("expected CherryPick = true")
		}
	})

	t.Run("detects REVERT_HEAD", func(t *testing.T) {
		tmp := t.TempDir()
		gitDir := filepath.Join(tmp, ".git")
		mustMkdirAll(t, gitDir)
		mustWriteFile(t, filepath.Join(gitDir, "REVERT_HEAD"), []byte("abc"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.Reverting {
			t.Error("expected Reverting = true")
		}
	})

	t.Run("detects BISECT_LOG", func(t *testing.T) {
		tmp := t.TempDir()
		gitDir := filepath.Join(tmp, ".git")
		mustMkdirAll(t, gitDir)
		mustWriteFile(t, filepath.Join(gitDir, "BISECT_LOG"), []byte("abc"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.Bisecting {
			t.Error("expected Bisecting = true")
		}
	})

	t.Run("detects rebase-merge", func(t *testing.T) {
		tmp := t.TempDir()
		mustMkdirAll(t, filepath.Join(tmp, ".git", "rebase-merge"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.RebaseHead {
			t.Error("expected RebaseHead = true")
		}
	})

	t.Run("detects rebase-apply", func(t *testing.T) {
		tmp := t.TempDir()
		mustMkdirAll(t, filepath.Join(tmp, ".git", "rebase-apply"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if !status.RebaseHead {
			t.Error("expected RebaseHead = true")
		}
	})

	t.Run("no special state in clean repo", func(t *testing.T) {
		tmp := t.TempDir()
		mustMkdirAll(t, filepath.Join(tmp, ".git"))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(tmp, status)

		if status.HasSpecialState() {
			t.Error("expected no special state")
		}
	})

	t.Run("worktree .git file", func(t *testing.T) {
		tmp := t.TempDir()
		realGitDir := filepath.Join(tmp, "real-gitdir")
		mustMkdirAll(t, realGitDir)
		mustWriteFile(t, filepath.Join(realGitDir, "MERGE_HEAD"), []byte("abc"))

		worktree := filepath.Join(tmp, "worktree")
		mustMkdirAll(t, worktree)
		mustWriteFile(t, filepath.Join(worktree, ".git"), []byte("gitdir: "+realGitDir))

		status := &model.RepoStatus{}
		reader.checkSpecialStates(worktree, status)

		if !status.MergeHead {
			t.Error("expected MergeHead = true via worktree gitdir")
		}
	})
}
