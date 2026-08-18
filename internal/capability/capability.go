// Package capability decides which rung of the render ladder a visitor gets,
// from the environment their SSH session arrives with.
//
// # What can actually be read
//
// An SSH session carries the visitor's terminal in two places, and they are not
// equally reliable. `TERM` rides in the pty-req and is always there, because
// OpenSSH sends it unconditionally. Everything else - `COLORTERM`,
// `TERM_PROGRAM` - only arrives if the client was configured to forward it,
// which almost none are: OpenSSH ships `SendEnv LANG LC_*` and nothing more.
//
// That asymmetry is what shapes the ladder below. `TERM` is the one signal
// that is always present and is therefore what the mainstream tier is decided
// on; the forwarded variables can only ever add information, never take it
// away, so they promote a session and never demote one.
//
// # The two questions, and why they collapse into one ladder
//
// A tier answers two questions at once: can this terminal paint truecolor, and
// which glyphs can it be trusted to draw.
//
// The colour question is binary here, and that was settled in ticket 04 rather
// than by this package: 256-colour quantisation of a colour master paints the
// jaw and the neck a saturated red, so it was measured and rejected outright.
// Truecolor or no colour, never the middle. So a terminal that cannot be
// credited with truecolor does not get a coarser render of the photograph - it
// gets the drawing, which is the whole reason a hand-drawn portrait was kept as
// an asset.
//
// The glyph question then chooses between the three renders of the photograph,
// and it is answered conservatively in one direction only: a terminal that does
// carry sextants but is not on the list loses a little sharpness, while a
// terminal on the list that does not carry them renders a wall of tofu. So the
// list grows on evidence and never on optimism.
//
// # Why `xterm-256color` is credited with truecolor
//
// Because it is the disguise every truecolor terminal arrives in. OpenSSH does
// not forward `COLORTERM`, so a visitor on a terminal that paints 16 million
// colours reaches this server announcing 256 of them, and there is no way to
// tell them apart from the environment alone. Given the choice between serving
// the mainstream visitor the rejected 256-colour render and serving them the
// tier the card was signed off in, this credits the `-256color` and `-direct`
// suffixes with truecolor and paints.
package capability

import (
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/art"
)

// sextantTerms are `TERM` values whose terminal draws Unicode 13's sextants
// from its own built-in geometry. Ticket 04 named these four; no font in
// general circulation carries the block, so a terminal that is not one of them
// cannot be given this tier on the strength of its font.
var sextantTerms = map[string]bool{
	"xterm-kitty":   true, // kitty
	"kitty":         true,
	"foot":          true, // foot
	"foot-extra":    true,
	"foot-direct":   true,
	"xterm-ghostty": true, // Ghostty
	"ghostty":       true,
	"wezterm":       true, // WezTerm, when it is configured to say so
}

// sextantPrograms are `TERM_PROGRAM` values for the same four terminals. Two of
// them - WezTerm and Ghostty - default to a `TERM` that says nothing about
// them, so this is the only way they can be recognised, and only when the
// visitor forwards it.
var sextantPrograms = map[string]bool{
	"wezterm": true,
	"ghostty": true,
}

// quadPrograms are `TERM_PROGRAM` values for terminals that are modern enough
// to be trusted with quadrant blocks and truecolor. They only ever matter when
// `TERM` did not already settle it.
var quadPrograms = map[string]bool{
	"iterm.app":      true,
	"apple_terminal": true,
	"vscode":         true,
	"hyper":          true,
	"tabby":          true,
	"warpterminal":   true,
	"rio":            true,
	"alacritty":      true,
}

// quadTerms are bare `TERM` names - no colour suffix to go on - belonging to
// terminals that are nonetheless known to be modern GPU terminals with full
// coverage of the block-element range.
var quadTerms = map[string]bool{
	"alacritty": true,
	"contour":   true,
	"rio":       true,
}

// Detect picks the tier for one session. term is the `TERM` from its pty-req
// and environ is the `KEY=VALUE` list the client forwarded; either may be
// empty, and an empty or unrecognised one degrades down the ladder rather than
// failing, because the bottom rung renders on anything.
func Detect(term string, environ []string) art.Tier {
	term = strings.ToLower(strings.TrimSpace(term))
	env := parseEnv(environ)

	// NO_COLOR is the one convention that maps exactly onto a rung this ladder
	// already has, so a visitor who asks for no colour is taken at their word
	// even on the terminal that would otherwise get the best tier. CLICOLOR
	// and CLICOLOR_FORCE are deliberately not read: they arbitrate between
	// colour and no colour for a program deciding whether to style its output,
	// which is not the question here.
	if env["no_color"] != "" {
		return art.Colorless
	}

	if sextantTerms[term] || sextantPrograms[env["term_program"]] {
		return art.Sextant
	}
	if !truecolor(term, env) {
		return art.Colorless
	}
	if quadGlyphs(term, env) {
		return art.Quad
	}
	// Colour, but nothing that vouches for the glyphs: half blocks are CP437
	// and predate every font this could be running against.
	return art.VHalf
}

// truecolor reports whether anything in the session vouches for 24-bit colour.
func truecolor(term string, env map[string]string) bool {
	switch env["colorterm"] {
	case "truecolor", "24bit":
		return true
	}
	return quadGlyphs(term, env)
}

// quadGlyphs reports whether the terminal named itself, as opposed to merely
// having its colour vouched for by a forwarded COLORTERM. Naming itself is
// what earns the quadrant blocks: `COLORTERM=truecolor` says what the terminal
// can paint and nothing at all about what it can draw.
func quadGlyphs(term string, env map[string]string) bool {
	if quadTerms[term] || quadPrograms[env["term_program"]] {
		return true
	}
	return strings.HasSuffix(term, "-256color") || strings.Contains(term, "-direct")
}

// parseEnv lowercases both halves of the forwarded environment. The keys
// because a client is free to send `Colorterm`, and the values because
// TERM_PROGRAM's casing is whatever each terminal felt like - `WezTerm`,
// `iTerm.app`, `Apple_Terminal`.
func parseEnv(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, kv := range environ {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		env[strings.ToLower(strings.TrimSpace(key))] = strings.ToLower(strings.TrimSpace(value))
	}
	return env
}
