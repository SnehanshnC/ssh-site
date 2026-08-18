package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// TestIdleTimeoutIsTenMinutes pins the duration D4 settled on: an application
// idle timer well inside the server's own 15m IdleTimeout, which stays a
// backstop a well-behaved session never reaches.
func TestIdleTimeoutIsTenMinutes(t *testing.T) {
	if idleTimeout != 10*time.Minute {
		t.Errorf("the idle timeout is %s, want 10m", idleTimeout)
	}
}

// TestIdleTimeoutEndsTheSessionLikeQ. An idle tick at the model's current
// generation - one nothing has reset since it was scheduled - ends the session
// exactly the way q does: quitting set, tea.Quit returned.
func TestIdleTimeoutEndsTheSessionLikeQ(t *testing.T) {
	m := shell(t, 80, 24)
	next, cmd := m.Update(idleTickMsg{gen: m.idleGen})
	if !quits(cmd) {
		t.Fatal("an idle tick at the current generation did not end the session")
	}
	if !next.(Model).quitting {
		t.Error("the idle timeout did not set quitting")
	}
}

// TestAnyKeypressResetsTheIdleTimer. A keypress bumps the generation, so a
// tick scheduled before it carries a generation that is now stale and is
// ignored, while a tick scheduled at the new generation still ends the
// session - the debounce the acceptance criteria calls "any keypress resets
// the timer".
func TestAnyKeypressResetsTheIdleTimer(t *testing.T) {
	m := shell(t, 80, 24)
	stale := m.idleGen
	m = press(m, "p")
	if m.idleGen == stale {
		t.Fatal("a keypress did not advance the idle generation")
	}

	if next, cmd := m.Update(idleTickMsg{gen: stale}); quits(cmd) || next.(Model).quitting {
		t.Error("a stale idle tick, scheduled before the keypress, ended the session")
	}
	if next, cmd := m.Update(idleTickMsg{gen: m.idleGen}); !quits(cmd) || !next.(Model).quitting {
		t.Error("a tick at the current generation, scheduled after the keypress, did not end the session")
	}
}

// TestQCtrlCAndIdleTimeoutPrintTheSameGoodbye is the acceptance criterion
// itself: one goodbye line, not three (or four, counting a page's own Quit
// action - see TestAPageCanRequestQuitPrintsTheSameGoodbye). Whichever of them
// triggers it, the rendered line is byte-for-byte identical.
func TestQCtrlCAndIdleTimeoutPrintTheSameGoodbye(t *testing.T) {
	base := shell(t, 80, 24)

	q := press(base, "q")
	ctrlC := press(base, "ctrl+c")
	idle, _ := base.Update(idleTickMsg{gen: base.idleGen})

	for name, m := range map[string]Model{"q": q, "ctrl+c": ctrlC, "idle timeout": idle.(Model)} {
		if !m.quitting {
			t.Fatalf("%s did not set quitting", name)
		}
		if got := m.View().Content; got != goodbye(base.pack) {
			t.Errorf("%s printed %q, want %q", name, got, goodbye(base.pack))
		}
	}
}

// TestAPageCanRequestQuitPrintsTheSameGoodbye covers the fourth trigger: a
// page's own Quit action (see TestAPageCanRequestQuit in shell_test.go), which
// runs through the same quit() as the other three.
func TestAPageCanRequestQuitPrintsTheSameGoodbye(t *testing.T) {
	m := shell(t, 80, 24)
	m.stack = []frame{{page: fakePage{action: Quit}}}
	next, cmd := m.Update(key("enter"))
	if !quits(cmd) {
		t.Fatal("a page's Quit action did not end the session")
	}
	nm := next.(Model)
	if !nm.quitting {
		t.Fatal("a page's Quit action did not set quitting")
	}
	if got := nm.View().Content; got != goodbye(m.pack) {
		t.Errorf("a page's Quit action printed %q, want %q", got, goodbye(m.pack))
	}
}

// TestGoodbyeDrawsOutsideTheAltScreen is what makes it survive scrollback:
// every ordinary screen draws into the alt screen, and quitting is the one
// transition that leaves it.
func TestGoodbyeDrawsOutsideTheAltScreen(t *testing.T) {
	m := shell(t, 80, 24)
	if !m.View().AltScreen {
		t.Fatal("the ordinary screen does not draw in the alt screen")
	}
	if got := press(m, "q").View(); got.AltScreen {
		t.Error("the goodbye line draws inside the alt screen")
	}
}

// TestGoodbyeCarriesTheNameTaglineAndGithubLink, and nothing else - most
// obviously not a phone or email, which the pack does not carry at all.
func TestGoodbyeCarriesTheNameTaglineAndGithubLink(t *testing.T) {
	pack := realPack(t)
	if len(pack.Identity.Taglines) == 0 {
		t.Fatal("the pack carries no taglines, so this test can prove nothing")
	}
	github, ok := pack.Link("github")
	if !ok {
		t.Fatal("the pack carries no github link, so this test can prove nothing")
	}

	got := plain(goodbye(pack))
	for _, fact := range []string{pack.Identity.Name, pack.Identity.Taglines[0], github.URL} {
		if !strings.Contains(got, fact) {
			t.Errorf("the goodbye line does not carry %q", fact)
		}
	}
}

// TestGoodbyeNameIsADimGradient: not painted plain, and not the bold ramp a
// page header uses - its own, dimmer variant, since it prints after the
// session has already ended.
func TestGoodbyeNameIsADimGradient(t *testing.T) {
	nameLine := strings.Split(goodbye(realPack(t)), "\n")[0]
	colours := map[string]bool{}
	for _, cell := range ansi.ParseLine(nameLine) {
		if cell.Rune == ' ' {
			continue
		}
		if cell.State.Attrs != "2" {
			t.Fatalf("a name cell is painted with attrs %q, want the dim attribute 2", cell.State.Attrs)
		}
		colours[cell.State.FG] = true
	}
	if len(colours) < 2 {
		t.Errorf("the name is painted in %d colour(s), want a gradient", len(colours))
	}
}

// TestGoodbyeIsComposedFromThePack proves the line is not written down in
// this repo: a pack carrying different facts prints a different line.
func TestGoodbyeIsComposedFromThePack(t *testing.T) {
	pack := fixturePack()
	got := plain(goodbye(pack))
	github, _ := pack.Link("github")
	for _, fact := range []string{pack.Identity.Name, pack.Identity.Taglines[0], github.URL} {
		if !strings.Contains(got, fact) {
			t.Errorf("a fixture pack's goodbye line does not carry %q", fact)
		}
	}
}

// TestNoGoodbyeFactIsWrittenInThisRepo: the pack is the source of every fact
// this line shows, the rule every other section on this surface follows too.
func TestNoGoodbyeFactIsWrittenInThisRepo(t *testing.T) {
	pack := realPack(t)
	facts := []string{pack.Identity.Name, pack.Identity.Taglines[0]}
	if github, ok := pack.Link("github"); ok {
		facts = append(facts, github.URL)
	}
	assertNoFactIsWritten(t, facts)
}
