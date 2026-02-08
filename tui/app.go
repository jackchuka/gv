package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackchuka/gv/internal/config"
	"github.com/jackchuka/gv/internal/model"
	"github.com/jackchuka/gv/internal/scanner"
	"github.com/jackchuka/gv/internal/status"
	"github.com/jackchuka/gv/internal/watcher"
)

type Phase int

const (
	PhaseIdle Phase = iota
	PhaseScanning
	PhaseLoading
	PhaseFetching
)

type ViewFilter int

const (
	ViewAll ViewFilter = iota
	ViewDirty
	ViewUnpushed
	ViewConflicts
)

type SortMode int

const (
	SortAlpha SortMode = iota
	SortDiff
	SortChurn
)

type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastError
)

type Toast struct {
	ID        int
	Message   string
	Level     ToastLevel
	CreatedAt time.Time
}

type TableRow struct {
	Repo *model.Repository
}

type AnimState struct {
	frame        int
	fetchShimmer map[string]bool
	glowFade     map[string]int
}

func newAnimState() AnimState {
	return AnimState{
		fetchShimmer: make(map[string]bool),
		glowFade:     make(map[string]int),
	}
}

type SummaryData struct {
	TotalRepos    int
	DirtyRepos    int
	AheadRepos    int
	BehindRepos   int
	InSyncRepos   int
	ConflictRepos int

	TotalStaged    int
	TotalModified  int
	TotalUntracked int

	TotalAdded   int
	TotalDeleted int
	TotalNet     int

	DailyCommits [7]int
}

type Model struct {
	cfg    *config.Config
	repos  []model.Repository
	rows   []TableRow
	cursor int

	width, height int
	scrollOffset  int

	phase       Phase
	filterMode  bool
	filterInput textinput.Model
	filterText  string
	viewFilter  ViewFilter
	sortMode    SortMode
	showHelp    bool
	showDetail  bool
	cdPath      string

	summary SummaryData
	anim    AnimState
	toasts  []Toast

	scanner     scanner.Scanner
	reader      status.Reader
	watcher     watcher.RepoWatcher
	watchCancel context.CancelFunc

	keys        keyMap
	fetchTarget string
	diffLoading bool
	nextToastID int
	animRunning bool
}

func NewModel(cfg *config.Config) *Model {
	ti := textinput.New()
	ti.Placeholder = "filter repos..."
	ti.CharLimit = 50

	var w watcher.RepoWatcher
	if cfg.AutoRefresh {
		w = watcher.NewPoller(cfg.PollInterval)
	}

	return &Model{
		cfg:         cfg,
		keys:        newKeyMap(),
		scanner:     scanner.NewWalker(cfg),
		reader:      status.NewGitReader(),
		filterInput: ti,
		viewFilter:  ViewAll,
		sortMode:    SortAlpha,
		showDetail:  true,
		watcher:     w,
		anim:        newAnimState(),
	}
}

func (m *Model) Init() tea.Cmd {
	m.phase = PhaseScanning
	// Send an immediate animTickMsg (no timer) so the first tick doesn't
	// depend on tea.Tick's timer surviving the Init→BatchMsg dispatch path.
	// Subsequent ticks use tea.Tick normally via the animTickMsg handler.
	m.animRunning = true
	cmds := []tea.Cmd{
		m.loadRepos(),
		func() tea.Msg { return animTickMsg{} },
	}

	if m.watcher != nil {
		cmds = append(cmds, m.startWatcher())
	}

	return tea.Batch(cmds...)
}

type reposLoadedMsg struct{ repos []model.Repository }
type statusUpdatedMsg struct{ statuses map[string]*model.RepoStatus }
type diffStatsLoadedMsg struct{ stats map[string]*model.DiffStats }
type fetchCompletedMsg struct {
	path   string
	status *model.RepoStatus
	err    error
}
type fetchAllCompletedMsg struct {
	statuses map[string]*model.RepoStatus
	errors   map[string]error
}
type repoChangedMsg struct {
	path         string
	statusOutput string // raw porcelain output from poller (avoids re-running git status)
}
type errMsg struct{ err error }
type animTickMsg struct{}
type toastExpiredMsg struct{ id int }

