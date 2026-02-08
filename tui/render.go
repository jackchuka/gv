package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jackchuka/gv/internal/model"
)

func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var sections []string
	sections = append(sections, m.renderHeader())

	if m.showHelp {
		sections = append(sections, m.renderHelp())
		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top,
			strings.Join(sections, "\n"))
	}

	sections = append(sections, m.renderSummaryPanel())
	sections = append(sections, m.renderTable())
	sections = append(sections, m.renderFooter())

	view := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top,
		strings.Join(sections, "\n"))

	// Overlay toasts on the view (bottom-right with padding)
	if len(m.toasts) > 0 {
		toast := m.renderToasts()
		tw := lipgloss.Width(toast)
		th := lipgloss.Height(toast)
		x := m.width - tw - 2
		y := m.height - th - 2
		view = placeOverlay(x, y, toast, view)
	}

	return view
}

func (m *Model) renderHeader() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("44")).Bold(true).Render("GitVision")

	// Spinner
	var spinner string
	switch m.phase {
	case PhaseScanning, PhaseLoading:
		spinner = "  " + renderSpinner(m.anim.frame) + " Scanning..."
	case PhaseFetching:
		target := "all"
		if m.fetchTarget != "" {
			for _, r := range m.repos {
				if r.Path == m.fetchTarget {
					target = r.DisplayName()
					break
				}
			}
		}
		spinner = "  " + renderSpinner(m.anim.frame) + " Fetching " + target + "..."
	}

	if m.diffLoading {
		spinner = "  " + renderSpinner(m.anim.frame) + " Loading diffs..."
	}

	// Spaced stats: dim label + bold colored number
	s := m.summary
	bold := lipgloss.NewStyle().Bold(true)
	stats := styleDim.Render("repos ") + bold.Foreground(lipgloss.Color("255")).Render(fmt.Sprintf("%d", s.TotalRepos))
	if s.DirtyRepos > 0 {
		stats += "  " + styleDim.Render("dirty ") + bold.Foreground(colorDirtyAmber).Render(fmt.Sprintf("%d", s.DirtyRepos))
	}
	if s.AheadRepos > 0 {
		stats += "  " + styleDim.Render("ahead ") + bold.Foreground(lipgloss.Color("73")).Render(fmt.Sprintf("%d", s.AheadRepos))
	}
	if s.ConflictRepos > 0 {
		stats += "  " + styleDim.Render("conflict ") + bold.Foreground(colorDangerRed).Render(fmt.Sprintf("%d", s.ConflictRepos))
	}

	// Filter display
	left := title + spinner
	if m.filterMode {
		left += "  " + m.filterInput.View()
	} else if m.filterText != "" {
		left += "  " + styleDim.Render("filter: "+m.filterText)
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(stats)
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + stats
	sep := styleDim.Render(strings.Repeat("─", m.width))

	return line + "\n" + sep
}

// --- Summary panel ---

type summaryRow struct {
	style lipgloss.Style
	label string
	value int
	color lipgloss.Color
}

func renderSummaryColumn(header string, rows []summaryRow, maxVal, barW int) string {
	col := styleTableHdr.Render(header)
	for _, r := range rows {
		col += "\n" + r.style.Render(r.label) + renderHBar(r.value, maxVal, barW, r.color)
	}
	return col
}

