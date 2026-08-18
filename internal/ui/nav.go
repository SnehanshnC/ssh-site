package ui

import "strings"

// navItem is one entry of the card's key legend. These are this surface's own
// navigation, not facts about anyone, so they live here rather than in the
// content pack.
type navItem struct {
	key   string
	label string
}

// navFull is the six-item legend the card was signed off with.
var navFull = []navItem{
	{"w", "work"}, {"p", "projects"}, {"a", "awards"},
	{"l", "links"}, {"?", "help"}, {"q", "quit"},
}

// navNarrow is the restack's legend. `[?] help` is the one item discoverable by
// pressing the key it names, so it is the one to drop; that takes the row from
// 70 columns to 55 and keeps every content section.
var navNarrow = []navItem{
	{"w", "work"}, {"p", "projects"}, {"a", "awards"},
	{"l", "links"}, {"q", "quit"},
}

const (
	navGap       = 3
	navNarrowGap = 2
)

// navRow renders the legend with one item live. It is the signed-off row
// unchanged - the same items, the same key colour, the same gaps, so the row
// keeps its width and the card keeps its geometry - with a ground painted under
// whichever item the visitor is on. A live index outside the row highlights
// nothing, which is the static legend the card arrived with.
func navRow(items []navItem, gap, live int) string {
	parts := make([]string, len(items))
	for i, item := range items {
		if i == live {
			parts[i] = paint(liveState, "["+item.key+"] "+item.label)
			continue
		}
		parts[i] = paint(keyState, "["+item.key+"]") + " " + paint(textState, item.label)
	}
	return strings.Join(parts, strings.Repeat(" ", gap))
}

func navWidth(items []navItem, gap int) int {
	w := gap * (len(items) - 1)
	for _, item := range items {
		w += len(item.key) + 3 + len(item.label) // "[k] label"
	}
	return w
}
