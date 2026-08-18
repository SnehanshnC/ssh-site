package ui

import (
	"strconv"
	"strings"
	"testing"
)

// TestDocumentCarriesEveryFactFromThePack. The piped session gets no page
// stack to drill through, so every fact a visitor could reach by pressing
// enter has to be in the document already: an employer's highlights and
// links, a project's stack and awards and links, every award, every link and
// every show and hobby, all in the one string.
//
// Facts are compared with their whitespace collapsed rather than as raw
// substrings, because a document this width wraps a long highlight or summary
// across several lines - wrap only ever turns a space into a line break, so
// collapsing both sides back to single spaces undoes exactly that and nothing
// else.
func TestDocumentCarriesEveryFactFromThePack(t *testing.T) {
	pack := realPack(t)
	doc := collapse(Document(pack))

	contains := func(t *testing.T, fact, from string) {
		t.Helper()
		if fact == "" {
			return
		}
		if !strings.Contains(doc, collapse(fact)) {
			t.Errorf("the document does not carry %q, from %s", fact, from)
		}
	}

	for _, job := range pack.Work {
		for _, fact := range []string{job.Company, job.Role, job.Location, span(job)} {
			contains(t, fact, job.Company)
		}
		for _, highlight := range job.Highlights {
			contains(t, highlight, job.Company)
		}
		for _, link := range job.Links {
			contains(t, link.Label, job.Company)
			contains(t, link.URL, job.Company)
		}
	}

	for _, project := range pack.Projects {
		facts := []string{project.Name, project.Summary, project.BuiltAt}
		facts = append(facts, project.Highlights...)
		facts = append(facts, project.Stack...)
		facts = append(facts, project.Notes...)
		for _, fact := range facts {
			contains(t, fact, project.Name)
		}
		for _, award := range pack.AwardsFor(project.Slug) {
			contains(t, awardHead(award), project.Name)
		}
		for _, link := range project.Links {
			contains(t, link.Label, project.Name)
			contains(t, link.URL, project.Name)
		}
	}

	for _, award := range pack.Awards {
		facts := []string{awardHead(award), award.Track, award.Participants, award.Prize}
		facts = append(facts, award.Extras...)
		if project, ok := pack.Project(award.Project); ok {
			facts = append(facts, project.Name)
		}
		for _, fact := range facts {
			contains(t, fact, "the awards section")
		}
	}

	for _, link := range pack.Links {
		contains(t, link.Label, "the links section")
		contains(t, link.URL, "the links section")
	}

	for _, show := range pack.Shows {
		for _, fact := range []string{show.Title, show.Category, show.Blurb} {
			contains(t, fact, "the show "+show.Title)
		}
	}
	for _, hobby := range pack.Hobbies {
		contains(t, hobby.Detail, "the hobbies section")
	}
}

// collapse squashes runs of whitespace - including the line breaks wrap and
// bullets insert - down to single spaces, so a fact that a document wrapped
// across several lines can still be found whole.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestDocumentSectionsAreInSpecOrder: work, then projects, then awards, then
// links, then hobbies - the order the interactive card's sections are read
// out, section 4 of the build spec.
func TestDocumentSectionsAreInSpecOrder(t *testing.T) {
	pack := realPack(t)
	doc := Document(pack)

	titles := []string{
		openWork(pack).Chrome().Title,
		openProjects(pack).Chrome().Title,
		openAwards(pack).Chrome().Title,
		openLinks(pack).Chrome().Title,
		openHobbies(pack).Chrome().Title,
	}
	last := -1
	for _, title := range titles {
		i := strings.Index(doc, title)
		if i < 0 {
			t.Fatalf("the document does not carry the %q heading", title)
		}
		if i < last {
			t.Errorf("%q appears before an earlier section, want spec order %v", title, titles)
		}
		last = i
	}
}

// TestDocumentHasNoANSIEscapes. D2 promises a piped session plain text with
// no escapes, not the coloured screen with the colour merely unread; every
// row this surface paints carries the ESC byte the moment it is coloured, so
// its absence is the whole test.
func TestDocumentHasNoANSIEscapes(t *testing.T) {
	if doc := Document(realPack(t)); strings.ContainsRune(doc, '\x1b') {
		t.Errorf("the document carries an ANSI escape:\n%s", doc)
	}
}

// TestDocumentEndsWithATrailingNewline, the shape a piped command's output is
// expected to have - `ssh ... | cat` should leave the terminal's own prompt on
// its own line rather than run into the document's last one.
func TestDocumentEndsWithATrailingNewline(t *testing.T) {
	doc := Document(realPack(t))
	if !strings.HasSuffix(doc, "\n") {
		t.Error("the document does not end on a newline")
	}
	if strings.HasSuffix(doc, "\n\n") {
		t.Error("the document ends on a blank line")
	}
}

// TestTheAwardCountIsUnaffected guards against a regression this slice could
// easily cause: awards read twice, once flat and once through every project
// that carries one, is by design (awards.go draws the same relation from both
// ends) - so this checks the pack's own count still shows up in the AWARDS
// heading, the way it does on the interactive page.
func TestTheAwardCountIsUnaffected(t *testing.T) {
	pack := realPack(t)
	want := strconv.Itoa(len(pack.Awards)) + " awards"
	if got := openAwards(pack).Chrome().Suffix; got != want {
		t.Fatalf("AWARDS is noted as %q, want %q - a pre-existing invariant, not this slice's to fix", got, want)
	}
}
