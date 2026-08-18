package ui

import (
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// documentCols is the column width the plain-text document composes its body
// against.
//
// A piped session has no viewport to fit, so this is not a terminal width -
// it is chosen wide enough that nothing a real entry writes ever runs into
// page.go's clip(), the same margin of safety the section tests already
// build their own wide renders around (140-200 columns) so that a missing
// fact reads as missing rather than clipped. Getting this wrong is a real
// defect and not a cosmetic one: unlike wrap, which only ever moves text to
// the next line, clip drops it - the awards section's About line, which
// concatenates a project's name with its track, its field size, its prize and
// every extra the pack recorded, is the one place in this surface's body text
// that goes through it, and it is exactly the kind of line a hackathon's
// sponsor-track name can push past 71 columns.
const documentCols = 200

// Document composes the whole portfolio - work, projects, awards, links and
// hobbies, in that order - as a single plain-text document with no ANSI
// escapes and no interactivity: what a session with no active PTY gets in
// place of the TUI (D2).
//
// It is built from the same pages every screen on the surface is drawn from -
// openWork, openProjects, openAwards, openLinks, openHobbies, and the job and
// project detail pages a visitor would otherwise press enter to reach - so
// there is no second copy of a fact here, nor of the formatting that presents
// it. What differs from a screen is only the colour and the cursor: Blocks is
// composed exactly the way it always is, with no block selected, and unpaint
// takes the colour back out of every row before it is written down.
func Document(pack *content.Pack) string {
	var work []string
	for i, job := range pack.Work {
		page := workDetail{job: job}
		if i > 0 {
			work = append(work, "")
		}
		work = append(work, page.Chrome().Title)
		work = append(work, plainBody(page, documentCols)...)
	}

	var projects []string
	for i, project := range pack.Projects {
		page := projectPage(pack, project, "projects")
		if i > 0 {
			projects = append(projects, "")
		}
		projects = append(projects, page.Chrome().Title)
		projects = append(projects, plainBody(page, documentCols)...)
	}

	sections := []string{
		documentSection(openWork(pack).Chrome().Title, work),
		documentSection(openProjects(pack).Chrome().Title, projects),
	}
	for _, page := range []Page{openAwards(pack), openLinks(pack), openHobbies(pack)} {
		sections = append(sections, documentSection(page.Chrome().Title, plainBody(page, documentCols)))
	}

	return strings.Join(sections, "\n\n") + "\n"
}

// documentSection is one heading of the document and the rows under it.
func documentSection(title string, body []string) string {
	return title + "\n\n" + strings.Join(body, "\n")
}

// plainBody flattens a page's own Blocks into the rows it renders as, with no
// block selected, then strips the SGR colour a live terminal would show them
// in and drops the blank rows a body trails off with - the two differences
// between a page on screen and a page in the document.
func plainBody(p Page, cols int) []string {
	flat, _ := flatten(p.Blocks(cols, -1))
	rows := make([]string, len(flat))
	for i, row := range flat {
		rows[i] = unpaint(row)
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

// unpaint strips one rendered row of its SGR colour, keeping the text a
// terminal cell would show. It is paint's inverse.
func unpaint(row string) string {
	var b strings.Builder
	for _, cell := range ansi.ParseLine(row) {
		if !cell.Filler {
			b.WriteRune(cell.Rune)
		}
	}
	return b.String()
}
