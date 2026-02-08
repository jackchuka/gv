// internal/scanner/detect_test.go
package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGitDir_NormalRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := detectGitDir(tmpDir)
	if err != nil {
		t.Fatalf("detectGitDir() error = %v", err)
	}

	if result == nil {
		t.Fatal("detectGitDir() returned nil")
	}

	if result.IsWorktree {
		t.Error("IsWorktree should be false for normal repo")
	}

	if result.Path != tmpDir {
		t.Errorf("Path = %q, want %q", result.Path, tmpDir)
	}
}

func TestDetectGitDir_Worktree(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main repo
	mainRepo := filepath.Join(tmpDir, "main")
	if err := os.MkdirAll(filepath.Join(mainRepo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create worktree with .git file pointing to main
	worktree := filepath.Join(tmpDir, "worktree")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatal(err)
	}

	gitFile := filepath.Join(worktree, ".git")
	content := "gitdir: " + filepath.Join(mainRepo, ".git", "worktrees", "worktree")
	if err := os.WriteFile(gitFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := detectGitDir(worktree)
	if err != nil {
		t.Fatalf("detectGitDir() error = %v", err)
	}

	if result == nil {
		t.Fatal("detectGitDir() returned nil")
	}

	if !result.IsWorktree {
		t.Error("IsWorktree should be true for worktree")
	}

	if result.MainWorktree != mainRepo {
		t.Errorf("MainWorktree = %q, want %q", result.MainWorktree, mainRepo)
	}
}

func TestDiscoverWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main repo with .git/worktrees/wt1/gitdir
	mainRepo := filepath.Join(tmpDir, "main")
	wtEntry := filepath.Join(mainRepo, ".git", "worktrees", "wt1")
	if err := os.MkdirAll(wtEntry, 0755); err != nil {
		t.Fatal(err)
	}

	// Create the worktree working directory with a .git file
	wtDir := filepath.Join(mainRepo, ".wt", "wt1")
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatal(err)
	}

	// gitdir file inside .git/worktrees/wt1 points to the worktree's .git file
	gitdirContent := filepath.Join(wtDir, ".git")
	if err := os.WriteFile(filepath.Join(wtEntry, "gitdir"), []byte(gitdirContent+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	repos := discoverWorktrees(mainRepo)
	if len(repos) != 1 {
		t.Fatalf("got %d worktrees, want 1", len(repos))
	}
	if repos[0].Path != wtDir {
		t.Errorf("Path = %q, want %q", repos[0].Path, wtDir)
	}
	if !repos[0].IsWorktree {
		t.Error("IsWorktree should be true")
	}
	if repos[0].MainWorktree != mainRepo {
		t.Errorf("MainWorktree = %q, want %q", repos[0].MainWorktree, mainRepo)
	}
}

func TestDiscoverWorktrees_NoWorktrees(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	repos := discoverWorktrees(tmpDir)
	if len(repos) != 0 {
		t.Errorf("got %d worktrees, want 0", len(repos))
	}
}

func TestDetectGitDir_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := detectGitDir(tmpDir)
	if err != nil {
		t.Fatalf("detectGitDir() error = %v", err)
	}

	if result != nil {
		t.Error("detectGitDir() should return nil for non-repo")
	}
}
