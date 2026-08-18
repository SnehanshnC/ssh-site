package ui

import (
	"strconv"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/art"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// The arrival card: a banner-topped profile card, signed off in ticket 04 from
// real renders in a real terminal.
//
//	rows 0-3    the wordmark, 46x4, centred on the frame so it spans both columns
//	rows 4-21   the portrait disc, 36x18, at column 2 | the copy column at 42
//	row 22      a rule
//	row 23      the nav legend
//
// The banner is centred on the whole frame rather than left-aligned over the
// portrait: left-aligned it reads as a header for the picture alone and its
// right end lands five columns inside the copy column's origin, at which point
// the two columns stop being two columns. There is no blank row under it - the
// wordmark's bottom row is a baseline slab and the disc's top row is a thin
// Braille cap, so the air is already there and a blank row would cost the face
// a row and buy nothing.
const (
	cardCols = 80 // the composition's own width; wider frames centre it
	cardRows = 24

	bannerRow = 0
	faceCol   = 2
	faceRow   = 4
	copyCol   = 42
	copyRule  = 36 // the copy column's rule, and so the copy column's own width
	ruleRow   = 22
	legendRow = 23

	narrowCols   = 60
	narrowFace   = 4
	narrowRole   = 20
	narrowSchool = 21

	// The spec's floor. Below this the body is replaced by a plea to enlarge:
	// 57 columns is where the narrow card consumes the full width, and one
	// column is reserved for right-edge chrome before any art is fitted.
	minCols = 58
	minRows = 20
)

// Copy-column rows, as offsets from the top of the portrait. Offsets rather
// than absolutes so the block can be re-hung if the face moves. The block is
// 11 rows inside the face's 18, which leaves 3 rows of air above it and 4
// below - the copy then sits on the disc's optical centre rather than on its
// bounding box.
const (
	roleOffset      = 3
	schoolOffset    = 4
	ruleOffset      = 6
	questHeadOffset = 8
	linksOffset     = 12
)

// Card composes the arrival screen for a terminal of the given size.
func Card(pack *content.Pack, width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	// Each layout gets a fresh canvas and is taken only if it drew everything
	// it wanted to. A layout that would have to clip is not a narrower card,
	// it is a broken one - a clipped URL is a dead link and a clipped nav row
	// is a lie about which keys work - so it is rejected and the next, smaller
	// layout is tried instead.
	for _, draw := range []func(*ansi.Canvas, *content.Pack) bool{
		drawWide, drawNarrow, drawCompact,
	} {
		cv := ansi.NewCanvas(width, height)
		if draw(cv, pack) && !cv.Clipped() {
			return cv.Render()
		}
	}
	cv := ansi.NewCanvas(width, height)
	drawPlea(cv)
	return cv.Render()
}

// drawWide draws the two-column card. It needs 24 rows, and enough columns for
// the nav legend and for every copy line to find a form that fits beside the
// portrait; the widest copy line is what decides the floor, and with a link
// that cannot shorten any further that floor is 71 columns.
func drawWide(cv *ansi.Canvas, pack *content.Pack) bool {
	if cv.Rows() < cardRows {
		return false
	}
	cardW := min(cv.Cols(), cardCols)
	if cardW <= navWidth(navFull, navGap) {
		return false
	}
	copyW := cardW - copyCol
	text, ok := composeCopy(pack, copyW)
	if !ok {
		return false
	}

	ox, oy := (cv.Cols()-cardW)/2, (cv.Rows()-cardRows)/2
	banner := art.Banner()
	cv.Put(ox+(cardW-ansi.BlockWidth(banner))/2, oy+bannerRow, banner)
	cv.Put(ox+faceCol, oy+faceRow, art.Portrait(art.Wide))

	// The two rules are a parameter, not art. Between 71 and 77 columns they
	// are the only thing that would overflow, so they shorten and the card
	// survives; the bottom rule ends where the copy column's rule ends, so the
	// card has one pair of vertical edges rather than two.
	rules := min(copyRule, copyW)
	drawCopyColumn(cv, ox+copyCol, oy+faceRow, text, rules)
	cv.Rule(oy+ruleRow, ox+faceCol, copyCol+rules-faceCol, '─', dimState)
	cv.PutLine(ox+(cardW-navWidth(navFull, navGap))/2, oy+legendRow,
		navRow(navFull, navGap))
	return true
}

func drawCopyColumn(cv *ansi.Canvas, x, top int, text cardCopy, rules int) {
	cv.PutLine(x, top+roleOffset, paint(textState, text.role))
	cv.PutLine(x, top+schoolOffset, paint(textState, text.school))
	cv.Rule(top+ruleOffset, x, rules, '─', dimState)
	for i, line := range text.quest {
		cv.PutLine(x, top+questHeadOffset+i, paint(accentState, line))
	}
	for i, line := range text.links {
		cv.PutLine(x, top+linksOffset+i, line)
	}
}

// drawNarrow draws the vertical restack. Height is the binding constraint here,
// not width: the wordmark costs 4 rows and the nav 1, so a circled face - which
// must be cols x cols/2 or the disc is an ellipse - can be at most 16 rows, and
// therefore 32 columns.
//
// What it drops, and why: the two-column split, because 32 columns of disc plus
// a 36-column copy column is 68 before any gutter; the quest line and the
// links, which have nowhere to go; and `[?] help` from the nav.
func drawNarrow(cv *ansi.Canvas, pack *content.Pack) bool {
	if cv.Rows() < cardRows || cv.Cols() < minCols {
		return false
	}
	cardW := min(cv.Cols(), narrowCols)
	if cardW < navWidth(navNarrow, navNarrowGap) {
		return false
	}
	role, ok := pick(roleForms(pack.Identity), cardW)
	if !ok {
		return false
	}
	school, ok := pick(schoolForms(pack.Identity), cardW)
	if !ok {
		return false
	}

	ox, oy := (cv.Cols()-cardW)/2, (cv.Rows()-cardRows)/2
	center := func(row int, block []string) {
		cv.Put(ox+(cardW-ansi.BlockWidth(block))/2, oy+row, block)
	}
	center(bannerRow, art.Banner())
	center(narrowFace, art.Portrait(art.Narrow))
	// The spare row goes under the copy, not over it: the two copy rows are the
	// portrait's caption and want to sit on it, while the nav is a different
	// kind of thing and wants the gap.
	center(narrowRole, []string{paint(textState, role)})
	center(narrowSchool, []string{paint(textState, school)})
	center(legendRow, []string{navRow(navNarrow, navNarrowGap)})
	return true
}

// drawCompact drops the art entirely. It is what a terminal wide enough for the
// card but too short for it gets - between 20 and 23 rows there is no portrait
// that fits, and the facts are worth more than the picture.
func drawCompact(cv *ansi.Canvas, pack *content.Pack) bool {
	if cv.Cols() < minCols || cv.Rows() < minRows {
		return false
	}
	cardW := min(cv.Cols(), narrowCols)
	text, ok := composeCopy(pack, cardW)
	if !ok {
		return false
	}

	block := []string{paint(textState, text.role), paint(textState, text.school), ""}
	for _, line := range text.quest {
		block = append(block, paint(accentState, line))
	}
	block = append(block, "")
	block = append(block, text.links...)
	block = append(block, "", navRow(navNarrow, navNarrowGap))

	banner := art.Banner()
	if cv.Rows() >= len(block)+art.BannerRows+1 {
		block = append(append(append([]string{}, banner...), ""), block...)
	}
	oy := (cv.Rows() - len(block)) / 2
	for i, line := range block {
		cv.PutLine((cv.Cols()-ansi.Width(line))/2, oy+i, line)
	}
	return true
}

// drawPlea is what a terminal too small for any layout gets: the card's own
// minimum, so a visitor knows what to resize to rather than seeing a broken
// screen.
func drawPlea(cv *ansi.Canvas) {
	lines := []string{
		paint(textState, "this card needs a bigger window"),
		paint(dimState, plural(minCols, "column")+" by "+plural(minRows, "row")),
	}
	oy := (cv.Rows() - len(lines)) / 2
	for i, line := range lines {
		cv.PutLine((cv.Cols()-ansi.Width(line))/2, max(oy+i, 0), line)
	}
}

func plural(n int, unit string) string {
	return strconv.Itoa(n) + " " + unit + "s"
}
