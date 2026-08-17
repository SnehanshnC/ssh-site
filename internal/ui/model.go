// Package ui holds the Bubble Tea model rendered to SSH visitors.
package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

var (
	nameStyle    = lipgloss.NewStyle().Bold(true)
	taglineStyle = lipgloss.NewStyle().Faint(false)
	hintStyle    = lipgloss.NewStyle().Faint(true)
)

// Model is the Bubble Tea model for the SSH site's landing view. It is
// deliberately minimal for now: name, role, the first tagline, and a quit
// hint - no banner art, that lands in a later build issue.
type Model struct {
	identity content.Identity
	width    int
	height   int
}

// New builds a Model from the loaded content pack's identity and the
// session's initial PTY window size.
func New(pack *content.Pack, width, height int) Model {
	return Model{
		identity: pack.Identity,
		width:    width,
		height:   height,
	}
}

// Init implements tea.Model. There is no initial command to run.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It quits on q, esc, or ctrl+c, and tracks the
// terminal size as it changes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var lines []string

	lines = append(lines, nameStyle.Render(m.identity.Name))
	lines = append(lines, fmt.Sprintf("%s at %s", m.identity.Role.Title, m.identity.Role.Company))

	if len(m.identity.Taglines) > 0 {
		lines = append(lines, "", taglineStyle.Render(m.identity.Taglines[0]))
	}

	lines = append(lines, "", hintStyle.Render("press q to quit"))

	body := strings.Join(lines, "\n")

	if m.width > 0 && m.height > 0 {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}

	return tea.NewView(body)
}
