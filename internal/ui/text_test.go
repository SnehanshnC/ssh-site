package ui

import (
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// TestWrapNeverSplitsOnHyphens is the prototype's finding as a test. Python's
// textwrap breaks on hyphens unless told not to, which is how the deck strand
// shipped "non-blocking" split across two lines; a hyphen inside a word is part
// of the word, and only a space is a place to break.
func TestWrapNeverSplitsOnHyphens(t *testing.T) {
	lines := wrap("a non-blocking async I/O engine", 12)
	for _, line := range lines {
		if strings.HasSuffix(line, "-") {
			t.Errorf("wrapped to %q, which broke a word at its hyphen", lines)
		}
	}
	if joined := strings.Join(lines, " "); !strings.Contains(joined, "non-blocking") {
		t.Errorf("wrapped to %q, which lost the whole word", lines)
	}
}

// TestWrapMeasuresColumnsNotBytes: one visible cell of this surface is twenty-odd
// bytes of truecolor SGR, so a wrap that counted bytes would break these lines
// after the first word and cut an escape sequence in half doing it.
func TestWrapMeasuresColumnsNotBytes(t *testing.T) {
	styled := paint(accentState, "alpha") + " " + paint(textState, "beta gamma delta")
	lines := wrap(styled, 12)
	if len(lines) != 2 {
		t.Fatalf("wrapped %d lines, want 2: %q", len(lines), lines)
	}
	for i, line := range lines {
		if got := ansi.Width(line); got > 12 {
			t.Errorf("line %d is %d columns, over the 12 asked for", i, got)
		}
	}
	if got := sgrRe.ReplaceAllString(strings.Join(lines, "|"), ""); got != "alpha beta|gamma delta" {
		t.Errorf("wrapped to %q", got)
	}
}

// TestWrapCarriesStyleAcrossTheBreak: a run that spans a wrap keeps its colour
// on the next line rather than falling back to the terminal's own.
func TestWrapCarriesStyleAcrossTheBreak(t *testing.T) {
	lines := wrap(paint(accentState, "one two three four"), 8)
	if len(lines) < 2 {
		t.Fatalf("wrapped %d lines, want more than one: %q", len(lines), lines)
	}
	for i, line := range lines[1:] {
		if !strings.Contains(line, accentState.Prefix()) {
			t.Errorf("line %d is %q, which lost the run's colour", i+1, line)
		}
	}
}

// TestWrapHangingIndentsEveryLineButTheFirst is the shape a bullet wants: the
// marker sits in the indent and the text lines up under its own first word.
func TestWrapHangingIndentsEveryLineButTheFirst(t *testing.T) {
	lines := wrapHanging("one two three four five six", 12, 2)
	if len(lines) < 3 {
		t.Fatalf("wrapped %d lines, want at least 3: %q", len(lines), lines)
	}
	if strings.HasPrefix(lines[0], " ") {
		t.Errorf("the first line %q is indented, and should be flush", lines[0])
	}
	for i, line := range lines[1:] {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d is %q, which is not hung under the first", i+1, line)
		}
		if got := ansi.Width(line); got > 12 {
			t.Errorf("line %d is %d columns, over the 12 asked for", i+1, got)
		}
	}
}

// TestWrapBreaksAWordNoLineCouldHold: the one place either helper splits
// something that is not a space, because the alternative is a row that runs off
// the screen.
func TestWrapBreaksAWordNoLineCouldHold(t *testing.T) {
	lines := wrap("supercalifragilistic", 6)
	if len(lines) != 4 {
		t.Fatalf("wrapped %d lines, want 4: %q", len(lines), lines)
	}
	for i, line := range lines {
		if got := ansi.Width(line); got > 6 {
			t.Errorf("line %d is %d columns, over the 6 asked for", i, got)
		}
	}
	if got := strings.Join(lines, ""); got != "supercalifragilistic" {
		t.Errorf("rejoined to %q, so the break lost something", got)
	}
}

func TestWrapEdges(t *testing.T) {
	if got := wrap("", 10); len(got) != 1 || got[0] != "" {
		t.Errorf("wrapping nothing gave %q, want one empty line", got)
	}
	if got := wrap("anything", 0); len(got) != 1 || got[0] != "anything" {
		t.Errorf("wrapping into no columns gave %q, want the text back", got)
	}
}

// TestClipMarksWhatItCut. Clipping is for chrome, where a shortened label is
// still true; nothing that has to stay complete to be useful goes through it.
func TestClipMarksWhatItCut(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"home / work", "home / work"},
		{"home / work / novaflow", "home / wo..."},
	} {
		if got := clip(tt.in, 12); got != tt.want {
			t.Errorf("clip(%q, 12) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := ansi.Width(clip(paint(textState, "home / work / novaflow"), 12)); got != 12 {
		t.Errorf("a styled string clipped to %d columns, want 12", got)
	}
}
