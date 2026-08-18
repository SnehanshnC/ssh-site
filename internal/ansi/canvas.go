package ansi

import "strings"

// Canvas is a fixed cols x rows grid of cells. Blocks are composed onto it by
// coordinate and it renders once, at the end, so a screen is assembled the way
// the ticket-04 prototype assembled its captures: nothing is measured by byte
// length and nothing is concatenated into a line before its neighbours are
// known.
//
// Writes outside the grid are dropped rather than wrapped or panicked on. A
// visitor with an unusual terminal should get a clipped card, not a crashed
// session; the layout in the ui package is responsible for choosing a variant
// that fits, and its tests assert that nothing is ever dropped.
type Canvas struct {
	cols, rows int
	grid       [][]Cell
	clipped    bool
}

// NewCanvas returns a blank cols x rows canvas.
func NewCanvas(cols, rows int) *Canvas {
	grid := make([][]Cell, rows)
	for r := range grid {
		grid[r] = make([]Cell, cols)
		for c := range grid[r] {
			grid[r][c] = Blank
		}
	}
	return &Canvas{cols: cols, rows: rows, grid: grid}
}

// Cols returns the canvas width in columns.
func (cv *Canvas) Cols() int { return cv.cols }

// Rows returns the canvas height in rows.
func (cv *Canvas) Rows() int { return cv.rows }

// Clipped reports whether any write fell outside the grid. It is the signal
// the layout tests assert on.
func (cv *Canvas) Clipped() bool { return cv.clipped }

// Put paints a block of possibly-coloured lines with its top-left corner at
// (x, y).
func (cv *Canvas) Put(x, y int, block []string) *Canvas {
	for dy, row := range ParseBlock(block) {
		ry := y + dy
		if ry < 0 || ry >= cv.rows {
			cv.clipped = cv.clipped || len(row) > 0
			continue
		}
		for dx, cell := range row {
			rx := x + dx
			if rx < 0 || rx >= cv.cols {
				cv.clipped = true
				continue
			}
			cv.grid[ry][rx] = cell
		}
	}
	return cv
}

// PutLine paints a single line at (x, y).
func (cv *Canvas) PutLine(x, y int, line string) *Canvas {
	return cv.Put(x, y, []string{line})
}

// Center paints a block horizontally centred on the canvas, starting at row y.
func (cv *Canvas) Center(y int, block []string) *Canvas {
	return cv.Put((cv.cols-BlockWidth(block))/2, y, block)
}

// Rule paints a horizontal run of r on row y, from column x, w columns wide,
// in the given SGR state.
func (cv *Canvas) Rule(y, x, w int, r rune, state State) *Canvas {
	if w <= 0 {
		return cv
	}
	return cv.PutLine(x, y, state.Prefix()+strings.Repeat(string(r), w))
}

// Render emits the canvas as a newline-joined block. Trailing blank cells are
// dropped from every row, so a screen carries no run of empty styled cells.
func (cv *Canvas) Render() string {
	lines := make([]string, cv.rows)
	for i, row := range cv.grid {
		end := len(row)
		for end > 0 && row[end-1].isBlank() {
			end--
		}
		lines[i] = EmitRow(row[:end])
	}
	return strings.Join(lines, "\n")
}
