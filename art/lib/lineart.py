"""The hand-drawn line-art portrait: the colorless tier's asset.

Ported from the ticket-04 prototype's `strand-4-lineart/draw.py`, candidate
`01-mass` - the one that strand's verdict ranked first, "the one to ship". It is
not a grayscale render of the photograph. Round 1 already proved that a
photographic conversion without colour reads as blurry pixels rather than as a
picture, which is why the line art was drawn at all and why it was kept as an
asset for exactly this tier.

Two things about the port are worth knowing before changing it.

**There is no photograph in this file, and that is not a shortcut.** The
prototype filled the hair mass algorithmically, from a CLAHE-equalised tone map
of the headshot, through a density ramp indexed by darkness. At the ramp the
shipped candidate uses - four blanks then six `M` - every cell of the hair
quantises above the blank band, so the fill emits `M` everywhere and the only
holes in the mass are the ones `SHEEN` punches by hand. Rebuilding `01-mass`
with the tone fill replaced by a solid fill reproduces the prototype's
committed output byte for byte, so the tone map, its three hand-traced mattes
and their ImageMagick pipeline are carried here as a fact about where the
highlight goes, not as code. `SHEEN` is that fact.

**The drawing keeps its own boundary; no ring is drawn round it.** The three
cell-render tiers clip to a disc because a photograph needs a frame to stop at.
Line art does not: the silhouette *is* the drawing, which is what strand 4
concluded, and shrinking the head to sit inside the ring would cost it the
detail it can least afford - at this size the nose is already one row and one
glyph. So the colorless tier is a bust in the same cell budget rather than a
disc, and it is the one tier whose portrait is pure ASCII, which also makes it
the safe floor for a terminal whose glyph coverage is as unknown as its colour.

Both sizes are hand-authored against the same landmark table, measured off the
prototype's gridded photograph. The narrow canvas is the wide one at 32/38 in
both axes - 0.842, the same factor either way, because 36x18 and 32x16 have the
same aspect - so its landmarks are the wide ones scaled, and every glyph is
then placed by hand against them.
"""

BS = "\\"  # one backslash, spelled out so the art tables below stay readable


class Art:
    """A fixed cell grid with hand-plotted segments and a solid mass fill."""

    def __init__(self, cols, rows):
        self.cols, self.rows = cols, rows
        self.g = [[" "] * cols for _ in range(rows)]

    def plot(self, rowlist, r0=0):
        """Plot (col, text) segments, one list per row.

        A `~` in the text leaves that cell alone, so segments can overlap
        safely and a later pass can restate an edge the fill ran over.
        """
        for i, segs in enumerate(rowlist):
            for col, text in segs:
                r = r0 + i
                for j, ch in enumerate(text):
                    c = col + j
                    if 0 <= r < self.rows and 0 <= c < self.cols and ch != "~":
                        self.g[r][c] = ch

    def fill(self, spans, ch="M"):
        """Fill every still-blank cell of each row's hair span.

        Solid, for the reason in the module docstring: the prototype's
        tone-driven ramp emits one glyph over this whole region anyway.
        """
        for r, (c0, c1) in spans.items():
            for c in range(c0, c1 + 1):
                if 0 <= r < self.rows and 0 <= c < self.cols and self.g[r][c] == " ":
                    self.g[r][c] = ch

    def lines(self):
        """The grid as exactly `rows` lines, each right-trimmed."""
        return ["".join(row).rstrip() for row in self.g]


# ===========================================================================
# Wide: 36x18, the two-column card's left column.
#
# The prototype drew this on a 38x19 canvas that trimmed to 36x19. The one
# change is the bottom: the jaw's three-row diagonal is redrawn over two so the
# crew-neck tee keeps both of its rows, because the tee's lower row is what
# gives the bust a base to sit on - dropping it leaves the shoulder line
# floating under the chin with nothing beneath it.
# ===========================================================================

