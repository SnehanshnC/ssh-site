package ui

import (
	"strconv"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/art"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// The arrival card: a banner-topped profile card, signed off in ticket 04 from
// real renders in a real terminal, with the one change the navigation shell
// makes to it.
//
//	rows 0-3    the wordmark, 46x4, centred on the frame so it spans both columns
//	rows 4-21   the portrait disc, 36x18, at column 2 | the copy column at 42
//	row 22      the live nav row
//
// The banner is centred on the whole frame rather than left-aligned over the
// portrait: left-aligned it reads as a header for the picture alone and its
// right end lands five columns inside the copy column's origin, at which point
// the two columns stop being two columns. There is no blank row under it - the
// wordmark's bottom row is a baseline slab and the disc's top row is a thin
// Braille cap, so the air is already there and a blank row would cost the face
// a row and buy nothing.
//
// What the shell changed: the card's own bottom two rows - a static rule and a
// static key legend - are gone, and one live nav row stands in their place. The
// row is the legend unchanged apart from a ground under the item the visitor is
// on, so the composition is a row shorter and nothing else about it moved.
const (
	cardCols = 80 // the composition's own width; wider frames centre it
	cardRows = 23

	bannerRow = 0
	faceCol   = 2
	faceRow   = 4
	copyCol   = 42
	copyRule  = 36 // the copy column's rule, and so the copy column's own width
	liveRow   = 22 // the live nav row, standing where the rule and legend were

	narrowCols   = 60
	narrowFace   = 4
	narrowRole   = 20
	narrowSchool = 21

	// chromeCol is the column held back at the right edge of every screen, at
	// every width, before any art is fitted. At exactly 57 columns the narrow
	// card consumes the full width and leaves nothing for persistent chrome,
	// which is why the floor below is 58 and not 57 - and why the reservation
	// has to happen before the fit, not after it.
	chromeCol = 1

	// The spec's floor, in the visitor's own columns and rows, and what is left
	// of it for the art once the chrome column is held back.
	minCols = 58
	minRows = 20
	minArt  = minCols - chromeCol

	// noHighlight is the live index that highlights nothing, which is the
	// static legend the card arrived with before the shell put a cursor on it.
	noHighlight = -1
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

// card is one visitor's arrival screen: the facts every visitor sees, and the
// rung of the render ladder this visitor's terminal earned. The two travel
// together because every layout below draws a portrait, and by the time the
// layout ladder runs, which portrait that is has already been settled from the
// session environment - see internal/capability.
type card struct {
	pack *content.Pack
	tier art.Tier
}

// Card composes the arrival screen for a terminal of the given size and render
// tier, with the nav row's live item at index live.
func Card(pack *content.Pack, tier art.Tier, width, height, live int) string {
	cv, _ := card{pack, tier}.fit(width, height, live)
	return cv.Render()
}

// cardNav returns the legend the card draws at a given size. The shell needs it
// to know how far the highlight can travel and what pressing enter over it
// means, and it has to come from the layout ladder rather than from the width
// alone, because which legend is drawn is decided by which card fits.
func cardNav(pack *content.Pack, tier art.Tier, width, height int) []navItem {
	_, nav := card{pack, tier}.fit(width, height, -1)
	return nav
}

// fit draws the first layout that fits and reports the legend it drew.
//
// Each layout gets a fresh canvas and is taken only if it drew everything it
// wanted to. A layout that would have to clip is not a narrower card, it is a
// broken one - a clipped URL is a dead link and a clipped nav row is a lie
// about which keys work - so it is rejected and the next, smaller layout is
// tried instead. The canvas is one column short of the terminal at every size:
// that is the chrome column, reserved before the art is offered any of it.
//
// The tier never enters this decision. Every tier of one size occupies the same
// cell budget, so which card fits is settled by the visitor's window alone and
// two visitors at the same size get the same composition however different
// their terminals are.
func (c card) fit(width, height, live int) (*ansi.Canvas, []navItem) {
	for _, layout := range []struct {
		nav  []navItem
		draw func(*ansi.Canvas, int) bool
	}{
		{navFull, c.drawWide},
		{navNarrow, c.drawNarrow},
		{navNarrow, c.drawCompact},
	} {
		cv := ansi.NewCanvas(max(width-chromeCol, 0), height)
		if layout.draw(cv, live) && !cv.Clipped() {
			return cv, layout.nav
		}
	}
	cv := ansi.NewCanvas(max(width-chromeCol, 0), height)
	drawPlea(cv)
	return cv, nil
}

// drawWide draws the two-column card. It needs 23 rows, and enough columns for
// the nav row and for every copy line to find a form that fits beside the
// portrait; the widest copy line is what decides the floor, and with a link
// that cannot shorten any further that floor is 71 columns of art.
func (c card) drawWide(cv *ansi.Canvas, live int) bool {
	if cv.Rows() < cardRows {
		return false
	}
	cardW := min(cv.Cols(), cardCols)
	if cardW <= navWidth(navFull, navGap) {
		return false
	}
	copyW := cardW - copyCol
	text, ok := composeCopy(c.pack, copyW)
	if !ok {
		return false
	}

	ox, oy := (cv.Cols()-cardW)/2, (cv.Rows()-cardRows)/2
	banner := art.Banner(c.tier)
	cv.Put(ox+(cardW-ansi.BlockWidth(banner))/2, oy+bannerRow, banner)
	cv.Put(ox+faceCol, oy+faceRow, art.Portrait(art.Wide, c.tier))

	// The copy column's rule is a parameter, not art. Between 71 and 77 columns
	// of art it is the only thing that would overflow, so it shortens and the
	// card survives.
	drawCopyColumn(cv, ox+copyCol, oy+faceRow, text, min(copyRule, copyW))
	cv.PutLine(ox+(cardW-navWidth(navFull, navGap))/2, oy+liveRow,
		navRow(navFull, navGap, live))
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
func (c card) drawNarrow(cv *ansi.Canvas, live int) bool {
	if cv.Rows() < cardRows || cv.Cols() < minArt {
		return false
	}
	cardW := min(cv.Cols(), narrowCols)
	if cardW < navWidth(navNarrow, navNarrowGap) {
		return false
	}
	role, ok := pick(roleForms(c.pack.Identity), cardW)
	if !ok {
		return false
	}
	school, ok := pick(schoolForms(c.pack.Identity), cardW)
	if !ok {
		return false
	}

	ox, oy := (cv.Cols()-cardW)/2, (cv.Rows()-cardRows)/2
	center := func(row int, block []string) {
		cv.Put(ox+(cardW-ansi.BlockWidth(block))/2, oy+row, block)
	}
	center(bannerRow, art.Banner(c.tier))
	center(narrowFace, art.Portrait(art.Narrow, c.tier))
	center(narrowRole, []string{paint(textState, role)})
	center(narrowSchool, []string{paint(textState, school)})
	center(liveRow, []string{navRow(navNarrow, navNarrowGap, live)})
	return true
}

// drawCompact drops the art entirely. It is what a terminal wide enough for the
// card but too short for it gets - between 20 and 22 rows there is no portrait
// that fits, and the facts are worth more than the picture.
func (c card) drawCompact(cv *ansi.Canvas, live int) bool {
	if cv.Cols() < minArt || cv.Rows() < minRows {
		return false
	}
	cardW := min(cv.Cols(), narrowCols)
	text, ok := composeCopy(c.pack, cardW)
	if !ok {
		return false
	}

	block := []string{paint(textState, text.role), paint(textState, text.school), ""}
	for _, line := range text.quest {
		block = append(block, paint(accentState, line))
	}
	block = append(block, "")
	block = append(block, text.links...)
	block = append(block, "", navRow(navNarrow, navNarrowGap, live))

	banner := art.Banner(c.tier)
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
