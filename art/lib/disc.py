"""Clip a rendered face to a disc and draw a Braille-dot ring around it.

Ported from the prototype's `strand-2-circle/clip-to-disc.py`, reduced to the
one ring style ticket 04 settled on (Braille, 2x4 subcells per cell) and the
one clip it uses.

Terminal cells are about 2:1 (tall:wide), so a visually round circle of N
columns needs about N/2 rows; every ellipse here is scaled accordingly.

The ring is emitted as a foreground-only cell state. That is safe *because*
`ansigrid.emit` writes a canonical prefix that always names a background too -
`clip-to-disc.py` on its own writes a bare foreground sequence, and the
right-hand arc then inherits the background of whichever face cell the terminal
painted before it.
"""

import math

from . import ansigrid

# Braille bit weights, U+2800 + bits, as a 4-row x 2-column subcell grid.
BRAILLE_BITS = [[0x01, 0x08],
                [0x02, 0x10],
                [0x04, 0x20],
                [0x40, 0x80]]

RING_SGR = "38;2;148;163;184"

# `clip_margin` is 0.06 * cols, not a flat constant: that is the value that
# keeps the kept disc at 0.44 of the frame width at EVERY diameter, which is
# the ratio the master is cropped for.
CLIP_MARGIN_PER_COL = 0.06
RING_INSET = 0.5


def clip_margin(cols):
    return CLIP_MARGIN_PER_COL * cols


def kept_radius_fraction():
    """Fraction of the frame width the disc clip keeps, as a radius."""
    return 0.5 - CLIP_MARGIN_PER_COL


def _fit_grid(grid, cols, rows):
    """Centre-crop / centre-pad the parsed grid into exactly cols x rows."""
    src_rows = len(grid)
    r0 = (src_rows - rows) // 2
    out = []
    for r in range(rows):
        sr = r0 + r
        src = grid[sr] if 0 <= sr < src_rows else []
        c0 = (len(src) - cols) // 2
        line = []
        for c in range(cols):
            sc = c0 + c
            line.append(src[sc] if 0 <= sc < len(src) else ansigrid.BLANK)
        out.append(line)
    return out


def _inside(cols, rows, r, c, margin):
    """True if cell (r,c) sits inside the disc, shrunk by `margin` cells."""
    rx = cols / 2.0 - margin
    ry = rows / 2.0 - margin / 2.0
    if rx <= 0 or ry <= 0:
        return False
    x = (c + 0.5 - cols / 2.0) / rx
    y = (r + 0.5 - rows / 2.0) / ry
    return x * x + y * y <= 1.0


def _ring_points(cols, rows, inset, sub_x, sub_y):
    """Rasterise the ring ellipse onto a (cols*sub_x) x (rows*sub_y) grid.

    Scans both by row and by column and unions the hits, so the curve stays
    8-connected instead of breaking apart where it runs near-horizontal. The
    far side is mirrored rather than rounded independently: rounding both ends
    separately makes half-integer boundaries land asymmetrically and the left
    and right of the circle stop matching.
    """
    w, h = cols * sub_x, rows * sub_y
    cx, cy = w / 2.0, h / 2.0
    rx = (cols / 2.0 - inset) * sub_x
    ry = (rows / 2.0 - inset / 2.0) * sub_y
    if rx <= 0 or ry <= 0:
        return set()

    hits = set()

    def add(sy, sx):
        if 0 <= sy < h and 0 <= sx < w:
            hits.add((sy, sx))

    for sy in range(h):
        y = (sy + 0.5 - cy) / ry
        if abs(y) >= 1.0:
            continue
        dx = math.sqrt(1.0 - y * y) * rx
        left = int(round(cx - dx - 0.5))
        add(sy, left)
        add(sy, (w - 1) - left)

    for sx in range(w):
        x = (sx + 0.5 - cx) / rx
        if abs(x) >= 1.0:
            continue
        dy = math.sqrt(1.0 - x * x) * ry
        top = int(round(cy - dy - 0.5))
        add(top, sx)
        add((h - 1) - top, sx)

    return hits


def braille_ring(cols, rows, inset=RING_INSET):
    acc = {}
    for (py, px) in _ring_points(cols, rows, inset, 2, 4):
        key = (py // 4, px // 2)
        acc[key] = acc.get(key, 0) | BRAILLE_BITS[py % 4][px % 2]
    return {key: chr(0x2800 + bits) for key, bits in acc.items()}


def clip(text, cols, rows, ring_sgr=RING_SGR):
    """Face render -> disc-clipped face with a Braille ring, as a list of lines."""
    grid = _fit_grid(ansigrid.parse_block(text), cols, rows)
    margin = clip_margin(cols)
    for r in range(rows):
        for c in range(cols):
            if not _inside(cols, rows, r, c, margin):
                grid[r][c] = ansigrid.BLANK

    state = ansigrid.apply_sgr(ansigrid.EMPTY, ring_sgr)
    for (r, c), ch in braille_ring(cols, rows).items():
        if 0 <= r < rows and 0 <= c < cols and ch != " ":
            grid[r][c] = (state, ch)

    return ansigrid.emit(grid).split("\n")
