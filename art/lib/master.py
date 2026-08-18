"""Headshot -> the 960x960 disc-centred master the portrait renders from.

Ported from the prototype's `variants/lib/facesrc-r4.py`, which is itself
`strand-5-sharpen/build-src.sh` re-run at a disc-centred crop. Every number in
here was measured during ticket 04 and is carried over unchanged; see
`art/README.md` for what each stage is for.

Three of the four cosmetic touch-ups ticket 04 deferred into the build live
here, because all three are artifacts of the photograph rather than of the
composition:

* the temple matte wedge - see `WEDGE_BOX` and `_make_matte`
* the earring reading as one pale cell - see `_retouch_earring`
* the collar step at the disc boundary - see `soften_disc_edge`

The fourth, the wordmark's row-gap lightness, is in `banner.py`.

Two ImageMagick 7 traps are load-bearing in this file and are easy to
reintroduce by "simplifying" it:

* `magick bg subj matte -compose over -composite` silently ignores the third
  image, handing back the untouched photo. The matte has to be copied into
  alpha with `-compose CopyOpacity` first.
* `-compose Minus` on masks computes an absolute difference, so subtracting one
  mask from another *exposes* what it was meant to remove.
"""

import math
import os
import subprocess
import sys
from collections import deque

BG = "#0a0c11"          # flat background fill, a touch darker than lifted hair
WK = 960                # working master resolution

# The shipped disc, picked in ticket 04 round 4 by rendering candidates at
# 36x18 and looking at them: about one cell of dark gap between the head and
# the ring at every angle, and only ~4% coarser than strand 5's rectangle.
DISC_CX, DISC_CY, DISC_R = 3460, 1860, 1350
KEEP_RATIO = 0.88       # fraction of the frame `disc.clip` keeps
KEEP_RATIO_MARGIN = 0.06  # `disc.CLIP_MARGIN_PER_COL`: the clip keeps radius 0.5 - this

# strand-5's crop, and its polygon's coordinate space (480 units across it)
S5_CROP = (2940, 2940, 2370, 320)

# strand-5 tools/make-matte.py POLY, verbatim, in strand-5's 480-space.
S5_POLY = [
    (66, 0), (58, 25), (45, 45), (35, 65), (38, 85), (48, 105),
    (40, 118), (33, 132), (30, 148), (24, 162), (20, 178),
    (22, 200), (28, 222), (28, 236),
    (36, 248), (48, 258), (52, 272), (56, 286),
    (60, 300), (60, 316), (63, 330),
    (65, 348), (67, 364), (70, 380), (76, 392),
    (80, 406), (80, 418), (68, 428), (52, 442), (40, 454),
    (30, 468), (24, 480),
    (480, 480),
    (480, 436), (455, 426), (424, 414),
    (408, 400), (400, 382), (396, 364), (398, 344), (400, 320),
    (415, 304), (440, 290), (458, 278), (470, 268), (480, 258),
    (480, 0),
]

# The hair's upper-left boundary above strand-5's crop top, in ORIGINAL image
# coordinates: scanned at 1500px wide, first x per row where luminance drops
# below 45 and stays under 70 for 5 samples, then pulled ~10px into the hair so
# the matte errs on cutting a wisp rather than admitting a bright background
# pixel (a bright pixel inside the matte becomes a halo after CLAHE).
HAIR_TOP = [
    (2774, 320),      # joins S5_POLY[0] exactly
    (2800, 270),
    (2960, 240),
    (3080, 200),
    (3160, 160),
    (3250, 110),
    (3330, 55),
    (3400, 0),
]


def _s5_to_orig(x, y):
    w, h, x0, y0 = S5_CROP
    return (x0 + x * w / 480.0, y0 + y * h / 480.0)


def _build_poly(crop):
    """strand-5's polygon + the hair-top splice, in the new crop's 480-space."""
    side, x0, y0 = crop
    k = 480.0 / side

    def to_new(ox, oy):
        return ((ox - x0) * k, (oy - y0) * k)

    pts = []
    # 1. the hair-top trace, right-to-left along the top, then down to the join
    top_y = to_new(0, 0)[1]                       # the photo's own top edge
    right_x = to_new(x0 + side + 40, 0)[0]        # just past the right frame edge
    pts.append((right_x, top_y))
    for ox, oy in reversed(HAIR_TOP):
        pts.append(to_new(ox, oy))
    # 2. strand-5's traced profile, bottom and right, transformed
    for x, y in S5_POLY[1:-1]:                    # drop the (66,0) dupe and (480,0)
        pts.append(to_new(*_s5_to_orig(x, y)))
    # 3. close along the right, beyond the frame, back up to the top
    pts.append((right_x, to_new(*_s5_to_orig(480, 258))[1]))
    return pts


