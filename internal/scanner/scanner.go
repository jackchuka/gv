// internal/scanner/scanner.go
package scanner

import (
	"context"

	"github.com/jackchuka/gv/internal/model"
)

type Scanner interface {
	Scan(ctx context.Context) ([]model.Repository, error)
	ScanPath(ctx context.Context, path string, maxDepth int) ([]model.Repository, error)
}

type ScanResult struct {
	Repos    []model.Repository
	Errors   []ScanError
	Duration int64 // milliseconds
}

type ScanError struct {
	Path  string
	Error error
}
