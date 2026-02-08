// internal/scanner/walker.go
package scanner

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/jackchuka/gv/internal/config"
	"github.com/jackchuka/gv/internal/model"
)

type Walker struct {
	cfg *config.Config
}

func NewWalker(cfg *config.Config) *Walker {
	return &Walker{cfg: cfg}
}

func (w *Walker) Scan(ctx context.Context) ([]model.Repository, error) {
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		seen = make(map[string]model.Repository)
	)

	for _, scanPath := range w.cfg.ScanPaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			found, err := w.ScanPath(ctx, path, w.cfg.MaxDepth)
			if err != nil {
				return
			}

			mu.Lock()
			for _, r := range found {
				// Deduplicate by path - scan paths may overlap
				if _, exists := seen[r.Path]; !exists {
					seen[r.Path] = r
				}
			}
			mu.Unlock()
		}(scanPath)
	}

	wg.Wait()

	repos := make([]model.Repository, 0, len(seen))
	for _, r := range seen {
		repos = append(repos, r)
	}
	return repos, nil
}

func (w *Walker) ScanPath(ctx context.Context, root string, maxDepth int) ([]model.Repository, error) {
	var repos []model.Repository

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)
		depth := 0
		if relPath != "." {
			for _, c := range relPath {
				if c == filepath.Separator {
					depth++
				}
			}
			depth++ // Add 1 for the final component
		}

		if depth > maxDepth {
			return fs.SkipDir
		}

		if d.Name() == ".git" {
			return fs.SkipDir
		}

		if w.cfg.ShouldIgnore(path) && path != root {
			return fs.SkipDir
		}

		repo, err := detectGitDir(path)
		if err != nil {
			return nil // Continue on errors
		}

		if repo != nil {
			repos = append(repos, *repo)
			if !repo.IsWorktree {
				repos = append(repos, discoverWorktrees(path)...)
			}
			return fs.SkipDir // Don't descend into git repos
		}

		return nil
	})

	return repos, err
}
