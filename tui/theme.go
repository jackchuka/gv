package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI 256 color palette
var (
	// Status colors (carried from V2)
	colorCleanGreen = lipgloss.Color("71")
	colorDirtyAmber = lipgloss.Color("179")
	colorDangerRed  = lipgloss.Color("167")
	colorCriticalRd = lipgloss.Color("196")

	// Accent
	colorCyan = lipgloss.Color("73")
	colorGold = lipgloss.Color("220")

	// Text
	colorFg  = lipgloss.Color("253")
	colorDim = lipgloss.Color("242")

	// Selection
	colorSelBg = lipgloss.Color("238")
	colorSelFg = lipgloss.Color("255")

	// V3-specific
	colorBlue     = lipgloss.Color("69")  // net delta, accent
	colorDiffAdd  = lipgloss.Color("71")  // added lines (same as clean green)
	colorDiffDel  = lipgloss.Color("167") // deleted lines (same as danger red)
	colorBarEmpty = lipgloss.Color("238") // ░ empty bar segments
	colorTableHdr = lipgloss.Color("245") // table header text
	colorRowAlt   = lipgloss.Color("234") // alternating row bg
	colorChurn    = lipgloss.Color("208") // churn/activity (orange)
)

// Left-border accent: flash bright/off, then fade out
var glowBorderColors = []lipgloss.Color{
	lipgloss.Color("46"),  // on
	lipgloss.Color("236"), // off
	lipgloss.Color("46"),  // on
	lipgloss.Color("236"), // off
	lipgloss.Color("46"),  // on
	lipgloss.Color("34"),  // fade
	lipgloss.Color("28"),  // fade
	lipgloss.Color("23"),  // fade
	lipgloss.Color("236"), // gone
}

// Braille spinner frames
var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// Unicode icons
const (
	iconClean    = "○"
	iconDirty    = "●"
	iconCleanWt  = "◇"
	iconDirtyWt  = "◆"
	iconConflict = "⚠"
	iconBranch   = "⟫"
	iconAhead    = "↑"
	iconBehind   = "↓"
	iconBolt     = "⚡"
	iconStar     = "★"
)

// Lipgloss styles
var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	styleDim      = lipgloss.NewStyle().Foreground(colorDim)
	styleRepoName = lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	styleBranch   = lipgloss.NewStyle().Foreground(colorCyan)
	styleAhead    = lipgloss.NewStyle().Foreground(colorDirtyAmber)
	styleBehind   = lipgloss.NewStyle().Foreground(colorDangerRed)
	styleCleanTxt = lipgloss.NewStyle().Foreground(colorCleanGreen)
	styleConflict = lipgloss.NewStyle().Foreground(colorCriticalRd).Bold(true)
	styleAmber    = lipgloss.NewStyle().Foreground(colorDirtyAmber)

	styleDiffAdd  = lipgloss.NewStyle().Foreground(colorDiffAdd)
	styleDiffDel  = lipgloss.NewStyle().Foreground(colorDiffDel)
	styleNetDelta = lipgloss.NewStyle().Foreground(colorBlue)
	styleBarEmpty = lipgloss.NewStyle().Foreground(colorBarEmpty)
	styleTableHdr = lipgloss.NewStyle().Foreground(colorTableHdr).Bold(true)

	styleKey       = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleActiveTab = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Underline(true)

	styleToastBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			Padding(0, 1)
)

func renderSpinner(frame int) string {
	f := spinnerFrames[frame%len(spinnerFrames)]
	return lipgloss.NewStyle().Foreground(colorCyan).Render(f)
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	// Truncate rune by rune
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
