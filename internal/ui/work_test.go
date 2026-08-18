package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// TestWorkListsEveryRoleFromThePack. The page is the pack's work section and
// nothing else: every employer it carries is on the list, described by the
// title held there.
func TestWorkListsEveryRoleFromThePack(t *testing.T) {
	pack := realPack(t)
	if len(pack.Work) == 0 {
		t.Fatal("the pack carries no work, so the page can prove nothing")
	}
	// Wide enough that no row has to shorten what it says, so a missing line is
	// a missing role rather than a clipped one.
	list := plain(screen(press(shell(t, 140, 40), "w")))
	for _, job := range pack.Work {
		for _, fact := range []string{job.Company, job.Role, job.Location, span(job)} {
			if !strings.Contains(list, fact) {
				t.Errorf("the list does not carry %q", fact)
			}
		}
	}
}

// TestTheRoleCountIsCounted, never written. It is the pack's own length, and it
// follows the pack rather than the page: hand the page one role and it says so,
// including in the singular.
func TestTheRoleCountIsCounted(t *testing.T) {
	pack := realPack(t)
	want := strconv.Itoa(len(pack.Work)) + " roles"
	if got := openWork(pack).Chrome().Suffix; got != want {
		t.Errorf("WORK is noted as %q, want %q", got, want)
	}
	if got := openWork(&content.Pack{Work: pack.Work[:1]}).Chrome().Suffix; got != "1 role" {
		t.Errorf("a one-role pack is noted as %q, want %q", got, "1 role")
	}
	if got := openWork(&content.Pack{}).Chrome().Suffix; got != "0 roles" {
		t.Errorf("an empty pack is noted as %q, want %q", got, "0 roles")
	}
}

// TestEnterDrillsToTheEmployerTheCursorIsOn, and the breadcrumb says where the
// visitor is by the employer's slug.
func TestEnterDrillsToTheEmployerTheCursorIsOn(t *testing.T) {
	pack := realPack(t)
	if len(pack.Work) < 2 {
		t.Skip("the pack carries one role, so there is nothing to move the cursor to")
	}
	m := press(press(press(shell(t, 80, 24), "w"), "down"), "enter")
	if len(m.stack) != 2 {
		t.Fatalf("enter left a stack %d deep, want the list and the employer", len(m.stack))
	}
	chrome := m.stack[1].page.Chrome()
	if got := chrome.Title; got != pack.Work[1].Company {
		t.Errorf("enter over the second role opened %q, want %q", got, pack.Work[1].Company)
	}
	if want := []string{"work", pack.Work[1].Slug}; strings.Join(chrome.Crumbs, "/") != strings.Join(want, "/") {
		t.Errorf("the breadcrumb is %v, want %v", chrome.Crumbs, want)
	}
	if chrome.Selectable {
		t.Error("the employer page is a list, want a document that scrolls")
	}
}

// TestTheListCursorSurvivesADrillDownAndAPopBack. The frame under the drill-down
// keeps its own cursor, so esc lands back on the row it was entered from - and
// on the list, not on the card.
func TestTheListCursorSurvivesADrillDownAndAPopBack(t *testing.T) {
	if len(realPack(t).Work) < 2 {
		t.Skip("the pack carries one role, so there is nothing to move the cursor to")
	}
	m := press(press(shell(t, 80, 24), "w"), "down")
	if got := m.stack[0].cursor; got != 1 {
		t.Fatalf("down put the cursor on row %d, want the second", got)
	}
	if got := press(m, "enter").stack[0].cursor; got != 1 {
		t.Errorf("drilling down moved the list's cursor to %d", got)
	}
	back := press(press(m, "enter"), "esc")
	if len(back.stack) != 1 {
		t.Fatalf("esc left a stack %d deep, want to be back on the list", len(back.stack))
	}
	if got := back.stack[0].cursor; got != 1 {
		t.Errorf("popping back landed the cursor on row %d, want the row it drilled from", got)
	}
}

