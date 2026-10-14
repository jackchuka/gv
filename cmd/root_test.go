package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.HasPrefix(out.String(), "gv version ") {
		t.Errorf("output = %q, want prefix %q", out.String(), "gv version ")
	}
}

func TestScanFlagExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgFile = filepath.Join(home, "missing.yaml")
	t.Cleanup(func() { cfgFile = "" })

	if err := rootCmd.ParseFlags([]string{"--scan=~/code,~"}); err != nil {
		t.Fatal(err)
	}
	initConfig()

	want := []string{filepath.Join(home, "code"), home}
	if len(cfg.ScanPaths) != len(want) {
		t.Fatalf("ScanPaths = %v, want %v", cfg.ScanPaths, want)
	}
	for i := range want {
		if cfg.ScanPaths[i] != want[i] {
			t.Errorf("ScanPaths[%d] = %q, want %q", i, cfg.ScanPaths[i], want[i])
		}
	}
}
