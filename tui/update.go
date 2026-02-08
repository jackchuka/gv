package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case animTickMsg:
		m.updateAnimState()
		if m.hasActiveAnimations() {
			return m, m.animTick()
		}
		m.animRunning = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case reposLoadedMsg:
		m.repos = msg.repos
		m.phase = PhaseLoading
		m.buildRows()

		if m.watcher != nil {
			for _, r := range m.repos {
				_ = m.watcher.Watch(r.Path)
			}
		}

		if len(m.repos) > 0 {
			return m, tea.Batch(m.loadStatuses(), m.ensureAnimTick())
		}
		m.phase = PhaseIdle
		return m, nil

	case statusUpdatedMsg:
		m.phase = PhaseIdle
		firstLoad := true
		for i := range m.repos {
			if m.repos[i].Status != nil {
				firstLoad = false
			}
			if s, ok := msg.statuses[m.repos[i].Path]; ok {
				m.repos[i].Status = s
				m.repos[i].LastScanned = time.Now()
			}
		}
		m.refresh()

		// Trigger diff stats load on first status load
		if firstLoad {
			m.diffLoading = true
			return m, tea.Batch(m.loadDiffStats(), m.ensureAnimTick())
		}
		return m, nil

	case diffStatsLoadedMsg:
		m.diffLoading = false
		for i := range m.repos {
			if ds, ok := msg.stats[m.repos[i].Path]; ok {
				m.repos[i].Diff = ds
			}
		}
		m.refresh()
		return m, nil

	case fetchCompletedMsg:
		m.phase = PhaseIdle
		m.fetchTarget = ""
		delete(m.anim.fetchShimmer, msg.path)

		var cmds []tea.Cmd
		if msg.err != nil {
			cmds = append(cmds, m.addToast("Fetch failed: "+msg.err.Error(), ToastError))
		} else if msg.status != nil {
			for i := range m.repos {
				if m.repos[i].Path == msg.path {
					m.repos[i].Status = msg.status
					m.repos[i].LastScanned = time.Now()
					break
				}
			}
			m.refresh()

			name := msg.path
			for _, r := range m.repos {
				if r.Path == msg.path {
					name = r.DisplayName()
					break
				}
			}
			cmds = append(cmds, m.addToast("Fetched "+name, ToastSuccess))
			cmds = append(cmds, m.refreshDiffStats(msg.path))
		}
		return m, tea.Batch(cmds...)

	case fetchAllCompletedMsg:
		m.phase = PhaseIdle
		m.fetchTarget = ""
		for k := range m.anim.fetchShimmer {
			delete(m.anim.fetchShimmer, k)
		}

		errCount := len(msg.errors)
		for i := range m.repos {
			if s, ok := msg.statuses[m.repos[i].Path]; ok {
				m.repos[i].Status = s
				m.repos[i].LastScanned = time.Now()
			}
		}
		m.refresh()

		toastMsg := "Fetched all repos"
		if errCount > 0 {
			toastMsg += fmt.Sprintf(" (%d errors)", errCount)
		}

		// Reload diff stats for all repos
		m.diffLoading = true
		return m, tea.Batch(
			m.addToast(toastMsg, ToastSuccess),
			m.loadDiffStats(),
			m.ensureAnimTick(),
		)

	case repoChangedMsg:
		m.anim.glowFade[msg.path] = 0
		var statusCmd tea.Cmd
		if msg.statusOutput != "" {
			statusCmd = m.refreshRepoFromOutput(msg.path, msg.statusOutput)
		} else {
			statusCmd = m.refreshRepo(msg.path)
		}
		return m, tea.Batch(
			statusCmd,
			m.refreshDiffStats(msg.path),
			m.listenForChanges(),
			m.ensureAnimTick(),
		)

	case errMsg:
		m.phase = PhaseIdle
		return m, m.addToast("Error: "+msg.err.Error(), ToastError)

	case toastExpiredMsg:
		for i, t := range m.toasts {
			if t.ID == msg.id {
				m.toasts = append(m.toasts[:i], m.toasts[i+1:]...)
				break
			}
		}
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Help overlay — any key closes
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Filter mode
	if m.filterMode {
		switch {
		case key.Matches(msg, m.keys.Escape):
			m.filterMode = false
			m.filterInput.Reset()
			m.filterText = ""
			m.buildRows()
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			m.filterMode = false
			m.filterText = m.filterInput.Value()
			m.buildRows()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filterText = m.filterInput.Value()
			m.buildRows()
			return m, cmd
		}
	}

	// Normal mode
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	// Navigation
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0

	case key.Matches(msg, m.keys.Bottom):
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}

	case key.Matches(msg, m.keys.HalfDown):
		m.cursor += m.visibleRows() / 2
		if m.cursor >= len(m.rows) {
			m.cursor = len(m.rows) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

	case key.Matches(msg, m.keys.HalfUp):
		m.cursor -= m.visibleRows() / 2
		if m.cursor < 0 {
			m.cursor = 0
		}

	// Filter
	case key.Matches(msg, m.keys.Filter):
		m.filterMode = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Escape):
		m.filterText = ""
		m.filterInput.Reset()
		m.buildRows()

	// Actions
	case key.Matches(msg, m.keys.Reload):
		m.phase = PhaseScanning
		return m, tea.Batch(m.loadRepos(), m.ensureAnimTick())

	case key.Matches(msg, m.keys.Fetch):
		repo := m.selectedRepo()
		if repo != nil {
			m.phase = PhaseFetching
			m.fetchTarget = repo.Path
			m.anim.fetchShimmer[repo.Path] = true
			return m, tea.Batch(m.fetchRepo(repo.Path), m.ensureAnimTick())
		}

	case key.Matches(msg, m.keys.FetchAll):
		m.phase = PhaseFetching
		m.fetchTarget = ""
		for _, row := range m.rows {
			if row.Repo != nil {
				m.anim.fetchShimmer[row.Repo.Path] = true
			}
		}
		return m, tea.Batch(m.fetchAllRepos(), m.ensureAnimTick())

	case key.Matches(msg, m.keys.Editor):
		repo := m.selectedRepo()
		if repo != nil {
			return m, m.openEditor(repo.Path)
		}

	case key.Matches(msg, m.keys.Open):
		repo := m.selectedRepo()
		if repo != nil {
			return m, m.openFinder(repo.Path)
		}

	case key.Matches(msg, m.keys.Shell):
		repo := m.selectedRepo()
		if repo != nil {
			return m, m.openShell(repo.Path)
		}

	case key.Matches(msg, m.keys.CopyPath):
		repo := m.selectedRepo()
		if repo != nil {
			return m, tea.Batch(
				m.copyToClipboard(repo.Path),
				m.addToast("Copied path", ToastInfo),
			)
		}

	// Views
	case key.Matches(msg, m.keys.ViewAll):
		m.viewFilter = ViewAll
		m.buildRows()

	case key.Matches(msg, m.keys.ViewDirty):
		m.viewFilter = ViewDirty
		m.buildRows()

	case key.Matches(msg, m.keys.ViewUnpushed):
		m.viewFilter = ViewUnpushed
		m.buildRows()

	case key.Matches(msg, m.keys.ViewConflicts):
		m.viewFilter = ViewConflicts
		m.buildRows()

	// V3 sort modes
	case key.Matches(msg, m.keys.SortDiff):
		if m.sortMode == SortDiff {
			m.sortMode = SortAlpha
		} else {
			m.sortMode = SortDiff
		}
		m.buildRows()

	case key.Matches(msg, m.keys.SortChurn):
		if m.sortMode == SortChurn {
			m.sortMode = SortAlpha
		} else {
			m.sortMode = SortChurn
		}
		m.buildRows()

	// V3 detail panel toggle
	case key.Matches(msg, m.keys.Detail):
		m.showDetail = !m.showDetail

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
	}

	return m, nil
}

func (m *Model) visibleRows() int {
	// header(2) + summary(5) + table header(1) + footer(2) = 10
	avail := m.height - 10
	if avail < 1 {
		avail = 1
	}
	return avail
}
