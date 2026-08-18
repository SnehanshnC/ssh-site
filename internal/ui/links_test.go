package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// TestLinksListsEveryLinkFromThePack. The page is the pack's links section and
// nothing else: every link it carries is on the page, its label and its
// address both.
func TestLinksListsEveryLinkFromThePack(t *testing.T) {
	pack := realPack(t)
	if len(pack.Links) == 0 {
		t.Fatal("the pack carries no links, so the page can prove nothing")
	}
	// Wide enough that no row has to shorten what it says, so a missing line is
	// a missing link rather than a clipped one.
	list := plain(screen(press(shell(t, 140, 40), "l")))
	for _, link := range pack.Links {
		for _, fact := range []string{link.Label, link.URL} {
			if !strings.Contains(list, fact) {
				t.Errorf("the page does not carry %q", fact)
			}
		}
	}
}

// TestTheLinkCountIsCounted, never written. It is the pack's own length, and it
// follows the pack rather than the page: hand the page one link and it says so,
// including in the singular.
func TestTheLinkCountIsCounted(t *testing.T) {
	pack := realPack(t)
	want := strconv.Itoa(len(pack.Links)) + " links"
	if got := openLinks(pack).Chrome().Suffix; got != want {
		t.Errorf("LINKS is noted as %q, want %q", got, want)
	}
	if got := openLinks(&content.Pack{Links: pack.Links[:1]}).Chrome().Suffix; got != "1 link" {
		t.Errorf("a one-link pack is noted as %q, want %q", got, "1 link")
	}
	if got := openLinks(&content.Pack{}).Chrome().Suffix; got != "0 links" {
		t.Errorf("an empty pack is noted as %q, want %q", got, "0 links")
	}
}

// TestLinksPageIsADocumentNotAList. The prototype leaves LINKS out of
// _SELECTABLE, so this page scrolls rather than carrying a cursor.
func TestLinksPageIsADocumentNotAList(t *testing.T) {
	if openLinks(realPack(t)).Chrome().Selectable {
		t.Error("the links page is selectable, want a document that scrolls")
	}
}

// TestLinksPageNeverCutsAnAddress. A truncated URL is a broken link, so the
// address is shown whole at every width the surface will ever draw it at,
// shortened only to a form that is still navigable.
func TestLinksPageNeverCutsAnAddress(t *testing.T) {
	for _, link := range realPack(t).Links {
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

// TestNoLinkFactIsWrittenInThisRepo. The pack is the source of every fact this
// surface shows, so no label or address may appear in the Go that renders
// them - a fact in a string literal here is a fact that stops following the
// pack the moment the pack changes.
func TestNoLinkFactIsWrittenInThisRepo(t *testing.T) {
	var facts []string
	for _, link := range realPack(t).Links {
		facts = append(facts, link.Label, link.URL)
	}
	assertNoFactIsWritten(t, facts)
}
