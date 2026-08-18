package ui

import "github.com/SnehanshnC/ssh-site/internal/ansi"

// The card's palette, carried over from the ticket-04 prototype unchanged.
// Every one is truecolor and every cell painted with them names a background
// too, so the card looks the same on a light terminal as on a dark one.
var (
	textState   = ansi.State{FG: "38;2;226;232;240"} // slate 200
	dimState    = ansi.State{FG: "38;2;100;116;139"} // slate 500
	accentState = ansi.State{FG: "38;2;34;211;238"}  // cyan 400
	keyState    = ansi.State{FG: "38;2;168;85;247"}  // violet 500
)

// paint prefixes s with a state and closes it, so runs can be concatenated on
// one line without bleeding into each other. The canvas resolves and re-emits
// every cell anyway, so a redundant reset here costs nothing in the output.
func paint(state ansi.State, s string) string {
	if s == "" {
		return ""
	}
	return state.Prefix() + s + "\x1b[0m"
}