def _raster_poly(pts, n):
    inside = bytearray(n * n)
    s = n / 480.0
    p = [(x * s, y * s) for x, y in pts]
    edges = [(p[i], p[(i + 1) % len(p)]) for i in range(len(p))]
    for y in range(n):
        yc = y + 0.5
        xs = []
        for (x0, y0), (x1, y1) in edges:
            if (y0 <= yc < y1) or (y1 <= yc < y0):
                xs.append(x0 + (yc - y0) * (x1 - x0) / (y1 - y0))
        xs.sort()
        row = y * n
        for k in range(0, len(xs) - 1, 2):
            for x in range(max(0, int(xs[k] + 0.5)), min(n, int(xs[k + 1] + 0.5))):
                inside[row + x] = 1
    return inside


def _read_rgb(path, n):
    return subprocess.run(["magick", path, "-depth", "8", "rgb:-"],
                          capture_output=True, check=True).stdout


def _crop_canvas(headshot, crop, n, out):
    """The crop square, padded with BG where it leaves the photograph."""
    side, x0, y0 = crop
    subprocess.run(["magick", "-size", f"{side}x{side}", f"xc:{BG}",
                    headshot, "-geometry", f"{-x0:+d}{-y0:+d}", "-composite",
                    "-resize", f"{n}x{n}!", out], check=True)


