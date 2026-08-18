package art

import (
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

func TestAssetsAreTheDeclaredSize(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		cols, rows int
	}{
		{"banner", Banner(), BannerCols, BannerRows},
		{"wide portrait", Portrait(Wide), WidePortraitCols, WidePortraitRows},
		{"narrow portrait", Portrait(Narrow), NarrowPortraitCols, NarrowPortraitRows},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(tt.lines); got != tt.rows {
				t.Errorf("%d rows, want %d", got, tt.rows)
			}
			if got := ansi.BlockWidth(tt.lines); got != tt.cols {
				t.Errorf("%d columns, want %d", got, tt.cols)
			}
		})
	}
}

// TestRingCellsRestateBothColours is the finding that cost ticket 04 a round.
// clip-to-disc.py writes the ring as a bare foreground sequence; if the asset
// pipeline ever stops canonicalising, the right-hand arc inherits the
// background of whichever face cell the terminal painted before it and the ring
// gets skin tone behind it.
func TestRingCellsRestateBothColours(t *testing.T) {
	for _, size := range []Size{Wide, Narrow} {
		rings := 0
		for row, line := range Portrait(size) {
			for col, cell := range ansi.ParseLine(line) {
				if !isBraille(cell.Rune) {
					continue
				}
				rings++
				if cell.State.FG == "" {
					t.Errorf("ring cell at %d,%d names no foreground", row, col)
				}
				if !strings.HasSuffix(cell.State.Prefix(), ";49m") &&
					!strings.Contains(cell.State.Prefix(), "48;") {
					t.Errorf("ring cell at %d,%d does not restate a background: %q",
						row, col, cell.State.Prefix())
				}
			}
		}
		if rings == 0 {
			t.Error("no Braille ring cells found in the portrait")
		}
	}
}

// TestPortraitsAreCellRenders keeps the tier honest: every painted cell carries
// an explicit foreground and background, which is what makes the render proof
// against the visitor's own terminal theme.
func TestPortraitsAreCellRenders(t *testing.T) {
	for _, line := range Portrait(Wide) {
		for col, cell := range ansi.ParseLine(line) {
			if cell.Rune == ' ' || isBraille(cell.Rune) {
				continue
			}
			if cell.State.FG == "" || cell.State.BG == "" {
				t.Fatalf("face cell at column %d is not fully painted: %+v",
					col, cell.State)
			}
		}
	}
}

func isBraille(r rune) bool { return r >= 0x2800 && r <= 0x28FF }