// TestBulletsHangAndNeverSplitOnHyphens is the finding a prototype round cost:
// Python's textwrap breaks on hyphens by default and shipped "non-blocking"
// across two lines. It is swept across the wrap boundary so the word is tested
// where it actually straddles it, and the hang is checked at the same time -
// the marker sits in the indent and every line of the item starts at one column.
func TestBulletsHangAndNeverSplitOnHyphens(t *testing.T) {
	const width, word = 40, "non-blocking"
	for shift := range 24 {
		text := strings.Repeat("xy ", shift) + word + " and a tail long enough to wrap again"
		rows := bullets(text, width)
		if len(rows) < 2 {
			t.Fatalf("shift %d wrapped to one row, so it proves nothing", shift)
		}

		whole := 0
		for i, row := range rows {
			if got := ansi.Width(row); got > width {
				t.Errorf("shift %d row %d is %d columns, over the %d it has", shift, i, got, width)
			}
			plain := plainRow(row)
			if strings.Contains(plain, word) {
				whole++
			}
			indent := bodyIndent + bulletHang
			if i == 0 {
				if !strings.HasPrefix(plain, strings.Repeat(" ", bodyIndent)+"- ") {
					t.Errorf("shift %d opens on %q, want the marker in the indent", shift, plain)
				}
			} else if !strings.HasPrefix(plain, strings.Repeat(" ", indent)) ||
				strings.HasPrefix(plain, strings.Repeat(" ", indent+1)) {
				t.Errorf("shift %d row %d is %q, want it hung at column %d", shift, i, plain, indent)
			}
		}
		if whole != 1 {
			t.Errorf("shift %d: %q is whole on %d rows, want exactly 1:\n%s",
				shift, word, whole, strings.Join(rows, "\n"))
		}
	}
}

// TestTheEmployerPageNeverCutsAnAddress. A truncated URL is a broken link, so
// the address is shown whole at every width the surface will ever draw it at,
// shortened only to a form that is still navigable.
func TestTheEmployerPageNeverCutsAnAddress(t *testing.T) {
	for _, job := range realPack(t).Work {
		for _, link := range job.Links {
			for width := minCols - chromeCol - 2*pageMarginWide; width <= 160; width++ {
				rows := urlRows(link.URL, width)
				shown := ""
				for _, row := range rows {
					shown += strings.TrimSpace(plainRow(row))
				}
				if shown != link.URL && !strings.Contains(link.URL, shown) {
					t.Fatalf("%s at %d columns shows %q, which is not its address",
						link.Slug, width, shown)
				}
				if strings.Contains(shown, "...") {
					t.Fatalf("%s at %d columns is cut: %q", link.Slug, width, shown)
				}
				for _, row := range rows {
					if got := ansi.Width(row); got > width {
						t.Fatalf("%s at %d columns rendered %d wide", link.Slug, width, got)
					}
				}
			}
		}
	}
}

// TestNoWorkFactIsWrittenInThisRepo. The pack is the source of every fact this
// surface shows, so no employer, title, date, place or highlight may appear in
// the Go that renders them - a fact in a string literal here is a fact that
// stops following the pack the moment the pack changes.
func TestNoWorkFactIsWrittenInThisRepo(t *testing.T) {
	var facts []string
	for _, job := range realPack(t).Work {
		facts = append(facts, job.Company, job.Role, job.Start, job.End, job.Location)
		facts = append(facts, job.Highlights...)
		for _, link := range job.Links {
			facts = append(facts, link.Label, link.URL)
		}
	}
	assertNoFactIsWritten(t, facts)
}

// plainRow strips the colour off one composed row.
func plainRow(row string) string {
	var b strings.Builder
	for _, cell := range ansi.ParseLine(row) {
		if !cell.Filler {
			b.WriteRune(cell.Rune)
		}
	}
	return b.String()
}
