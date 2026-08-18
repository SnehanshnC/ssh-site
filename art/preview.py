#!/usr/bin/env python3
"""Rasterise a checked-in `.ans` asset to a PNG, the way a terminal draws it.

    python3 art/preview.py internal/art/assets/portrait-wide-sextant.ans out.png

A developer tool for looking at the render ladder, and the only way to look at
the top of it. Sextant glyphs live in Unicode 13's Symbols for Legacy Computing
block and **no local font carries them** - kitty, foot, Ghostty and WezTerm draw
them from their own built-in geometry, which is exactly why they are the tier
those four terminals get and nobody else does. So `cat`ting a sextant asset into
an ordinary terminal shows a wall of tofu whether the asset is right or wrong.

This paints each cell the way those terminals do: fill the cell with its
background colour, then fill the subcells the glyph names with its foreground.
Block glyphs are defined as exact rectangles, so the result is not an
approximation of the terminal's output - for the block and sextant glyphs it is
the same picture, at whatever cell size you ask for.

Braille is the one glyph class drawn by analogy rather than by definition: the
ring's dots are round in a font and square here. The ring's shape and colour are
what a review of it is about, and both survive.
"""

import argparse
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from lib import ansigrid   # noqa: E402

# Cell size in pixels. 12x24 is the smallest cell that divides evenly by every
# subcell grid on the ladder - sextant 2x3, quad 2x2, vhalf 1x2, braille 2x4 -
# so no glyph has to be rounded onto the pixel grid.
CELL_W, CELL_H = 12, 24

# Sextant subcell positions, as (row, col) on a 2-wide, 3-tall grid, in the
# order Unicode numbers them.
SEXTANT_CELLS = [(0, 0), (0, 1), (1, 0), (1, 1), (2, 0), (2, 1)]

# The 2x2 quadrant glyphs, as (row, col) sets on a 2x2 grid.
UL, UR, LL, LR = (0, 0), (0, 1), (1, 0), (1, 1)
QUADRANTS = {
    "▖": [LL], "▗": [LR], "▘": [UL], "▙": [UL, LL, LR],
    "▚": [UL, LR], "▛": [UL, UR, LL], "▜": [UL, UR, LR],
    "▝": [UR], "▞": [UR, LL], "▟": [UR, LL, LR],
    "▀": [UL, UR], "▄": [LL, LR],
    "▌": [UL, LL], "▐": [UR, LR],
}

# Braille dot bit weights, as (row, col) on the 2-wide, 4-tall dot grid.
BRAILLE_DOTS = {0x01: (0, 0), 0x02: (1, 0), 0x04: (2, 0), 0x40: (3, 0),
                0x08: (0, 1), 0x10: (1, 1), 0x20: (2, 1), 0x80: (3, 1)}


def sextant_map():
    """Codepoint -> lit subcell list, for U+1FB00..U+1FB3B.

    The block encodes all 64 patterns except the four that already had
    characters: empty is a space, left and right columns are the vertical half
    blocks, and full is a full block. So the code points run in pattern order
    with those four skipped.
    """
    out = {}
    cp = 0x1FB00
    for bits in range(1, 63):
        if bits in (0b010101, 0b101010):     # left column, right column
            continue
        out[chr(cp)] = [SEXTANT_CELLS[i] for i in range(6) if bits >> i & 1]
        cp += 1
    return out


SEXTANTS = sextant_map()


def sgr_rgb(param, fallback):
    """An SGR colour parameter list as (r, g, b).

    Only truecolor and the eight basic colours appear in these assets; anything
    else falls back rather than guessing, because a preview that invents a
    colour is worse than one that admits it did not know.
    """
    if not param:
        return fallback
    parts = param.split(";")
    if len(parts) == 5 and parts[1] == "2":
        return tuple(int(p) for p in parts[2:5])
    if len(parts) == 1 and parts[0].isdigit():
        basic = [(0, 0, 0), (205, 49, 49), (13, 188, 121), (229, 229, 16),
                 (36, 114, 200), (188, 63, 188), (17, 168, 205), (229, 229, 229)]
        n = int(parts[0])
        if 30 <= n <= 37:
            return basic[n - 30]
        if 40 <= n <= 47:
            return basic[n - 40]
    return fallback


def paint(pix, width, x0, y0, w, h, rgb):
    for y in range(y0, y0 + h):
        row = (y * width + x0) * 3
        for i in range(w):
            pix[row + i * 3:row + i * 3 + 3] = bytes(rgb)


def render(lines, fg_default, bg_default):
    """The parsed asset as (width, height, RGB bytes)."""
    grid = ansigrid.parse_block("\n".join(lines))
    rows = len(grid)
    cols = max(len(r) for r in grid)
    width, height = cols * CELL_W, rows * CELL_H
    pix = bytearray(bytes(bg_default) * (width * height))

    for r, row in enumerate(grid):
        for c, (state, ch) in enumerate(row):
            x0, y0 = c * CELL_W, r * CELL_H
            fg = sgr_rgb(state[1], fg_default)
            bg = sgr_rgb(state[2], bg_default)
            paint(pix, width, x0, y0, CELL_W, CELL_H, bg)
            if ch == " ":
                continue
            if ch == "█":
                paint(pix, width, x0, y0, CELL_W, CELL_H, fg)
            elif ch in QUADRANTS:
                for sr, sc in QUADRANTS[ch]:
                    paint(pix, width, x0 + sc * CELL_W // 2,
                          y0 + sr * CELL_H // 2, CELL_W // 2, CELL_H // 2, fg)
            elif ch in SEXTANTS:
                for sr, sc in SEXTANTS[ch]:
                    paint(pix, width, x0 + sc * CELL_W // 2,
                          y0 + sr * CELL_H // 3, CELL_W // 2, CELL_H // 3, fg)
            elif "⠀" <= ch <= "⣿":
                bits = ord(ch) - 0x2800
                for weight, (sr, sc) in BRAILLE_DOTS.items():
                    if bits & weight:
                        paint(pix, width, x0 + sc * CELL_W // 2 + 1,
                              y0 + sr * CELL_H // 4 + 1,
                              CELL_W // 2 - 2, CELL_H // 4 - 2, fg)
            else:
                # Text - the line-art tier, and any glyph this tool does not
                # model. Drawn as a bar so its ink shows in the silhouette.
                paint(pix, width, x0 + 2, y0 + 4, CELL_W - 4, CELL_H - 8, fg)
    return width, height, bytes(pix)


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("asset")
    ap.add_argument("out")
    ap.add_argument("--scale", type=int, default=2)
    ap.add_argument("--background", default="#0f172a",
                    help="what the visitor's own terminal paints behind the "
                         "asset; every cell render should be proof against it")
    a = ap.parse_args()

    bg = a.background.lstrip("#")
    bg_default = tuple(int(bg[i:i + 2], 16) for i in (0, 2, 4))

    with open(a.asset, encoding="utf-8") as fh:
        lines = fh.read().rstrip("\n").split("\n")
    width, height, pix = render(lines, (226, 232, 240), bg_default)

    subprocess.run(["magick", "ppm:-", "-filter", "point",
                    "-resize", f"{a.scale * 100}%", a.out],
                   input=b"P6\n%d %d\n255\n" % (width, height) + pix, check=True)
    print(f"{a.out}  {width * a.scale}x{height * a.scale} "
          f"({len(lines)} rows of {re.sub(r'.*/', '', a.asset)})", file=sys.stderr)


if __name__ == "__main__":
    main()
