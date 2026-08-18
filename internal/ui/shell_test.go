package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// TestArrivalIsTheCardWithALiveNavRow. The card is ticket 04's, unchanged
// except where the shell was asked to change it: the static rule and the static
// legend are gone and one live nav row stands there instead, carrying the same
// six items in the same style with a ground under the one the visitor is on.
func TestArrivalIsTheCardWithALiveNavRow(t *testing.T) {
	m := shell(t, 80, 24)
	row := plainRows(screen(m))[liveRow]
	for _, item := range navFull {
		if !strings.Contains(row, "["+item.key+"] "+item.label) {
			t.Errorf("the nav row %q is missing %q", row, item.label)
		}
	}

	for i, want := range []string{"[w] work", "[p] projects", "[a] awards"} {
		if got := live(screen(m)); got != want {
			t.Errorf("after %d moves the highlight is on %q, want %q", i, got, want)
		}
		m = press(m, "right")
	}
	m = shell(t, 80, 24)
	if got := live(screen(press(m, "left"))); got != "[q] quit" {
		t.Errorf("moving left off the first item lands on %q, want it to wrap to quit", got)
	}
	if got := live(screen(press(shell(t, 80, 24), "tab"))); got != "[p] projects" {
		t.Errorf("tab moved the highlight to %q, want the next item", got)
	}
}

// TestEnterOpensTheHighlightedSection, and the page it opens is a page on a
// stack rather than a screen that replaced the card.
func TestEnterOpensTheHighlightedSection(t *testing.T) {
	m := press(press(shell(t, 80, 24), "right"), "enter")
	if depth := len(m.stack); depth != 1 {
		t.Fatalf("the stack is %d deep, want 1", depth)
	}
	if got := m.stack[0].page.Chrome().Title; got != "PROJECTS" {
		t.Errorf("enter over [p] projects opened %q", got)
	}
}

// TestPoppingWalksBackToTheCard: esc and backspace each pop one level, and
// popping the first page is what returns the visitor to arrival.
func TestPoppingWalksBackToTheCard(t *testing.T) {
	for _, back := range []string{"esc", "backspace"} {
		m := press(press(shell(t, 80, 24), "w"), "enter") // work, then a role
		if depth := len(m.stack); depth != 2 {
			t.Fatalf("%s: the stack is %d deep before popping, want 2", back, depth)
		}
		if m = press(m, back); len(m.stack) != 1 {
			t.Errorf("%s: popped to %d levels, want 1", back, len(m.stack))
		}
		if m = press(m, back); len(m.stack) != 0 {
			t.Errorf("%s: popped to %d levels, want the card", back, len(m.stack))
		}
		if !strings.Contains(screen(m), "[w] work") {
			t.Errorf("%s: popping the first page did not land back on the card", back)
		}
		if m = press(m, back); len(m.stack) != 0 {
			t.Errorf("%s: popping from the card left %d levels", back, len(m.stack))
		}
	}
}

// TestAPageCanRequestQuit is the defect the page protocol was written to fix.
// In the prototype only an unconsumed literal `q` ended a session, so a page
// could never offer quit as something the cursor lands on and presses enter
// over; an explicit action means it can.
func TestAPageCanRequestQuit(t *testing.T) {
	m := shell(t, 80, 24)
	m.stack = []frame{{page: fakePage{action: Quit}}}
	if _, cmd := m.Update(key("enter")); !quits(cmd) {
		t.Error("a page that returned Quit did not end the session")
	}
}

// TestTheCardsQuitItemQuits is the same capability where a visitor meets it:
// the highlight lands on `[q] quit` like any other item and enter over it ends
// the session, which the prototype's highlight had to skip.
func TestTheCardsQuitItemQuits(t *testing.T) {
	m := press(shell(t, 80, 24), "end")
	if got := live(screen(m)); got != "[q] quit" {
		t.Fatalf("end put the highlight on %q, want the last item", got)
	}
	if _, cmd := m.Update(key("enter")); !quits(cmd) {
		t.Error("enter over [q] quit did not end the session")
	}
}

