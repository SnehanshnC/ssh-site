package capability

import (
	"testing"

	"github.com/SnehanshnC/ssh-site/internal/art"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		environ []string
		want    art.Tier
		why     string
	}{{
		name: "kitty advertises itself in TERM",
		term: "xterm-kitty",
		want: art.Sextant,
		why:  "kitty draws sextants itself and says so in the one variable SSH always forwards",
	}, {
		name: "foot advertises itself in TERM",
		term: "foot-extra",
		want: art.Sextant,
	}, {
		name: "Ghostty advertises itself in TERM",
		term: "xterm-ghostty",
		want: art.Sextant,
	}, {
		name:    "WezTerm only advertises itself in TERM_PROGRAM",
		term:    "xterm-256color",
		environ: []string{"TERM_PROGRAM=WezTerm"},
		want:    art.Sextant,
		why:     "WezTerm's default TERM says nothing about it, so the forwarded program name is the only signal - and it promotes, never demotes",
	}, {
		name: "the mainstream terminal gets the mainstream tier",
		term: "xterm-256color",
		want: art.Quad,
		why:  "SSH does not forward COLORTERM, so every truecolor terminal arrives looking exactly like this one",
	}, {
		name: "a direct-colour terminfo entry gets quad",
		term: "xterm-direct",
		want: art.Quad,
	}, {
		name: "tmux inside a modern terminal gets quad",
		term: "tmux-256color",
		want: art.Quad,
	}, {
		name:    "iTerm2 with nothing else forwarded gets quad",
		term:    "xterm",
		environ: []string{"TERM_PROGRAM=iTerm.app"},
		want:    art.Quad,
	}, {
		name:    "colour vouched for but glyphs not gets the half-block floor",
		term:    "xterm",
		environ: []string{"COLORTERM=truecolor"},
		want:    art.VHalf,
		why:     "COLORTERM says what the terminal can paint and nothing about what it can draw",
	}, {
		name:    "an unknown terminal that forwards COLORTERM still gets a colour tier",
		term:    "somebody-new",
		environ: []string{"COLORTERM=24bit"},
		want:    art.VHalf,
	}, {
		name: "a terminal advertising no truecolor gets the colorless tier",
		term: "xterm",
		want: art.Colorless,
	}, {
		name: "an absent TERM degrades rather than erroring",
		term: "",
		want: art.Colorless,
	}, {
		name: "an unrecognised TERM degrades rather than erroring",
		term: "some-terminal-nobody-has-heard-of",
		want: art.Colorless,
	}, {
		name: "TERM=dumb gets the drawing",
		term: "dumb",
		want: art.Colorless,
	}, {
		name: "the Linux console gets the drawing",
		term: "linux",
		want: art.Colorless,
		why:  "16 colours, and a console font that carries neither quadrants nor sextants",
	}, {
		name:    "NO_COLOR beats the best terminal on the ladder",
		term:    "xterm-kitty",
		environ: []string{"NO_COLOR=1"},
		want:    art.Colorless,
	}, {
		name:    "an empty NO_COLOR is not set, per the convention",
		term:    "xterm-kitty",
		environ: []string{"NO_COLOR="},
		want:    art.Sextant,
	}, {
		name:    "TERM_PROGRAM's casing is whatever each terminal felt like",
		term:    "xterm-256color",
		environ: []string{"term_program=wezterm"},
		want:    art.Sextant,
	}, {
		name:    "a malformed environment entry is skipped, not fatal",
		term:    "xterm-256color",
		environ: []string{"NOT_AN_ASSIGNMENT", "COLORTERM=truecolor"},
		want:    art.Quad,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.term, tt.environ)
			if got != tt.want {
				t.Errorf("Detect(%q, %v) = %s, want %s\n%s",
					tt.term, tt.environ, got, tt.want, tt.why)
			}
		})
	}
}

// TestEveryRungIsReachable is the check that the ladder is a ladder. A
// detection rule that quietly stopped firing would leave a tier with an asset
// nobody is ever served, and every test above would still pass if the rung it
// asserts were the only one left.
func TestEveryRungIsReachable(t *testing.T) {
	reached := map[art.Tier]string{
		Detect("xterm-kitty", nil):                       "xterm-kitty",
		Detect("xterm-256color", nil):                    "xterm-256color",
		Detect("xterm", []string{"COLORTERM=truecolor"}): "xterm + COLORTERM",
		Detect("vt100", nil):                             "vt100",
	}
	for _, tier := range art.Tiers {
		if _, ok := reached[tier]; !ok {
			t.Errorf("no session in this test reaches the %s tier", tier)
		}
	}
}

// TestDetectIsTotal guards the promise the slice makes about hostile input:
// whatever a client sends, a session ends up on a rung that renders, because
// every rung has an asset and the bottom one is plain ASCII.
func TestDetectIsTotal(t *testing.T) {
	terms := []string{"", " ", "\x00", "XTERM-256COLOR", "-direct", "256color",
		"screen", "screen.xterm-256color", "putty", "ansi", "unknown"}
	environs := [][]string{nil, {}, {""}, {"="}, {"COLORTERM="}, {"TERM_PROGRAM="},
		{"COLORTERM=maybe"}, {"NO_COLOR=0"}}
	for _, term := range terms {
		for _, environ := range environs {
			tier := Detect(term, environ)
			if len(art.Portrait(art.Wide, tier)) == 0 {
				t.Errorf("Detect(%q, %v) = %s, which has no portrait to serve",
					term, environ, tier)
			}
		}
	}
}
