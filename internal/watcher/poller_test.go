package watcher

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPoller_MinimumInterval(t *testing.T) {
	tests := []struct {
		name         string
		interval     time.Duration
		wantInterval time.Duration
	}{
		{"below minimum clamps to 1s", 100 * time.Millisecond, time.Second},
		{"zero clamps to 1s", 0, time.Second},
		{"exact 1s preserved", time.Second, time.Second},
		{"above minimum preserved", 5 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPoller(tt.interval)
			if p.interval != tt.wantInterval {
				t.Errorf("interval = %v, want %v", p.interval, tt.wantInterval)
			}
		})
	}
}

func TestPoller_Events(t *testing.T) {
	p := NewPoller(time.Second)
	ch := p.Events()
	if ch == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestPoller_WatchAndUnwatch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	p := NewPoller(time.Second)

	if err := p.Watch(tmpDir); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	p.mu.RLock()
	_, exists := p.repos[tmpDir]
	p.mu.RUnlock()
	if !exists {
		t.Error("repo should be registered after Watch()")
	}

	p.Unwatch(tmpDir)
	p.mu.RLock()
	_, exists = p.repos[tmpDir]
	p.mu.RUnlock()
	if exists {
		t.Error("repo should be removed after Unwatch()")
	}
}

func TestPoller_WatchDuplicate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	p := NewPoller(time.Second)
	if err := p.Watch(tmpDir); err != nil {
		t.Fatalf("first Watch() error = %v", err)
	}
	if err := p.Watch(tmpDir); err != nil {
		t.Fatalf("second Watch() error = %v", err)
	}

	p.mu.RLock()
	count := len(p.repos)
	p.mu.RUnlock()

	if count != 1 {
		t.Errorf("repos count = %d after duplicate Watch, want 1", count)
	}
}

func TestPoller_RunCancellation(t *testing.T) {
	p := NewPoller(time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		p.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Run returned after cancellation
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestPoller_DetectsChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Commit an initial file so status has a baseline
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, tmpDir, "add", "test.txt")
	gitCmd(t, tmpDir, "commit", "-m", "initial")

	p := NewPoller(time.Second)
	if err := p.Watch(tmpDir); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Modify the file to change status
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Trigger a poll manually
	ctx := context.Background()
	p.poll(ctx)

	select {
	case ev := <-p.events:
		if ev.RepoPath != tmpDir {
			t.Errorf("event RepoPath = %q, want %q", ev.RepoPath, tmpDir)
		}
		if len(ev.StatusOutput) == 0 {
			t.Error("event StatusOutput should not be empty")
		}
	default:
		t.Error("expected a change event after modifying a tracked file")
	}
}

func TestPoller_NoEventWhenUnchanged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	p := NewPoller(time.Second)
	if err := p.Watch(tmpDir); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Poll without making changes
	ctx := context.Background()
	p.poll(ctx)

	select {
	case ev := <-p.events:
		t.Errorf("unexpected event: %+v", ev)
	default:
		// No event expected
	}
}

func TestPoller_Close(t *testing.T) {
	p := NewPoller(time.Second)
	if err := p.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Reading from closed channel should return zero value immediately
	_, ok := <-p.events
	if ok {
		t.Error("expected channel to be closed")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "config", "user.name", "Test")
}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