func (m *Model) renderSummaryPanel() string {
	s := m.summary
	if s.TotalRepos == 0 {
		return ""
	}

	barW := 12

	changesMax := s.TotalStaged + s.TotalModified + s.TotalUntracked
	if changesMax == 0 {
		changesMax = 1
	}
	changesCol := renderSummaryColumn("CHANGES", []summaryRow{
		{styleDiffAdd, fmt.Sprintf(" staged  %3d ", s.TotalStaged), s.TotalStaged, colorDiffAdd},
		{styleAmber, fmt.Sprintf(" mod     %3d ", s.TotalModified), s.TotalModified, colorDirtyAmber},
		{styleDim, fmt.Sprintf(" untrack %3d ", s.TotalUntracked), s.TotalUntracked, colorDim},
	}, changesMax, barW)

	syncMax := s.TotalRepos
	if syncMax == 0 {
		syncMax = 1
	}
	syncCol := renderSummaryColumn("SYNC", []summaryRow{
		{styleAhead, fmt.Sprintf(" ahead  %3d ", s.AheadRepos), s.AheadRepos, colorDirtyAmber},
		{styleBehind, fmt.Sprintf(" behind %3d ", s.BehindRepos), s.BehindRepos, colorDangerRed},
		{styleCleanTxt, fmt.Sprintf(" sync   %3d ", s.InSyncRepos), s.InSyncRepos, colorCleanGreen},
	}, syncMax, barW)

	// DIFF column — unique last row uses renderDiffBar instead of renderHBar
	diffMax := s.TotalAdded + s.TotalDeleted
	if diffMax == 0 {
		diffMax = 1
	}
	diffCol := styleTableHdr.Render("DIFF") + "\n"
	diffCol += styleDiffAdd.Render(fmt.Sprintf(" +added %4d ", s.TotalAdded)) +
		renderHBar(s.TotalAdded, diffMax, barW, colorDiffAdd) + "\n"
	diffCol += styleDiffDel.Render(fmt.Sprintf(" -del   %4d ", s.TotalDeleted)) +
		renderHBar(s.TotalDeleted, diffMax, barW, colorDiffDel) + "\n"
	sign := "+"
	if s.TotalNet < 0 {
		sign = ""
	}
	diffCol += styleNetDelta.Render(fmt.Sprintf(" net   %s%4d ", sign, s.TotalNet)) +
		renderDiffBar(s.TotalAdded, s.TotalDeleted, barW)

	// ACTIVITY column
	commits := s.DailyCommits[:]
	actCol := styleTableHdr.Render("ACTIVITY (7d)") + "\n"
	actCol += " " + renderSparkline(commits, colorCyan) + "\n"
	totalCommits := 0
	for _, c := range commits {
		totalCommits += c
	}
	actCol += styleDim.Render(fmt.Sprintf(" %d commits", totalCommits))

	gap := "   "
	panel := lipgloss.JoinHorizontal(lipgloss.Top,
		changesCol, gap, syncCol, gap, diffCol, gap, actCol,
	)

	sep := styleDim.Render(strings.Repeat("─", m.width))
	return panel + "\n" + sep
}

// --- Table ---

