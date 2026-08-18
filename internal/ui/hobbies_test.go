package ui

import (
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// TestHobbiesListsShowsAndHobbiesFromThePack. The page is the pack's hobbies
// section, shows folded in, and nothing else: every fact either list carries
// is on the page.
func TestHobbiesListsShowsAndHobbiesFromThePack(t *testing.T) {
	pack := realPack(t)
	if len(pack.Shows) == 0 || len(pack.Hobbies) == 0 {
		t.Fatal("the pack carries no shows or no hobbies, so the page can prove nothing")
	}
	// Wide and tall enough that no row has to shorten or scroll away what it
	// says, so a missing fact is a missing fact rather than a clipped one.
	page := plain(screen(press(shell(t, 200, 120), "h")))
	for _, show := range pack.Shows {
		facts := []string{show.Title, show.Category}
		if show.Blurb != "" {
			facts = append(facts, show.Blurb)
		}
		for _, fact := range facts {
			if !strings.Contains(page, fact) {
				t.Errorf("the page does not carry %q", fact)
			}
		}
	}
	for _, hobby := range pack.Hobbies {
		if !strings.Contains(page, hobby.Detail) {
			t.Errorf("the page does not carry %q", hobby.Detail)
		}
	}
}

// TestTheHobbiesHeaderSuffixIsOffTheClock, not a count. Every other section's
// suffix is count() derived from the pack's own length; this one is a fixed
// phrase the spec names outright, so it does not move with either list's size.
func TestTheHobbiesHeaderSuffixIsOffTheClock(t *testing.T) {
	pack := realPack(t)
	if len(pack.Hobbies) == 0 {
		t.Fatal("the pack carries no hobbies, so the suffix check can prove nothing")
	}
	if got, want := openHobbies(pack).Chrome().Suffix, "off the clock"; got != want {
		t.Errorf("HOBBIES is noted as %q, want %q", got, want)
	}

	trimmed := &content.Pack{Shows: pack.Shows, Hobbies: pack.Hobbies[:1]}
	if got := openHobbies(trimmed).Chrome().Suffix; got != "off the clock" {
		t.Errorf("a trimmed pack is noted as %q, want the suffix unchanged", got)
	}
	if got := openHobbies(&content.Pack{}).Chrome().Suffix; got != "off the clock" {
		t.Errorf("an empty pack is noted as %q, want the suffix unchanged", got)
	}
}

// TestHobbiesPageIsADocumentNotAList. Like LINKS, there is nothing under a show
// or a hobby to drill into, so this page scrolls rather than carrying a cursor.
func TestHobbiesPageIsADocumentNotAList(t *testing.T) {
	if openHobbies(realPack(t)).Chrome().Selectable {
		t.Error("the hobbies page is selectable, want a document that scrolls")
	}
}

// TestShowsFoldIntoHobbiesNotItsOwnSection. There is no jump, no card item, and
// no page anywhere on this surface named for shows alone - they render inside
// the hobbies page's own body, under their own heading there.
func TestShowsFoldIntoHobbiesNotItsOwnSection(t *testing.T) {
	pack := realPack(t)
	if len(pack.Shows) == 0 {
		t.Fatal("the pack carries no shows, so the fold-in can prove nothing")
	}
	for _, key := range []string{"s", "S"} {
		if openSection(pack, key) != nil {
			t.Errorf("%q opens a page, want shows to have no section of their own", key)
		}
	}
	page := plain(screen(press(shell(t, 200, 120), "h")))
	if !strings.Contains(page, "shows") {
		t.Error("the hobbies page does not carry a shows heading")
	}
}

// TestHReachesHobbiesFromAnywhere. `h` has no seat on the card's nav row, so
// the letter jump is its only door in - and it opens the same page whether the
// visitor presses it from the card or from inside a drill-down.
func TestHReachesHobbiesFromAnywhere(t *testing.T) {
	deep := press(press(shell(t, 80, 24), "w"), "enter")
	for _, from := range []Model{shell(t, 80, 24), deep} {
		m := press(from, "h")
		if len(m.stack) != 1 {
			t.Fatalf("%q left a stack %d deep, want 1", "h", len(m.stack))
		}
		chrome := m.stack[0].page.Chrome()
		if chrome.Title != "HOBBIES" {
			t.Errorf("h opened %q, want HOBBIES", chrome.Title)
		}
		if chrome.Suffix != "off the clock" {
			t.Errorf("h opened a page noted %q, want %q", chrome.Suffix, "off the clock")
		}
	}
}

// TestNoHobbyFactIsWrittenInThisRepo. The pack is the source of every fact this
// surface shows, so no show's title, category or blurb, and no hobby's detail,
// may appear in the Go that renders them.
func TestNoHobbyFactIsWrittenInThisRepo(t *testing.T) {
	pack := realPack(t)
	var facts []string
	for _, show := range pack.Shows {
		facts = append(facts, show.Title, show.Category, show.Blurb)
	}
	for _, hobby := range pack.Hobbies {
		facts = append(facts, hobby.Detail)
	}
	assertNoFactIsWritten(t, facts)
}
