// Package ui holds the Bubble Tea model rendered to SSH visitors.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// Model is the Bubble Tea model for the SSH site's landing view: the arrival
// card, composed for whatever size the visitor's terminal is. Navigation
// arrives in a later build issue - for now the nav legend is a legend.
type Model struct {
	pack   *content.Pack
	width  int
	height int
}

// New builds a Model from the loaded content pack and the session's initial
// PTY window size.
func New(pack *content.Pack, width, height int) Model {
	return Model{pack: pack, width: width, height: height}
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
	return tea.NewView(Card(m.pack, m.width, m.height))
}