// TestTheShellAppliesItsDefaultToWhatAPageIgnores. A page gets first refusal on
// every key the shell has not already claimed, and what it hands back falls
// through to the shell's own default for that key.
func TestTheShellAppliesItsDefaultToWhatAPageIgnores(t *testing.T) {
	m := shell(t, 80, 24)
	m.stack = []frame{{page: fakePage{action: Ignored}}}
	if got := press(m, "esc"); len(got.stack) != 0 {
		t.Error("esc a page ignored did not fall through to the shell's pop")
	}

	m.stack = []frame{{page: fakePage{action: Consumed}}}
	if got := press(m, "esc"); len(got.stack) != 1 {
		t.Error("esc a page consumed still popped")
	}

	m.stack = []frame{{page: fakePage{action: Pop}}}
	if got := press(m, "x"); len(got.stack) != 0 {
		t.Error("a page that returned Pop did not pop")
	}

	m.stack = []frame{{page: fakePage{action: Push, next: fakePage{}}}}
	if got := press(m, "x"); len(got.stack) != 2 {
		t.Errorf("a page that returned Push left %d levels, want 2", len(got.stack))
	}
}

// TestLetterJumpsWorkFromAnywhere, including from inside a drill-down, and they
// change subject rather than going deeper: a jump leaves one level to pop.
func TestLetterJumpsWorkFromAnywhere(t *testing.T) {
	want := map[string]string{
		"w": "WORK", "p": "PROJECTS", "a": "AWARDS", "l": "LINKS", "h": "HOBBIES",
	}
	deep := press(press(shell(t, 80, 24), "w"), "enter")
	for _, from := range []Model{shell(t, 80, 24), deep} {
		for jump, title := range want {
			m := press(from, jump)
			if len(m.stack) != 1 {
				t.Errorf("%q left a stack %d deep, want 1", jump, len(m.stack))
				continue
			}
			if got := m.stack[0].page.Chrome().Title; got != title {
				t.Errorf("%q opened %q, want %q", jump, got, title)
			}
		}
	}
}

// TestHelpOverlay: `?` toggles it, five keys dismiss it, and it lies over
// whatever it was opened from rather than replacing it.
func TestHelpOverlay(t *testing.T) {
	m := press(shell(t, 80, 24), "?")
	if !m.help {
		t.Fatal("? did not open the help overlay")
	}
	if !strings.Contains(plain(screen(m)), "this help, esc closes") {
		t.Error("the overlay is open but its keys are not on the screen")
	}
	if !strings.Contains(plain(screen(m)), "[w] work") {
		t.Error("the overlay replaced the card instead of lying over it")
	}
	for _, dismiss := range []string{"esc", "enter", "backspace", "space", "?"} {
		if press(m, dismiss).help {
			t.Errorf("%q did not dismiss the overlay", dismiss)
		}
	}
	if !press(m, "down").help {
		t.Error("a key that dismisses nothing dismissed the overlay")
	}

	// Opened over a page, it leaves the stack exactly as it found it.
	page := press(shell(t, 80, 24), "w")
	if got := press(press(page, "?"), "esc"); len(got.stack) != 1 {
		t.Error("dismissing the overlay over a page popped the page too")
	}
}

// TestQuitFromAnywhere, the overlay included: `q` and `ctrl+c` are the one
// promise the shell makes over every screen, so neither a page nor the overlay
// ever gets the chance to swallow them.
func TestQuitFromAnywhere(t *testing.T) {
	card := shell(t, 80, 24)
	for name, m := range map[string]Model{
		"the card":    card,
		"a page":      press(card, "w"),
		"a drilldown": press(press(card, "w"), "enter"),
		"the overlay": press(card, "?"),
		"too small":   shell(t, 40, 10),
	} {
		for _, quit := range []string{"q", "ctrl+c"} {
			if _, cmd := m.Update(key(quit)); !quits(cmd) {
				t.Errorf("%q did not quit from %s", quit, name)
			}
		}
	}
}