func (m *Model) renderTable() string {
	visRows := m.visibleRows()
	tableHeight := visRows + 1 // +1 for header

	if len(m.rows) == 0 {
		msg := "No repos found"
		if m.filterText != "" {
			msg = "No repos match filter"
		}
		content := "\n " + styleDim.Render(msg)
		// Pad to fill table area so footer stays at bottom
		lines := strings.Split(content, "\n")
		for len(lines) < tableHeight {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	contentWidth := m.width
	detailWidth := 0
	if m.showDetail && m.width >= 100 {
		detailWidth = m.width * 35 / 100
		contentWidth = m.width - detailWidth - 1
	}

	// Column widths
	cols := computeColumns(contentWidth)

	// Header
	hdr := " " +
		styleTableHdr.Render(padRight("REPO", cols.repo)) +
		styleTableHdr.Render(padRight("BRANCH", cols.branch)) +
		styleTableHdr.Render(padRight("SYNC", cols.sync)) +
		styleTableHdr.Render(padRight("CHANGES", cols.changes)) +
		styleTableHdr.Render(padRight("DIFF", cols.diff))

	// Keep cursor in view
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visRows {
		m.scrollOffset = m.cursor - visRows + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	end := m.scrollOffset + visRows
	if end > len(m.rows) {
		end = len(m.rows)
	}

	var tableLines []string
	tableLines = append(tableLines, hdr)

	// Find max diff volume for bar scaling
	maxDiff := 1
	maxChurn := 1
	for _, row := range m.rows {
		if row.Repo != nil {
			dv := diffVolume(row.Repo)
			if dv > maxDiff {
				maxDiff = dv
			}
			ct := churnTotal(row.Repo)
			if ct > maxChurn {
				maxChurn = ct
			}
		}
	}

	for i := m.scrollOffset; i < end; i++ {
		row := m.rows[i]
		selected := i == m.cursor
		parentAbove := false
		if row.Repo != nil && row.Repo.IsWorktree && i > 0 {
			prev := m.rows[i-1].Repo
			parentAbove = prev != nil && prev.Path == row.Repo.MainWorktree
		}
		line := m.renderTableRow(row, cols, selected, i%2 == 1, maxDiff, contentWidth, parentAbove)
		tableLines = append(tableLines, line)
	}

	// Pad table to fill available height so footer stays at bottom
	for len(tableLines) < tableHeight {
		tableLines = append(tableLines, "")
	}

	tableContent := strings.Join(tableLines, "\n")

	// Detail panel
	if detailWidth > 0 {
		detail := m.renderDetailPanel(detailWidth, tableHeight)
		sepLines := make([]string, tableHeight)
		for i := range sepLines {
			sepLines[i] = styleDim.Render("│")
		}
		sep := strings.Join(sepLines, "\n")
		return lipgloss.JoinHorizontal(lipgloss.Top, tableContent, sep, detail)
	}

	return tableContent
}

type columnWidths struct {
	repo    int
	branch  int
	changes int
	sync    int
	diff    int
}

func computeColumns(width int) columnWidths {
	// Allocate proportionally, minimum widths
	usable := width - 2 // leading space + margin
	if usable < 40 {
		usable = 40
	}

	c := columnWidths{
		repo:    usable * 28 / 100,
		branch:  usable * 25 / 100,
		sync:    usable * 8 / 100,
		changes: usable * 18 / 100,
		diff:    usable * 21 / 100,
	}

	// Minimum widths
	if c.repo < 10 {
		c.repo = 10
	}
	if c.branch < 8 {
		c.branch = 8
	}
	if c.sync < 8 {
		c.sync = 8
	}

	return c
}

// --- Row rendering ---

// rowRenderer holds per-row styling state shared across cell renderers.
type rowRenderer struct {
	bg      func(lipgloss.Style) lipgloss.Style
	rowBg   lipgloss.Style
	hasGlow bool
	bgColor lipgloss.Color // empty when no row background
	prefix  string         // prepended to the row (border/dot styles)
}

func (m *Model) newRowRenderer(repo *model.Repository, selected, alt bool) rowRenderer {
	step, hasGlow := m.anim.glowFade[repo.Path]

	var bgColor lipgloss.Color
	if selected {
		bgColor = colorSelBg
	} else if alt {
		bgColor = colorRowAlt
	}

	bg := func(base lipgloss.Style) lipgloss.Style {
		if selected {
			return base.Background(colorSelBg)
		}
		if alt {
			return base.Background(colorRowAlt)
		}
		return base
	}

	var prefix string
	if hasGlow {
		prefix = lipgloss.NewStyle().Foreground(glowBorderColors[step]).Render("▎")
	}

	return rowRenderer{
		bg:      bg,
		rowBg:   bg(lipgloss.NewStyle()),
		hasGlow: hasGlow,
		bgColor: bgColor,
		prefix:  prefix,
	}
}

func (r rowRenderer) repoCell(repo *model.Repository, width int, selected, parentAbove bool) string {
	s := repo.Status
	wt := repo.IsWorktree

	var dot string
	switch {
	case s != nil && s.HasSpecialState():
		dot = r.bg(styleConflict).Render(iconConflict)
	case s != nil && s.IsDirty() && wt:
		dot = r.bg(styleAmber).Render(iconDirtyWt)
	case s != nil && s.IsDirty():
		dot = r.bg(styleAmber).Render(iconDirty)
	case s != nil && wt:
		dot = r.bg(styleCleanTxt).Render(iconCleanWt)
	case s != nil:
		dot = r.bg(styleCleanTxt).Render(iconClean)
	default:
		dot = r.bg(styleDim).Render("·")
	}

	nameStyle := r.bg(styleRepoName)
	if selected && !r.hasGlow {
		nameStyle = nameStyle.Foreground(colorSelFg)
	}

	prefix := ""
	nameWidth := width - 3
	if repo.IsWorktree && repo.MainWorktree != "" {
		parentName := filepath.Base(repo.MainWorktree) + "/"
		if parentAbove {
			prefix = r.bg(styleDim).Render("└ ")
			nameWidth -= 2
		} else {
			prefix = r.bg(styleDim).Render(parentName)
			nameWidth -= len(parentName)
		}
	}
	name := truncateWithEllipsis(repo.DisplayName(), nameWidth)
	return r.rowBg.Width(width).Render(dot + r.rowBg.Render(" ") + prefix + nameStyle.Render(name))
}

func (r rowRenderer) branchCell(s *model.RepoStatus, width int) string {
	if s == nil {
		return r.bg(styleDim).Width(width).Render("...")
	}
	branch := s.Branch
	if branch == "" && s.DetachedHead {
		branch = s.CommitHash[:7]
	}
	if branch == "" {
		branch = "???"
	}
	return r.bg(styleBranch).Width(width).Render(truncateWithEllipsis(branch, width-1))
}

func (r rowRenderer) syncCell(s *model.RepoStatus, width int) string {
	var content string
	if s != nil {
		if s.Ahead > 0 {
			content += r.bg(styleAhead).Render(fmt.Sprintf("%s%d", iconAhead, s.Ahead))
		}
		if s.Behind > 0 {
			if content != "" {
				content += r.rowBg.Render(" ")
			}
			content += r.bg(styleBehind).Render(fmt.Sprintf("%s%d", iconBehind, s.Behind))
		}
		if s.HasSpecialState() {
			content = r.bg(styleConflict).Render(specialStateLabel(s))
		}
		if content == "" {
			content = r.bg(styleDim).Render("──")
		}
	} else {
		content = r.bg(styleDim).Render("...")
	}
	return r.rowBg.Width(width).Render(content)
}

func (r rowRenderer) changesCell(s *model.RepoStatus, width int) string {
	var content string
	if s != nil {
		barW := width - 10
		if barW < 3 {
			barW = 3
		}
		bar := renderStackedBar(s.Staged, s.Modified, s.Untracked, barW, r.bgColor)
		total := s.Staged + s.Modified + s.Untracked
		if total > 0 {
			content = bar + r.rowBg.Render(" ") + r.bg(styleDim).Render(fmt.Sprintf("%d", total))
		} else {
			content = r.bg(styleBarEmpty).Render(strings.Repeat("░", barW)) + r.rowBg.Render(" ") + r.bg(styleDim).Render("0")
		}
	} else {
		content = r.bg(styleDim).Render("...")
	}
	return r.rowBg.Width(width).Render(content)
}

func (r rowRenderer) diffCell(repo *model.Repository, width int, loading bool) string {
	var content string
	if repo.Diff != nil {
		barW := width - 14
		if barW < 3 {
			barW = 3
		}
		bar := renderDiffBar(repo.Diff.TotalAdded, repo.Diff.TotalDeleted, barW, r.bgColor)
		content = bar + r.rowBg.Render(" ") +
			r.bg(styleDiffAdd).Render(fmt.Sprintf("+%d", repo.Diff.TotalAdded)) + r.rowBg.Render(" ") +
			r.bg(styleDiffDel).Render(fmt.Sprintf("-%d", repo.Diff.TotalDeleted))
	} else if loading {
		content = r.bg(styleDim).Render("...")
	} else {
		content = r.bg(styleDim).Render("─")
	}
	return r.rowBg.Width(width).Render(content)
}

func (m *Model) renderTableRow(row TableRow, cols columnWidths, selected, alt bool, maxDiff, rowWidth int, parentAbove bool) string {
	repo := row.Repo
	if repo == nil {
		return ""
	}

	r := m.newRowRenderer(repo, selected, alt)

	leading := r.rowBg.Render(" ")
	if r.prefix != "" {
		leading = r.prefix
	}

	line := leading +
		r.repoCell(repo, cols.repo, selected, parentAbove) +
		r.branchCell(repo.Status, cols.branch) +
		r.syncCell(repo.Status, cols.sync) +
		r.changesCell(repo.Status, cols.changes) +
		r.diffCell(repo, cols.diff, m.diffLoading)

	return r.rowBg.Width(rowWidth).Render(line)
}

// --- Detail panel ---

func renderDetailStatus(s *model.RepoStatus, innerW int) []string {
	var lines []string

	branch := s.Branch
	if branch == "" && s.DetachedHead {
		branch = s.CommitHash
	}
	lines = append(lines, styleBranch.Render(" "+iconBranch+" "+branch))

	if s.Owner != "" {
		lines = append(lines, styleDim.Render(" owner: "+s.Owner))
	}

	if s.Ahead > 0 || s.Behind > 0 {
		sync := " "
		if s.Ahead > 0 {
			sync += styleAhead.Render(fmt.Sprintf("%s%d ahead ", iconAhead, s.Ahead))
		}
		if s.Behind > 0 {
			sync += styleBehind.Render(fmt.Sprintf("%s%d behind", iconBehind, s.Behind))
		}
		lines = append(lines, sync)
	}

	if s.Stashes > 0 {
		lines = append(lines, styleDim.Render(fmt.Sprintf(" stashes: %d", s.Stashes)))
	}

	lines = append(lines, "")

	if s.IsDirty() {
		lines = append(lines, styleTableHdr.Render(" FILE CHANGES"))
		barW := innerW - 16
		if barW < 4 {
			barW = 4
		}
		bar := renderStackedBar(s.Staged, s.Modified, s.Untracked, barW)
		lines = append(lines, " "+bar)
		if s.Staged > 0 {
			lines = append(lines, styleDiffAdd.Render(fmt.Sprintf("  staged:    %d", s.Staged)))
		}
		if s.Modified > 0 {
			lines = append(lines, styleAmber.Render(fmt.Sprintf("  modified:  %d", s.Modified)))
		}
		if s.Untracked > 0 {
			lines = append(lines, styleDim.Render(fmt.Sprintf("  untracked: %d", s.Untracked)))
		}
		lines = append(lines, "")
	}

	if s.HasSpecialState() {
		label := specialStateLabel(s)
		lines = append(lines, styleConflict.Render(" "+iconBolt+" "+label))
		lines = append(lines, "")
	}

	return lines
}

func renderDetailFileList(header string, files []model.FileDiffStat, innerW int, countStyle lipgloss.Style) []string {
	if len(files) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, styleTableHdr.Render(" "+header))
	maxLines := 5
	for i, f := range files {
		if i >= maxLines {
			lines = append(lines, styleDim.Render(fmt.Sprintf("  ...and %d more", len(files)-maxLines)))
			break
		}
		fname := truncateWithEllipsis(filepath.Base(f.Path), innerW-20)
		miniBar := renderMiniDiffBar(f.Added, f.Deleted, 6)
		lines = append(lines, fmt.Sprintf("  %s %s %s",
			miniBar,
			countStyle.Render(fmt.Sprintf("+%d", f.Added)),
			styleDim.Render(fname)))
	}
	lines = append(lines, "")
	return lines
}

func renderDetailDiff(d *model.DiffStats, innerW int) []string {
	var lines []string

	lines = append(lines, styleTableHdr.Render(" DIFF BREAKDOWN"))
	barW := innerW - 4
	if barW < 6 {
		barW = 6
	}
	lines = append(lines, " "+renderDiffBar(d.TotalAdded, d.TotalDeleted, barW))
	lines = append(lines,
		styleDiffAdd.Render(fmt.Sprintf("  +%d added", d.TotalAdded))+"  "+
			styleDiffDel.Render(fmt.Sprintf("-%d deleted", d.TotalDeleted)))

	sign := "+"
	if d.NetDelta < 0 {
		sign = ""
	}
	lines = append(lines, styleNetDelta.Render(fmt.Sprintf("  net: %s%d lines", sign, d.NetDelta)))
	lines = append(lines, "")

	if d.StagedAdded+d.StagedDeleted > 0 || d.UnstagedAdded+d.UnstagedDeleted > 0 {
		lines = append(lines, styleDim.Render(fmt.Sprintf("  staged:   +%d -%d", d.StagedAdded, d.StagedDeleted)))
		lines = append(lines, styleDim.Render(fmt.Sprintf("  unstaged: +%d -%d", d.UnstagedAdded, d.UnstagedDeleted)))
		lines = append(lines, "")
	}

	lines = append(lines, renderDetailFileList("STAGED FILES", d.StagedFiles, innerW, styleDiffAdd)...)
	lines = append(lines, renderDetailFileList("MODIFIED FILES", d.UnstagedFiles, innerW, styleAmber)...)

	// Activity sparkline
	lines = append(lines, styleTableHdr.Render(" ACTIVITY (7d)"))
	commits := d.DailyCommits[:]
	lines = append(lines, " "+renderSparkline(commits, colorCyan))
	totalCommits := 0
	for _, c := range commits {
		totalCommits += c
	}
	lines = append(lines, styleDim.Render(fmt.Sprintf("  %d commits this week", totalCommits)))
	lines = append(lines, "")

	// Top file churn
	topChurn := d.TopChurnFiles(5)
	if len(topChurn) > 0 {
		lines = append(lines, styleTableHdr.Render(" HOT FILES"))
		maxChurn := topChurn[0].Count
		for _, entry := range topChurn {
			fname := truncateWithEllipsis(filepath.Base(entry.Path), innerW-12)
			bar := renderChurnBar(entry.Count, maxChurn, 6)
			lines = append(lines, fmt.Sprintf("  %s %s %s",
				bar,
				lipgloss.NewStyle().Foreground(colorChurn).Render(fmt.Sprintf("%d", entry.Count)),
				styleDim.Render(fname)))
		}
	}

	return lines
}

func (m *Model) renderDetailPanel(width, height int) string {
	repo := m.selectedRepo()
	if repo == nil {
		return padLines(styleDim.Render(" No selection"), width, height)
	}

	var lines []string
	innerW := width - 2

	lines = append(lines, styleRepoName.Render(" "+repo.DisplayName()))
	lines = append(lines, styleDim.Render(" "+repo.Path))
	lines = append(lines, "")

	if repo.Status != nil {
		lines = append(lines, renderDetailStatus(repo.Status, innerW)...)
	}

	if repo.Diff != nil {
		lines = append(lines, renderDetailDiff(repo.Diff, innerW)...)
	} else if m.diffLoading {
		lines = append(lines, styleDim.Render(" Loading diff stats..."))
	}

	return padLines(strings.Join(lines, "\n"), width, height)
}

// --- Footer, toasts, help ---

func (m *Model) renderFooter() string {
	sep := styleDim.Render(strings.Repeat("─", m.width))

	type viewTab struct {
		key    string
		label  string
		filter ViewFilter
	}
	tabs := []viewTab{
		{"1", "all", ViewAll},
		{"2", "dirty", ViewDirty},
		{"3", "ahead", ViewUnpushed},
		{"4", "conflict", ViewConflicts},
	}

	var parts []string
	parts = append(parts, styleKey.Render("/")+" search")
	parts = append(parts, styleKey.Render("f")+" fetch")
	parts = append(parts, styleKey.Render("e")+" editor")

	for _, t := range tabs {
		if m.viewFilter == t.filter {
			parts = append(parts, styleActiveTab.Render(t.key+" "+t.label))
		} else {
			parts = append(parts, styleKey.Render(t.key)+" "+t.label)
		}
	}

	// Sort mode indicators
	switch m.sortMode {
	case SortDiff:
		parts = append(parts, styleActiveTab.Render("5 diff"))
	default:
		parts = append(parts, styleKey.Render("5")+" diff")
	}
	switch m.sortMode {
	case SortChurn:
		parts = append(parts, styleActiveTab.Render("6 churn"))
	default:
		parts = append(parts, styleKey.Render("6")+" churn")
	}

	if m.showDetail {
		parts = append(parts, styleActiveTab.Render("d detail"))
	} else {
		parts = append(parts, styleKey.Render("d")+" detail")
	}

	parts = append(parts, styleKey.Render("?")+" help")
	parts = append(parts, styleKey.Render("q")+" quit")

	return sep + "\n " + truncateWithEllipsis(strings.Join(parts, "  "), m.width-2)
}

func (m *Model) renderToasts() string {
	var toastStrs []string
	for _, t := range m.toasts {
		var bc lipgloss.Color
		var icon string
		switch t.Level {
		case ToastSuccess:
			bc = colorGold
			icon = iconStar + " "
		case ToastError:
			bc = colorDangerRed
			icon = iconConflict + " "
		default:
			bc = colorCyan
			icon = ""
		}
		box := styleToastBox.BorderForeground(bc).Render(icon + t.Message)
		toastStrs = append(toastStrs, box)
	}
	return strings.Join(toastStrs, "\n")
}

func (m *Model) renderHelp() string {
	content := m.keys.helpText()

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorCyan).
		Padding(1, 2).
		Width(50).
		Render(styleTitle.Render("HELP") + "\n\n" + content + "\n\n" + styleDim.Render("press any key to close"))

	availH := m.height - 4
	if availH < 10 {
		availH = 10
	}
	return lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, box)
}

// --- Layout utilities ---

// placeOverlay writes fg on top of bg at the given column (x) and row (y).
// It handles ANSI-styled strings correctly using ansi.Cut.
func placeOverlay(x, y int, fg, bg string) string {
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	for i, fgLine := range fgLines {
		bgIdx := y + i
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}
		bgLine := bgLines[bgIdx]
		fgW := ansi.StringWidth(fgLine)
		bgW := ansi.StringWidth(bgLine)

		if x < 0 {
			x = 0
		}
		if x >= bgW {
			// Toast starts beyond the background line; just append
			bgLines[bgIdx] = bgLine + strings.Repeat(" ", x-bgW) + fgLine
			continue
		}

		left := ansi.Cut(bgLine, 0, x)
		var right string
		if x+fgW < bgW {
			right = ansi.Cut(bgLine, x+fgW, bgW)
		}
		bgLines[bgIdx] = left + fgLine + right
	}
	return strings.Join(bgLines, "\n")
}

func padLines(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			lines[i] = line + strings.Repeat(" ", width-w)
		}
	}
	return strings.Join(lines, "\n")
}
