// internal/scanner/walker_test.go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackchuka/gv/internal/config"
)

func TestWalker_FindsRepos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two repos
	repo1 := filepath.Join(tmpDir, "project1")
	repo2 := filepath.Join(tmpDir, "project2")

	for _, r := range []string{repo1, repo2} {
		if err := os.MkdirAll(filepath.Join(r, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.NewConfig()
	cfg.ScanPaths = []string{tmpDir}

	w := NewWalker(cfg)
	repos, err := w.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Found %d repos, want 2", len(repos))
	}
}

func TestWalker_RespectsMaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create repo at depth 3
	deepRepo := filepath.Join(tmpDir, "a", "b", "c", "deep-project")
	if err := os.MkdirAll(filepath.Join(deepRepo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	cfg.ScanPaths = []string{tmpDir}
	cfg.MaxDepth = 2

	w := NewWalker(cfg)
	repos, err := w.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("Found %d repos, want 0 (depth limit)", len(repos))
	}
}

func TestWalker_SkipsIgnoredDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create repo inside node_modules (should be skipped)
	ignored := filepath.Join(tmpDir, "project", "node_modules", "some-pkg")
	if err := os.MkdirAll(filepath.Join(ignored, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create normal repo (should be found)
	normal := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(filepath.Join(normal, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	cfg.ScanPaths = []string{tmpDir}

	w := NewWalker(cfg)
	repos, err := w.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Found %d repos, want 1", len(repos))
	}

	if repos[0].Path != normal {
		t.Errorf("Found repo at %q, want %q", repos[0].Path, normal)
	}
}
