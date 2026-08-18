// Package art holds the SSH site's terminal art: the wordmark banner and the
// portrait discs, as pre-rendered ANSI assets.
//
// The art is a build-time asset and these files are checked in. `make art`
// regenerates them from the source headshot through ImageMagick, chafa and
// figlet; CI never runs it and builds from what is committed here. The reason
// is the difference in how often the two halves of the card change: the
// photograph changes when a human decides to change it, while the facts beside
// it change on every content-pack push, so a pack push must never have to
// re-render a photograph. See art/README.md.
package art

import (
	"embed"
	"fmt"
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

//go:embed assets/*.ans
var assetsFS embed.FS

// Size selects which portrait a layout wants. The two are not the same picture
// scaled: each is rendered from the master at its own cell budget and sharpened
// at its own subcell grid.
type Size int

const (
	// Wide is the 36x18 disc the two-column card carries.
	Wide Size = iota
	// Narrow is the 32x16 disc the vertical restack carries, where height,
	// not width, is the binding constraint.
	Narrow
)

// Tier is a rung of the render ladder: how much picture one cell of the
// visitor's terminal can be asked to carry. Declared best to worst, so a lower
// tier is literally a larger Tier and the zero value is the best one.
//
// The three cell tiers are the same photograph at three subcell resolutions,
// and every cell of all three paints both a foreground and a background, which
// is what makes them proof against the visitor's own terminal theme. The
// bottom rung is not that photograph with its colour removed - a photographic
// conversion without colour was tried in ticket 04 round 1 and read as blurry
// pixels - it is the hand-drawn line art that round 2 produced and that was
// kept as an asset for exactly this.
//
// Which rung a visitor gets is [github.com/SnehanshnC/ssh-site/internal/capability]'s
// decision, from their session environment. Nothing here guesses.
type Tier int

const (
	// Sextant is 2x3 pixels per cell, the sharpest tier. It needs a terminal
	// that draws Unicode 13's Symbols for Legacy Computing itself, because no
	// font in general circulation carries them: kitty, foot, Ghostty, WezTerm.
	Sextant Tier = iota
	// Quad is 2x2 pixels per cell, the mainstream default. Quadrant blocks are
	// old enough that the fonts terminals actually ship with carry them, and
	// they cost nothing over half blocks.
	Quad
	// VHalf is 1x2 pixels per cell, on the two half blocks every font has
	// carried since CP437. The floor for a terminal that can paint colour but
	// whose glyph coverage cannot be assumed.
	VHalf
	// Colorless is the hand-drawn line-art portrait, in plain ASCII with no
	// colour at all - what a terminal gets when nothing about it says it can
	// paint truecolor.
	Colorless
)

// Tiers is the ladder in order, best to worst.
var Tiers = []Tier{Sextant, Quad, VHalf, Colorless}

// Sizes is every portrait size, for callers that walk them all.
var Sizes = []Size{Wide, Narrow}

// CellTiers is the part of the ladder that is a render of the photograph, as
// opposed to the drawing. These are the tiers that paint every cell and carry
// the Braille ring.
var CellTiers = []Tier{Sextant, Quad, VHalf}

var tierNames = map[Tier]string{
	Sextant: "sextant", Quad: "quad", VHalf: "vhalf", Colorless: "colorless",
}

var sizeNames = map[Size]string{Wide: "wide", Narrow: "narrow"}

// String names the tier the way the asset files and the logs do.
func (t Tier) String() string {
	if name, ok := tierNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Tier(%d)", int(t))
}

// String names the size the way the asset files do.
func (s Size) String() string {
	if name, ok := sizeNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Size(%d)", int(s))
}

// Dimensions of each asset, in cells. They are constants rather than measured
// at load time because the layout budgets rows and columns against them before
// it decides which card to draw. Every tier of one size is that same budget:
// the tiers differ in how much picture a cell carries, never in how many cells
// the picture takes.
const (
	BannerCols = 46
	BannerRows = 4

	WidePortraitCols, WidePortraitRows     = 36, 18
	NarrowPortraitCols, NarrowPortraitRows = 32, 16
)

type portrait struct {
	size Size
	tier Tier
}

var (
	banner          = mustLoad("banner", BannerCols, BannerRows)
	colorlessBanner = mustLoad("banner-colorless", BannerCols, BannerRows)
	portraits       = loadPortraits()
)

// Banner returns the figlet smslant SNEHANSHN wordmark, 46x4, painted with the
// cyan-violet gradient.
//
// The bottom rung gets its own, for the same reason it gets a drawing instead
// of a photograph. The wordmark's shipped form closes figlet's row gaps by
// swapping each stroke for the box or block glyph that fills the whole cell,
// and a terminal that reached that rung is one nothing in the session vouched
// for - so it gets the strokes figlet drew, in the same 46x4 and the same
// place, out of characters that predate the problem.
func Banner(tier Tier) []string {
	if tier == Colorless {
		return colorlessBanner
	}
	return banner
}

// Portrait returns the portrait at the given size, rendered for the given tier.
func Portrait(size Size, tier Tier) []string {
	return portraits[portrait{size, tier}]
}

// PortraitSize reports the cell budget a portrait of the given size occupies,
// which is the same at every tier.
func PortraitSize(size Size) (cols, rows int) {
	if size == Narrow {
		return NarrowPortraitCols, NarrowPortraitRows
	}
	return WidePortraitCols, WidePortraitRows
}

func loadPortraits() map[portrait][]string {
	out := make(map[portrait][]string, len(Sizes)*len(Tiers))
	for _, size := range Sizes {
		cols, rows := PortraitSize(size)
		for _, tier := range Tiers {
			name := "portrait-" + size.String() + "-" + tier.String()
			out[portrait{size, tier}] = mustLoad(name, cols, rows)
		}
	}
	return out
}

func mustLoad(name string, cols, rows int) []string {
	lines, err := load(name, cols, rows)
	if err != nil {
		panic(err)
	}
	return lines
}

func load(name string, cols, rows int) ([]string, error) {
	b, err := assetsFS.ReadFile("assets/" + name + ".ans")
	if err != nil {
		return nil, fmt.Errorf("read art asset %s: %w", name, err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) != rows {
		return nil, fmt.Errorf("art asset %s: %d rows, want %d", name, len(lines), rows)
	}
	if w := ansi.BlockWidth(lines); w != cols {
		return nil, fmt.Errorf("art asset %s: %d columns, want %d", name, w, cols)
	}
	return lines, nil
}