// TestCursorOrScrollByPageKind. A list counts in items, because items are what
// the visitor is looking at; a document counts in rows, and `space` pages it
// down.
func TestCursorOrScrollByPageKind(t *testing.T) {
	list := press(shell(t, 80, 24), "p")
	for _, tt := range []struct {
		keys []string
		want int
	}{
		{[]string{"down", "down"}, 2},
		{[]string{"down", "up"}, 0},
		{[]string{"pgdown"}, pageStep},
		{[]string{"end"}, len(list.stack[0].page.Blocks(70, 0)) - 1},
		{[]string{"end", "home"}, 0},
	} {
		m := list
		for _, k := range tt.keys {
			m = press(m, k)
		}
		if got := m.stack[0].cursor; got != tt.want {
			t.Errorf("%v moved the cursor to %d, want %d", tt.keys, got, tt.want)
		}
		if got := m.stack[0].scroll; tt.want == 0 && got != 0 {
			t.Errorf("%v scrolled a list to %d with the cursor at the top", tt.keys, got)
		}
	}

	// A detail page, which is a document rather than a list, and one whose
	// highlights run past a screen so there is something below the fold.
	doc := press(press(shell(t, 80, 24), "w"), "enter")
	cols, rows := pageBody(80, 24)
	flat, _ := flatten(doc.stack[1].page.Blocks(cols, 0))
	bottom := max(len(flat)-rows, 0)
	if bottom == 0 {
		t.Fatal("the detail page fits on one screen, so it can prove nothing about scrolling")
	}
	for _, tt := range []struct {
		keys []string
		want int
	}{
		{[]string{"down", "down"}, 2},
		{[]string{"down", "up"}, 0},
		{[]string{"space"}, min(rows, bottom)},
		{[]string{"pgdown", "pgup"}, 0},
		{[]string{"end"}, bottom},
		{[]string{"end", "home"}, 0},
	} {
		m := doc
		for _, k := range tt.keys {
			m = press(m, k)
		}
		if got := m.stack[1].scroll; got != tt.want {
			t.Errorf("%v scrolled to row %d, want %d", tt.keys, got, tt.want)
		}
		if got := m.stack[1].cursor; got != 0 {
			t.Errorf("%v moved a cursor on a page that has none", tt.keys)
		}
	}
}

// TestSpaceDoesNothingToAList: there is no partially-seen row for it to bring
// into view, and taking the cursor five items down on a key that says "page" is
// a different promise than the one it makes on a document.
func TestSpaceDoesNothingToAList(t *testing.T) {
	list := press(shell(t, 80, 24), "p")
	if got := press(list, "space"); got.stack[0].cursor != 0 || got.stack[0].scroll != 0 {
		t.Errorf("space moved a list to cursor %d, scroll %d",
			got.stack[0].cursor, got.stack[0].scroll)
	}
}

// TestResizeKeepsTheVisitorWhereTheyWere is the finding a prototype round cost:
// a model that reset on resize threw the visitor back to arrival, so the stack,
// every page's cursor and every page's scroll offset all survive a
// WindowSizeMsg.
func TestResizeKeepsTheVisitorWhereTheyWere(t *testing.T) {
	m := press(press(press(shell(t, 80, 24), "w"), "down"), "enter")
	m = press(press(press(m, "down"), "down"), "down")
	before := m.stack

	resized, _ := m.Update(tea.WindowSizeMsg{Width: 76, Height: 22})
	after := resized.(Model)

	if len(after.stack) != len(before) {
		t.Fatalf("the stack is %d deep after the resize, want %d", len(after.stack), len(before))
	}
	for i := range before {
		if after.stack[i].cursor != before[i].cursor {
			t.Errorf("page %d's cursor moved from %d to %d",
				i, before[i].cursor, after.stack[i].cursor)
		}
		if after.stack[i].scroll != before[i].scroll {
			t.Errorf("page %d's scroll moved from %d to %d",
				i, before[i].scroll, after.stack[i].scroll)
		}
	}
	if strings.Contains(plain(screen(after)), "[w] work") {
		t.Error("the resize returned the visitor to arrival")
	}
}