# Outer silhouette: spiky crown, hair overhanging the brow, the mass dropping
# low behind the ear.
WIDE_SIL = [
    # 0         1         2         3
    # 012345678901234567890123456789012345
    [(12, ",^-,_/^" + BS + ",-^-,_")],                # r0  wet-look spikes
    [(8, "_,-'"), (26, "`-,_")],                      # r1
    [(6, ",-'"), (30, "`-.")],                        # r2
    [(5, ",'"), (32, "`.")],                          # r3
    [(4, "/"), (34, BS)],                             # r4
    [(4, "|"), (34, "|")],                            # r5
    [(35, "|")],                                      # r6
    [(35, "|")],                                      # r7
    [(35, "|")],                                      # r8
    [(35, "|")],                                      # r9
    [(35, "/")],                                      # r10
    [(34, "/")],                                      # r11
    [(32, "_/")],                                     # r12
    [(28, "`---'")],                                  # r13 hair ends behind jaw
]

# Left profile: fringe tip, forehead, brow, nose, lips, chin, jaw. This is the
# identity carrier and it has to read as one unbroken line.
WIDE_PROFILE = [
    [], [], [], [], [], [],
    [(4, BS + "_,")],                                 # r6  fringe juts past brow
    [(6, "/")],                                       # r7  forehead
    [(5, "(")],                                       # r8  brow
    [(5, "|")],                                       # r9  nose bridge
    [(4, "<")],                                       # r10 NOSE TIP
    [(5, BS + "_")],                                  # r11 under nose
    [(6, BS)],                                        # r12 upper lip
    [],                                               # r13 lower lip, from mouth
    [(8, "`-.__")],                                   # r14 chin
    [(12, "`-----.._")],                              # r15 jaw -> neck
]

# The crew-neck tee. Its right-hand slope starts exactly where the jaw run
# lands, so the two read as one continuous edge down the neck.
WIDE_BUST = [
    [], [], [], [], [], [], [], [], [], [], [], [], [], [], [], [],
    [(3, "__,---" + "'" * 7), (21, "`--.")],          # r16
    [(1, ",-'"), (27, "`-.")],                        # r17
]

# The fringe: the hair/face boundary, the second strongest line in the photo.
WIDE_FRINGE = [
    [], [], [], [], [], [],
    [(7, "--..__")],                                  # r6
    [(14, "`--.._")],                                 # r7
    [(18, "`-.._")],                                  # r8
    [(20, "`-.")],                                    # r9  taper to the ear
]

# Feature glyphs, plotted last with padding spaces so the fill cannot eat them.
WIDE_FEATURES = [
    [], [], [], [], [], [], [],
    [(10, "___")],                                    # r7  brow bar
    [(10, " <@)  ")],                                 # r8  EYE, profile wedge
    [(23, " ,-. ")],                                  # r9  ear
    [(23, " ( | ")],                                  # r10 ear
    [(23, " `-' ")],                                  # r11 ear
    [(6, BS + "wwwww._"), (23, " o ")],               # r12 SMILE + EARRING
    [(7, "`-----'")],                                 # r13 lower lip closes it
]

WIDE_HAIR = {0: (13, 24), 1: (9, 28), 2: (7, 31), 3: (6, 33), 4: (5, 34),
             5: (5, 34), 6: (10, 35), 7: (14, 35), 8: (19, 35), 9: (23, 35),
             10: (26, 35), 11: (27, 34), 12: (27, 33), 13: (28, 32)}

# The glossy highlight running through the crown. Its columns come from the
# bright band in the photo's tone map, but it is plotted as a clean diagonal by
# hand: letting the tone punch the holes gives moth-eaten speckle, while a drawn
# streak reads as light on hair. This is the single change that moved the
# candidate from "dithered photo" to "drawing".
WIDE_SHEEN = [
    [], [],
    [(19, " " * 4)],                                  # r2
    [(17, " " * 4)],                                  # r3
    [(14, " " * 4)],                                  # r4
    [(12, " " * 4)],                                  # r5
]


# ===========================================================================
# Narrow: 32x16, the vertical restack.
#
# The wide drawing at 32/38 in both axes. Everything the smaller canvas cannot
# hold goes from the hair, never from the face: the crown loses a row of spikes
# and the mass behind the ear loses two columns, while the profile keeps every
# one of its rows, because the profile is what carries the likeness and the
# hair is a mass either way.
# ===========================================================================

