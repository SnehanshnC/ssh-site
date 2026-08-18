// Package ansi is an ANSI-aware fixed canvas: it parses coloured terminal art
// into cells, composes cells onto a grid, and emits the grid back out.
//
// Counting raw bytes is not an option here. One visible cell of the portrait is
// twenty-odd bytes of truecolor SGR, so anything that measures or slices art by
// string length is wrong the moment it meets a face. Everything in this package
// works on (state, rune) cells instead, where a state is a resolved
// (attributes, foreground, background) triple.
//
// Two findings from the ticket-04 prototype are baked into [State.Prefix], and
// both cost a prototype round to learn:
//
//   - A state prefix always names *both* foreground and background. Ring cells
//     that set only a foreground inherit the background of whichever face cell
//     the terminal painted before them, which puts skin tone behind the ring.
//   - Resolving each cell's state and re-emitting it canonically is what keeps
//     a composed screen around 18 KB. Carrying raw SGR prefixes forward by
//     concatenation - which is what a naive parser does with chafa output,
//     since chafa emits fg+bg per cell and never resets - grows an ever longer
//     prefix per cell and balloons the same screen to megabytes.
package ansi

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

// sgrRe matches one escape sequence: a CSI sequence, an OSC string, or a
// two-byte escape. Only CSI sequences ending in 'm' change cell state; the
// rest are recognised so they can be skipped without being mistaken for text.
var sgrRe = regexp.MustCompile(
	"^(?:\x1b\\[[0-9;:?]*[a-zA-Z]|\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)|\x1b[@-Z\\\\-_])")

// State is a resolved SGR state. Attrs holds the attribute parameters in the
// order they were first seen, semicolon-joined, so State stays comparable.
type State struct {
	Attrs string
	FG    string
	BG    string
}

// Cell is one terminal cell: a state and the rune painted in it. A wide rune
// occupies two cells, the second of which is a filler carrying no rune.
type Cell struct {
	State  State
	Rune   rune
	Filler bool
}

// Blank is an unstyled space, the value every untouched canvas cell holds.
var Blank = Cell{Rune: ' '}

func (c Cell) isBlank() bool { return c == Blank }

// Apply folds one SGR parameter list into a state.
func (s State) Apply(params string) State {
	attrs := splitAttrs(s.Attrs)
	fg, bg := s.FG, s.BG

	ps := strings.Split(params, ";")
	if params == "" {
		ps = []string{"0"}
	}
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		if p == "" {
			p = "0"
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		switch {
		case n == 0:
			attrs, fg, bg = nil, "", ""
		case n == 38 || n == 48:
			// 38;5;N (indexed) or 38;2;R;G;B (truecolor)
			val := p
			switch {
			case i+1 < len(ps) && ps[i+1] == "5" && i+2 < len(ps):
				val = strings.Join(ps[i:i+3], ";")
				i += 2
			case i+1 < len(ps) && ps[i+1] == "2" && i+4 < len(ps):
				val = strings.Join(ps[i:i+5], ";")
				i += 4
			}
			if n == 38 {
				fg = val
			} else {
				bg = val
			}
		case n == 39:
			fg = ""
		case n == 49:
			bg = ""
		case (n >= 30 && n <= 37) || (n >= 90 && n <= 97):
			fg = p
		case (n >= 40 && n <= 47) || (n >= 100 && n <= 107):
			bg = p
		case n == 21 || n == 22:
			attrs = without(attrs, "1", "2")
		case n >= 23 && n <= 29 && n != 26:
			attrs = without(attrs, strconv.Itoa(n-20))
		default:
			if !contains(attrs, p) {
				attrs = append(attrs, p)
			}
		}
	}
	return State{Attrs: strings.Join(attrs, ";"), FG: fg, BG: bg}
}

// Prefix is the canonical, self-sufficient escape sequence for a state. It
// always names a foreground and a background, so a cell can never inherit
// either from whatever the terminal painted before it.
func (s State) Prefix() string {
	if s == (State{}) {
		return ""
	}
	var b strings.Builder
	b.WriteString("\x1b[")
	if s.Attrs != "" {
		b.WriteString(s.Attrs)
		b.WriteByte(';')
	}
	if s.FG != "" {
		b.WriteString(s.FG)
	} else {
		b.WriteString("39")
	}
	b.WriteByte(';')
	if s.BG != "" {
		b.WriteString(s.BG)
	} else {
		b.WriteString("49")
	}
	b.WriteByte('m')
	return b.String()
}

func splitAttrs(a string) []string {
	if a == "" {
		return nil
	}
	return strings.Split(a, ";")
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func without(xs []string, drop ...string) []string {
	out := xs[:0]
	for _, v := range xs {
		if !contains(drop, v) {
			out = append(out, v)
		}
	}
	return out
}

// ParseLine splits one line of possibly-coloured text into cells.
func ParseLine(line string) []Cell {
	var cells []Cell
	var state State
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if loc := sgrRe.FindStringIndex(line[i:]); loc != nil {
				seq := line[i : i+loc[1]]
				if strings.HasSuffix(seq, "m") && strings.HasPrefix(seq, "\x1b[") {
					state = state.Apply(seq[2 : len(seq)-1])
				}
				i += loc[1]
				continue
			}
		}
		r, size := decodeRune(line[i:])
		i += size
		if r == '\r' || r == '\n' {
			continue
		}
		switch runewidth.RuneWidth(r) {
		case 0:
			continue
		case 2:
			cells = append(cells, Cell{State: state, Rune: r},
				Cell{State: state, Filler: true})
		default:
			cells = append(cells, Cell{State: state, Rune: r})
		}
	}
	return cells
}

// ParseBlock splits a block of lines into rows of cells.
func ParseBlock(lines []string) [][]Cell {
	rows := make([][]Cell, len(lines))
	for i, line := range lines {
		rows[i] = ParseLine(line)
	}
	return rows
}

// EmitRow renders one row of cells back to a string.
func EmitRow(row []Cell) string {
	var b strings.Builder
	var cur State
	for _, cell := range row {
		if cell.Filler {
			continue
		}
		if cell.State != cur {
			// A reset is needed to turn attributes off, and to return to the
			// terminal's own colours - Prefix is empty for the zero state, so
			// without this an unstyled cell would keep painting with whatever
			// the cell before it set. Every other transition costs exactly one
			// sequence, because the canonical prefix restates both colours.
			if cur != (State{}) && (cell.State == (State{}) || dropsAttrs(cur, cell.State)) {
				b.WriteString("\x1b[0m")
			}
			b.WriteString(cell.State.Prefix())
			cur = cell.State
		}
		b.WriteRune(cell.Rune)
	}
	if cur != (State{}) {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func dropsAttrs(from, to State) bool {
	for _, a := range splitAttrs(from.Attrs) {
		if !contains(splitAttrs(to.Attrs), a) {
			return true
		}
	}
	return false
}

// Width returns the number of terminal columns a possibly-coloured string
// occupies.
func Width(line string) int { return len(ParseLine(line)) }

// BlockWidth returns the widest line of a block, in columns.
func BlockWidth(lines []string) int {
	w := 0
	for _, line := range lines {
		if n := Width(line); n > w {
			w = n
		}
	}
	return w
}
