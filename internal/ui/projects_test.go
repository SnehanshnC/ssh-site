package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// TestProjectsListsEveryProjectFromThePack. The page is the pack's projects
// list and nothing else: every project it carries is on the list, described by
// what the pack says the project is.
func TestProjectsListsEveryProjectFromThePack(t *testing.T) {
	pack := realPack(t)
	if len(pack.Projects) == 0 {
		t.Fatal("the pack carries no projects, so the page can prove nothing")
	}
	// Wide enough that no row has to shorten what it says, so a missing line is
	// a missing project rather than a clipped one.
	list := plain(screen(press(shell(t, 200, 40), "p")))
	for _, project := range pack.Projects {
		for _, fact := range []string{project.Name, project.Summary} {
			if !strings.Contains(list, fact) {
				t.Errorf("the list does not carry %q", fact)
			}
		}
	}
}

// TestTheProjectCountIsCounted, never written. It is the pack's own length, and
// it follows the pack rather than the page: hand the page one project and it
// says so, including in the singular.
func TestTheProjectCountIsCounted(t *testing.T) {
	pack := realPack(t)
	want := strconv.Itoa(len(pack.Projects)) + " projects"
	if got := openProjects(pack).Chrome().Suffix; got != want {
		t.Errorf("PROJECTS is noted as %q, want %q", got, want)
	}
	one := &content.Pack{Projects: pack.Projects[:1]}
	if got := openProjects(one).Chrome().Suffix; got != "1 project" {
		t.Errorf("a one-project pack is noted as %q, want %q", got, "1 project")
	}
	if got := openProjects(&content.Pack{}).Chrome().Suffix; got != "0 projects" {
		t.Errorf("an empty pack is noted as %q, want %q", got, "0 projects")
	}
}

// TestEnterDrillsToTheProjectTheCursorIsOn, and the breadcrumb says where the
// visitor is by the project's slug.
func TestEnterDrillsToTheProjectTheCursorIsOn(t *testing.T) {
	pack := realPack(t)
	if len(pack.Projects) < 2 {
		t.Skip("the pack carries one project, so there is nothing to move the cursor to")
	}
	m := press(press(press(shell(t, 80, 24), "p"), "down"), "enter")
	if len(m.stack) != 2 {
		t.Fatalf("enter left a stack %d deep, want the list and the project", len(m.stack))
	}
	chrome := m.stack[1].page.Chrome()
	if got := chrome.Title; got != pack.Projects[1].Name {
		t.Errorf("enter over the second project opened %q, want %q", got, pack.Projects[1].Name)
	}
	if want := []string{"projects", pack.Projects[1].Slug}; strings.Join(chrome.Crumbs, "/") != strings.Join(want, "/") {
		t.Errorf("the breadcrumb is %v, want %v", chrome.Crumbs, want)
	}
	if chrome.Selectable {
		t.Error("the project page is a list, want a document that scrolls")
	}
}

// TestAProjectPageShowsItsOwnAwardsAndNoOthers is the relation this slice
// exists to draw: the awards list is flat and names the project each award was
// won for, so a project page is the one place that slug is resolved back into
// the awards it stands for. A project with none shows none, which is the half
// of the rule a page carrying every award would still pass.
func TestAProjectPageShowsItsOwnAwardsAndNoOthers(t *testing.T) {
	pack := realPack(t)
	if len(pack.Awards) == 0 {
		t.Fatal("the pack carries no awards, so a project page can prove nothing")
	}
	// Tall and wide enough that the whole page is on one screen and no award
	// line wraps, so an absence is an absence rather than a scroll or a break.
	base := shell(t, 200, 120)
	awarded := 0
	for i, project := range pack.Projects {
		m := press(base, "p")
		for range i {
			m = press(m, "down")
		}
		m = press(m, "enter")
		if got := m.stack[1].page.Chrome().Title; got != project.Name {
			t.Fatalf("%d downs opened %q, want %q", i, got, project.Name)
		}

		page := plain(screen(m))
		for _, award := range pack.Awards {
			own := award.Project == project.Slug
			if own {
				awarded++
			}
			if shown := strings.Contains(page, awardHead(award)); shown != own {
				verb := "does not show"
				if shown {
					verb = "shows"
				}
				t.Errorf("%s %s %q, an award of %q",
					project.Slug, verb, awardHead(award), award.Project)
			}
		}
	}
	if awarded != len(pack.Awards) {
		t.Errorf("%d of the pack's %d awards reached a project page",
			awarded, len(pack.Awards))
	}
}