// TestPageChrome checks the frame every section page is drawn in.
func TestPageChrome(t *testing.T) {
	m := press(press(shell(t, 80, 24), "w"), "enter")
	rows := plainRows(screen(m))
	margin := pageMargin(80)
	cols, _ := pageBody(80, 24)

	if got := strings.TrimSpace(rows[0]); got != "" {
		t.Errorf("the page opens on %q, want a blank row", got)
	}
	suffix := m.stack[len(m.stack)-1].page.Chrome().Suffix
	if !strings.HasSuffix(strings.TrimRight(rows[pageTitleRow], " "), suffix) {
		t.Errorf("the title row %q does not end on its dim note %q", rows[pageTitleRow], suffix)
	}
	if crumb := strings.TrimSpace(rows[pageCrumbRow]); !strings.HasPrefix(crumb, "home / work / ") {
		t.Errorf("the breadcrumb is %q, want it to start at home", crumb)
	}
	if got := strings.Count(rows[pageRuleRow], "─"); got != cols {
		t.Errorf("the rule is %d columns, want %d", got, cols)
	}
	for _, row := range []int{pageTitleRow, pageCrumbRow, pageRuleRow, pageBodyRow} {
		if first, _ := ink(rows[row]); first != margin {
			t.Errorf("row %d starts at column %d, want the %d-column margin",
				row, first, margin)
		}
	}
	if hint := strings.TrimSpace(rows[len(rows)-1]); !strings.Contains(hint, "esc back") {
		t.Errorf("the hint row is %q", hint)
	}

	// The header is the wordmark's own gradient, so a page belongs to the card
	// it was opened from: more than one colour across one title.
	colours := map[string]bool{}
	for _, cell := range ansi.ParseLine(strings.Split(screen(press(shell(t, 80, 24), "w")), "\n")[pageTitleRow]) {
		if cell.Rune != ' ' {
			colours[cell.State.FG] = true
		}
	}
	if len(colours) < 3 {
		t.Errorf("the title is painted in %d colours, want a gradient", len(colours))
	}
}

// TestEveryScreenFitsItsFrame sweeps the sizes a visitor can arrive with and
// resize to, with a page open rather than the card, and holds every screen to
// the same two rules the card is held to: it fills the frame exactly, and it
// leaves the reserved chrome column alone.
func TestEveryScreenFitsItsFrame(t *testing.T) {
	base := press(press(shell(t, 80, 24), "p"), "enter")
	for width := 1; width <= 200; width++ {
		for _, height := range []int{1, 5, 19, 20, 23, 24, 40} {
			resized, _ := base.Update(tea.WindowSizeMsg{Width: width, Height: height})
			for _, help := range []bool{false, true} {
				m := resized.(Model)
				m.help = help
				rows := plainRows(screen(m))
				if len(rows) != height {
					t.Fatalf("%dx%d: %d rows, want %d", width, height, len(rows), height)
				}
				for i, row := range rows {
					if got := ansi.Width(row); got > width-chromeCol {
						t.Fatalf("%dx%d: row %d is %d columns wide, over the %d it gets",
							width, height, i, got, width-chromeCol)
					}
				}
			}
		}
	}
}

