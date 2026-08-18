package ui

import (
	"strconv"
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// Action is what a page asks the shell to do with a key it was offered. It is
// an explicit action rather than a handled/not-handled bool, and that is the
// fix for a defect the ticket-06 prototype exposed: there, the only thing that
// could end a session was an *unconsumed* literal `q`, so a page could never
// render `[q] quit` as something the cursor lands on and presses enter over.
// The card's nav row had to skip its own quit item as a workaround.
type Action int

const (
	// Ignored means the page did not handle the key; the shell applies its own
	// default for it.
	Ignored Action = iota
	// Consumed means the page handled the key and wants no navigation.
	Consumed
	// Push means open the page returned alongside the action.
	Push
	// Pop means go back one level.
	Pop
	// Quit means end the session.
	Quit
)

// Page is one screen on the stack: a gradient header, a dim breadcrumb, a body,
// and a dim row of key hints.
//
// The shell owns the chrome, the viewport, the cursor and the scroll offset; a
// page owns its body and what its own keys mean. Keeping the position out of
// the page is what lets a resize recompose every body without losing where the
// visitor was.
type Page interface {
	// Chrome names the page in its header and breadcrumb, and says whether its
	// body is a list with a cursor or a document that scrolls.
	Chrome() Chrome
	// Blocks composes the body into blocks of rendered rows, each row already
	// fitted to width columns. One block is one item on a selectable page - the
	// cursor addresses blocks, not rows, so a two-row entry moves as one thing -
	// and cursor is the block the shell has selected, for a page that marks it.
	Blocks(width, cursor int) [][]string
	// Key offers the page a key the shell has not already claimed, along with
	// the cursor the shell holds for it.
	Key(key string, cursor int) (Action, Page)
}

// Chrome is a page's description of itself.
type Chrome struct {
	Title      string   // the gradient header
	Suffix     string   // the dim note beside it, usually a count
	Crumbs     []string // the breadcrumb under it, after "home"
	Selectable bool     // its body is a list with a cursor, not a document
}

// frame is one page on the stack together with the position the visitor left it
// in. Every frame on the stack keeps its own, so walking back out of a
// drill-down returns to the row it was entered from.
type frame struct {
	page   Page
	cursor int
	scroll int
}

// The page frame, as the prototype signed it off:
//
//	row 0        blank
//	row 1        the gradient title, with a dim note beside it
//	row 2        the dim breadcrumb
//	row 3        a dim rule
//	row 4        blank
//	rows 5..     the body's viewport
//	second-last  blank
//	last row     the dim key hints
const (
	pageTitleRow = 1
	pageCrumbRow = 2
	pageRuleRow  = 3
	pageBodyRow  = 5

	// pageChromeRows is what the five header rows, the gap and the hint row
	// cost, and so what the body does not get.
	pageChromeRows = 7

	// The body's side margins. The wider one is taken from the width at which
	// the card itself stops growing, so a page indents to the same air the card
	// has around it.
	pageMarginWide   = 4
	pageMarginNarrow = 2
	pageWideCols     = 78

	// pageStep is how far pgup/pgdn moves a cursor. A page of rows is the wrong
	// step for a list, where the items are what the visitor is counting.
	pageStep = 5

	// pageSuffixCols caps the dim note beside a title, so a long one can never
	// push the title's own gradient off the row.
	pageSuffixCols = 24
)

// pageMargin is the body's left and right margin at a given terminal width.
func pageMargin(width int) int {
	if width >= pageWideCols {
		return pageMarginWide
	}
	return pageMarginNarrow
}

// pageBody is the geometry a body is composed and scrolled against: the columns
// its rows are fitted to, and the rows its viewport shows. The renderer and the
// key handling both take it from here, so a cursor can never be moved against a
// different body than the one on screen.
func pageBody(width, height int) (cols, rows int) {
	margin := pageMargin(width)
	return max(width-chromeCol-2*margin, 8), max(height-pageChromeRows, 1)
}

// renderPage draws a frame's page onto a canvas sized for the terminal's own
// width and height.
func renderPage(cv *ansi.Canvas, f frame, width, height int) {
	margin := pageMargin(width)
	cols, rows := pageBody(width, height)
	chrome := f.page.Chrome()

	head := gradient(chrome.Title)
	if chrome.Suffix != "" {
		head += "  " + paint(dimState, clip(chrome.Suffix, pageSuffixCols))
	}
	cv.PutLine(margin, pageTitleRow, clip(head, cols))
	cv.PutLine(margin, pageCrumbRow,
		paint(dimState, clip(strings.Join(append([]string{"home"}, chrome.Crumbs...), " / "), cols)))
	cv.Rule(pageRuleRow, margin, cols, '─', dimState)

	blocks := f.page.Blocks(cols, f.cursor)
	for i, row := range window(blocks, f.scroll, rows) {
		cv.PutLine(margin, pageBodyRow+i, row)
	}
	cv.PutLine(margin, cv.Rows()-1, paint(dimState, hints(f, blocks, rows, cols)))
}

// window returns the rows of a body visible at a scroll offset, padded out to
// the full viewport. It bounds the offset it is given rather than trusting it,
// so a stale offset renders a real screen instead of an empty one.
func window(blocks [][]string, scroll, rows int) []string {
	flat, _ := flatten(blocks)
	scroll = min(max(scroll, 0), max(len(flat)-rows, 0))

	out := make([]string, rows)
	for i := range out {
		if j := scroll + i; j < len(flat) {
			out[i] = flat[j]
		}
	}
	return out
}

// flatten runs the blocks together into the rows they render as, and reports
// where each block starts.
func flatten(blocks [][]string) (rows []string, starts []int) {
	starts = make([]int, len(blocks))
	for i, block := range blocks {
		starts[i] = len(rows)
		rows = append(rows, block...)
	}
	return rows, starts
}

// settle bounds a frame's cursor and scroll offset against the body it is
// looking at, and pulls the selected block fully into view. It is the one place
// either is written, so every key that moves and every resize that recomposes
// end up in a position the screen can actually show.
func (f *frame) settle(blocks [][]string, rows int, selectable bool) {
	flat, starts := flatten(blocks)
	if selectable && len(blocks) > 0 {
		f.cursor = min(max(f.cursor, 0), len(blocks)-1)
		first := starts[f.cursor]
		last := first + len(blocks[f.cursor]) - 1
		if first < f.scroll {
			f.scroll = first
		}
		if last >= f.scroll+rows {
			f.scroll = last - rows + 1
		}
	} else {
		f.cursor = 0
	}
	f.scroll = min(max(f.scroll, 0), max(len(flat)-rows, 0))
}

// hints is the dim row along the bottom: what the keys do here, and where in
// the page the visitor is. Parts are dropped from the right until the row fits,
// rather than the row being cut, so a narrow terminal loses whole hints instead
// of being told about half a key. They are ordered by what a visitor who can
// only be told one thing should be told.
func hints(f frame, blocks [][]string, rows, width int) string {
	var parts []string
	if f.page.Chrome().Selectable {
		parts = []string{"up/down move", "enter open", "esc back"}
		if len(blocks) > 0 {
			parts = append(parts,
				strconv.Itoa(f.cursor+1)+" of "+strconv.Itoa(len(blocks)))
		}
		parts = append(parts, "? keys", "h hobbies")
	} else {
		parts = []string{"up/down scroll", "esc back"}
		if flat, _ := flatten(blocks); f.scroll+rows < len(flat) {
			parts = append(parts, "more")
		}
		parts = append(parts, "? keys", "h hobbies")
	}
	for len(parts) > 1 && ansi.Width(strings.Join(parts, " · ")) > width {
		parts = parts[:len(parts)-1]
	}
	return clip(strings.Join(parts, " · "), width)
}
