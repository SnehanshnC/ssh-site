package ui

import "github.com/SnehanshnC/ssh-site/internal/content"

// The links section: every link the pack carries, in the order it wrote them.
//
// LINKS is a document rather than a list. There is nothing under a link to
// drill into - the address is the whole of what a link is - so the page
// scrolls rather than carrying a cursor, and Chrome leaves Selectable at its
// zero value the way work.go's employer detail and projects.go's project
// detail do for the same reason.

// openLinks returns the page `l` and the card's `[l] links` item open.
func openLinks(pack *content.Pack) Page { return linksPage{links: pack.Links} }

// linksPage is the section: one block per link, the label at the body's own
// left edge and the address wrapped whole underneath it at bodyIndent.
type linksPage struct{ links []content.Link }

func (p linksPage) Chrome() Chrome {
	return Chrome{
		Title:  "LINKS",
		Suffix: count(len(p.links), "link", "links"),
		Crumbs: []string{"links"},
	}
}

func (p linksPage) Key(string, int) (Action, Page) { return Ignored, nil }

// Blocks lays each link out as its label and its address, with the air under
// them as part of the same block: the cursor does not address this page, but
// a block is still one link's worth of rows, kept together the way every
// other section's blocks are.
func (p linksPage) Blocks(width, _ int) [][]string {
	blocks := make([][]string, 0, len(p.links))
	for _, link := range p.links {
		rows := append([]string{bodyLine(textState, link.Label, 0, width)},
			urlRows(link.URL, width)...)
		blocks = append(blocks, append(rows, ""))
	}
	return blocks
}