// TestTooSmallAsksForRoom: below the floor the body is replaced by a plea
// wherever the visitor happens to be, page or card, and a window with no room
// for a page has none for a box of keys over it either.
func TestTooSmallAsksForRoom(t *testing.T) {
	deep := press(press(shell(t, 80, 24), "w"), "enter")
	for _, size := range [][2]int{{minCols - 1, minRows}, {minCols, minRows - 1}} {
		resized, _ := deep.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m := resized.(Model)
		m.help = true
		if got := plain(screen(m)); !strings.Contains(got, "bigger window") {
			t.Errorf("%dx%d drew a screen instead of the plea", size[0], size[1])
		} else if strings.Contains(got, "this help") {
			t.Errorf("%dx%d drew the help overlay over the plea", size[0], size[1])
		}
	}
	resized, _ := deep.Update(tea.WindowSizeMsg{Width: minCols, Height: minRows})
	if got := plain(screen(resized.(Model))); strings.Contains(got, "bigger window") {
		t.Errorf("%dx%d is the floor and still drew the plea", minCols, minRows)
	}
}

// TestTheNarrowCardsHighlightStaysOnTheRowItHas. The restack's legend is five
// items, not six, so a highlight parked on the sixth has to come back inside
// the row when the window shrinks under it rather than pointing at nothing.
func TestTheNarrowCardsHighlightStaysOnTheRowItHas(t *testing.T) {
	m := press(shell(t, 80, 24), "end")
	if got := len(cardNav(m.pack, 80, 24)); got != len(navFull) {
		t.Fatalf("the wide card drew %d nav items, want %d", got, len(navFull))
	}
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	narrow := resized.(Model)
	if narrow.nav >= len(navNarrow) {
		t.Errorf("the highlight is on item %d of a %d-item row", narrow.nav, len(navNarrow))
	}
	if got := live(screen(narrow)); got != "[q] quit" {
		t.Errorf("the highlight came back onto %q, want it to stay on quit", got)
	}
}

// --- helpers ---

// namedKeys are the keys that arrive as something other than the character
// typed. Everything else is its own literal.
var namedKeys = map[string]tea.KeyPressMsg{
	"up":        {Code: tea.KeyUp},
	"down":      {Code: tea.KeyDown},
	"left":      {Code: tea.KeyLeft},
	"right":     {Code: tea.KeyRight},
	"enter":     {Code: tea.KeyEnter},
	"esc":       {Code: tea.KeyEscape},
	"backspace": {Code: tea.KeyBackspace},
	"tab":       {Code: tea.KeyTab},
	"space":     {Code: tea.KeySpace, Text: " "},
	"pgup":      {Code: tea.KeyPgUp},
	"pgdown":    {Code: tea.KeyPgDown},
	"home":      {Code: tea.KeyHome},
	"end":       {Code: tea.KeyEnd},
	"ctrl+c":    {Code: 'c', Mod: tea.ModCtrl},
}

func key(name string) tea.KeyPressMsg {
	if k, ok := namedKeys[name]; ok {
		return k
	}
	r := []rune(name)[0]
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func shell(t *testing.T, width, height int) Model {
	t.Helper()
	return New(realPack(t), width, height)
}

func screen(m Model) string { return m.View().Content }

func press(m Model, k string) Model {
	next, _ := m.Update(key(k))
	return next.(Model)
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// live reports which nav item a rendered card has the ground under.
func live(card string) string {
	var found strings.Builder
	for _, cell := range ansi.ParseLine(strings.Split(card, "\n")[liveRow]) {
		if cell.State == liveState {
			found.WriteRune(cell.Rune)
		}
	}
	return found.String()
}

func plain(screen string) string { return strings.Join(plainRows(screen), "\n") }

// fakePage answers every key with one action, so each branch of the protocol
// can be exercised without a section page that means something.
type fakePage struct {
	action Action
	next   Page
}

func (p fakePage) Chrome() Chrome                 { return Chrome{Title: "FAKE", Crumbs: []string{"fake"}} }
func (p fakePage) Blocks(int, int) [][]string     { return [][]string{{"row"}} }
func (p fakePage) Key(string, int) (Action, Page) { return p.action, p.next }
