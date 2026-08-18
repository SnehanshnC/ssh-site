# cast

The recorded walkthrough embedded in the root README, for the visitor who won't open a terminal.

```sh
bash scripts/record-cast.sh
```

## Why this is a developer tool

Like `make art`, this needs something the CI box doesn't have: `asciinema`, `agg`, `expect`, and a live network path to the box.
It records against whatever is actually deployed, so it's a snapshot taken by a human running it on demand, not a build step.
CI never runs it; it embeds whatever `demo.gif` is already committed.

## What it records

`scripts/record-cast.sh` drives the walkthrough itself with `expect`, so the pacing is exact and nobody has to sit and type: arrival on the card, a drill into work, back out, into an award that drills to its project, then `q` to exit.
It waits on the app's own page-kind chrome - the list footer, the detail footer, the arrival card's key legend - never on any one person's facts, so a content-pack push (a new job, a reordered award) never breaks the driver.

## Why a GIF, not the raw `.cast`

GitHub's Markdown doesn't play `.cast` files - it renders them as a raw JSON download, not a terminal.
The options were an asciinema.org upload (an external account this repo doesn't control, and a link that can rot), an SVG conversion (the tooling is unmaintained and inconsistent about animating in GitHub's own Markdown sandbox), or a GIF.
GIF won: it's a normal image GitHub already knows how to render inline and animate on load, needs nothing external, and stays byte-for-byte reproducible from the `.cast`.

`demo.cast` is kept alongside it anyway, as the source recording and in case a better embedding exists later - `agg` (or any other asciicast player) can always regenerate the GIF from it.

## What it emits

| asset | what it is |
| --- | --- |
| `demo.cast` | the raw asciinema v3 recording, real per-keystroke timing |
| `demo.gif` | `demo.cast` run through `agg`, idle gaps (mostly connection setup) capped at 1.5s so the embedded version doesn't sit on a blank screen |
