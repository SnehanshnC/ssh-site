package ui

import "github.com/SnehanshnC/ssh-site/internal/content"

// The work section: the employers, and the role held at each one.
//
// This is the first section the scaffold's generic stub hands over to, and the
// shape the four after it copy - a selectable index whose count is derived from
// the pack, drilling into a document that is one entry written out in full.
// What generalises lives in page.go beside the chrome; only what is true of
// work is here.

// openWork returns the page `w` and the card's `[w] work` item open.
func openWork(pack *content.Pack) Page { return workList{jobs: pack.Work} }

// workList is the section's index: the employer and the title held there, with
// the span and the place under it.
type workList struct{ jobs []content.Job }

func (p workList) Chrome() Chrome {
	return Chrome{
		Title:      "WORK",
		Suffix:     count(len(p.jobs), "role", "roles"),
		Crumbs:     []string{"work"},
		Selectable: true,
	}
}

func (p workList) Key(key string, cursor int) (Action, Page) {
	if key == "enter" && cursor >= 0 && cursor < len(p.jobs) {
		return Push, workDetail{job: p.jobs[cursor]}
	}
	return Ignored, nil
}

func (p workList) Blocks(width, cursor int) [][]string {
	names := make([]string, len(p.jobs))
	for i, job := range p.jobs {
		names[i] = job.Company
	}
	column := nameColumn(names, width)

	blocks := make([][]string, 0, len(p.jobs))
	for i, job := range p.jobs {
		// Two rows and the air under them, as one block: the cursor addresses
		// blocks, so a role's dates never scroll away from the role's name.
		blocks = append(blocks, []string{
			listRow(width, i == cursor, job.Company, job.Role, column),
			bodyLine(dimState, span(job)+" · "+job.Location, subIndent, width),
			"",
		})
	}
	return blocks
}

// workDetail is one employer: the role, when and where it was held, what was
// done there, and where to go and look at it.
type workDetail struct{ job content.Job }

func (p workDetail) Chrome() Chrome {
	return Chrome{
		Title:  p.job.Company,
		Suffix: "work",
		Crumbs: []string{"work", p.job.Slug},
	}
}

func (p workDetail) Key(string, int) (Action, Page) { return Ignored, nil }

func (p workDetail) Blocks(width, _ int) [][]string {
	var blocks [][]string
	add := func(rows ...string) {
		if len(rows) > 0 {
			blocks = append(blocks, rows)
		}
	}

	// The title and the span are wrapped rather than shortened. A job title cut
	// short is a different job title, and unlike a list row - where the column
	// beside the name is all the room there is - a page has the width to wrap
	// them into.
	add(wrap(paint(textState, p.job.Role), width)...)
	add(wrap(paint(dimState, span(p.job)+" · "+p.job.Location), width)...)
	add("")

	if len(p.job.Highlights) > 0 {
		add(paint(dimState, "highlights"))
		for _, highlight := range p.job.Highlights {
			add(bullets(highlight, width)...)
			add("")
		}
	}
	if len(p.job.Links) > 0 {
		add(paint(dimState, "links"))
		for _, link := range p.job.Links {
			add(append([]string{bodyLine(textState, link.Label, bodyIndent, width)},
				urlRows(link.URL, width)...)...)
			add("")
		}
	}
	return blocks
}

// span is a role's dates as this surface says them: the pack's own start, and
// its end or "present" where the role is still held. The pack writes no such
// word - an open-ended role is one whose end date is simply absent, and the
// word for that is chosen here, where the reading happens.
func span(job content.Job) string {
	end := job.End
	if end == "" {
		end = "present"
	}
	return job.Start + " - " + end
}

// urlRows is an address, set under the label that names it.
//
// A truncated URL is a broken link, so this never shortens one by cutting it.
// It shows the address the pack wrote wherever that fits, falls back to the
// fullest of the card's own shortened forms that does fit - every one of which
// is still somewhere a visitor can go - and wraps the address whole rather than
// cut it if even the shortest form is too wide for the column.
func urlRows(url string, width int) []string {
	column := max(width-bodyIndent, 1)
	if shown, ok := pick(append([]string{url}, urlForms(url)...), column); ok {
		return []string{bodyLine(accentState, shown, bodyIndent, width)}
	}
	return indentRows(wrap(paint(accentState, url), column), bodyIndent)
}
