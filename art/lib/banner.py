"""The `smslant` SNEHANSHN banner: 46x4, cyan-violet horizontal gradient.

Ported from the prototype's `strand-3-banner/lib/gradient.py` plus
`variants/lib/build-final.py::figlet_banner`, with one deliberate change - the
row-gap touch-up, below.

Row-gap touch-up
----------------
Ticket 04 signed the banner off but flagged it as the softest element on the
screen and deferred the fix here. Slant figlet fonts are drawn from `/`, `\\`,
`|` and `_`, and in a real monospace cell those strokes do not meet across a row
boundary: `_` sits just under its own baseline and the `/` on the row below
starts near its own cap height, so every pair of banner rows has a visible gap
and the wordmark reads lighter than the face and the copy beside it.

The prototype's suggested cure was strand 3's half-block hero, which is 72x6 and
costs two rows the 24-row screen does not have. This does it inside the same
46x4 by swapping each stroke for the box/block glyph that occupies the *whole*
cell in that direction, so the strokes tile edge to edge and the letterforms
close up:

    /  ->  U+2571 BOX DRAWINGS LIGHT DIAGONAL UPPER RIGHT TO LOWER LEFT
    \\  ->  U+2572 BOX DRAWINGS LIGHT DIAGONAL UPPER LEFT TO LOWER RIGHT
    |  ->  U+2502 BOX DRAWINGS LIGHT VERTICAL
    _  ->  U+2581 LOWER ONE EIGHTH BLOCK

Coverage is no worse than what the card already requires: all four are in Menlo
and SF Mono (checked against their cmaps), terminals that draw box characters
themselves (kitty, Ghostty, WezTerm, foot) rasterise them corner to corner, and
the disc's Braille ring already depends on system font fallback.

The gradient is untouched: `t` still runs across the 46-column frame, so the
banner's colours are the ones ticket 04 signed off, cell for cell.
"""

import subprocess

# strand-3-banner/lib/gradient.py, PALETTES["cyan-violet"]: teal -> indigo -> violet
CYAN_VIOLET = ["#22d3ee", "#6366f1", "#a855f7"]

# The row-gap touch-up. Keys are every non-space rune `smslant` emits.
TILING_GLYPHS = {"/": "╱", "\\": "╲", "|": "│", "_": "▁"}


def _hexrgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


def _ramp(stops, t):
    """t in [0,1] -> rgb tuple, piecewise-linear across N stops."""
    if len(stops) == 1:
        return stops[0]
    t = min(max(t, 0.0), 1.0)
    seg = t * (len(stops) - 1)
    i = min(int(seg), len(stops) - 2)
    f = seg - i
    a, b = stops[i], stops[i + 1]
    return tuple(round(a[k] + (b[k] - a[k]) * f) for k in range(3))


def paint(lines, stops, bold=True):
    """Per-column foreground ramp. Only non-space runes are painted, so the
    gradient tracks the letterforms rather than the padding."""
    stops = [_hexrgb(s) for s in stops]
    w = max((len(line) for line in lines), default=1)
    out = []
    pre = "\033[1m" if bold else ""
    for line in lines:
        buf = pre
        last = None
        for x, ch in enumerate(line):
            if ch == " ":
                buf += ch
                continue
            rgb = _ramp(stops, x / max(w - 1, 1))
            if rgb != last:
                buf += "\033[38;2;%d;%d;%dm" % rgb
                last = rgb
            buf += ch
        out.append(buf + "\033[0m")
    return out


def figlet_lines(text, font):
    out = subprocess.run(["figlet", "-f", font, text],
                         capture_output=True, check=True).stdout.decode("utf-8")
    lines = [line.rstrip("\n") for line in out.split("\n")]
    while lines and not lines[-1].strip():
        lines.pop()
    w = max(len(line) for line in lines)
    return [line.ljust(w) for line in lines]


def close_row_gaps(lines):
    return ["".join(TILING_GLYPHS.get(ch, ch) for ch in line) for line in lines]


def render(text="SNEHANSHN", font="smslant", tiling=True):
    lines = figlet_lines(text, font)
    if tiling:
        lines = close_row_gaps(lines)
    return paint(lines, CYAN_VIOLET)
