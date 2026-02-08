// internal/status/numstat.go
package status

import (
	"strconv"
	"strings"
	"time"

	"github.com/jackchuka/gv/internal/model"
)

// parseNumstat parses output of `git diff --numstat`.
// Each line: "added\tdeleted\tfilename" or "-\t-\tfilename" (binary).
func parseNumstat(output string) ([]model.FileDiffStat, int, int) {
	if strings.TrimSpace(output) == "" {
		return nil, 0, 0
	}

	var files []model.FileDiffStat
	totalAdded, totalDeleted := 0, 0

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		fds := model.FileDiffStat{Path: parts[2]}

		if parts[0] == "-" && parts[1] == "-" {
			fds.Binary = true
		} else {
			fds.Added, _ = strconv.Atoi(parts[0])
			fds.Deleted, _ = strconv.Atoi(parts[1])
			totalAdded += fds.Added
			totalDeleted += fds.Deleted
		}

		files = append(files, fds)
	}

	return files, totalAdded, totalDeleted
}

// parseDailyCommits parses git log date output into a [7]int array.
// Input: one date per line in YYYY-MM-DD format.
// Output: index 0 = 6 days ago, index 6 = today.
func parseDailyCommits(output string) [7]int {
	var result [7]int
	if strings.TrimSpace(output) == "" {
		return result
	}

	today := time.Now().Truncate(24 * time.Hour)
	counts := make(map[string]int)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}

	for i := range 7 {
		day := today.AddDate(0, 0, -(6 - i))
		key := day.Format("2006-01-02")
		result[i] = counts[key]
	}

	return result
}

// parseFileChurn parses git log --name-only output into a churn map.
// Input: file paths separated by blank lines (one commit's files per block).
// Output: map of file path -> number of commits touching that file.
func parseFileChurn(output string) map[string]int {
	churn := make(map[string]int)
	if strings.TrimSpace(output) == "" {
		return churn
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		churn[line]++
	}

	return churn
}
