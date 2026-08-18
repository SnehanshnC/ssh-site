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

// Dimensions of each asset, in cells. They are constants rather than measured
// at load time because the layout budgets rows and columns against them before
// it decides which card to draw.
const (
	BannerCols = 46
	BannerRows = 4

	WidePortraitCols, WidePortraitRows     = 36, 18
	NarrowPortraitCols, NarrowPortraitRows = 32, 16
)

var (
	banner    = mustLoad("banner", BannerCols, BannerRows)
	portraits = map[Size][]string{
		Wide:   mustLoad("portrait-wide-quad", WidePortraitCols, WidePortraitRows),
		Narrow: mustLoad("portrait-narrow-quad", NarrowPortraitCols, NarrowPortraitRows),
	}
)

// Banner returns the figlet smslant SNEHANSHN wordmark, 46x4, painted with the
// cyan-violet gradient.
func Banner() []string { return banner }

// Portrait returns the disc-clipped, Braille-ringed portrait at the given size.
func Portrait(size Size) []string { return portraits[size] }

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
