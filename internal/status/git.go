// internal/status/git.go
package status

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackchuka/gv/internal/model"
)

type GitReader struct {
	concurrency int
}

func NewGitReader() *GitReader {
	return &GitReader{
		concurrency: 8,
	}
}

func (r *GitReader) GetStatus(ctx context.Context, repoPath string) (*model.RepoStatus, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := r.runGit(cmdCtx, repoPath, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil, err
	}

	return r.GetStatusFromOutput(ctx, repoPath, output)
}

// GetStatusFromOutput builds a full RepoStatus from pre-fetched porcelain output,
// running supplementary commands (stash, log, remote) in parallel.
func (r *GitReader) GetStatusFromOutput(ctx context.Context, repoPath string, porcelainOutput string) (*model.RepoStatus, error) {
	status, err := parsePorcelainV2(porcelainOutput)
	if err != nil {
		return nil, err
	}

	// Supplementary commands are independent — run in parallel
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		stashOutput, err := r.runGit(cmdCtx, repoPath, "stash", "list")
		if err == nil {
			status.Stashes = countLines(stashOutput)
		}
	}()

	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		logOutput, err := r.runGit(cmdCtx, repoPath, "log", "-1", "--format=%ct")
		if err == nil {
			if ts, err := strconv.ParseInt(strings.TrimSpace(logOutput), 10, 64); err == nil {
				status.LastCommit = time.Unix(ts, 0)
			}
		}
	}()

	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		remoteURL, err := r.runGit(cmdCtx, repoPath, "remote", "get-url", "origin")
		if err == nil {
			status.Owner = parseOwnerFromURL(strings.TrimSpace(remoteURL))
		}
	}()

	// checkSpecialStates is filesystem-only, fast — no goroutine needed
	r.checkSpecialStates(repoPath, status)

	wg.Wait()
	return status, nil
}

func (r *GitReader) GetStatusBatch(ctx context.Context, paths []string) (map[string]*model.RepoStatus, map[string]error) {
	results := make(map[string]*model.RepoStatus)
	errors := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, r.concurrency)

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			status, err := r.GetStatus(ctx, p)
			mu.Lock()
			if err != nil {
				errors[p] = err
			}
			if status != nil {
				results[p] = status
			}
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	return results, errors
}

func (r *GitReader) RunAlias(ctx context.Context, repoPath string, cmd string) (string, error) {
	args := strings.Fields(cmd)
	return r.runGit(ctx, repoPath, args...)
}

func (r *GitReader) runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}

func (r *GitReader) checkSpecialStates(repoPath string, status *model.RepoStatus) {
	gitDir := filepath.Join(repoPath, ".git")

	info, err := os.Stat(gitDir)
	if err == nil && !info.IsDir() {
		content, err := os.ReadFile(gitDir)
		if err == nil {
			line := strings.TrimSpace(string(content))
			if strings.HasPrefix(line, "gitdir:") {
				gitDir = strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
			}
		}
	}

	checks := []struct {
		path string
		flag *bool
	}{
		{filepath.Join(gitDir, "MERGE_HEAD"), &status.MergeHead},
		{filepath.Join(gitDir, "CHERRY_PICK_HEAD"), &status.CherryPick},
		{filepath.Join(gitDir, "REVERT_HEAD"), &status.Reverting},
		{filepath.Join(gitDir, "BISECT_LOG"), &status.Bisecting},
	}

	for _, c := range checks {
		if _, err := os.Stat(c.path); err == nil {
			*c.flag = true
		}
	}

	// Check for rebase (can be in multiple locations)
	rebasePaths := []string{
		filepath.Join(gitDir, "rebase-merge"),
		filepath.Join(gitDir, "rebase-apply"),
	}
	for _, p := range rebasePaths {
		if _, err := os.Stat(p); err == nil {
			status.RebaseHead = true
			break
		}
	}
}

