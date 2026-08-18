package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// idleTimeout is how long a session can sit untouched before the shell ends
// it exactly the way `q` would. D4 set this at the application level as a
// visible ten minutes, well inside the SSH server's own IdleTimeout (15m) and
// MaxTimeout (1h) in cmd/ssh-site/main.go - those stay put as backstops a
// session that respects this timer never reaches.
const idleTimeout = 10 * time.Minute

// idleTickMsg is a scheduled check for whether the session has gone
// idleTimeout without a keypress. See idleGen on Model for how a stale one -
// scheduled before a keypress that has since reset the clock - is told apart
// from a live one.
type idleTickMsg struct{ gen int }

// idleTick schedules the next idle check, tagged with the keypress generation
// it was scheduled at.
func idleTick(gen int) tea.Cmd {
	return tea.Tick(idleTimeout, func(time.Time) tea.Msg {
		return idleTickMsg{gen: gen}
	})
}

// quit is the one path every way of ending a session runs through: `q`,
// `ctrl+c`, a page's own Quit action, and the idle timeout all call this
// rather than returning tea.Quit themselves, so the goodbye line has exactly
// one place it is decided rather than three (or four) kept in sync by hand.
func (m Model) quit() (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}

// goodbye is the line every one of those exits shows, composed fresh from the
// pack rather than written down here: the name in a dim run of the banner's
// own gradient, the identity section's leading tagline, and the github link.
// It is deliberately plain text past the name - the prototype it was signed
// off from prints it the same way - because it is read after the session has
// already ended, outside the chrome that styles everything else on this
// surface.
func goodbye(pack *content.Pack) string {
	lines := []string{dimGradient(pack.Identity.Name)}
	if len(pack.Identity.Taglines) > 0 {
		lines = append(lines, pack.Identity.Taglines[0])
	}
	if github, ok := pack.Link("github"); ok {
		lines = append(lines, github.URL)
	}
	return strings.Join(lines, "\n")
}
