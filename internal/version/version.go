package version

import (
	"fmt"
	"runtime/debug"
)

// Set via -ldflags "-X github.com/jackchuka/gv/internal/version.Version=...".
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns Version, falling back to the module version for go install builds.
func String() string {
	info, _ := debug.ReadBuildInfo()
	return resolve(Version, info)
}

// Info returns String plus the commit and build date when they were set at build time.
func Info() string {
	if Commit == "none" {
		return String()
	}
	return fmt.Sprintf("%s (commit %s, built %s)", String(), Commit, Date)
}

func resolve(version string, info *debug.BuildInfo) string {
	if version != "dev" || info == nil {
		return version
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return version
}
