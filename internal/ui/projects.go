package ui

import (
	"strconv"
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// The projects section: what was built, and what each one won.
//
// It is work.go's shape with one thing added - a project page carries facts
// from a second list, the awards the pack files under that project's slug. The
// lookup itself is the pack's (content.AwardsFor), so the awards section can
// read the same relation from the other end; what is here is only how an award
// reads on a project's own page.
//
// Programs share projects.yaml with both lists and appear on no screen this
// file draws. That is the spec's decision, not an omission: programs and skills
// are off this surface.

// openProjects returns the page `p` and the card's `[p] projects` item open.
func openProjects(pack *content.Pack) Page { return projectsList{pack: pack} }

// projectsList is the section's index: the project and what it is, one row
// each. Unlike a role, which needs a second line for its span and its place, a
// project says everything the list has to say beside its own name.
type projectsList struct{ pack *content.Pack }

func (p projectsList) Chrome() Chrome {
	return Chrome{
		Title:      "PROJECTS",
		Suffix:     count(len(p.pack.Projects), "project", "projects"),
		Crumbs:     []string{"projects"},
		Selectable: true,
	}
}

func (p projectsList) Key(key string, cursor int) (Action, Page) {
	if key == "enter" && cursor >= 0 && cursor < len(p.pack.Projects) {
		return Push, projectPage(p.pack, p.pack.Projects[cursor])
	}
	return Ignored, nil
}

func (p projectsList) Blocks(width, cursor int) [][]string {
	names := make([]string, len(p.pack.Projects))
	for i, project := range p.pack.Projects {
		names[i] = project.Name
	}
	column := nameColumn(names, width)

	blocks := make([][]string, 0, len(p.pack.Projects))
	for i, project := range p.pack.Projects {
		blocks = append(blocks, []string{
			listRow(width, i == cursor, project.Name, project.Summary, column),
		})
	}
	return blocks
}

// projectPage opens one project: the project as the pack wrote it, together
// with the awards the pack files under its slug, resolved once here so the page
// itself holds facts rather than a way of finding them.
//
// It is the constructor rather than a struct literal because a project page is
// reached from two directions - down from this section's list, and across from
// an award that names the project it was won for.
func projectPage(pack *content.Pack, project content.Project) Page {
	return projectDetail{project: project, awards: pack.AwardsFor(project.Slug)}
}

// projectDetail is one project: what it does, what it is made of, what it won,
// and where to go and look at it.
type projectDetail struct {
	project content.Project
	awards  []content.Award
}

func (p projectDetail) Chrome() Chrome {
	return Chrome{
		Title:  p.project.Name,
		Suffix: "project",
		Crumbs: []string{"projects", p.project.Slug},
	}
}

func (p projectDetail) Key(string, int) (Action, Page) { return Ignored, nil }

func (p projectDetail) Blocks(width, _ int) [][]string {
	var blocks [][]string
	add := func(rows ...string) {
		if len(rows) > 0 {
			blocks = append(blocks, rows)
		}
	}
	// The labelled fields under the summary are all set in by the same step, so
	// the label is the only thing at the body's own left edge.
	indented := func(text string) []string {
		return indentRows(wrap(paint(textState, text), max(width-bodyIndent, 1)), bodyIndent)
	}

	add(wrap(paint(textState, p.project.Summary), width)...)
	add("")

	if len(p.project.Highlights) > 0 {
		add(paint(dimState, "highlights"))
		for _, highlight := range p.project.Highlights {
			add(bullets(highlight, width)...)
		}
		add("")
	}
	if len(p.project.Stack) > 0 {
		// One run rather than a bulleted list: the stack is a set of names, and
		// wrapping them as a sentence keeps a long one from costing ten rows.
		add(append([]string{paint(dimState, "stack")},
			indented(strings.Join(p.project.Stack, " · "))...)...)
		add("")
	}
	if len(p.awards) > 0 {
		add(paint(dimState, "awards"))
		for _, award := range p.awards {
			add(indented(awardHead(award))...)
		}
		add("")
	}
	if len(p.project.Links) > 0 {
		add(paint(dimState, "links"))
		for _, link := range p.project.Links {
			add(append([]string{bodyLine(textState, link.Label, bodyIndent, width)},
				urlRows(link.URL, width)...)...)
			add("")
		}
	}
	if p.project.BuiltAt != "" {
		add(paint(dimState, "built at "+p.project.BuiltAt))
	}
	if len(p.project.Notes) > 0 {
		add("")
		for _, note := range p.project.Notes {
			add(wrap(paint(dimState, note), width)...)
		}
	}
	return blocks
}

// awardHead is one award said in a line: how it placed, at what, and when.
//
// The one word this composes with is the surface's, not the pack's. An award
// that neither placed nor was selected out of a field is one the pack recorded
// for having been entered, and "entered" is the word for that here, the way
// "present" is the word for a role with no end date.
//
// The year is left off where the event's own name already carries it, which
// several of them do: an event named for its season and year would otherwise
// have that year said twice on one line.
func awardHead(award content.Award) string {
	place := award.Placement
	if place == "" {
		place = award.Selection
	}
	if place == "" {
		place = "entered"
	}
	head := place + " - " + award.Event
	if year := strconv.Itoa(award.Year); award.Year != 0 && !strings.Contains(award.Event, year) {
		head += " " + year
	}
	return head
}
