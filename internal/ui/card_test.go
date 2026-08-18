package ui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/art"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// fixturePack is a pack with nothing real in it, so the tests that check where
// copy comes from cannot pass by accident on a hardcoded string. Its link is
// deliberately the awkward shape: a long path under a host, with a trailing
// slash and a `www.`, which is the case that decides the card's widths.
func fixturePack() *content.Pack {
	return &content.Pack{
		Identity: content.Identity{
			Name: "Test Person",
			Role: content.Role{
				Title: "Test Engineer", Company: "TestCo", Program: "Test Program S00",
			},
			Education: content.Education{
				Institution: "Testfield University - Test Campus",
				Degree:      "BS in Test Science and Testonomy",
			},
			Taglines: []string{"Current test: Testing all the things."},
		},
		Links: []content.Link{
			{Slug: "github", Label: "GitHub", URL: "https://github.com/TestPerson"},
			{Slug: "linkedin", Label: "LinkedIn",
				URL: "https://www.linkedin.com/in/test-person-0a1b2c3d4/"},
		},
	}
}

func realPack(t *testing.T) *content.Pack {
	t.Helper()
	pack, err := content.Load()
	if err != nil {
		t.Fatalf("load content pack: %v", err)
	}
	return pack
}

// TestCardMatchesTheReferenceGeometry checks the card against the layout
// ticket 04 signed off, as the navigation shell left it: banner rows 0-3
// centred on the frame, the disc at column 2 under it, the copy column at 42,
// and the live nav row on row 22, where the static rule and the static legend
// used to be.
//
// The card is centred on 79 columns rather than 80 because one column is
// reserved for right-edge chrome at every width before any art is fitted, so an
// 80-column terminal offers the composition 79.
//
// It is geometry rather than bytes because the four cosmetic touch-ups the
// build was asked to make - the banner's row gaps, the earring, the temple
// wedge, the collar step - all change pixels of the reference on purpose. What
// must not move is where anything sits.
func TestCardMatchesTheReferenceGeometry(t *testing.T) {
	rows := plainRows(Card(realPack(t), art.Quad, 80, 24, noHighlight))
	if len(rows) != 24 {
		t.Fatalf("card is %d rows, want 24", len(rows))
	}

	for row := range art.BannerRows {
		assertInk(t, rows, row, 16, 61) // 46 columns centred on 79
	}
	for row := faceRow; row < faceRow+art.WidePortraitRows; row++ {
		if first, _ := ink(rows[row]); first < faceCol {
			t.Errorf("row %d starts at column %d, left of the disc's column %d",
				row, first, faceCol)
		}
	}
	if got := strings.Count(rows[faceRow+ruleOffset], "─"); got != copyRule {
		t.Errorf("the copy column's rule is %d wide, want %d", got, copyRule)
	}

	// The copy column starts where the reference puts it, on every row it uses.
	for _, row := range []int{
		faceRow + roleOffset, faceRow + schoolOffset, faceRow + ruleOffset,
		faceRow + questHeadOffset, faceRow + questHeadOffset + 1,
		faceRow + linksOffset, faceRow + linksOffset + 1,
	} {
		if r := []rune(rows[row])[copyCol]; r == ' ' {
			t.Errorf("row %d has no copy at column %d", row, copyCol)
		}
	}

	nav := rows[liveRow]
	for _, item := range navFull {
		if !strings.Contains(nav, "["+item.key+"] "+item.label) {
			t.Errorf("nav row %q is missing %q", nav, item.label)
		}
	}
	if first, last := ink(nav); first != 4 || last != 73 {
		t.Errorf("nav spans columns %d..%d, want 4..73", first, last)
	}

	// Nothing above the nav row is the card's own bottom edge any more: the
	// static rule that used to close the card off went with the static legend.
	if row := rows[liveRow-1]; strings.Contains(row, strings.Repeat("─", 8)) {
		t.Errorf("row %d still carries the card's old bottom rule", liveRow-1)
	}
}

// TestScreenIsKilobytesNotMegabytes is the prototype's other engineering
// finding as a budget. A composed truecolor screen is around 18 KB when every
// cell's state is resolved and re-emitted canonically, and megabytes when SGR
// prefixes are carried forward by concatenation instead.
func TestScreenIsKilobytesNotMegabytes(t *testing.T) {
	size := len(Card(realPack(t), art.Quad, 80, 24, noHighlight))
	if size < 8*1024 || size > 32*1024 {
		t.Errorf("a composed screen is %d bytes, want it on the order of 18 KB", size)
	}
}

