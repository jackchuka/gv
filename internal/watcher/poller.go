// internal/watcher/poller.go
package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os/exec"
	"sync"
	"time"
)

// Poller monitors git repositories using polling instead of fsnotify.
// Uses git status to detect working tree changes.
type Poller struct {
	interval time.Duration
	events   chan Event
	repos    map[string]string // path -> hash of last git status output
	mu       sync.RWMutex
}

type Event struct {
	RepoPath     string
	Time         time.Time
	StatusOutput []byte // Raw porcelain v2 output for reuse by the reader
}

func NewPoller(interval time.Duration) *Poller {
	if interval < time.Second {
		interval = time.Second
	}
	return &Poller{
		interval: interval,
		events:   make(chan Event, 100),
		repos:    make(map[string]string),
	}
}

func (p *Poller) Events() <-chan Event {
	return p.events
}

func (p *Poller) Watch(repoPath string) error {
	p.mu.RLock()
	_, exists := p.repos[repoPath]
	p.mu.RUnlock()

	if exists {
		return nil
	}

	// Get initial status hash outside the lock (runs git commands)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hash, _ := p.getStatusHash(ctx, repoPath)

	p.mu.Lock()
	// Double-check after acquiring write lock
	if _, exists := p.repos[repoPath]; !exists {
		p.repos[repoPath] = hash
	}
	p.mu.Unlock()
	return nil
}

func (p *Poller) Unwatch(repoPath string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.repos, repoPath)
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	// Snapshot current repos and hashes under the lock
	p.mu.RLock()
	snapshot := make(map[string]string, len(p.repos))
	maps.Copy(snapshot, p.repos)
	p.mu.RUnlock()

	// Poll repos concurrently with a semaphore
	type change struct {
		path    string
		newHash string
		output  []byte
	}

	var (
		changes []change
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	sem := make(chan struct{}, 4)

	for repoPath, lastHash := range snapshot {
		wg.Add(1)
		go func(path, last string) {
			defer wg.Done()

			// Respect context cancellation while waiting for semaphore
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			currentHash, output := p.getStatusHash(ctx, path)
			if currentHash == "" {
				return // git failed, skip to avoid phantom changes
			}
			if currentHash != last {
				mu.Lock()
				changes = append(changes, change{path: path, newHash: currentHash, output: output})
				mu.Unlock()
			}
		}(repoPath, lastHash)
	}

	wg.Wait()

	if len(changes) == 0 {
		return
	}

	// Emit events, only updating hashes for successfully sent events.
	// If the channel is full, the old hash is kept so the change is
	// re-detected on the next poll cycle.
	p.mu.Lock()
	for _, c := range changes {
		select {
		case p.events <- Event{RepoPath: c.path, Time: time.Now(), StatusOutput: c.output}:
			if _, ok := p.repos[c.path]; ok {
				p.repos[c.path] = c.newHash
			}
		default:
			// Channel full — keep old hash so change is re-detected next cycle
		}
	}

	p.mu.Unlock()
}

// getStatusHash returns a hash of git status output for change detection.
// Returns both the hash and the raw output so callers can reuse it.
func (p *Poller) getStatusHash(ctx context.Context, repoPath string) (string, []byte) {
	statusCtx, statusCancel := context.WithTimeout(ctx, 5*time.Second)
	defer statusCancel()

	cmd := exec.CommandContext(statusCtx, "git", "-C", repoPath, "status", "--porcelain=v2", "--branch")
	output, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	stashCtx, stashCancel := context.WithTimeout(ctx, 3*time.Second)
	defer stashCancel()

	stashCmd := exec.CommandContext(stashCtx, "git", "-C", repoPath, "rev-parse", "refs/stash")
	stashOutput, _ := stashCmd.Output()

	h := sha256.New()
	h.Write(output)
	h.Write(stashOutput)
	return hex.EncodeToString(h.Sum(nil)), output
}

func (p *Poller) Close() error {
	close(p.events)
	return nil
}
