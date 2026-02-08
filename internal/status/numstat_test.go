// internal/status/numstat_test.go
package status

import (
	"testing"
)

func TestParseNumstat(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantFiles    int
		wantAdded    int
		wantDeleted  int
		wantBinaries int
	}{
		{
			name:        "empty output",
			input:       "",
			wantFiles:   0,
			wantAdded:   0,
			wantDeleted: 0,
		},
		{
			name:        "single file",
			input:       "10\t5\tsrc/main.go\n",
			wantFiles:   1,
			wantAdded:   10,
			wantDeleted: 5,
		},
		{
			name:        "multiple files",
			input:       "10\t5\tsrc/main.go\n3\t1\tREADME.md\n",
			wantFiles:   2,
			wantAdded:   13,
			wantDeleted: 6,
		},
		{
			name:         "binary file",
			input:        "-\t-\timage.png\n",
			wantFiles:    1,
			wantAdded:    0,
			wantDeleted:  0,
			wantBinaries: 1,
		},
		{
			name:         "mixed binary and text",
			input:        "22\t5\tsrc/app.go\n-\t-\tlogo.png\n8\t0\tgo.mod\n",
			wantFiles:    3,
			wantAdded:    30,
			wantDeleted:  5,
			wantBinaries: 1,
		},
		{
			name:        "whitespace only",
			input:       "  \n  \n",
			wantFiles:   0,
			wantAdded:   0,
			wantDeleted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, added, deleted := parseNumstat(tt.input)
			if len(files) != tt.wantFiles {
				t.Errorf("files count = %d, want %d", len(files), tt.wantFiles)
			}
			if added != tt.wantAdded {
				t.Errorf("totalAdded = %d, want %d", added, tt.wantAdded)
			}
			if deleted != tt.wantDeleted {
				t.Errorf("totalDeleted = %d, want %d", deleted, tt.wantDeleted)
			}
			binaries := 0
			for _, f := range files {
				if f.Binary {
					binaries++
				}
			}
			if binaries != tt.wantBinaries {
				t.Errorf("binaries = %d, want %d", binaries, tt.wantBinaries)
			}
		})
	}
}

func TestParseDailyCommits(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result [7]int)
	}{
		{
			name:  "empty output",
			input: "",
			check: func(t *testing.T, result [7]int) {
				for i, v := range result {
					if v != 0 {
						t.Errorf("result[%d] = %d, want 0", i, v)
					}
				}
			},
		},
		{
			name:  "whitespace only",
			input: "  \n  \n",
			check: func(t *testing.T, result [7]int) {
				total := 0
				for _, v := range result {
					total += v
				}
				if total != 0 {
					t.Errorf("total = %d, want 0", total)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDailyCommits(tt.input)
			tt.check(t, result)
		})
	}
}

func TestParseFileChurn(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFiles int
		wantMax   int
	}{
		{
			name:      "empty output",
			input:     "",
			wantFiles: 0,
		},
		{
			name:      "single file once",
			input:     "src/main.go\n",
			wantFiles: 1,
			wantMax:   1,
		},
		{
			name:      "same file multiple times",
			input:     "src/main.go\n\nsrc/main.go\n\nsrc/main.go\n",
			wantFiles: 1,
			wantMax:   3,
		},
		{
			name:      "multiple files",
			input:     "src/main.go\nREADME.md\n\nsrc/main.go\ngo.mod\n",
			wantFiles: 3,
			wantMax:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			churn := parseFileChurn(tt.input)
			if len(churn) != tt.wantFiles {
				t.Errorf("file count = %d, want %d", len(churn), tt.wantFiles)
			}
			maxCount := 0
			for _, c := range churn {
				if c > maxCount {
					maxCount = c
				}
			}
			if maxCount != tt.wantMax {
				t.Errorf("max churn = %d, want %d", maxCount, tt.wantMax)
			}
		})
	}
}
