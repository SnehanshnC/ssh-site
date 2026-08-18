# Context

Glossary for the SSH site.
Terms only - decisions live on the wayfinder map, maintained locally in `.scratch/ssh-site/` and deliberately kept out of this repo's public history.

## Terms

**Surface** - one of the three places Snehanshn's facts render: the SSH site (this repo), [snehanshn-site](https://snehanshn-site.vercel.app), and the SnehanshnC profile README.
Surfaces share facts only, never code, rendering, or deploys.

**Content pack** - the single shared data source every surface consumes, so no fact about Snehanshn is ever written twice.
Facts only, every entry slug-keyed; it carries no tone, curation, or styling, and lives in its own repo owned by no surface.

**Fact** - a canonical piece of content about Snehanshn (a job, a project, an award, a link, a hobby) that may render on more than one surface.
A fact lives only in the content pack; a fact hardcoded inside a surface's repo is a defect.
Phone number and email are never facts; they stay out of the pack entirely.

**Surface copy** - content one surface owns deliberately: tone, curation and ordering, visual metaphor, easter eggs, jokes.
Surface copy may play with facts but never contradict them; aggregate claims (counts, records) are derived from itemized facts, never hand-written.

**Slug** - the permanent identifier a fact carries in the pack.
Surfaces key their display mappings to slugs, never to display text, so rewording a fact breaks nothing.

**App server** - the site's own SSH server: the Go process (Wish) that answers visitors on port 22 and serves the TUI.
It is not sshd, performs no authentication, and guards no shell.

**Admin sshd** - the VM's OpenSSH daemon for administering the box, listening on a high non-standard port.
Never visitor-facing; the only listener on the machine that guards real credentials.

**Address** - the literal string a visitor types after `ssh `.
Currently `snehanshn.duckdns.org`; swappable without downstream rework beyond copy.

**Render ladder** - the four ways the portrait is drawn, best to worst: sextant, quad, vertical half-blocks, colorless.
The top three are the same photograph at three subcell resolutions; the bottom one is a hand-drawn line-art portrait, not that photograph with its colour removed.

**Render tier** - the one rung of the ladder a given visitor is served, decided once per session from the environment that session arrives with and never from its window size.
A property of the visitor's terminal, so unlike the layout it does not change when they resize.

**Visitor** - an anonymous SSH client connecting to the app server.
Never authenticated, never gated; a presented public key may personalize but never blocks.
