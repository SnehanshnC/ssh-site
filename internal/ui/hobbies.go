package ui

import "github.com/SnehanshnC/ssh-site/internal/content"

// The hobbies section: what happens off the clock, shows included.
//
// Shows fold into this page as their own subsection rather than carrying a
// section of their own - there is no `s` jump, no [s] on the card, nothing on
// this surface named "shows" outside this page's own body. `h` itself has no
// seat on the card's six-item nav row either, settled art from the
// experience prototype's review; it is a hidden jump, advertised only in the
// hints row and the help overlay.
//
// HOBBIES is a document like LINKS is, not a list: nothing under a show or a
// hobby to drill into, so the page scrolls rather than carrying a cursor, and
// Chrome leaves Selectable at its zero value the way links.go's does.

// openHobbies returns the page `h` opens.
func openHobbies(pack *content.Pack) Page {
	return hobbiesPage{shows: pack.Shows, hobbies: pack.Hobbies}
}

// hobbiesPage is the section: the shows under their own dim heading, then the
// hobbies under theirs.
type hobbiesPage struct {
	shows   []content.Show
	hobbies []content.Hobby
}

func (p hobbiesPage) Chrome() Chrome {
	return Chrome{
		Title:  "HOBBIES",
		Suffix: "off the clock",
		Crumbs: []string{"hobbies"},
	}
}

func (p hobbiesPage) Key(string, int) (Action, Page) { return Ignored, nil }

// Blocks lays the shows out first, one block per show - the title and its
// category on a shared list row, the same idiom work.go's index lines names up
// in, with the blurb wrapped dim underneath where the pack wrote one - then the
// hobbies, one block per hobby, each wrapped whole under the heading that
// names them.
func (p hobbiesPage) Blocks(width, _ int) [][]string {
	var blocks [][]string

	if len(p.shows) > 0 {
		blocks = append(blocks, []string{bodyLine(dimState, "shows", 0, width)})
		titles := make([]string, len(p.shows))
		for i, show := range p.shows {
			titles[i] = show.Title
		}
		column := nameColumn(titles, width)
		for _, show := range p.shows {
			rows := []string{listRow(width, false, show.Title, show.Category, column)}
			if show.Blurb != "" {
				rows = append(rows, indentRows(
					wrap(paint(dimState, show.Blurb), max(width-subIndent, 1)), subIndent)...)
			}
			blocks = append(blocks, append(rows, ""))
		}
	}

	if len(p.hobbies) > 0 {
		blocks = append(blocks, []string{bodyLine(dimState, "hobbies", 0, width)})
		for _, hobby := range p.hobbies {
			blocks = append(blocks, indentRows(
				wrap(paint(textState, hobby.Detail), max(width-bodyIndent, 1)), bodyIndent))
		}
	}

	return blocks
}
