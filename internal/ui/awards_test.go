package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// TestAwardsListsEveryAwardFromThePack. The page is the pack's awards list and
// nothing else: one block per award, each carrying how it placed and every fact
// the pack files beside the placing.
func TestAwardsListsEveryAwardFromThePack(t *testing.T) {
	pack := realPack(t)
	if len(pack.Awards) == 0 {
		t.Fatal("the pack carries no awards, so the page can prove nothing")
	}
	// Wide enough that no row has to shorten what it says, so a missing line is
	// a missing award rather than a clipped one.
	const width = 200
	cols, _ := pageBody(width, 120)
	if got := len(openAwards(pack).Blocks(cols, 0)); got != len(pack.Awards) {
		t.Errorf("the page lays out %d blocks for %d awards", got, len(pack.Awards))
	}

	list := plain(screen(press(shell(t, width, 120), "a")))
	for _, award := range pack.Awards {
		facts := []string{awardHead(award), award.Track, award.Participants, award.Prize}
		if project, ok := pack.Project(award.Project); ok {
			facts = append(facts, project.Name)
		}
		facts = append(facts, award.Extras...)
		for _, fact := range facts {
			if fact != "" && !strings.Contains(list, fact) {
				t.Errorf("the list does not carry %q", fact)
			}
		}
	}
}

// TestTheAwardCountIsCounted, never written. It is the pack's own length, and
// it follows the pack rather than the page: hand the page one award and it says
// so, including in the singular. A "3x HackRU" line, if this surface ever draws
// one, is this same counting and not a sentence anyone typed.
func TestTheAwardCountIsCounted(t *testing.T) {
	pack := realPack(t)
	want := strconv.Itoa(len(pack.Awards)) + " awards"
	if got := openAwards(pack).Chrome().Suffix; got != want {
		t.Errorf("AWARDS is noted as %q, want %q", got, want)
	}
	one := &content.Pack{Awards: pack.Awards[:1]}
	if got := openAwards(one).Chrome().Suffix; got != "1 award" {
		t.Errorf("a one-award pack is noted as %q, want %q", got, "1 award")
	}
	if got := openAwards(&content.Pack{}).Chrome().Suffix; got != "0 awards" {
		t.Errorf("an empty pack is noted as %q, want %q", got, "0 awards")
	}
}

// TestEnterDrillsToTheProjectTheAwardWasWonFor, for every award the pack
// carries - and the breadcrumb roots at awards, because that is the way the
// visitor came in and the path is what a breadcrumb is for.
func TestEnterDrillsToTheProjectTheAwardWasWonFor(t *testing.T) {
	pack := realPack(t)
	if len(pack.Awards) == 0 {
		t.Fatal("the pack carries no awards, so the drill can prove nothing")
	}
	drilled := 0
	for i, award := range pack.Awards {
		project, ok := pack.Project(award.Project)
		if !ok {
			continue
		}
		drilled++

		m := press(shell(t, 80, 24), "a")
		for range i {
			m = press(m, "down")
		}
		m = press(m, "enter")
		if len(m.stack) != 2 {
			t.Fatalf("%s: enter left a stack %d deep, want the list and the project",
				award.Slug, len(m.stack))
		}
		chrome := m.stack[1].page.Chrome()
		if chrome.Title != project.Name {
			t.Errorf("%s opened %q, want %q", award.Slug, chrome.Title, project.Name)
		}
		want := []string{"awards", project.Slug}
		if strings.Join(chrome.Crumbs, "/") != strings.Join(want, "/") {
			t.Errorf("%s: the breadcrumb is %v, want %v", award.Slug, chrome.Crumbs, want)
		}
		if chrome.Selectable {
			t.Errorf("%s opened a list, want the project's document", award.Slug)
		}
	}
	if drilled == 0 {
		t.Fatal("no award in the pack names a project it carries, so nothing was drilled")
	}
}