NARROW_SIL = [
    # 0         1         2         3
    # 01234567890123456789012345678901
    [(10, ",^-,_/^" + BS + ",-^,_")],                 # r0  crown spikes
    [(7, "_,-'"), (23, "`-,_")],                      # r1
    [(5, ",-'"), (27, "`-.")],                        # r2
    [(4, ",'"), (29, "`.")],                          # r3
    [(4, "|"), (30, BS)],                             # r4
    [(31, "|")],                                      # r5
    [(31, "|")],                                      # r6
    [(31, "|")],                                      # r7
    [(31, "/")],                                      # r8
    [(30, "/")],                                      # r9
    [(29, "/")],                                      # r10
    [(27, "_/")],                                     # r11
    [(24, "`--'")],                                   # r12 hair ends behind jaw
]

NARROW_PROFILE = [
    [], [], [], [], [],
    [(3, BS + "_,")],                                 # r5  fringe juts past brow
    [(5, "/")],                                       # r6  forehead
    [(4, "(")],                                       # r7  brow, nose bridge
    [(3, "<")],                                       # r8  NOSE TIP
    [(4, BS + "_")],                                  # r9  under nose
    [(5, BS)],                                        # r10 upper lip
    [],                                               # r11 lower lip, from mouth
    [(7, "`-.__")],                                   # r12 chin
    [(11, "`---..__")],                               # r13 jaw -> neck
]

NARROW_BUST = [
    [], [], [], [], [], [], [], [], [], [], [], [], [], [],
    [(2, "_,--" + "'" * 6), (19, "`--.")],            # r14
    [(0, ",-'"), (23, "`-.")],                        # r15
]

NARROW_FRINGE = [
    [], [], [], [], [],
    [(6, "-..__")],                                   # r5
    [(11, "`-.._")],                                  # r6
    [(16, "`-.")],                                    # r7  taper to the ear
]

NARROW_FEATURES = [
    [], [], [], [], [], [],
    [(8, "__")],                                      # r6  brow bar
    [(8, " <@) "), (19, " ,-. ")],                    # r7  EYE + ear top
    [(19, " ( | ")],                                  # r8  ear
    [(19, " `-' ")],                                  # r9  ear
    [(5, BS + "wwww._"), (19, " o ")],                # r10 SMILE + EARRING
    [(6, "`----'")],                                  # r11 lower lip closes it
]

NARROW_HAIR = {0: (11, 21), 1: (8, 25), 2: (6, 28), 3: (5, 29), 4: (5, 29),
               5: (9, 30), 6: (12, 30), 7: (19, 30), 8: (20, 30),
               9: (23, 29), 10: (24, 28), 11: (24, 26), 12: (24, 27)}

NARROW_SHEEN = [
    [], [],
    [(15, " " * 4)],                                  # r2
    [(12, " " * 4)],                                  # r3
    [(10, " " * 4)],                                  # r4
]


# ===========================================================================

SIZES = {
    (36, 18): (WIDE_SIL, WIDE_PROFILE, WIDE_FRINGE, WIDE_BUST,
               WIDE_HAIR, WIDE_SHEEN, WIDE_FEATURES),
    (32, 16): (NARROW_SIL, NARROW_PROFILE, NARROW_FRINGE, NARROW_BUST,
               NARROW_HAIR, NARROW_SHEEN, NARROW_FEATURES),
}


def build(cols, rows):
    """The line-art portrait at one of the two authored sizes, as lines.

    Order matters and is the prototype's: structure, then the mass fill, then
    the sheen, then the structure again - the sheen is plotted as blanks and
    would otherwise cut the silhouette it crosses - and the feature glyphs last
    of all, so nothing can swallow the eye or the earring.
    """
    if (cols, rows) not in SIZES:
        raise ValueError(f"no line art authored at {cols}x{rows}; "
                         f"have {sorted(SIZES)}")
    sil, profile, fringe, bust, hair, sheen, features = SIZES[(cols, rows)]

    a = Art(cols, rows)
    structure = (sil, profile, fringe, bust)
    for block in structure:
        a.plot(block)
    a.fill(hair)
    a.plot(sheen)
    for block in structure:
        a.plot(block)
    a.plot(features)
    return a.lines()
