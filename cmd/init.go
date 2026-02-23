package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jackchuka/gv/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up gv config interactively",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

type initStep int

const (
	stepWelcome   initStep = iota
	stepOverwrite          // only if config exists
	stepPaths
	stepConfirm
	stepDone
)

var (
	styleInitTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("73"))
	styleInitSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("71"))
	styleInitWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	styleInitDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
)

type initModel struct {
	step         initStep
	input        textinput.Model
	paths        []string
	warnings     map[int]string // index → warning message
	configPath   string
	configExists bool
	err          error
	cancelled    bool
	needOnePath  bool
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	_, err := os.Stat(configPath)
	configExists := err == nil

	ti := textinput.New()
	ti.Placeholder = "~/ghq/github.com"
	ti.CharLimit = 256
	ti.Width = 50

	m := &initModel{
		step:         stepWelcome,
		input:        ti,
		warnings:     make(map[int]string),
		configPath:   configPath,
		configExists: configExists,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}

	if final, ok := result.(*initModel); ok && final.err != nil {
		return final.err
	}

	return nil
}

func (m *initModel) Init() tea.Cmd {
	return nil
}

func (m *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Global quit
		if key == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.step {
		case stepWelcome:
			if key == "enter" {
				if m.configExists {
					m.step = stepOverwrite
				} else {
					m.step = stepPaths
					m.input.Focus()
					return m, textinput.Blink
				}
			}
			if key == "q" || key == "esc" {
				m.cancelled = true
				return m, tea.Quit
			}

		case stepOverwrite:
			if key == "y" || key == "Y" {
				m.step = stepPaths
				m.input.Focus()
				return m, textinput.Blink
			}
			m.cancelled = true
			return m, tea.Quit

		case stepPaths:
			if key == "enter" {
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					if isDuplicate(m.paths, val) {
						m.input.Reset()
						return m, nil
					}
					expanded, exists := expandAndCheck(val)
					m.paths = append(m.paths, val)
					if !exists {
						m.warnings[len(m.paths)-1] = fmt.Sprintf("  %s does not exist yet", expanded)
					}
					m.input.Reset()
					m.needOnePath = false
				} else if len(m.paths) > 0 {
					m.step = stepConfirm
				} else {
					m.needOnePath = true
				}
				return m, nil
			}
			if key == "esc" {
				m.cancelled = true
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		case stepConfirm:
			if key == "enter" {
				cfg := config.NewConfig()
				cfg.ScanPaths = m.paths
				if err := config.Save(cfg, m.configPath); err != nil {
					m.err = err
				}
				m.step = stepDone
				return m, tea.Quit
			}
			if key == "esc" {
				m.step = stepPaths
				m.input.Focus()
				return m, textinput.Blink
			}

		case stepDone:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *initModel) View() string {
	var b strings.Builder

	switch m.step {
	case stepWelcome:
		b.WriteString(styleInitTitle.Render("Welcome to gv!"))
		b.WriteString("\n\n")
		b.WriteString("Config will be saved to ")
		b.WriteString(styleInitDim.Render(m.configPath))
		b.WriteString("\n\n")
		b.WriteString(styleInitDim.Render("Press Enter to continue, Esc to cancel"))
		b.WriteString("\n")

	case stepOverwrite:
		b.WriteString(styleInitWarn.Render("Config already exists"))
		b.WriteString(" at ")
		b.WriteString(styleInitDim.Render(m.configPath))
		b.WriteString("\n\n")
		b.WriteString("Overwrite? ")
		b.WriteString(styleInitDim.Render("[y/N]"))
		b.WriteString("\n")

	case stepPaths:
		b.WriteString(styleInitTitle.Render("Scan paths"))
		b.WriteString("\n\n")
		if len(m.paths) > 0 {
			for i, p := range m.paths {
				b.WriteString(styleInitSuccess.Render("  + " + p))
				b.WriteString("\n")
				if w, ok := m.warnings[i]; ok {
					b.WriteString(styleInitWarn.Render(w))
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
		}
		if len(m.paths) == 0 {
			b.WriteString("Enter a directory to scan for git repos:\n")
		} else {
			b.WriteString("Enter another path (or press Enter to finish):\n")
		}
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.needOnePath {
			b.WriteString(styleInitWarn.Render("  Add at least one path"))
			b.WriteString("\n")
		}

	case stepConfirm:
		b.WriteString(styleInitTitle.Render("Ready to write config"))
		fmt.Fprintf(&b, " with %d scan path(s):\n\n", len(m.paths))
		for _, p := range m.paths {
			b.WriteString("  - " + p + "\n")
		}
		b.WriteString("\n")
		b.WriteString(styleInitDim.Render("[Enter] Write config  [Esc] Go back"))
		b.WriteString("\n")

	case stepDone:
		if m.err != nil {
			b.WriteString(styleInitWarn.Render("Error: " + m.err.Error()))
			b.WriteString("\n")
		} else {
			b.WriteString(styleInitSuccess.Render("Config saved to " + m.configPath))
			b.WriteString("\n\n")
			b.WriteString("Run ")
			b.WriteString(styleInitTitle.Render("gv"))
			b.WriteString(" to start monitoring your repos!\n")
		}
	}

	return b.String()
}

func expandAndCheck(path string) (expanded string, exists bool) {
	expanded = config.ExpandHome(path)
	_, err := os.Stat(expanded)
	return expanded, err == nil
}

func isDuplicate(paths []string, candidate string) bool {
	return slices.Contains(paths, candidate)
}
