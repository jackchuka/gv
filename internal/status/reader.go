// internal/status/reader.go
package status

import (
	"context"

	"github.com/jackchuka/gv/internal/model"
)

type Reader interface {
	GetStatus(ctx context.Context, repoPath string) (*model.RepoStatus, error)
	GetStatusFromOutput(ctx context.Context, repoPath string, porcelainOutput string) (*model.RepoStatus, error)
	GetStatusBatch(ctx context.Context, paths []string) (map[string]*model.RepoStatus, map[string]error)

	// Fetch returns the fetch error alongside the status so callers can
	// distinguish "fetch failed but here's the old status" from "status read failed".
	Fetch(ctx context.Context, repoPath string) (*model.RepoStatus, error)
	FetchBatch(ctx context.Context, paths []string) (map[string]*model.RepoStatus, map[string]error)

	// GetDiffStats always returns a result (possibly partial) — never nil.
	GetDiffStats(ctx context.Context, repoPath string) *model.DiffStats
	GetDiffStatsBatch(ctx context.Context, paths []string) map[string]*model.DiffStats

	RunAlias(ctx context.Context, repoPath string, cmd string) (string, error)
}
