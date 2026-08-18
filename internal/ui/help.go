package ui

import (
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// helpRows is what `?` shows. It is the whole key surface, including the keys
// that are not advertised anywhere else: `h`, which has no seat on the card's
// six-item nav row, and the page keys, which only exist once a page is open.
var helpRows = []struct{ keys, does string }{
	{"arrows", "move the highlight, select, scroll"},
	{"tab", "next nav item"},
	{"enter", "open the selected item"},
	{"esc", "back one level"},
	{"pgup pgdn", "page through long pages"},
	{"space", "page down a long page"},
	{"w p a l h", "work projects awards links hobbies"},
	{"?", "this help, esc closes"},
	{"q", "quit"},
}

// helpKeys are the keys that dismiss the overlay. Everything a visitor might
// reach for to get out of a thing gets them out of this one; `q` is not here
// because `q` quits from inside the overlay just as it does from anywhere else.
var helpKeys = map[string]bool{
	"esc": true, "enter": true, "backspace": true, "space": true, "?": true,
}

const helpPad = 2 // the air between the box's border and its contents

// drawHelp lays the key overlay over whatever is already on the canvas. It
// paints every cell of the box, including the empty ones, so the screen behind
// it cannot show through the gaps.
func drawHelp(cv *ansi.Canvas) {
	keys := 0
	for _, row := range helpRows {
		keys = max(keys, len(row.keys))
	}
	keys += helpPad

	inner := 0
	for _, row := range helpRows {
		inner = max(inner, keys+len(row.does))
	}
	inner = min(inner+2*helpPad, max(cv.Cols()-2*helpPad, 12))
	does := max(inner-2*helpPad-keys, 8)

	box := []string{paint(dimState, "┌"+strings.Repeat("─", inner)+"┐")}
	box = append(box, helpLine(gradient("keys"), inner))
	box = append(box, helpLine("", inner))
	for _, row := range helpRows {
		box = append(box, helpLine(
			paint(accentState, pad(row.keys, keys))+paint(textState, clip(row.does, does)), inner))
	}
	box = append(box, helpLine("", inner))
	box = append(box, paint(dimState, "└"+strings.Repeat("─", inner)+"┘"))

	cv.Put((cv.Cols()-inner-2)/2, max((cv.Rows()-len(box))/2, 0), box)
}

// helpLine is one row inside the box: the borders, the air, and the body padded
// out so the row is opaque all the way across.
func helpLine(body string, inner int) string {
	border := paint(dimState, "│")
	air := strings.Repeat(" ", helpPad)
	return border + air + body + strings.Repeat(" ", max(inner-2*helpPad-ansi.Width(body), 0)) +
		air + border
}

// pad runs a string out to width columns, counting columns rather than bytes
// so a name with anything but ASCII in it still lines up.
func pad(s string, width int) string {
	return s + strings.Repeat(" ", max(width-ansi.Width(s), 0))
}
