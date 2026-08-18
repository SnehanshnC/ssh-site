package ui

import (
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// The awards section: the same relation the projects section draws, read from
// the other end.
//
// The pack files awards flat, each naming the project it was won for, so a
// project page resolves that slug into the awards that point at it
// (content.AwardsFor) and this section resolves it the other way
// (content.Project). Awards being their own section rather than a heading
// inside projects is what the experience prototype settled and what `a` means.
//
// So `enter` here opens projects.go's page rather than a page of its own. An
// award has no detail the project it was won for does not already carry, and a
// second copy of that page would be a second place for it to drift; what this
// section adds is the way in, which the breadcrumb then says.

// openAwards returns the page `a` and the card's `[a] awards` item open.
func openAwards(pack *content.Pack) Page { return awardsList{pack: pack} }

// awardsList is the section: every award the pack carries, in the order it
// wrote them - how each placed on its own row, and what it was won for under it.
//
// It holds the whole pack rather than the awards alone because an award is a
// reference: both what a row says about it and where enter goes from it are on
// the far end of the slug it names.
type awardsList struct{ pack *content.Pack }

func (p awardsList) Chrome() Chrome {
	return Chrome{
		Title:      "AWARDS",
		Suffix:     count(len(p.pack.Awards), "award", "awards"),
		Crumbs:     []string{"awards"},
		Selectable: true,
	}
}

// Key drills into the project the award under the cursor was won for: the
// projects section's own page, opened with awards at the root of its breadcrumb
// because that is the way the visitor came in.
//
// An award whose slug names no project on this pack opens nothing. The award is
// still listed - it happened, and the pack says so - but a reference that does
// not resolve is the pack's to fix, and inventing a page for it here would hide
// exactly the fact that it needs fixing.
func (p awardsList) Key(key string, cursor int) (Action, Page) {
	if key != "enter" || cursor < 0 || cursor >= len(p.pack.Awards) {
		return Ignored, nil
	}
	project, ok := p.pack.Project(p.pack.Awards[cursor].Project)
	if !ok {
		return Ignored, nil
	}
	return Push, projectPage(p.pack, project, "awards")
}

// Blocks lays each award out in the shared list idiom, with no name column
// beside it: an award's placing and event are one phrase, not a name and a
// description, so the row runs to its own length rather than being split in two.
func (p awardsList) Blocks(width, cursor int) [][]string {
	blocks := make([][]string, 0, len(p.pack.Awards))
	for i, award := range p.pack.Awards {
		// The rows and the air under them as one block: the cursor addresses
		// blocks, so what an award was won for never scrolls away from it.
		rows := []string{listRow(width, i == cursor, awardHead(award), "", 0)}
		if about := p.about(award); about != "" {
			rows = append(rows, bodyLine(dimState, about, subIndent, width))
		}
		blocks = append(blocks, append(rows, ""))
	}
	return blocks
}

// about is the dim line under an award: what it was won for, and what the pack
// files beside the placing itself.
//
// The project comes first because it is what the award is for. It is left off
// where the event's own name is already the project's - a competition entered
// as itself would otherwise be named twice, once on each row. An award naming a
// project this pack does not carry is said by the slug it names instead, so an
// unresolved reference reads as the one thing that is missing rather than as a
// row with nothing under it.
func (p awardsList) about(award content.Award) string {
	var bits []string
	name := award.Project
	if project, ok := p.pack.Project(award.Project); ok {
		name = project.Name
	}
	if name != "" && name != award.Event {
		bits = append(bits, name)
	}
	for _, bit := range []string{award.Track, award.Participants, award.Prize} {
		if bit != "" {
			bits = append(bits, bit)
		}
	}
	bits = append(bits, award.Extras...)
	return strings.Join(bits, " · ")
}
