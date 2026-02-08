package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	HalfDown key.Binding
	HalfUp   key.Binding

	// Filter & input
	Filter key.Binding
	Escape key.Binding
	Enter  key.Binding

	// Actions
	Reload   key.Binding
	Fetch    key.Binding
	FetchAll key.Binding
	Open     key.Binding
	Editor   key.Binding
	Shell    key.Binding
	CopyPath key.Binding

	// Views
	ViewAll       key.Binding
	ViewDirty     key.Binding
	ViewUnpushed  key.Binding
	ViewConflicts key.Binding

	// V3 sort modes
	SortDiff  key.Binding
	SortChurn key.Binding

	// V3 detail toggle
	Detail key.Binding

	// Meta
	Help key.Binding
	Quit key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "bottom"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "½ page down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "½ page up"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
		Fetch: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fetch"),
		),
		FetchAll: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "fetch all"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open finder"),
		),
		Editor: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "editor"),
		),
		Shell: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "shell"),
		),
		CopyPath: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy path"),
		),
		ViewAll: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "all"),
		),
		ViewDirty: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "dirty"),
		),
		ViewUnpushed: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "ahead"),
		),
		ViewConflicts: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "conflict"),
		),
		SortDiff: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "sort:diff"),
		),
		SortChurn: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "sort:churn"),
		),
		Detail: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "detail"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) helpText() string {
	format := func(b key.Binding) string {
		h := b.Help()
		return "  " + padRight(h.Key, 12) + h.Desc
	}

	return `Navigation
` + format(k.Up) + `
` + format(k.Down) + `
` + format(k.Top) + `
` + format(k.Bottom) + `
` + format(k.HalfDown) + `
` + format(k.HalfUp) + `
` + format(k.Filter) + `
` + format(k.Escape) + `

Actions
` + format(k.Reload) + `
` + format(k.Fetch) + `
` + format(k.FetchAll) + `
` + format(k.Editor) + `
` + format(k.Open) + `
` + format(k.CopyPath) + `
` + format(k.Shell) + `

Views & Sort
` + format(k.ViewAll) + `
` + format(k.ViewDirty) + `
` + format(k.ViewUnpushed) + `
` + format(k.ViewConflicts) + `
` + format(k.SortDiff) + `
` + format(k.SortChurn) + `
` + format(k.Detail) + `

` + format(k.Help) + `
` + format(k.Quit)
}