def _make_matte(crop, n, base, out, touchups=True):
    """The traced silhouette plus a chromatic border flood fill.

    Classifying background by hue alone is not safe - the same test fires on
    moles and on dark warm hair - so only classified pixels *reachable from the
    frame border* through other killed pixels are removed. Background always
    touches the border; a mole in the middle of a cheek never does.
    """
    inside = _raster_poly(_build_poly(crop), n)
    raw = _read_rgb(base, n)

    kill = bytearray(n * n)
    flag = bytearray(n * n)
    for i in range(n * n):
        j = i * 3
        r, g, b = raw[j], raw[j + 1], raw[j + 2]
        if (r > 40 and r > g * 1.30 and b >= g) or (min(r, g, b) > 115 and b >= r * 0.95):
            flag[i] = 1
    if touchups:
        for i in _wedge_flags(raw, n):
            flag[i] = 1

    q = deque()
    for x in range(n):
        for i in (x, (n - 1) * n + x, x * n, x * n + n - 1):
            if (not inside[i] or flag[i]) and not kill[i]:
                kill[i] = 1
                q.append(i)
    while q:
        i = q.popleft()
        y, x = divmod(i, n)
        for ny, nx in ((y - 1, x), (y + 1, x), (y, x - 1), (y, x + 1)):
            if 0 <= ny < n and 0 <= nx < n:
                k = ny * n + nx
                if not kill[k] and (not inside[k] or flag[k]):
                    kill[k] = 1
                    q.append(k)

    for _ in range(max(2, n // 200)):
        add = []
        for i in range(n * n):
            if kill[i]:
                continue
            y, x = divmod(i, n)
            for ny, nx in ((y - 1, x), (y + 1, x), (y, x - 1), (y, x + 1)):
                if 0 <= ny < n and 0 <= nx < n and kill[ny * n + nx] and flag[ny * n + nx]:
                    add.append(i)
                    break
        for i in add:
            kill[i] = 1

    m = bytes(0 if kill[i] else 255 for i in range(n * n))
    subprocess.run(["magick", "pgm:-", "-colorspace", "gray", "-alpha", "off", out],
                   input=b"P5\n%d %d\n255\n" % (n, n) + m, check=True)
    print(f"  matte: {100 * (1 - sum(kill) / (n * n)):.1f}% subject, "
          f"{sum(1 for i in range(n * n) if kill[i] and inside[i])} px reclaimed by hue",
          file=sys.stderr)


# ---------------------------------------------------------------- touch-ups

# Ticket 04 rough edge: "a dark violet-grey wedge sits between the temple and
# the hairline, inside the disc, on every circled capture ... the traced polygon
# runs a little wide of the profile there and admits a slice of the blurred
# background, which the flat fill then darkens instead of removing."
#
# The wedge really is background - a bright grey blur pokes through the hair
# fringe there - so the cure is to remove it, not to keep it. It survives the
# existing chromatic cleanup because that classifier only fires on the maroon
# awning and on bright cool pixels, and the wedge is neither: sampled off
# base.png it runs (52,49,60)..(62,58,73), a *dark* cool grey. Hair beside it is
# (1,1,1)..(11,13,25) and the forehead is warm, so a third classifier separates
# it cleanly:
#
#     b >= r + 6 and b >= g + 6 and 30 <= max(r,g,b) <= 110
#
# That test is not safe frame-wide - the navy tee passes it - so it is fenced to
# the wedge's own box, measured by mapping the classifier over the crop (the
# blob runs x 100..280, y 120..380 in 960-space and touches the killed region on
# its left, which is what lets the existing border flood fill consume it).
# Coordinates are in the crop's 480-space, like every other polygon here.
WEDGE_BOX = (48, 52, 160, 200)      # x0, y0, x1, y1

# Ticket 04 rough edge: "the hoop earring is a single pale cell at the lobe, not
# a resolvable hoop ... the one anchor that a reader who did not know it was
# there would not name."
#
# At 36x18 the hoop is about 1.2 cells wide and 1 cell tall, so no amount of
# preprocessing resolves it as a ring - the medium cannot carry it. What it can
# carry is a bright subcell against a dark one, which quad glyphs render as a
# two-tone cell with an edge in it rather than one washed-out pale cell. So the
# retouch works at the subcell scale, not the pixel scale: lift the metal, sink
# the skin immediately around it, and let the downsample keep the step between
# them. Anything finer is averaged away by the 13x27 master pixels behind one
# subcell, which is why strand 5's "sharpen at the output grid" rule applies to
# retouching too.
#
# Both adjustments scale all three channels by the same factor, so they move
# luminance and leave hue alone. An S-curve per channel (`-sigmoidal-contrast`)
# does not: it drove the warm lobe to saturated orange.
#
# Geometry measured off the master: the hoop occupies x 705..737, y 528..583 in
# 960-space. Quoted here in the crop's 480-space, like every other polygon.
EARRING = (360.5, 277.5)        # centre
EARRING_METAL = (9.0, 15.0)     # radii of the lift, about the hoop itself
EARRING_SURROUND = (17.0, 25.0) # radii of the sink, about one subcell beyond
EARRING_LIFT = 1.30
EARRING_SINK = 0.68


def _mask_from_bits(bits, n, out):
    subprocess.run(["magick", "pgm:-", "-colorspace", "gray", "-alpha", "off", out],
                   input=b"P5\n%d %d\n255\n" % (n, n) + bytes(bits), check=True)


def _apply_through_mask(base, variant, mask, out):
    """Composite `variant` over `base` through `mask`.

    Deliberately not the three-image `magick base variant mask -composite`
    form: on ImageMagick 7 that silently ignores the mask and hands back the
    variant. The mask has to be copied into the variant's alpha first.
    """
    tmp = out + ".masked.png"
    subprocess.run(["magick", variant, mask, "-alpha", "off",
                    "-compose", "CopyOpacity", "-composite", tmp], check=True)
    subprocess.run(["magick", base, tmp, "-compose", "over", "-composite",
                    "-alpha", "off", out], check=True)
    os.remove(tmp)


def _wedge_flags(raw, n):
    """The temple-wedge classifier, fenced to WEDGE_BOX."""
    x0, y0, x1, y1 = (int(round(v * n / 480.0)) for v in WEDGE_BOX)
    flags = []
    for y in range(y0, min(y1, n)):
        for x in range(x0, min(x1, n)):
            j = (y * n + x) * 3
            r, g, b = raw[j], raw[j + 1], raw[j + 2]
            if b >= r + 6 and b >= g + 6 and 30 <= max(r, g, b) <= 110:
                flags.append(y * n + x)
    return flags


def _ellipse_mask(n, cx, cy, rx, ry, feather=0.25, inner=None):
    """A soft-edged ellipse (optionally an annulus), as 0..255 mask bytes."""
    cx, cy, rx, ry = (v * n / 480.0 for v in (cx, cy, rx, ry))
    bits = bytearray(n * n)
    for y in range(max(0, int(cy - ry) - 2), min(n, int(cy + ry) + 3)):
        for x in range(max(0, int(cx - rx) - 2), min(n, int(cx + rx) + 3)):
            d = math.sqrt(((x + 0.5 - cx) / rx) ** 2 + ((y + 0.5 - cy) / ry) ** 2)
            if d >= 1.0:
                continue
            v = min(1.0, (1.0 - d) / feather)
            if inner is not None:
                irx, iry = (r * n / 480.0 for r in inner)
                di = math.sqrt(((x + 0.5 - cx) / irx) ** 2 + ((y + 0.5 - cy) / iry) ** 2)
                v = min(v, min(1.0, max(0.0, di - 1.0) / feather))
            bits[y * n + x] = int(round(255 * v))
    return bits


def _retouch_earring(src, n, out):
    cx, cy = EARRING
    steps = [
        (_ellipse_mask(n, cx, cy, *EARRING_SURROUND, inner=EARRING_METAL), EARRING_SINK),
        (_ellipse_mask(n, cx, cy, *EARRING_METAL), EARRING_LIFT),
    ]
    cur = src
    for i, (bits, factor) in enumerate(steps):
        mask = f"{out}.ear{i}.pgm"
        layer = f"{out}.ear{i}.png"
        dst = out if i == len(steps) - 1 else f"{out}.ear{i}-out.png"
        _mask_from_bits(bits, n, mask)
        subprocess.run(["magick", cur, "-evaluate", "multiply", str(factor), layer],
                       check=True)
        _apply_through_mask(cur, layer, mask, dst)
        for f in (mask, layer):
            os.remove(f)
        if cur is not src:
            os.remove(cur)
        cur = dst


def soften_disc_edge(src, out, cols, cells=1.5):
    """Blend the outermost `cells` of the disc toward the flat fill.

    Ticket 04 rough edge: "the tee clips into a chunky step where the collar
    meets the disc boundary". The clip in `disc.py` is per *cell*, so where the
    disc boundary crosses a high-contrast edge every kept cell is a solid block
    of tee blue and every dropped one is empty, and the boundary reads as a
    staircase instead of a curve.

    Ramping the last cell and a half toward the background fill fixes it at the
    only place it can be fixed - before the downsample - and it is self-limiting:
    blending toward the fill is invisible wherever the picture is already near
    the fill, which is the whole top and right of the disc where the hair is.
    The collar and the neck, the two places the disc actually cuts something
    bright, are exactly the places it does anything.
    """
    n = subprocess.run(["magick", src, "-format", "%w", "info:"],
                       capture_output=True, check=True).stdout.decode()
    n = int(n)
    r_out = (0.5 - KEEP_RATIO_MARGIN) * n
    r_in = r_out - cells * (n / cols)
    bits = bytearray(n * n)
    cx = cy = n / 2.0
    for y in range(n):
        dy = y + 0.5 - cy
        for x in range(n):
            dx = x + 0.5 - cx
            d = math.hypot(dx, dy)
            if d <= r_in:
                v = 255
            elif d >= r_out:
                v = 0
            else:
                t = (r_out - d) / (r_out - r_in)
                v = int(round(255 * t * t * (3 - 2 * t)))     # smoothstep
            bits[y * n + x] = v
    mask = out + ".discmask.pgm"
    _mask_from_bits(bits, n, mask)
    plate = out + ".plate.png"
    subprocess.run(["magick", "-size", f"{n}x{n}", f"xc:{BG}", plate], check=True)
    _apply_through_mask(plate, src, mask, out)
    for f in (mask, plate):
        os.remove(f)



def build(headshot, out, work, touchups=True):
    """headshot.jpg -> the toned, matted, retouched 960x960 disc master."""
    side = int(round(2 * DISC_R / KEEP_RATIO))
    crop = (side, DISC_CX - side // 2, DISC_CY - side // 2)
    print(f"  crop {side}x{side}+{crop[1]}+{crop[2]}  disc r={DISC_R} "
          f"at ({DISC_CX},{DISC_CY})", file=sys.stderr)

    base = os.path.join(work, "base.png")
    matte = os.path.join(work, "matte.png")
    _crop_canvas(headshot, crop, WK, base)
    _make_matte(crop, WK, base, matte, touchups)

    toned = os.path.join(work, "toned.png")
    subprocess.run(["magick", base,
                    "-modulate", "104,108,100",
                    "-clahe", "25x25%+128+1.2",
                    "-sigmoidal-contrast", "2,45%",
                    "+level", "7%,100%", toned], check=True)

    plate = os.path.join(work, "bg.png")
    subj = os.path.join(work, "subj.png")
    matted = os.path.join(work, "matted.png")
    subprocess.run(["magick", "-size", f"{WK}x{WK}", f"xc:{BG}", plate], check=True)
    subprocess.run(["magick", toned, matte, "-alpha", "off",
                    "-compose", "CopyOpacity", "-composite", subj], check=True)
    subprocess.run(["magick", plate, subj, "-compose", "over", "-composite",
                    "-alpha", "off", matted], check=True)

    if touchups:
        _retouch_earring(matted, WK, out)
    else:
        subprocess.run(["magick", matted, out], check=True)
    print(f"  wrote {out}", file=sys.stderr)
    return out