func countLines(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// parseOwnerFromURL extracts the owner/org from a git remote URL
// Handles:
//   - SSH colon format: git@github.com:owner/repo.git
//   - SSH URL format: ssh://git@github.com/owner/repo.git
//   - HTTPS format: https://github.com/owner/repo.git
func parseOwnerFromURL(url string) string {
	if strings.Contains(url, "@") && strings.Contains(url, ":") && !strings.HasPrefix(url, "ssh://") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 && !strings.Contains(parts[0], "/") {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 1 {
				return pathParts[0]
			}
		}
	}

	url = strings.TrimPrefix(url, "ssh://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	if idx := strings.Index(url, "@"); idx != -1 {
		url = url[idx+1:]
	}

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}

// Fetch runs git fetch and returns updated status.
// Returns the fetch error (if any) alongside the current status so callers
// can distinguish "fetch failed but here's the old status" from "status read failed".
func (r *GitReader) Fetch(ctx context.Context, repoPath string) (*model.RepoStatus, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, fetchErr := r.runGit(fetchCtx, repoPath, "fetch", "--quiet")

	// Get updated status — use parent ctx so cancellation propagates
	status, statusErr := r.GetStatus(ctx, repoPath)
	if statusErr != nil {
		return nil, statusErr
	}

	return status, fetchErr
}

func (r *GitReader) FetchBatch(ctx context.Context, paths []string) (map[string]*model.RepoStatus, map[string]error) {
	results := make(map[string]*model.RepoStatus)
	errors := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	fetchSem := make(chan struct{}, r.concurrency)

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			fetchSem <- struct{}{}
			defer func() { <-fetchSem }()

			status, err := r.Fetch(ctx, p)
			mu.Lock()
			if err != nil {
				errors[p] = err
			}
			if status != nil {
				results[p] = status
			}
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	return results, errors
}

// GetDiffStats reads line-level diff stats, commit activity, and file churn for a repo.
// All 4 git commands run in parallel with independent timeouts.
// Always returns a result (possibly with partial data) — never returns an error
// since all sub-commands are non-fatal.
func (r *GitReader) GetDiffStats(ctx context.Context, repoPath string) *model.DiffStats {
	ds := &model.DiffStats{
		FileChurn:   make(map[string]int),
		CollectedAt: time.Now(),
	}

	var wg sync.WaitGroup
	wg.Add(4)

	// 1. Unstaged diff
	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		unstaged, err := r.runGit(cmdCtx, repoPath, "diff", "--numstat")
		if err == nil {
			ds.UnstagedFiles, ds.UnstagedAdded, ds.UnstagedDeleted = parseNumstat(unstaged)
		}
	}()

	// 2. Staged diff
	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		staged, err := r.runGit(cmdCtx, repoPath, "diff", "--cached", "--numstat")
		if err == nil {
			ds.StagedFiles, ds.StagedAdded, ds.StagedDeleted = parseNumstat(staged)
		}
	}()

	// 3. Commit activity (7-day history)
	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		logOutput, err := r.runGit(cmdCtx, repoPath, "log", "--since=7.days", "--format=%cd", "--date=format:%Y-%m-%d")
		if err == nil {
			ds.DailyCommits = parseDailyCommits(logOutput)
		}
	}()

	// 4. File churn (7-day history)
	go func() {
		defer wg.Done()
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		churnOutput, err := r.runGit(cmdCtx, repoPath, "log", "--since=7.days", "--name-only", "--format=")
		if err == nil {
			ds.FileChurn = parseFileChurn(churnOutput)
		}
	}()

	wg.Wait()

	// Aggregate totals after all goroutines complete
	ds.TotalAdded = ds.UnstagedAdded + ds.StagedAdded
	ds.TotalDeleted = ds.UnstagedDeleted + ds.StagedDeleted
	ds.NetDelta = ds.TotalAdded - ds.TotalDeleted

	return ds
}

func (r *GitReader) GetDiffStatsBatch(ctx context.Context, paths []string) map[string]*model.DiffStats {
	results := make(map[string]*model.DiffStats)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, r.concurrency)

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			ds := r.GetDiffStats(ctx, p)

			mu.Lock()
			results[p] = ds
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	return results
}