func (m *Model) buildRows() {
	filtered := filterRepos(m.repos, m.viewFilter)

	// Text filter
	if m.filterText != "" {
		var textFiltered []model.Repository
		for _, r := range filtered {
			if containsIgnoreCase(r.DisplayName(), m.filterText) ||
				containsIgnoreCase(r.Path, m.filterText) {
				textFiltered = append(textFiltered, r)
			}
		}
		filtered = textFiltered
	}

	// Sort
	switch m.sortMode {
	case SortDiff:
		sort.Slice(filtered, func(i, j int) bool {
			di := diffVolume(&filtered[i])
			dj := diffVolume(&filtered[j])
			if di != dj {
				return di > dj
			}
			return filtered[i].DisplayName() < filtered[j].DisplayName()
		})
	case SortChurn:
		sort.Slice(filtered, func(i, j int) bool {
			ci := churnTotal(&filtered[i])
			cj := churnTotal(&filtered[j])
			if ci != cj {
				return ci > cj
			}
			return filtered[i].DisplayName() < filtered[j].DisplayName()
		})
	default:
		sortRepos(filtered)
	}

	// Build flat row list
	rows := make([]TableRow, len(filtered))
	for i := range filtered {
		rows[i] = TableRow{Repo: &filtered[i]}
	}

	m.rows = rows

	// Clamp cursor
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) refresh() {
	m.computeSummary()
	m.buildRows()
}

func (m *Model) computeSummary() {
	s := SummaryData{}

	for _, r := range m.repos {
		s.TotalRepos++
		if r.Status == nil {
			continue
		}
		if r.Status.IsDirty() {
			s.DirtyRepos++
		}
		if r.Status.Ahead > 0 {
			s.AheadRepos++
		}
		if r.Status.Behind > 0 {
			s.BehindRepos++
		}
		if r.Status.Remote != "" && !r.Status.IsDirty() && r.Status.Ahead == 0 && r.Status.Behind == 0 {
			s.InSyncRepos++
		}
		if r.Status.HasSpecialState() {
			s.ConflictRepos++
		}

		s.TotalStaged += r.Status.Staged
		s.TotalModified += r.Status.Modified
		s.TotalUntracked += r.Status.Untracked

		if r.Diff != nil {
			s.TotalAdded += r.Diff.TotalAdded
			s.TotalDeleted += r.Diff.TotalDeleted

			for i := 0; i < 7; i++ {
				s.DailyCommits[i] += r.Diff.DailyCommits[i]
			}
		}
	}

	s.TotalNet = s.TotalAdded - s.TotalDeleted
	m.summary = s
}

func (m *Model) selectedRepo() *model.Repository {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].Repo
}

func (m *Model) addToast(msg string, level ToastLevel) tea.Cmd {
	id := m.nextToastID
	m.nextToastID++
	t := Toast{
		ID:        id,
		Message:   msg,
		Level:     level,
		CreatedAt: time.Now(),
	}
	m.toasts = append(m.toasts, t)
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return toastExpiredMsg{id}
	})
}

func (m *Model) updateAnimState() {
	m.anim.frame++

	// Step the border flash every 3 frames (300ms) for snappy blink
	if m.anim.frame%3 == 0 {
		for path, step := range m.anim.glowFade {
			if step >= len(glowBorderColors)-1 {
				delete(m.anim.glowFade, path)
			} else {
				m.anim.glowFade[path] = step + 1
			}
		}
	}
}

func (m *Model) animTick() tea.Cmd {
	m.animRunning = true
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return animTickMsg{}
	})
}

func (m *Model) hasActiveAnimations() bool {
	return len(m.anim.glowFade) > 0 || len(m.anim.fetchShimmer) > 0 || m.phase != PhaseIdle || m.diffLoading
}

func (m *Model) ensureAnimTick() tea.Cmd {
	if m.animRunning {
		return nil
	}
	return m.animTick()
}

func (m *Model) loadRepos() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		repos, err := m.scanner.Scan(ctx)
		if err != nil {
			return errMsg{err}
		}
		return reposLoadedMsg{repos}
	}
}

func (m *Model) loadStatuses() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		paths := make([]string, len(m.repos))
		for i, r := range m.repos {
			paths[i] = r.Path
		}

		statuses, _ := m.reader.GetStatusBatch(ctx, paths)
		return statusUpdatedMsg{statuses}
	}
}

func (m *Model) loadDiffStats() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		paths := make([]string, len(m.repos))
		for i, r := range m.repos {
			paths[i] = r.Path
		}

		stats := m.reader.GetDiffStatsBatch(ctx, paths)
		return diffStatsLoadedMsg{stats: stats}
	}
}

