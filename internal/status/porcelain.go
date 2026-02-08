// internal/status/porcelain.go
package status

import (
	"strconv"
	"strings"

	"github.com/jackchuka/gv/internal/model"
)

func parsePorcelainV2(output string) (*model.RepoStatus, error) {
	status := &model.RepoStatus{
		Aliases: make(map[string]string),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Header lines start with #
		if strings.HasPrefix(line, "# ") {
			parseHeaderLine(line, status)
			continue
		}

		// Tracked file changes (ordinary or renamed)
		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			parseChangeLine(line, status)
			continue
		}

		// Untracked files
		if strings.HasPrefix(line, "? ") {
			status.Untracked++
			continue
		}

		// Ignored files (we don't track these)
		if strings.HasPrefix(line, "! ") {
			continue
		}

		// Unmerged files
		if strings.HasPrefix(line, "u ") {
			status.Modified++ // Count as modified
			continue
		}
	}

	return status, nil
}

func parseHeaderLine(line string, status *model.RepoStatus) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return
	}

	key := parts[1]
	value := parts[2]

	switch key {
	case "branch.oid":
		if len(value) >= 7 {
			status.CommitHash = value[:7]
		}
	case "branch.head":
		if value == "(detached)" {
			status.DetachedHead = true
		} else {
			status.Branch = value
		}
	case "branch.upstream":
		status.Remote = value
	case "branch.ab":
		// Parse "+N -M"
		abParts := strings.Fields(value)
		for _, p := range abParts {
			if strings.HasPrefix(p, "+") {
				status.Ahead, _ = strconv.Atoi(p[1:])
			} else if strings.HasPrefix(p, "-") {
				status.Behind, _ = strconv.Atoi(p[1:])
			}
		}
	}
}

func parseChangeLine(line string, status *model.RepoStatus) {
	// Format: "1 XY ..."
	// X = index status, Y = worktree status
	if len(line) < 4 {
		return
	}

	xy := line[2:4]
	indexStatus := xy[0]
	worktreeStatus := xy[1]

	// Staged changes (index has changes)
	if indexStatus != '.' && indexStatus != '?' {
		status.Staged++
	}

	// Unstaged changes (worktree has changes)
	if worktreeStatus != '.' && worktreeStatus != '?' {
		status.Modified++
	}
}
