package ansi

import (
	"strings"
	"testing"
)

func TestApplyResolvesColoursAndAttributes(t *testing.T) {
	tests := []struct {
		name   string
		params []string
		want   State
	}{
		{"truecolor pair", []string{"38;2;1;2;3", "48;2;4;5;6"},
			State{FG: "38;2;1;2;3", BG: "48;2;4;5;6"}},
		{"indexed", []string{"38;5;208"}, State{FG: "38;5;208"}},
		{"basic", []string{"31", "44"}, State{FG: "31", BG: "44"}},
		{"bold then colour", []string{"1", "38;2;1;2;3"},
			State{Attrs: "1", FG: "38;2;1;2;3"}},
		{"default colours", []string{"38;2;1;2;3;48;2;4;5;6", "39", "49"}, State{}},
		{"reset", []string{"1;38;2;1;2;3", "0"}, State{}},
		{"bold off keeps colour", []string{"1;38;2;1;2;3", "22"},
			State{FG: "38;2;1;2;3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got State
			for _, p := range tt.params {
				got = got.Apply(p)
			}
			if got != tt.want {
				t.Errorf("state = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestPrefixAlwaysNamesBothColours is the ring finding from the ticket-04
// prototype, as an invariant: a cell that names only a foreground inherits the
// background of whichever cell the terminal painted before it, which puts the
// face's skin tone behind the disc's ring.
func TestPrefixAlwaysNamesBothColours(t *testing.T) {
	for _, params := range []string{"38;2;148;163;184", "48;2;1;2;3", "1", "31"} {
		prefix := State{}.Apply(params).Prefix()
		if !hasForeground(prefix) {
			t.Errorf("Apply(%q).Prefix() = %q, names no foreground", params, prefix)
		}
		if !hasBackground(prefix) {
			t.Errorf("Apply(%q).Prefix() = %q, names no background", params, prefix)
		}
	}
	if got := (State{}).Prefix(); got != "" {
		t.Errorf("zero state Prefix() = %q, want empty", got)
	}
}

func TestParseLineCountsColumnsNotBytes(t *testing.T) {
	line := "\x1b[38;2;1;2;3;48;2;4;5;6m▛\x1b[38;2;7;8;9m▙\x1b[0m ab"
	if got, want := Width(line), 5; got != want {
		t.Fatalf("Width = %d, want %d", got, want)
	}
	cells := ParseLine(line)
	if cells[0].State.BG != "48;2;4;5;6" {
		t.Errorf("cell 0 background = %q, want the one it was given", cells[0].State.BG)
	}
	// chafa never resets between cells, so an unstated background carries.
	if cells[1].State.BG != "48;2;4;5;6" {
		t.Errorf("cell 1 background = %q, want it carried forward", cells[1].State.BG)
	}
	if cells[2].State != (State{}) {
		t.Errorf("cell 2 state = %+v, want cleared by the reset", cells[2].State)
	}
}

func TestParseLineHandlesWideAndCombiningRunes(t *testing.T) {
	if got, want := Width("漢字"), 4; got != want {
		t.Errorf("Width(wide) = %d, want %d", got, want)
	}
	if got, want := Width("é"), 1; got != want {
		t.Errorf("Width(combining) = %d, want %d", got, want)
	}
}

// TestEmitRowDoesNotGrowTruecolorRuns is the other half of the prototype's
// canonicalisation finding: a parser that carries raw SGR prefixes forward by
// concatenation turns a screen of per-cell truecolor into megabytes. Resolving
// the state and re-emitting it canonically has to be idempotent in size, or
// composing a screen twice would double it.
func TestEmitRowDoesNotGrowTruecolorRuns(t *testing.T) {
	var b strings.Builder
	for i := range 200 {
		b.WriteString("\x1b[38;2;" + itoa(i) + ";0;0;48;2;0;0;" + itoa(i) + "m▀")
	}
	line := b.String()

	once := EmitRow(ParseLine(line))
	twice := EmitRow(ParseLine(once))
	if once != twice {
		t.Error("re-emitting a canonical line changed it")
	}
	// The only thing the canonical form adds is the reset that closes the row.
	if len(once) > len(line)+len("\x1b[0m") {
		t.Errorf("emitted %d bytes from %d: canonical form must not grow",
			len(once), len(line))
	}
	if Width(once) != 200 {
		t.Errorf("emitted width = %d, want 200", Width(once))
	}
}

// TestEmitRowClosesStateBeforeUnstyledCells guards the failure that makes a
// background leak: an unstyled cell has an empty prefix, so without an explicit
// reset it keeps painting with whatever the cell before it set.
func TestEmitRowClosesStateBeforeUnstyledCells(t *testing.T) {
	row := append(ParseLine("\x1b[48;2;9;9;9mX"), Blank, Blank)
	got := EmitRow(row)
	if !strings.Contains(got, "X\x1b[0m") {
		t.Errorf("EmitRow = %q, want the background closed before the spaces", got)
	}
}

func TestCanvasComposesByColumn(t *testing.T) {
	cv := NewCanvas(10, 3)
	cv.Put(2, 0, []string{"ab", "cd"})
	cv.Center(2, []string{"xy"})
	if got, want := cv.Render(), "  ab\n  cd\n    xy"; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
	if cv.Clipped() {
		t.Error("Clipped() = true, want false")
	}
}

func TestCanvasReportsClipping(t *testing.T) {
	cv := NewCanvas(4, 2)
	cv.PutLine(2, 0, "abcd")
	if !cv.Clipped() {
		t.Error("Clipped() = false after writing past the right edge")
	}
	if got, want := Width(cv.Render()), 4; got > want {
		t.Errorf("clipped row is %d columns wide, want at most %d", got, want)
	}
}

func TestCanvasRuleSpansExactly(t *testing.T) {
	cv := NewCanvas(10, 1)
	cv.Rule(0, 2, 5, '─', State{FG: "38;2;1;2;3"})
	row := ParseLine(cv.Render())
	if got, want := len(row), 7; got != want {
		t.Fatalf("rule row is %d columns, want %d", got, want)
	}
	for i := 2; i < 7; i++ {
		if row[i].Rune != '─' {
			t.Errorf("column %d = %q, want the rule rune", i, row[i].Rune)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func hasForeground(prefix string) bool { return sgrHas(prefix, 30, 38, 39, 90) }
func hasBackground(prefix string) bool { return sgrHas(prefix, 40, 48, 49, 100) }

// sgrHas reports whether a prefix names a colour of the given family: a basic
// code in [base, base+7], the extended selector, or the default selector.
func sgrHas(prefix string, base, extended, def, bright int) bool {
	params := strings.Split(strings.TrimSuffix(strings.TrimPrefix(prefix, "\x1b["), "m"), ";")
	for _, p := range params {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				n = -1
				break
			}
			n = n*10 + int(r-'0')
		}
		switch {
		case n == extended || n == def:
			return true
		case n >= base && n <= base+7:
			return true
		case n >= bright && n <= bright+7:
			return true
		}
	}
	return false
}