func (m *Model) refreshRepo(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s, err := m.reader.GetStatus(ctx, path)
		if err != nil {
			return errMsg{err}
		}
		return statusUpdatedMsg{map[string]*model.RepoStatus{path: s}}
	}
}

func (m *Model) refreshRepoFromOutput(path string, output string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s, err := m.reader.GetStatusFromOutput(ctx, path, output)
		if err != nil {
			return errMsg{err}
		}
		return statusUpdatedMsg{map[string]*model.RepoStatus{path: s}}
	}
}

func (m *Model) refreshDiffStats(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ds := m.reader.GetDiffStats(ctx, path)
		return diffStatsLoadedMsg{stats: map[string]*model.DiffStats{path: ds}}
	}
}

func (m *Model) fetchRepo(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()

		st, err := m.reader.Fetch(ctx, path)
		return fetchCompletedMsg{path: path, status: st, err: err}
	}
}

func (m *Model) fetchAllRepos() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		paths := make([]string, len(m.repos))
		for i, r := range m.repos {
			paths[i] = r.Path
		}

		statuses, errors := m.reader.FetchBatch(ctx, paths)
		return fetchAllCompletedMsg{statuses: statuses, errors: errors}
	}
}

func (m *Model) startWatcher() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.watchCancel = cancel
	go m.watcher.Run(ctx)
	return m.listenForChanges()
}

func (m *Model) listenForChanges() tea.Cmd {
	return func() tea.Msg {
		if m.watcher == nil {
			return nil
		}
		event, ok := <-m.watcher.Events()
		if !ok {
			return nil
		}
		return repoChangedMsg{path: event.RepoPath, statusOutput: string(event.StatusOutput)}
	}
}

func (m *Model) openShell(path string) tea.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell)
	cmd.Dir = path
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return nil })
}

func (m *Model) openEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return nil })
}

func (m *Model) openFinder(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			cmd = exec.Command("explorer", path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
		_ = cmd.Run()
		return nil
	}
}

func (m *Model) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "windows":
			cmd = exec.Command("clip")
		default:
			// Try xclip first, fall back to xsel
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			}
		}
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return nil
	}
}

func filterRepos(repos []model.Repository, filter ViewFilter) []model.Repository {
	if filter == ViewAll {
		return repos
	}
	var filtered []model.Repository
	for _, r := range repos {
		if r.Status == nil {
			continue
		}
		switch filter {
		case ViewDirty:
			if r.Status.IsDirty() {
				filtered = append(filtered, r)
			}
		case ViewUnpushed:
			if r.Status.Ahead > 0 {
				filtered = append(filtered, r)
			}
		case ViewConflicts:
			if r.Status.HasSpecialState() {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

func sortRepos(repos []model.Repository) {
	sort.Slice(repos, func(i, j int) bool {
		keyi := repos[i].Path
		if repos[i].IsWorktree && repos[i].MainWorktree != "" {
			keyi = repos[i].MainWorktree
		}
		keyj := repos[j].Path
		if repos[j].IsWorktree && repos[j].MainWorktree != "" {
			keyj = repos[j].MainWorktree
		}
		if keyi == keyj {
			if !repos[i].IsWorktree && repos[j].IsWorktree {
				return true
			}
			if repos[i].IsWorktree && !repos[j].IsWorktree {
				return false
			}
			return repos[i].DisplayName() < repos[j].DisplayName()
		}
		return keyi < keyj
	})
}

func diffVolume(r *model.Repository) int {
	if r.Diff == nil {
		return 0
	}
	return r.Diff.TotalDiffVolume()
}

func churnTotal(r *model.Repository) int {
	if r.Diff == nil {
		return 0
	}
	total := 0
	for _, c := range r.Diff.FileChurn {
		total += c
	}
	return total
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func specialStateLabel(s *model.RepoStatus) string {
	if s.MergeHead {
		return "MERGE"
	}
	if s.RebaseHead {
		return "REBASE"
	}
	if s.CherryPick {
		return "CHERRY-PICK"
	}
	if s.Reverting {
		return "REVERT"
	}
	if s.Bisecting {
		return "BISECT"
	}
	return ""
}

func Run(cfg *config.Config) error {
	m := NewModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()

	// Cleanup
	if m.watchCancel != nil {
		m.watchCancel()
	}
	if m.watcher != nil {
		_ = m.watcher.Close()
	}

	if err != nil {
		return err
	}

	// cd action
	if mdl, ok := result.(*Model); ok && mdl.cdPath != "" {
		fmt.Println(mdl.cdPath)
	}

	return nil
}
