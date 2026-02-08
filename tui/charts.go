package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline characters ordered by magnitude
var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// withBg applies a background color to a style when bg is non-empty.
func withBg(s lipgloss.Style, bg lipgloss.Color) lipgloss.Style {
	if bg != "" {
		return s.Background(bg)
	}
	return s
}

// renderHBar renders a single-color horizontal bar.
// Returns: "████░░░░" with value/maxValue proportion filled.
func renderHBar(value, maxValue, width int, fg lipgloss.Color) string {
	if maxValue <= 0 || width <= 0 {
		return ""
	}
	if value < 0 {
		value = 0
	}
	if value > maxValue {
		value = maxValue
	}

	filled := value * width / maxValue
	empty := width - filled

	var b strings.Builder
	if filled > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(fg).Render(strings.Repeat("█", filled)))
	}
	if empty > 0 {
		b.WriteString(styleBarEmpty.Render(strings.Repeat("░", empty)))
	}
	return b.String()
}

// renderStackedBar renders a stacked bar with three segments (staged, modified, untracked).
// Green + Amber + Gray segments proportional to their counts.
// Optional bgColor applies a background to each segment for consistent row backgrounds.
func renderStackedBar(staged, modified, untracked, width int, bgColor ...lipgloss.Color) string {
	var bg lipgloss.Color
	if len(bgColor) > 0 {
		bg = bgColor[0]
	}

	total := staged + modified + untracked
	if total == 0 || width <= 0 {
		return withBg(styleBarEmpty, bg).Render(strings.Repeat("░", width))
	}

	// Calculate segment widths proportionally
	stagedW := staged * width / total
	modifiedW := modified * width / total
	untrackedW := untracked * width / total

	// Distribute remainder to largest segment
	remainder := width - stagedW - modifiedW - untrackedW
	if staged >= modified && staged >= untracked {
		stagedW += remainder
	} else if modified >= untracked {
		modifiedW += remainder
	} else {
		untrackedW += remainder
	}

	var b strings.Builder
	if stagedW > 0 {
		b.WriteString(withBg(styleDiffAdd, bg).Render(strings.Repeat("█", stagedW)))
	}
	if modifiedW > 0 {
		b.WriteString(withBg(styleAmber, bg).Render(strings.Repeat("█", modifiedW)))
	}
	if untrackedW > 0 {
		b.WriteString(withBg(styleDim, bg).Render(strings.Repeat("█", untrackedW)))
	}

	return b.String()
}

// renderDiffBar renders a green/red split bar for +added/-deleted lines.
// Optional bgColor applies a background to each segment for consistent row backgrounds.
func renderDiffBar(added, deleted, width int, bgColor ...lipgloss.Color) string {
	var bg lipgloss.Color
	if len(bgColor) > 0 {
		bg = bgColor[0]
	}

	total := added + deleted
	if total == 0 || width <= 0 {
		return withBg(styleBarEmpty, bg).Render(strings.Repeat("░", width))
	}

	addedW := added * width / total
	deletedW := width - addedW

	var b strings.Builder
	if addedW > 0 {
		b.WriteString(withBg(styleDiffAdd, bg).Render(strings.Repeat("█", addedW)))
	}
	if deletedW > 0 {
		b.WriteString(withBg(styleDiffDel, bg).Render(strings.Repeat("▒", deletedW)))
	}
	return b.String()
}

// renderSparkline renders a sparkline from values using block chars ▁▂▃▄▅▆▇█.
func renderSparkline(values []int, color lipgloss.Color) string {
	if len(values) == 0 {
		return ""
	}

	maxVal := 0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	var b strings.Builder
	style := lipgloss.NewStyle().Foreground(color)
	dimStyle := lipgloss.NewStyle().Foreground(colorBarEmpty)

	for _, v := range values {
		if maxVal == 0 || v == 0 {
			b.WriteString(dimStyle.Render("▁"))
		} else {
			idx := v * (len(sparkChars) - 1) / maxVal
			b.WriteString(style.Render(string(sparkChars[idx])))
		}
	}
	return b.String()
}

func renderChurnBar(count, maxCount, width int) string {
	if maxCount <= 0 || width <= 0 {
		return ""
	}

	filled := count * width / maxCount
	if filled == 0 && count > 0 {
		filled = 1
	}

	var b strings.Builder
	style := lipgloss.NewStyle().Foreground(colorChurn)
	for i := 0; i < filled; i++ {
		b.WriteString(style.Render("█"))
	}
	return b.String()
}

func renderMiniDiffBar(added, deleted, width int) string {
	total := added + deleted
	if total == 0 || width <= 0 {
		return ""
	}

	addedW := added * width / total
	deletedW := width - addedW
	if addedW == 0 && added > 0 {
		addedW = 1
		deletedW = width - 1
	}

	var b strings.Builder
	if addedW > 0 {
		b.WriteString(styleDiffAdd.Render(strings.Repeat("█", addedW)))
	}
	if deletedW > 0 {
		b.WriteString(styleDiffDel.Render(strings.Repeat("▒", deletedW)))
	}
	return b.String()
}
