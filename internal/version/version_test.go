package version

import (
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		version string
		info    *debug.BuildInfo
		want    string
	}{
		{"ldflags set", "1.2.3", &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, "1.2.3"},
		{"go install", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v1.4.0"}}, "v1.4.0"},
		{"local build", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev"},
		{"no build info", "dev", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.version, tt.info); got != tt.want {
				t.Errorf("resolve(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}
