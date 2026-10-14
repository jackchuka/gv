package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackchuka/gv/internal/config"
)

func TestHelpFitsShortTerminal(t *testing.T) {
	m := NewModel(config.NewConfig())
	m.width, m.height = 80, 20
	m.showHelp = true

	if h := lipgloss.Height(m.View()); h > m.height {
		t.Fatalf("help view height = %d, want <= %d", h, m.height)
	}
	if !strings.Contains(m.View(), "up") {
		t.Errorf("help view should start with the first bindings")
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if !m.showHelp {
		t.Fatal("scroll key closed the help overlay")
	}
	view := m.View()
	if h := lipgloss.Height(view); h > m.height {
		t.Fatalf("scrolled help view height = %d, want <= %d", h, m.height)
	}
	if !strings.Contains(view, "quit") {
		t.Errorf("scrolling to the bottom should reveal the last binding, got:\n%s", view)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.showHelp {
		t.Error("non-scroll key should close the help overlay")
	}
}
