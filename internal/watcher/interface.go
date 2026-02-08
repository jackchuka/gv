// internal/watcher/interface.go
package watcher

import "context"

type RepoWatcher interface {
	Events() <-chan Event
	Watch(repoPath string) error
	Unwatch(repoPath string)
	Run(ctx context.Context)
	Close() error
}
