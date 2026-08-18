package ui

import (
	"strings"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
)

// The surface's text measuring and breaking, written once here and shared by
// every page. Two rules, both learned from the ticket-06 prototype:
//
//   - Wrapping breaks on spaces and on nothing else. Python's textwrap splits
//     on hyphens by default, which is how the prototype's deck strand shipped
//     "non-blocking" broken across two lines. A hyphen inside a word is part of
//     the word.
//   - Everything works on cells, never on bytes. One visible cell of this
//     surface is twenty-odd bytes of truecolor SGR, so a byte-counting wrap
//     would break a line in the middle of an escape sequence and paint the rest
//     of the screen the wrong colour.
//
// Both helpers carry the active SGR state across the break, so a styled run
// that spans a wrap keeps its colour on the next line.

// wrap breaks text into lines of at most width columns, splitting only on
// spaces.
func wrap(text string, width int) []string {
	return wrapHanging(text, width, 0)
}

// wrapHanging wraps text with its first line flush and every line after it
// indented by indent columns, which is the shape a bullet or a labelled field
// wants: the marker sits in the indent and the text lines up under itself.
func wrapHanging(text string, width, indent int) []string {
	if width <= 0 {
		return []string{text}
	}
	if indent < 0 || indent >= width {
		indent = 0
	}

	var lines []string
	var line []ansi.Cell

	// The first line has the whole column; every line after it gives up the
	// hanging indent, which is why both depend on how many lines are already
	// out rather than on a counter of their own.
	limit := func() int {
		if len(lines) == 0 {
			return width
		}
		return width - indent
	}
	flush := func() {
		prefix := ""
		if len(lines) > 0 {
			prefix = strings.Repeat(" ", indent)
		}
		lines = append(lines, prefix+ansi.EmitRow(line))
		line = nil
	}

	for _, word := range splitWords(ansi.ParseLine(text), max(width-indent, 1)) {
		space := 0
		if len(line) > 0 {
			space = 1
		}
		if len(line)+space+len(word) > limit() && len(line) > 0 {
			flush()
			space = 0
		}
		if space == 1 {
			// The joining space carries the state of the run it follows, so a
			// styled phrase stays one uninterrupted run of cells.
			line = append(line, ansi.Cell{State: line[len(line)-1].State, Rune: ' '})
		}
		line = append(line, word...)
	}
	if len(line) > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

// splitWords cuts cells into space-separated words, breaking any word that no
// line could hold into max-column pieces. That break is the only place either
// helper splits something that is not a space, and it never lands between a
// wide rune and the filler cell that follows it.
func splitWords(cells []ansi.Cell, max int) [][]ansi.Cell {
	var words [][]ansi.Cell
	var word []ansi.Cell
	push := func() {
		for len(word) > max {
			n := max
			if word[n].Filler {
				n--
			}
			if n <= 0 {
				n = max
			}
			words = append(words, word[:n])
			word = word[n:]
		}
		if len(word) > 0 {
			words = append(words, word)
		}
		word = nil
	}
	for _, cell := range cells {
		if cell.Rune == ' ' && !cell.Filler {
			push()
			continue
		}
		word = append(word, cell)
	}
	push()
	return words
}

// clip cuts a string to width columns, marking the cut with an ellipsis. It is
// truncation, not wrapping: it belongs to chrome - a breadcrumb, a count, a row
// of key hints - where a shortened label is still true. Nothing that has to
// stay complete to be useful, a URL above all, is ever passed through it.
func clip(text string, width int) string {
	if width <= 0 {
		return ""
	}
	cells := ansi.ParseLine(text)
	if len(cells) <= width {
		return text
	}
	const ellipsis = "..."
	if width <= len(ellipsis) {
		return ansi.EmitRow(cells[:width])
	}
	return ansi.EmitRow(cells[:width-len(ellipsis)]) + ellipsis
}