// TestTheAwardsDrillOpensTheProjectsOwnPage, rather than a second copy of it.
// An award has no detail the project it was won for does not already carry, so
// the two paths reach one page: the same body, the same title, and a breadcrumb
// that differs at its root and nowhere else.
func TestTheAwardsDrillOpensTheProjectsOwnPage(t *testing.T) {
	pack := realPack(t)
	cols, _ := pageBody(200, 120)
	body := func(page Page) string {
		rows, _ := flatten(page.Blocks(cols, 0))
		return strings.Join(rows, "\n")
	}
	// The row a slug sits on in each section, so each page is opened the way a
	// visitor opens it rather than by calling its constructor.
	open := func(jump string, row int) Page {
		m := press(shell(t, 200, 120), jump)
		for range row {
			m = press(m, "down")
		}
		if m = press(m, "enter"); len(m.stack) != 2 {
			t.Fatalf("%q at row %d opened a stack %d deep", jump, row, len(m.stack))
		}
		return m.stack[1].page
	}

	compared := 0
	for i, award := range pack.Awards {
		for j, project := range pack.Projects {
			if project.Slug != award.Project {
				continue
			}
			compared++

			viaAward, viaProject := open("a", i), open("p", j)
			if got, want := body(viaAward), body(viaProject); got != want {
				t.Errorf("%s reaches a different page than the projects list:\n%s\n---\n%s",
					award.Slug, got, want)
			}
			a, p := viaAward.Chrome(), viaProject.Chrome()
			if a.Title != p.Title || a.Suffix != p.Suffix || a.Selectable != p.Selectable {
				t.Errorf("%s reaches %+v, want the projects list's %+v", award.Slug, a, p)
			}
			if a.Crumbs[len(a.Crumbs)-1] != p.Crumbs[len(p.Crumbs)-1] {
				t.Errorf("%s ends its breadcrumb at %v, want %v", award.Slug, a.Crumbs, p.Crumbs)
			}
			if a.Crumbs[0] == p.Crumbs[0] {
				t.Errorf("%s roots its breadcrumb at %q, the same as the projects list",
					award.Slug, a.Crumbs[0])
			}
		}
	}
	if compared == 0 {
		t.Fatal("no award in the pack names a project it carries, so nothing was compared")
	}
}

// TestAnAwardNamingNoProjectIsListedAndOpensNothing. The award happened and the
// pack says so, so the row stays and says it - by the slug it names, since that
// is the only thing left to say about what it was won for. What it cannot do is
// open a page for a project the pack does not carry, and pressing enter over it
// is a key that does nothing rather than a crash or an empty screen.
func TestAnAwardNamingNoProjectIsListedAndOpensNothing(t *testing.T) {
	pack := realPack(t)
	if len(pack.Awards) == 0 {
		t.Fatal("the pack carries no awards, so there is nothing to stand a broken one beside")
	}
	orphan := content.Award{
		Slug:      "orphan-award",
		Project:   "no-project-of-that-name",
		Event:     "Test Orphan Cup",
		Placement: "1st",
	}
	// The real pack with only its awards list replaced, so the card behind the
	// page is still the card and only the reference under test is broken.
	broken := *pack
	broken.Awards = []content.Award{orphan, pack.Awards[0]}

	list := press(New(&broken, 200, 120), "a")
	if got := list.stack[0].page.Chrome().Suffix; got != "2 awards" {
		t.Errorf("the broken pack is noted as %q, want both awards counted", got)
	}
	shown := plain(screen(list))
	if !strings.Contains(shown, awardHead(orphan)) {
		t.Errorf("the award naming no project was dropped from the list:\n%s", shown)
	}
	if !strings.Contains(shown, orphan.Project) {
		t.Errorf("the row does not say the slug it could not resolve:\n%s", shown)
	}

	if got := press(list, "enter"); len(got.stack) != 1 {
		t.Errorf("enter over it left a stack %d deep, want the list alone", len(got.stack))
	}
	// The award beside it resolves, so the section still opens what it can.
	if got := press(press(list, "down"), "enter"); len(got.stack) != 2 {
		t.Errorf("enter over the resolvable award left a stack %d deep, want the project",
			len(got.stack))
	}
}
