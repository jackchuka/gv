// internal/scanner/detect.go
package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jackchuka/gv/internal/model"
)

func detectGitDir(path string) (*model.Repository, error) {
	gitPath := filepath.Join(path, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	repo := &model.Repository{
		Path: path,
	}

	if info.IsDir() {
		// Normal git repository
		repo.IsWorktree = false
		return repo, nil
	}

	// .git is a file - this is a worktree
	repo.IsWorktree = true

	content, err := os.ReadFile(gitPath)
	if err != nil {
		return nil, err
	}

	// Parse "gitdir: /path/to/main/.git/worktrees/name"
	line := strings.TrimSpace(string(content))
	if strings.HasPrefix(line, "gitdir:") {
		gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		// Extract main repo path from .git/worktrees/name
		if idx := strings.Index(gitdir, "/.git/worktrees/"); idx != -1 {
			repo.MainWorktree = gitdir[:idx]
		}
	}

	return repo, nil
}

// discoverWorktrees finds linked worktrees registered in .git/worktrees/.
// Each entry contains a "gitdir" file pointing to the worktree's working directory.
func discoverWorktrees(repoPath string) []model.Repository {
	wtDir := filepath.Join(repoPath, ".git", "worktrees")
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		return nil
	}

	var repos []model.Repository
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitdirFile := filepath.Join(wtDir, e.Name(), "gitdir")
		content, err := os.ReadFile(gitdirFile)
		if err != nil {
			continue
		}
		wtPath := strings.TrimSpace(string(content))
		// gitdir file points to the worktree's .git file location;
		// the working directory is its parent
		wtPath = filepath.Dir(wtPath)
		if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
			continue
		}
		repos = append(repos, model.Repository{
			Path:         wtPath,
			IsWorktree:   true,
			MainWorktree: repoPath,
		})
	}
	return repos
}