// TestEverySequenceNamesBothColours checks the ring finding at the level that
// actually matters - the bytes a visitor's terminal receives.
func TestEverySequenceNamesBothColours(t *testing.T) {
	sgr := regexp.MustCompile("\x1b\\[([0-9;]*)m")
	for _, match := range sgr.FindAllStringSubmatch(Card(realPack(t), art.Quad, 80, 24, noHighlight), -1) {
		params := match[1]
		if params == "0" || params == "" {
			continue
		}
		if !namesColour(params, 30, 38, 39, 90) {
			t.Errorf("sequence ESC[%sm names no foreground", params)
		}
		if !namesColour(params, 40, 48, 49, 100) {
			t.Errorf("sequence ESC[%sm names no background", params)
		}
	}
}

// TestFitsEveryTerminalSize sweeps the sizes a visitor can actually arrive
// with. Whatever the card decides to draw, it must fill the frame exactly and
// never overflow it.
func TestFitsEveryTerminalSize(t *testing.T) {
	pack := fixturePack()
	for width := 1; width <= 200; width++ {
		for _, height := range []int{1, 5, 19, 20, 23, 24, 25, 40, 60} {
			card := Card(pack, art.Quad, width, height, noHighlight)
			rows := plainRows(card)
			if len(rows) != height {
				t.Fatalf("%dx%d: %d rows, want %d", width, height, len(rows), height)
			}
			for i, row := range rows {
				// One column short of the frame, not level with it: the chrome
				// column is reserved before the art is fitted, at every width.
				if got := ansi.Width(row); got > width-chromeCol {
					t.Fatalf("%dx%d: row %d is %d columns wide, over the %d the art gets",
						width, height, i, got, width-chromeCol)
				}
			}
		}
	}
}

// TestResponsiveLadder walks the widths the responsive rule names, each one
// column wider than the art budget it names because of the reserved chrome
// column: the card is pixel-correct from 79 terminal columns; between 72 and 78
// the copy column's rule shortens, which is a parameter rather than a change to
// the art; below that it restacks.
func TestResponsiveLadder(t *testing.T) {
	pack := fixturePack()
	tests := []struct {
		width  int
		layout string
		rule   int
	}{
		{200, "wide", 36}, {80, "wide", 36}, {79, "wide", 36},
		{78, "wide", 35}, {75, "wide", 32}, {72, "wide", 29},
		{71, "narrow", 0}, {60, "narrow", 0}, {58, "narrow", 0},
		{57, "plea", 0},
	}
	for _, tt := range tests {
		rows := plainRows(Card(pack, art.Quad, tt.width, 24, noHighlight))
		if got := layoutOf(rows); got != tt.layout {
			t.Errorf("%d columns: drew the %s layout, want %s", tt.width, got, tt.layout)
		}
		if tt.rule == 0 {
			continue
		}
		if got := strings.Count(rows[faceRow+ruleOffset], "─"); got != tt.rule {
			t.Errorf("%d columns: the copy column's rule is %d wide, want %d",
				tt.width, got, tt.rule)
		}
	}
}

// TestHeightLadder: below 23 rows - the composition's height now that its
// bottom two rows are one live nav row - there is no portrait that fits, so the
// card drops the art rather than the facts, and below the spec's floor it asks
// to be made bigger rather than showing a broken screen.
func TestHeightLadder(t *testing.T) {
	pack := fixturePack()
	for _, tt := range []struct {
		height int
		want   string
	}{
		{40, "wide"}, {24, "wide"}, {23, "wide"}, {22, "compact"}, {20, "compact"},
		{19, "plea"},
	} {
		if got := layoutOf(plainRows(Card(pack, art.Quad, 80, tt.height, noHighlight))); got != tt.want {
			t.Errorf("%d rows: drew the %s layout, want %s", tt.height, got, tt.want)
		}
	}
}

