# art

The pipeline behind the SSH site's terminal art.

```sh
make content    # the wordmark's text is a fact, and comes from the pack
make art        # regenerates internal/art/assets/*.ans
```

## Why this is a developer tool

The card has two halves that change at completely different rates.
The photograph and the wordmark change when a human decides to change them; the facts beside them change on every content-pack push.
So the art is a **build-time asset**: `make art` renders it, the `.ans` files under `internal/art/assets/` are committed, and the Go build embeds those.
CI never runs this directory.
A pack push therefore rebuilds the site without re-rendering a photograph, and neither ImageMagick nor Python is anywhere near the Go build.

Everything else on the card - the role, the school, the quest line, the links - composes at runtime in `internal/ui`.

## What it needs

`figlet`, `chafa`, ImageMagick 7 (`magick`), and Python 3 with nothing installed beyond the standard library.
Plus the source headshot, which is **not in this repo**.
It is private source material, and the matte geometry in `lib/master.py` is traced for that one photograph anyway, so it is passed in:

```sh
make art ART_HEADSHOT=/path/to/headshot.jpg
```

## What it emits

| asset | size | what it is |
| --- | --- | --- |
| `banner.ans` | 46x4 | figlet `smslant`, cyan-violet horizontal gradient |
| `portrait-wide-quad.ans` | 36x18 | the disc for the two-column card |
| `portrait-narrow-quad.ans` | 32x16 | the disc for the vertical restack |

The tier is `quad` (2x2 pixels per cell), the mainstream default.
The rest of the render ladder - sextant, vertical half-blocks, colorless - is a later slice; `lib/portrait.py` already takes the tier as a parameter.

## The pipeline

Ported from the ticket-04 prototype, which settled every number in it over five rounds of real renders reviewed in a real terminal.

```
headshot.jpg
  │  crop to the disc-centred square, pad past the frame edge with the fill
  │  trace the silhouette matte, then a chromatic border flood fill
  │  tone: modulate / CLAHE / sigmoidal contrast / lift the black point
  │  composite the subject over a flat #0a0c11 plate
  │  retouch the earring                                    lib/master.py
  ▼
master.png (960x960)
  │  ramp the outermost cell and a half toward the fill     lib/master.py
  │  resize to the mode's subcell grid, sharpen THERE, point-upscale
  │  chafa -f symbols -c full --symbols quad+space+solid    lib/portrait.py
  │  clip to the disc, draw the Braille ring                lib/disc.py
  ▼
portrait-*.ans
```

The shape of the render stage is the point: **we** own every pixel decision and chafa only picks a glyph and two colours.
The master is resized to exactly the subcell grid the mode can represent, sharpened at that resolution - the only place sharpening does anything, because sharpening the 960px master is averaged away by the downsample - then point-upscaled by an integer factor so chafa's own scaler cannot put the blur back.

## Traps

Four things in here look like they could be simplified and cannot.

**Ring cells must restate both foreground and background.**
`lib/disc.py` writes the ring as a foreground-only state and `lib/ansigrid.py` emits a canonical prefix that always names a background too.
Drop the canonicalisation and the right-hand arc inherits the background of whichever face cell the terminal painted before it, which puts skin tone behind the ring.

**Composed ANSI needs SGR-state canonicalisation.**
chafa emits a foreground and a background per cell and never resets, so a parser that carries raw prefixes forward by concatenation grows an ever longer prefix per cell: 3 MB per screen instead of 18 KB, for byte-identical output.
Resolve the state, re-emit it canonically.

**`magick bg subj matte -compose over -composite` silently ignores the matte** on ImageMagick 7 and hands back the untouched photo.
Copy the matte into alpha with `-compose CopyOpacity` first, then composite.
`lib/master.py::_apply_through_mask` is the only way any mask is applied here.

**`-compose Minus` on masks computes an absolute difference**, so subtracting one mask from another *exposes* what it was meant to remove.

## The four cosmetic touch-ups

Ticket 04 signed the look off with four rough edges deferred into the build.
All four are fixed here, and each is fixed at the stage that owns it.

**The `smslant` row-gap lightness.**
Slant figlet fonts are drawn from `/`, `\`, `|` and `_`, and in a real monospace cell those strokes do not meet across a row boundary: `_` sits just under its own baseline and the `/` on the row below starts near its own cap height.
The wordmark therefore has a visible gap between every pair of rows and reads lighter than the face and the copy beside it.
The prototype's suggested cure was a half-block hero at 72x6, which costs two rows the 24-row screen does not have.
`lib/banner.py` fixes it inside the same 46x4 by swapping each stroke for the box or block glyph that fills the whole cell in that direction - `╱ ╲ │ ▁` - so the strokes tile edge to edge and the letterforms close up.
The gradient is untouched, so the colours are the ones ticket 04 signed off, cell for cell.

**The earring reading as one pale cell.**
At 36x18 the hoop is about 1.2 cells wide, so no amount of preprocessing resolves it as a ring - the medium cannot carry it.
What the medium can carry is a bright subcell against a dark one, which quad glyphs render as a two-tone cell with an edge in it.
`lib/master.py::_retouch_earring` lifts the metal and sinks the skin immediately around it, at the subcell scale rather than the pixel scale, because anything finer is averaged away by the 13x27 master pixels behind one subcell.
Both adjustments scale all three channels equally, so they move luminance and leave hue alone; an S-curve per channel drove the warm lobe to saturated orange.

**The temple matte wedge.**
A dark violet-grey wedge between the temple and the hairline, inherited from the prototype's matte: a bright grey blur pokes through the hair fringe there, the traced polygon admits it, and the flat fill darkens it instead of removing it.
It survived the existing chromatic cleanup because that classifier fires on the maroon awning and on bright cool pixels and the wedge is neither.
Sampled off the crop it runs `(52,49,60)..(62,58,73)`, a *dark* cool grey, while the hair beside it is `(1,1,1)..(11,13,25)` and the forehead is warm - so a third classifier separates it, fenced to the wedge's own measured box because the navy tee passes the same test.

**The collar step at the disc boundary.**
The disc clip is per *cell*, so where the boundary crosses a high-contrast edge every kept cell is a solid block of tee blue and every dropped one is empty, and the curve reads as a staircase.
`lib/master.py::soften_disc_edge` ramps the outermost cell and a half toward the background fill, before the downsample, which is the only place it can be fixed.
It is self-limiting: blending toward the fill is invisible wherever the picture is already near the fill, which is the whole top and right of the disc where the hair is, so the collar and the neck are the only places it does anything.

## Relationship to the reference render

`variant-final-quad.ans` in the ticket-04 prototype is the reference.
Running this pipeline with the touch-ups disabled reproduces its portrait pixel for pixel - the port was checked that way, against the prototype's own `work/final/` captures.
With the touch-ups on it deliberately does not, and those four deltas are the whole difference.
The geometry - 46x4 banner, 36x18 and 32x16 discs, the ring, the disc radius - is unchanged, and `internal/ui` asserts it.
