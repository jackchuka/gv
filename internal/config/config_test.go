// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg := NewConfig()

	if cfg.MaxDepth != 10 {
		t.Errorf("MaxDepth = %d, want 10", cfg.MaxDepth)
	}

	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", cfg.PollInterval)
	}
}

func TestConfig_ShouldIgnore(t *testing.T) {
	cfg := &Config{
		IgnorePatterns: []string{
			"**/node_modules/**",
			"**/vendor/**",
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/code/project/src", false},
		{"/home/user/code/project/node_modules/pkg", true},
		{"/home/user/code/project/vendor/lib", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := cfg.ShouldIgnore(tt.path)
			if got != tt.expected {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestLoad_CreatesDefaultIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.MaxDepth != 10 {
		t.Errorf("MaxDepth = %d, want 10", cfg.MaxDepth)
	}
}

func TestLoad_ParsesYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
scan_paths:
  - ~/code
  - ~/projects
max_depth: 5
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.ScanPaths) != 2 {
		t.Errorf("ScanPaths length = %d, want 2", len(cfg.ScanPaths))
	}

	if cfg.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5", cfg.MaxDepth)
	}
}

func TestSave_and_Load_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sub", "dir", "config.yaml")

	cfg := NewConfig()
	cfg.ScanPaths = []string{"/home/user/code", "/tmp/repos"}
	cfg.MaxDepth = 7
	cfg.PollInterval = 10 * time.Second
	cfg.AutoRefresh = false

	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.ScanPaths) != 2 {
		t.Errorf("ScanPaths length = %d, want 2", len(loaded.ScanPaths))
	}
	if loaded.MaxDepth != 7 {
		t.Errorf("MaxDepth = %d, want 7", loaded.MaxDepth)
	}
	if loaded.PollInterval != 10*time.Second {
		t.Errorf("PollInterval = %v, want 10s", loaded.PollInterval)
	}
	if loaded.AutoRefresh != false {
		t.Error("AutoRefresh should be false")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde only", "~", home},
		{"tilde with path", "~/code", filepath.Join(home, "code")},
		{"absolute path unchanged", "/usr/local/bin", "/usr/local/bin"},
		{"empty string", "", ""},
		{"relative path unchanged", "some/path", "some/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultConfigPath(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		got := DefaultConfigPath()
		expected := "/custom/config/gv/config.yaml"
		if got != expected {
			t.Errorf("DefaultConfigPath() = %q, want %q", got, expected)
		}
	})

	t.Run("falls back to ~/.config when XDG unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		home, _ := os.UserHomeDir()
		got := DefaultConfigPath()
		expected := filepath.Join(home, ".config", "gv", "config.yaml")
		if got != expected {
			t.Errorf("DefaultConfigPath() = %q, want %q", got, expected)
		}
	})
}

func TestShouldIgnore_AdditionalCases(t *testing.T) {
	cfg := &Config{
		IgnorePatterns: []string{
			"**/node_modules/**",
			"*.tmp",
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/src/main.go", false},
		{"/home/user/project/node_modules/pkg", true},
		{"file.tmp", true},
		{"file.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := cfg.ShouldIgnore(tt.path)
			if got != tt.expected {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestNewConfig_DefaultIgnorePatterns(t *testing.T) {
	cfg := NewConfig()

	if !cfg.AutoRefresh {
		t.Error("AutoRefresh should default to true")
	}

	if len(cfg.IgnorePatterns) == 0 {
		t.Error("IgnorePatterns should not be empty by default")
	}

	if len(cfg.ScanPaths) != 0 {
		t.Errorf("ScanPaths should be empty by default, got %d", len(cfg.ScanPaths))
	}
}
