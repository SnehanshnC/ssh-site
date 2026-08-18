"""Master -> a COLSxROWS disc-clipped portrait, as a checked-in `.ans` asset.

Ported from the prototype's `strand-5-sharpen/render.sh`. The downsample is
ours, not chafa's: each glyph mode has a different number of pixels per cell
(vhalf 1x2, quad 2x2, sextant 2x3), so the master is resized to exactly that
subcell grid, sharpened *at* that resolution, then point-upscaled by an integer
factor so chafa's own scaler cannot blur anything back. chafa is left with two
decisions - which glyph, and which two colours - and every pixel decision is
ours.

Sharpening at the output grid is the single change that most decides "is this
him"; sharpening the 960px master does nothing because the downsample averages
it away.
"""

import os
import subprocess

from . import disc, master

MODES = {
    #          subcell x, subcell y, chafa symbol set
    "vhalf":   (1, 2, "vhalf+space+solid"),
    "quad":    (2, 2, "quad+space+solid"),
    "sextant": (2, 3, "sextant+space+solid"),
}

# Sharpen strength scales down as the grid grows: at 40 cells the downsample is
# ~24:1 and needs a lot of help; at 48 with quad glyphs it is ~10:1 and the same
# unsharp rings.
UNSHARP = ((1000, "0x0.6+0.9+0"), (2600, "0x0.6+0.8+0"), (5200, "0x0.7+0.65+0"))
UNSHARP_LARGE = "0x0.8+0.5+0"


def _unsharp(px):
    for limit, spec in UNSHARP:
        if px <= limit:
            return spec
    return UNSHARP_LARGE


def render(src, cols, rows, mode, work):
    """The master at `src`, as a raw COLSxROWS chafa capture."""
    sub_x, sub_y, symbols = MODES[mode]
    sw, sh = cols * sub_x, rows * sub_y
    grid = os.path.join(work, f"grid-{cols}x{rows}-{mode}.png")
    subprocess.run(["magick", src,
                    "-resize", f"{sw}x{sh}!", "-unsharp", _unsharp(sw * sh),
                    # point-upscale to the exact pixel size that maps onto
                    # cols x rows cells at chafa's default 1/2 font ratio
                    "-filter", "point", "-resize", f"{cols * 24}x{rows * 48}!",
                    grid], check=True)
    return subprocess.run(
        ["chafa", "-f", "symbols", "-c", "full", "--symbols", symbols,
         "-w", "9", "--size", f"{cols}x{rows}", "--polite", "on",
         "--relative", "off", grid],
        capture_output=True, check=True).stdout.decode("utf-8")


def build(src, cols, rows, mode, work, touchups=True):
    """Master -> disc-clipped, ringed portrait lines, ready to write as an asset.

    The disc-edge softening is per size, because its width is quoted in cells
    and a cell is `960 / cols` master pixels wide.
    """
    if touchups:
        softened = os.path.join(work, f"soft-{cols}.png")
        master.soften_disc_edge(src, softened, cols)
        src = softened
    return disc.clip(render(src, cols, rows, mode, work), cols, rows)
