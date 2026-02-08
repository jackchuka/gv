// internal/status/porcelain_test.go
package status

import (
	"testing"
)

func TestParsePorcelainV2(t *testing.T) {
	output := `# branch.oid abc123def456
# branch.head main
# branch.upstream origin/main
# branch.ab +2 -1
1 M. N... 100644 100644 abc123 def456 src/main.go
1 .M N... 100644 100644 abc123 abc123 README.md
? untracked.txt
`

	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatalf("parsePorcelainV2() error = %v", err)
	}

	if status.Branch != "main" {
		t.Errorf("Branch = %q, want %q", status.Branch, "main")
	}

	if status.Remote != "origin/main" {
		t.Errorf("Remote = %q, want %q", status.Remote, "origin/main")
	}

	if status.Ahead != 2 {
		t.Errorf("Ahead = %d, want 2", status.Ahead)
	}

	if status.Behind != 1 {
		t.Errorf("Behind = %d, want 1", status.Behind)
	}

	if status.Staged != 1 {
		t.Errorf("Staged = %d, want 1", status.Staged)
	}

	if status.Modified != 1 {
		t.Errorf("Modified = %d, want 1", status.Modified)
	}

	if status.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", status.Untracked)
	}
}

func TestParsePorcelainV2_DetachedHead(t *testing.T) {
	output := `# branch.oid abc123def456
# branch.head (detached)
`

	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatalf("parsePorcelainV2() error = %v", err)
	}

	if !status.DetachedHead {
		t.Error("DetachedHead should be true")
	}
}

func TestParsePorcelainV2_NoUpstream(t *testing.T) {
	output := `# branch.oid abc123def456
# branch.head feature-branch
`

	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatalf("parsePorcelainV2() error = %v", err)
	}

	if status.Branch != "feature-branch" {
		t.Errorf("Branch = %q, want %q", status.Branch, "feature-branch")
	}

	if status.Remote != "" {
		t.Errorf("Remote = %q, want empty", status.Remote)
	}
}
