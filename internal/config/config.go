// internal/config/config.go
package config

import (
	"path/filepath"
	"time"
)

type Config struct {
	// Scanning
	ScanPaths      []string `yaml:"scan_paths"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
	MaxDepth       int      `yaml:"max_depth"`

	// Watcher
	PollInterval time.Duration `yaml:"poll_interval"`
	AutoRefresh  bool          `yaml:"auto_refresh"`
}

func NewConfig() *Config {
	return &Config{
		ScanPaths: []string{},
		IgnorePatterns: []string{
			"**/node_modules/**",
			"**/vendor/**",
			"**/.cache/**",
			"**/.npm/**",
			"**/.pnpm/**",
			"**/__pycache__/**",
			"**/.venv/**",
			"**/venv/**",
			"**/.tox/**",
			"**/target/**",
			"**/build/**",
			"**/dist/**",
		},
		MaxDepth:     10,
		PollInterval: 5 * time.Second,
		AutoRefresh:  true,
	}
}

func (c *Config) ShouldIgnore(path string) bool {
	for _, pattern := range c.IgnorePatterns {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Try matching against each path segment for ** patterns
		if containsDoublestar(pattern) {
			if matchDoublestar(pattern, path) {
				return true
			}
		}
	}
	return false
}

func containsDoublestar(pattern string) bool {
	for i := 0; i < len(pattern)-1; i++ {
		if pattern[i] == '*' && pattern[i+1] == '*' {
			return true
		}
	}
	return false
}

func matchDoublestar(pattern, path string) bool {
	// Simple implementation: check if path contains the middle part
	// e.g., "**/node_modules/**" matches if path contains "/node_modules/"
	if len(pattern) < 5 {
		return false
	}
	// Extract middle part between **/ and /**
	middle := pattern[3 : len(pattern)-3]
	return filepath.Base(filepath.Dir(path)) == middle ||
		filepath.Base(path) == middle
}
