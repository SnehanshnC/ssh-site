#!/usr/bin/env python3
"""Regenerate the SSH site's checked-in terminal art.

    make art                       # from the default headshot path
    make art ART_HEADSHOT=/path/to/headshot.jpg

This is a developer tool, never a build or CI step: the art is a build-time
asset and the checked-in `.ans` files under `internal/art/assets/` are what the
Go build embeds. Run it when the photograph or the wordmark changes, look at
the result, and commit the assets.

It needs `figlet`, `chafa` and ImageMagick 7 on PATH, and a headshot. The
headshot is not in this repo - it is private source material, and the matte
geometry in `lib/master.py` is traced for that one photograph anyway.
"""

import argparse
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from lib import banner, lineart, master, portrait   # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
ASSETS = os.path.join(REPO, "internal", "art", "assets")

# The portrait matrix: one card layout by one render tier.
#
# The wide disc is the two-column card's left column; the narrow one is the
# restack below 71 columns, where height is the binding constraint and a circled
# face must be cols x cols/2 or it is an ellipse.
#
# The tiers are the render ladder, best to worst. The three cell tiers differ
# only in how many pixels one cell can carry - sextant 2x3, quad 2x2, vhalf 1x2
# - so all three come off the same master through the same pipeline, and each is
# sharpened at its own subcell grid. The colorless tier is not that picture
# without its colour: it is the hand-drawn line art, which is why it comes from
# `lib/lineart.py` and needs no photograph at all.
LAYOUTS = [
    ("wide", 36, 18),
    ("narrow", 32, 16),
]
CELL_TIERS = ["sextant", "quad", "vhalf"]

# The wordmark's text is a fact, so it comes from the content pack rather than
# from this file - `make content` has to have run. Only the given name goes in
# the banner: `smslant` renders SNEHANSHN at 46 columns and a full name would
# not fit the frame at any signed-off size.
PACK_IDENTITY = os.path.join(REPO, "internal", "content", "pack", "identity.yaml")


def wordmark():
    """The given name from the pack's identity section, uppercased.

    Read with a regex rather than a YAML parser so `make art` needs nothing
    installed beyond the image tools; `name` is a plain scalar at the top level
    of a file this repo fetches whole, so there is no ambiguity to resolve.
    """
    if not os.path.exists(PACK_IDENTITY):
        sys.exit(f"art: no content pack at {os.path.relpath(PACK_IDENTITY, REPO)}\n"
                 f"     run `make content` first - the wordmark is a fact, not a "
                 f"string in this repo")
    with open(PACK_IDENTITY, encoding="utf-8") as fh:
        match = re.search(r"^name:\s*(.+?)\s*$", fh.read(), re.MULTILINE)
    if not match:
        sys.exit(f"art: no `name` in {os.path.relpath(PACK_IDENTITY, REPO)}")
    return match.group(1).strip("\"'").split()[0].upper()


def write(name, lines):
    path = os.path.join(ASSETS, name + ".ans")
    with open(path, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines) + "\n")
    print(f"  wrote {os.path.relpath(path, REPO)} "
          f"({os.path.getsize(path)} bytes)", file=sys.stderr)


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--headshot", required=True)
    ap.add_argument("--work", default=os.path.join(HERE, "work"))
    ap.add_argument("--baseline", action="store_true",
                    help="skip the four cosmetic touch-ups, reproducing the "
                         "ticket-04 reference render as it was signed off")
    a = ap.parse_args()

    if not os.path.exists(a.headshot):
        sys.exit(f"art: no headshot at {a.headshot}\n"
                 f"     pass ART_HEADSHOT=/path/to/headshot.jpg")

    os.makedirs(a.work, exist_ok=True)
    os.makedirs(ASSETS, exist_ok=True)

    touchups = not a.baseline
    if a.baseline:
        print("baseline: the four cosmetic touch-ups are OFF", file=sys.stderr)

    text = wordmark()
    print(f"banner ({text}):", file=sys.stderr)
    write("banner", banner.render(text, tiling=touchups))
    # The colorless tier's wordmark. The row-gap touch-up closes the letterforms
    # by swapping figlet's strokes for the box and block glyphs that fill a whole
    # cell, and a terminal that reached the bottom rung is one nothing vouched
    # for - so it gets `smslant` as figlet drew it, in the same 46x4, out of the
    # `/ \ | _` that every terminal has drawn since figlet was written. The
    # gradient is painted on either way and stripped at the writer.
    write("banner-colorless", banner.render(text, tiling=False))

    print("master:", file=sys.stderr)
    src = master.build(a.headshot, os.path.join(a.work, "master.png"),
                       a.work, touchups)

    for layout, cols, rows in LAYOUTS:
        for tier in CELL_TIERS:
            print(f"portrait {layout} {tier} ({cols}x{rows}):", file=sys.stderr)
            write(f"portrait-{layout}-{tier}",
                  portrait.build(src, cols, rows, tier, a.work, touchups))
        print(f"portrait {layout} colorless ({cols}x{rows}):", file=sys.stderr)
        write(f"portrait-{layout}-colorless", lineart.build(cols, rows))


if __name__ == "__main__":
    main()