// TestAnAwardIsSaidAsPlacementEventYear covers the three ways the pack writes
// an award and the one word the surface adds. A placing is said; an award that
// was selected out of a field rather than placed says the selection; an award
// that did neither is one that was entered, which is the surface's word for the
// pack's silence. The year is dropped where the event's own name carries it.
func TestAnAwardIsSaidAsPlacementEventYear(t *testing.T) {
	for _, tt := range []struct {
		name  string
		award content.Award
		want  string
	}{
		{"a placing", content.Award{Placement: "First", Event: "Test Cup", Year: 2020},
			"First - Test Cup 2020"},
		{"a selection", content.Award{Selection: "Picked, 4 of 400", Event: "Test Cup"},
			"Picked, 4 of 400 - Test Cup"},
		{"neither", content.Award{Event: "Test Cup"}, "entered - Test Cup"},
		{"a placing over a selection",
			content.Award{Placement: "First", Selection: "Picked", Event: "Test Cup"},
			"First - Test Cup"},
		{"an event that carries its own year",
			content.Award{Placement: "First", Event: "Test Cup 2020", Year: 2020},
			"First - Test Cup 2020"},
	} {
		if got := awardHead(tt.award); got != tt.want {
			t.Errorf("%s reads %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestProgramsNeverReachAScreen. The pack's projects section carries programs
// alongside the projects and the awards; they are parsed so the section can be
// read at all, and the spec keeps them off this surface along with skills. So
// every screen a visitor can walk to is swept, not just the projects pages.
//
// A program's name and what the pack says about it are what identify it. Its
// date is not: a month the pack happens to write against a program is the same
// month it writes against a role, and a screen showing that role would fail a
// check that could not tell the two apart.
func TestProgramsNeverReachAScreen(t *testing.T) {
	pack := realPack(t)
	if len(pack.Programs) == 0 {
		t.Fatal("the pack carries no programs, so the sweep proves nothing")
	}
	for _, screen := range everyScreen(t, shell(t, 200, 120)) {
		for _, program := range pack.Programs {
			for _, fact := range []string{program.Name, program.Detail} {
				if fact != "" && strings.Contains(screen, fact) {
					t.Errorf("a screen shows %q, which belongs to a program", fact)
				}
			}
		}
	}
}

// TestNoProjectFactIsWrittenInThisRepo, the same rule the work section is held
// to, over the whole of the section's file: no project, award or program fact
// may appear in the Go that renders them.
func TestNoProjectFactIsWrittenInThisRepo(t *testing.T) {
	pack := realPack(t)
	var facts []string
	for _, project := range pack.Projects {
		facts = append(facts, project.Name, project.Summary, project.BuiltAt, project.Date)
		facts = append(facts, project.Highlights...)
		facts = append(facts, project.Stack...)
		facts = append(facts, project.Teammates...)
		facts = append(facts, project.Notes...)
		for _, link := range project.Links {
			// A link that carries no label of its own is shown under its key,
			// and a key is a slug - the one thing this rule allows to be named
			// in the Go, because keying a display mapping to a slug is the rule
			// being followed rather than broken.
			if link.Label != link.Slug {
				facts = append(facts, link.Label)
			}
			facts = append(facts, link.URL)
		}
	}
	for _, award := range pack.Awards {
		facts = append(facts, award.Event, award.Placement, award.Selection,
			award.Track, award.Participants, award.Prize)
		facts = append(facts, award.Extras...)
		facts = append(facts, award.Notes...)
	}
	for _, program := range pack.Programs {
		facts = append(facts, program.Name, program.Detail)
	}
	assertNoFactIsWritten(t, facts)
}

// everyScreen walks the whole surface from the card and returns every screen a
// visitor can reach: arrival, each section's own page, and each row of each
// section drilled into.
func everyScreen(t *testing.T, card Model) []string {
	t.Helper()

	screens := []string{plain(screen(card))}
	cols, _ := pageBody(card.width, card.height)
	for _, jump := range sectionKeys {
		list := press(card, jump)
		if len(list.stack) == 0 {
			t.Fatalf("%q opened nothing", jump)
		}
		screens = append(screens, plain(screen(list)))

		top := list.stack[0]
		if !top.page.Chrome().Selectable {
			continue
		}
		for row := range len(top.page.Blocks(cols, 0)) {
			m := list
			for range row {
				m = press(m, "down")
			}
			if m = press(m, "enter"); len(m.stack) > 1 {
				screens = append(screens, plain(screen(m)))
			}
		}
	}
	return screens
}
