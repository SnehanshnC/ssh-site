"""ANSI-aware cell grid: parse a coloured line into cells, emit it canonically.

Ported from the prototype's `variants/lib/ansicanvas.py`. Only the parts the
art build needs are here - the Go side owns composition, this side only has to
clip a render to a disc and write the result back out.

The one thing worth restating, because it is the finding that cost a prototype
round: `state_prefix` always names *both* foreground and background. A ring cell
that only sets a foreground inherits the background of whichever face cell the
terminal painted before it, which puts the face's skin tone behind the ring.
Resolving each cell to a full (attrs, fg, bg) state and re-emitting it
canonically also keeps a composed screen around 18 KB instead of the megabytes
you get by concatenating raw SGR prefixes forward, cell after cell.
"""

import re
import unicodedata

SGR_RE = re.compile(
    r"\x1b\[[0-9;:?]*[a-zA-Z]|\x1b][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\-_]")

EMPTY = ((), None, None)  # (attrs, fg params, bg params)
BLANK = (EMPTY, " ")


def char_width(ch):
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1


def apply_sgr(state, params):
    """Fold one SGR parameter list into (attrs, fg, bg)."""
    attrs, fg, bg = list(state[0]), state[1], state[2]
    ps = params.split(";") if params else ["0"]
    i = 0
    while i < len(ps):
        p = ps[i] or "0"
        n = int(p)
        if n == 0:
            attrs, fg, bg = [], None, None
        elif n in (38, 48):
            if i + 1 < len(ps) and ps[i + 1] == "5":
                val = ";".join(ps[i:i + 3])
                i += 2
            elif i + 1 < len(ps) and ps[i + 1] == "2":
                val = ";".join(ps[i:i + 5])
                i += 4
            else:
                val = p
            if n == 38:
                fg = val
            else:
                bg = val
        elif n == 39:
            fg = None
        elif n == 49:
            bg = None
        elif (30 <= n <= 37) or (90 <= n <= 97):
            fg = p
        elif (40 <= n <= 47) or (100 <= n <= 107):
            bg = p
        elif n in (21, 22):
            attrs = [a for a in attrs if a not in ("1", "2")]
        elif n in (23, 24, 25, 27, 28, 29):
            attrs = [a for a in attrs if int(a) != n - 20]
        else:
            if p not in attrs:
                attrs.append(p)
        i += 1
    return (tuple(attrs), fg, bg)


def state_prefix(state):
    """Canonical, self-sufficient escape for a state: always names fg and bg."""
    attrs, fg, bg = state
    if not attrs and fg is None and bg is None:
        return ""
    return "\x1b[" + ";".join(list(attrs) + [fg or "39", bg or "49"]) + "m"


def parse_line(line):
    """One line -> [(state, char)]. Wide chars get a (state, None) filler."""
    cells = []
    state = EMPTY
    i = 0
    while i < len(line):
        m = SGR_RE.match(line, i)
        if m:
            seq = m.group(0)
            if seq.endswith("m"):
                state = apply_sgr(state, seq[2:-1])
            i = m.end()
            continue
        ch = line[i]
        i += 1
        if ch in ("\r", "\n"):
            continue
        w = char_width(ch)
        if w == 0:
            continue
        cells.append((state, ch))
        if w == 2:
            cells.append((state, None))
    return cells


def parse_block(text):
    if isinstance(text, list):
        text = "\n".join(text)
    return [parse_line(ln) for ln in text.split("\n")]


def emit_row(row):
    cur = EMPTY
    buf = []
    for state, ch in row:
        if ch is None:
            continue
        if state != cur:
            # A reset is needed to turn attributes off, and to return to the
            # terminal's own colours - state_prefix is empty for EMPTY, so
            # without this a cleared cell keeps painting with whatever the cell
            # before it set, and the disc leaks a background outside its own
            # edge. Every other transition is one sequence, because the
            # canonical prefix restates both colours.
            if cur != EMPTY and (state == EMPTY or set(cur[0]) - set(state[0])):
                buf.append("\x1b[0m")
            buf.append(state_prefix(state))
            cur = state
        buf.append(ch)
    line = "".join(buf)
    if cur != EMPTY:
        line += "\x1b[0m"
    return line


def emit(grid):
    out = []
    for row in grid:
        end = len(row)
        while end and row[end - 1] == BLANK:
            end -= 1
        out.append(emit_row(row[:end]))
    return "\n".join(out)


def visible_width(line):
    return len(parse_line(line))
