package ui

import (
	"strconv"
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// The card's palette, carried over from the ticket-04 prototype unchanged.
// Every one is truecolor and every cell painted with them names a background
// too, so the card looks the same on a light terminal as on a dark one.
var (
	textState   = ansi.State{FG: "38;2;226;232;240"} // slate 200
	dimState    = ansi.State{FG: "38;2;100;116;139"} // slate 500
	accentState = ansi.State{FG: "38;2;34;211;238"}  // cyan 400
	keyState    = ansi.State{FG: "38;2;168;85;247"}  // violet 500

	// liveState is the ground under whatever the visitor has selected: the nav
	// row's live item, and a list page's cursor row.
	//
	// Reverse video is the usual way to do this and is wrong here. Reversing
	// hands the highlight's text colour to whatever the visitor's terminal
	// paints as its background, which is exactly the inheritance every other
	// cell on this surface is written to avoid, so the highlight names both of
	// its colours like everything else.
	liveState = ansi.State{Attrs: "1", FG: "38;2;15;23;42", BG: "48;2;34;211;238"} // slate 900 on cyan 400

	// markerState is the cursor on a list page. A row is not a chip: a ground
	// under a whole row of a page reads as a block of colour rather than as a
	// selection, so a list marks its row the way a menu does, with a marker and
	// the accent colour.
	markerState = ansi.State{Attrs: "1", FG: "38;2;34;211;238"} // cyan 400, bold
)

// gradStops are the banner's own ramp, cyan through indigo to violet. The
// wordmark is a pre-rendered asset painted with it; a page header is composed
// at runtime and paints itself with it, so the two belong to one surface.
var gradStops = [3][3]int{{34, 211, 238}, {99, 102, 241}, {168, 85, 247}}

// paint prefixes s with a state and closes it, so runs can be concatenated on
// one line without bleeding into each other. The canvas resolves and re-emits
// every cell anyway, so a redundant reset here costs nothing in the output.
func paint(state ansi.State, s string) string {
	if s == "" {
		return ""
	}
	return state.Prefix() + s + "\x1b[0m"
}

// gradient paints text bold, with the banner's horizontal ramp spread over
// the text's own width - a page header's own colour, and every other place
// this surface uses the ramp at full strength.
func gradient(text string) string { return rampText(text, "1") }

// dimGradient paints text with the same ramp, dimmed rather than bold. It is
// the goodbye line's name: dim because that line prints after the session has
// already ended, an afterthought in the visitor's scrollback rather than a
// headline on screen.
func dimGradient(text string) string { return rampText(text, "2") }

// rampText spreads the banner's horizontal ramp over text's own width,
// carrying attrs on every painted rune. Spaces are left unpainted: a gradient
// is only legible on ink, and painting the gaps would put a background
// behind them.
func rampText(text, attrs string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	span := max(len(runes)-1, 1)

	var b strings.Builder
	var cur ansi.State
	for i, r := range runes {
		state := ansi.State{}
		if r != ' ' {
			state = ansi.State{Attrs: attrs, FG: ramp(float64(i) / float64(span))}
		}
		if state != cur {
			b.WriteString("\x1b[0m")
			b.WriteString(state.Prefix())
			cur = state
		}
		b.WriteRune(r)
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// ramp samples the gradient at t in [0, 1] and returns it as an SGR foreground
// parameter list.
func ramp(t float64) string {
	t = min(max(t, 0), 1)
	seg := t * float64(len(gradStops)-1)
	i := min(int(seg), len(gradStops)-2)
	f := seg - float64(i)

	out := "38;2"
	for c := range 3 {
		a, b := float64(gradStops[i][c]), float64(gradStops[i+1][c])
		out += ";" + strconv.Itoa(int(a+(b-a)*f+0.5))
	}
	return out
}