// TestLinksAreNeverTruncated is the card's hard promise. A URL that is cut to
// fit is a dead link, so at every width the card either shows a form that is
// complete on its own or it stops using this layout. Nothing in between.
func TestLinksAreNeverTruncated(t *testing.T) {
	pack := fixturePack()
	link, _ := pack.Link("linkedin")
	complete := map[string]bool{}
	for _, form := range urlForms(link.URL) {
		complete[form] = true
	}

	found := 0
	for width := 1; width <= 200; width++ {
		for _, row := range plainRows(Card(pack, art.Quad, width, 24, noHighlight)) {
			for _, field := range strings.Fields(row) {
				if !strings.Contains(field, "test-person") {
					continue
				}
				found++
				if !complete[field] {
					t.Errorf("%d columns: shows %q, which is not a complete form",
						width, field)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("the link never appeared at any width")
	}
}

// TestTheCardDrawsTheTierItIsGiven. The render ladder is only real if it
// reaches the screen: the card has to draw the portrait for this visitor's
// tier, at both sizes, and nothing else about the composition may move with it.
func TestTheCardDrawsTheTierItIsGiven(t *testing.T) {
	pack := realPack(t)
	sizes := []struct {
		name          string
		size          art.Size
		width, height int
	}{
		{"the two-column card", art.Wide, 80, 24},
		{"the vertical restack", art.Narrow, 60, 24},
	}
	for _, sz := range sizes {
		t.Run(sz.name, func(t *testing.T) {
			for _, tier := range art.Tiers {
				drawn := Card(pack, tier, sz.width, sz.height, noHighlight)
				want := art.Portrait(sz.size, tier)
				for _, line := range want {
					if trimmed := strings.TrimSpace(line); trimmed != "" &&
						!strings.Contains(drawn, trimmed) {
						t.Fatalf("the %s card does not carry a row of the %s portrait",
							tier, tier)
					}
				}
			}
		})
	}
}

// TestOnlyTheArtChangesWithTheTier. The tier decides how much picture a cell
// carries and never how many cells the picture takes, so swapping rungs must
// leave the composition alone: the copy column beside the portrait, the nav row
// under it, and where every one of them sits.
//
// Inside the art's own cells the tiers genuinely differ, and deliberately. The
// three cell renders are clipped to a disc because a photograph needs a frame
// to stop at, while the drawing keeps its own silhouette; and the bottom rung's
// wordmark is figlet's strokes rather than the glyphs that close them up. So
// this checks what surrounds the art; the budget itself is the next test.
func TestOnlyTheArtChangesWithTheTier(t *testing.T) {
	pack := realPack(t)
	var want []string
	for _, tier := range art.Tiers {
		outside := aroundTheArt(plainRows(Card(pack, tier, 80, 24, noHighlight)))
		if want == nil {
			want = outside
			continue
		}
		for row := range outside {
			if outside[row] != want[row] {
				t.Errorf("at the %s tier, row %d outside the art reads\n%q\nwant\n%q",
					tier, row, outside[row], want[row])
			}
		}
	}
}

// aroundTheArt blanks the cells the two art assets are allowed to paint - the
// wordmark's rows, which carry nothing else, and the portrait's column band -
// leaving every other cell of the card where it was.
func aroundTheArt(rows []string) []string {
	out := make([]string, len(rows))
	for row, line := range rows {
		runes := []rune(line)
		switch {
		case row < art.BannerRows:
			out[row] = ""
		case row < faceRow || row >= faceRow+art.WidePortraitRows:
			out[row] = line
		default:
			for col := faceCol; col < faceCol+art.WidePortraitCols && col < len(runes); col++ {
				runes[col] = ' '
			}
			out[row] = string(runes)
		}
	}
	return out
}

// TestTheColorlessCardIsDrawableAnywhere. The bottom rung is where a terminal
// lands when nothing in its session vouched for it, so nothing the card puts on
// screen at that rung may assume more than the oldest terminal has: the
// portrait is a drawing rather than a photograph, and the wordmark is figlet's
// own `/ \ | _` rather than the box glyphs that close its row gaps.
//
// The colour is not stripped here - that happens at the writer, in cmd - so
// this is about glyphs alone.
func TestTheColorlessCardIsDrawableAnywhere(t *testing.T) {
	pack := realPack(t)
	for _, row := range plainRows(Card(pack, art.Colorless, 80, 24, noHighlight)) {
		for col, r := range []rune(row) {
			// The copy column composes at runtime and draws its own rule; the
			// art is what this tier is responsible for.
			if col >= copyCol {
				continue
			}
			if r > '~' {
				t.Errorf("the colorless card draws %q at column %d, which an "+
					"old terminal has no glyph for", r, col)
			}
		}
	}
}

// TestEveryTierStaysInsideThePortraitBudget is the other half of the same
// promise, from the portrait's side: whatever a tier does with its silhouette,
// none may put ink outside the 36x18 the layout budgeted - the gutter it would
// spill into is the only thing keeping it off the copy column at 42.
func TestEveryTierStaysInsideThePortraitBudget(t *testing.T) {
	pack := realPack(t)
	for _, tier := range art.Tiers {
		rows := plainRows(Card(pack, tier, 80, 24, noHighlight))
		for row := faceRow; row < faceRow+art.WidePortraitRows; row++ {
			gutter := []rune(rows[row])[faceCol+art.WidePortraitCols : copyCol]
			if first, _ := ink(rows[row]); first >= 0 && first < faceCol {
				t.Errorf("the %s tier puts ink on row %d at column %d, left of the portrait's %d",
					tier, row, first, faceCol)
			}
			if first, _ := ink(string(gutter)); first >= 0 {
				t.Errorf("the %s tier puts ink on row %d in the gutter between the portrait and the copy column",
					tier, row)
			}
		}
	}
}

// TestCopyComesFromThePack: every fact on the card is composed from the pack at
// render time. Nothing here asserts a string about a person, only that what the
// pack says arrives on the screen.
func TestCopyComesFromThePack(t *testing.T) {
	pack := fixturePack()
	card := strings.Join(plainRows(Card(pack, art.Quad, 80, 24, noHighlight)), "\n")
	for _, want := range []string{
		"Test Engineer",           // role title
		"TestCo",                  // company
		"S00",                     // the program, shortened but not lost
		"Testfield",               // the institution
		"Current test:",           // the tagline, split at its own colon
		"Testing all the things.", //
		"github.com/TestPerson",   // the links, host and path
		"test-person-0a1b2c3d4",   //
	} {
		if !strings.Contains(card, want) {
			t.Errorf("card does not carry %q from the pack", want)
		}
	}
}

// TestCopyLadders checks each shortening rule at the width where it has to give
// something up, so a rule that quietly drops a fact too early is caught.
func TestCopyLadders(t *testing.T) {
	id := fixturePack().Identity
	tests := []struct {
		name  string
		forms []string
		width int
		want  string
	}{
		{"role, room for everything", roleForms(id), 60,
			"Test Engineer @ TestCo (Test Program S00)"},
		{"role, program shortened", roleForms(id), 36, "Test Engineer @ TestCo (TP S00)"},
		{"role, program dropped", roleForms(id), 25, "Test Engineer @ TestCo"},
		{"school, room for everything", schoolForms(id), 60,
			"Test Science and Testonomy @ Testfield"},
		{"school, fields shortened", schoolForms(id), 36, "TS + Testonomy @ Testfield"},
		{"school, institution outlives the fields", schoolForms(id), 12, "Testfield"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pick(tt.forms, tt.width)
			if !ok {
				t.Fatalf("nothing fits %d columns", tt.width)
			}
			if got != tt.want {
				t.Errorf("at %d columns = %q, want %q", tt.width, got, tt.want)
			}
		})
	}
}

func TestInitialise(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"Y Combinator S25", "YC S25"},
		{"Computer Science", "CS"},
		{"Mathematics", "Mathematics"},
		{"", ""},
	} {
		if got := initialise(tt.in); got != tt.want {
			t.Errorf("initialise(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestShortInstitution(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"Rutgers University - New Brunswick", "Rutgers"},
		{"Testfield College, Somewhere", "Testfield"},
		{"University", "University"},
		{"", ""},
	} {
		if got := shortInstitution(tt.in); got != tt.want {
			t.Errorf("shortInstitution(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- helpers ---

var sgrRe = regexp.MustCompile("\x1b\\[[0-9;:?]*[a-zA-Z]")

// plainRows strips the colour and pads every row out to its full column count,
// so a test can index a row by column the way the layout does.
func plainRows(card string) []string {
	rows := strings.Split(card, "\n")
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = sgrRe.ReplaceAllString(row, "")
	}
	width := 0
	for _, row := range out {
		width = max(width, len([]rune(row)))
	}
	for i, row := range out {
		out[i] = row + strings.Repeat(" ", width-len([]rune(row)))
	}
	return out
}

// namesColour reports whether an SGR parameter list selects a colour of the
// given family: a basic code, its bright twin, the extended selector, or the
// explicit default. "Default" counts - the point of the invariant is that the
// sequence says which background it wants, not that it wants a painted one.
func namesColour(params string, base, extended, def, bright int) bool {
	for _, p := range strings.Split(params, ";") {
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		switch {
		case n == extended || n == def,
			n >= base && n <= base+7,
			n >= bright && n <= bright+7:
			return true
		}
	}
	return false
}

func ink(row string) (first, last int) {
	runes := []rune(row)
	first, last = -1, -1
	for i, r := range runes {
		if r != ' ' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	return first, last
}

func assertInk(t *testing.T, rows []string, row, wantFirst, wantLast int) {
	t.Helper()
	first, last := ink(rows[row])
	if first < wantFirst || last > wantLast {
		t.Errorf("row %d has ink in columns %d..%d, want it inside %d..%d",
			row, first, last, wantFirst, wantLast)
	}
}

// layoutOf names which of the card's layouts produced a screen, by the one
// element each has that the others do not.
func layoutOf(rows []string) string {
	joined := strings.Join(rows, "\n")
	switch {
	case strings.Contains(joined, "bigger window"):
		return "plea"
	case strings.Contains(joined, "─"):
		return "wide"
	case strings.Contains(joined, "[?] help"):
		return "wide"
	case strings.Contains(rows[min(liveRow, len(rows)-1)], "[q] quit"):
		return "narrow"
	default:
		return "compact"
	}
}
