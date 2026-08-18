package art

import (
	"strings"
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

func TestAssetsAreTheDeclaredSize(t *testing.T) {
	for _, tier := range Tiers {
		t.Run("banner "+tier.String(), func(t *testing.T) {
			assertSize(t, Banner(tier), BannerCols, BannerRows)
		})
	}
	// Every tier of one size occupies the same cell budget. That is what lets
	// the layout decide which card fits before it knows anything about the
	// visitor's terminal: swapping tiers can never move the copy column.
	for _, size := range Sizes {
		cols, rows := PortraitSize(size)
		for _, tier := range Tiers {
			t.Run(size.String()+" "+tier.String(), func(t *testing.T) {
				assertSize(t, Portrait(size, tier), cols, rows)
			})
		}
	}
}

func assertSize(t *testing.T, lines []string, cols, rows int) {
	t.Helper()
	if got := len(lines); got != rows {
		t.Errorf("%d rows, want %d", got, rows)
	}
	if got := ansi.BlockWidth(lines); got != cols {
		t.Errorf("%d columns, want %d", got, cols)
	}
}

// TestEveryTierHasItsOwnAsset guards against the ladder quietly collapsing:
// a Portrait lookup that missed would return nil and a tier wired to the wrong
// file would serve a neighbour's render, and both would look like a working
// card at the tier above.
func TestEveryTierHasItsOwnAsset(t *testing.T) {
	for _, size := range Sizes {
		seen := make(map[string]Tier, len(Tiers))
		for _, tier := range Tiers {
			body := strings.Join(Portrait(size, tier), "\n")
			if body == "" {
				t.Fatalf("%s %s has no asset", size, tier)
			}
			if other, ok := seen[body]; ok {
				t.Errorf("%s %s is byte-identical to %s", size, tier, other)
			}
			seen[body] = tier
		}
	}
}

// TestRingCellsRestateBothColours is the finding that cost ticket 04 a round.
// clip-to-disc.py writes the ring as a bare foreground sequence; if the asset
// pipeline ever stops canonicalising, the right-hand arc inherits the
// background of whichever face cell the terminal painted before it and the ring
// gets skin tone behind it.
func TestRingCellsRestateBothColours(t *testing.T) {
	for _, size := range Sizes {
		for _, tier := range CellTiers {
			rings := 0
			for row, line := range Portrait(size, tier) {
				for col, cell := range ansi.ParseLine(line) {
					if !isBraille(cell.Rune) {
						continue
					}
					rings++
					if cell.State.FG == "" {
						t.Errorf("%s %s: ring cell at %d,%d names no foreground",
							size, tier, row, col)
					}
					if !strings.HasSuffix(cell.State.Prefix(), ";49m") &&
						!strings.Contains(cell.State.Prefix(), "48;") {
						t.Errorf("%s %s: ring cell at %d,%d does not restate a background: %q",
							size, tier, row, col, cell.State.Prefix())
					}
				}
			}
			if rings == 0 {
				t.Errorf("%s %s: no Braille ring cells found in the portrait", size, tier)
			}
		}
	}
}

// TestCellTiersAreFullyPainted keeps the ladder honest: on every tier that is a
// render of the photograph, every painted cell carries an explicit foreground
// and background, which is what makes the render proof against the visitor's
// own terminal theme.
func TestCellTiersAreFullyPainted(t *testing.T) {
	for _, size := range Sizes {
		for _, tier := range CellTiers {
			for row, line := range Portrait(size, tier) {
				for col, cell := range ansi.ParseLine(line) {
					if cell.Rune == ' ' || isBraille(cell.Rune) {
						continue
					}
					if cell.State.FG == "" || cell.State.BG == "" {
						t.Fatalf("%s %s: cell at %d,%d is not fully painted: %+v",
							size, tier, row, col, cell.State)
					}
				}
			}
		}
	}
}

// TestColorlessTierIsPlainASCII is the whole point of the bottom rung. A
// terminal reaches it because nothing about it said it could paint truecolor,
// and a terminal nothing vouches for cannot be credited with glyph coverage
// either - so its portrait names no colour at all and its wordmark, which does
// carry the gradient because the writer strips it, still has to draw itself out
// of characters no font has ever been without.
func TestColorlessTierIsPlainASCII(t *testing.T) {
	assertASCII(t, "wordmark", Banner(Colorless), true)
	for _, size := range Sizes {
		assertASCII(t, size.String()+" portrait", Portrait(size, Colorless), false)
	}
}

func assertASCII(t *testing.T, what string, lines []string, allowSGR bool) {
	t.Helper()
	for row, line := range lines {
		if !allowSGR && strings.ContainsRune(line, '\x1b') {
			t.Errorf("the colorless %s carries an ANSI escape on row %d", what, row)
		}
		for col, cell := range ansi.ParseLine(line) {
			if r := cell.Rune; r > '~' || (r < ' ' && r != 0) {
				t.Errorf("the colorless %s has %q at %d,%d, which is not ASCII",
					what, r, row, col)
			}
		}
	}
}

func isBraille(r rune) bool { return r >= 0x2800 && r <= 0x28FF }
