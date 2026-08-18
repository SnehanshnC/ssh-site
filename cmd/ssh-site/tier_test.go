package main

import (
	"testing"

	"charm.land/ssh"
	"charm.land/wish/v2/testsession"
	"github.com/charmbracelet/colorprofile"
	gossh "golang.org/x/crypto/ssh"

	"github.com/SnehanshnC/ssh-site/internal/art"
)

// TestSessionTierComesFromTheRealSession is the acceptance criterion "the tier
// is chosen from the session environment, not hardcoded or guessed from width"
// proved from the outside: a real client makes a real pty-req and real `env`
// requests over a real connection, and the tier the server settles on is read
// back off the session it actually got.
//
// Reading TERM out of the pty-req rather than out of `env` is the load-bearing
// part. OpenSSH sends the pty-req unconditionally and forwards almost nothing
// through `env`, so a test that set TERM the second way would pass while the
// server saw nothing at all from a real visitor.
func TestSessionTierComesFromTheRealSession(t *testing.T) {
	tests := []struct {
		name string
		term string
		env  map[string]string
		want art.Tier
	}{
		{"kitty", "xterm-kitty", nil, art.Sextant},
		{"the mainstream terminal", "xterm-256color", nil, art.Quad},
		{"WezTerm, which only says so if the client forwards it", "xterm-256color",
			map[string]string{"TERM_PROGRAM": "WezTerm"}, art.Sextant},
		{"colour forwarded, glyphs unvouched for", "xterm",
			map[string]string{"COLORTERM": "truecolor"}, art.VHalf},
		{"a terminal advertising no truecolor", "xterm", nil, art.Colorless},
		{"an unrecognised terminal", "not-a-real-terminal", nil, art.Colorless},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make(chan art.Tier, 1)
			srv := &ssh.Server{Handler: func(sess ssh.Session) {
				got <- sessionTier(sess)
			}}

			sess := testsession.New(t, srv, nil)
			for key, value := range tt.env {
				if err := sess.Setenv(key, value); err != nil {
					t.Fatalf("could not forward %s: %v", key, err)
				}
			}
			if err := sess.RequestPty(tt.term, 24, 80, gossh.TerminalModes{}); err != nil {
				t.Fatalf("could not request a pty: %v", err)
			}
			if _, err := sess.Output(""); err != nil {
				t.Fatalf("the session did not exit cleanly: %v", err)
			}

			if tier := <-got; tier != tt.want {
				t.Errorf("TERM=%s %v got the %s tier, want %s",
					tt.term, tt.env, tier, tt.want)
			}
		})
	}
}

// TestSessionWithNoPtyStillPicksATier: the document path never asks for a tier,
// but nothing should be able to panic its way through a session that made no
// pty-req at all - a client that sends no pty sends no TERM either, and that is
// the emptiest input the ladder ever sees.
func TestSessionWithNoPtyStillPicksATier(t *testing.T) {
	got := make(chan art.Tier, 1)
	srv := &ssh.Server{Handler: func(sess ssh.Session) { got <- sessionTier(sess) }}

	if _, err := testsession.New(t, srv, nil).Output(""); err != nil {
		t.Fatalf("the session did not exit cleanly: %v", err)
	}
	if tier := <-got; tier != art.Colorless {
		t.Errorf("a session with no pty got the %s tier, want the bottom rung", tier)
	}
}

// TestColorProfileFollowsTheTier. The bottom rung is reached because nothing
// vouched for truecolor, so it has to be served under a profile that strips
// colour from the runtime-composed half of the screen too - the portrait names
// none, but the wordmark's gradient and the copy column beside it do.
func TestColorProfileFollowsTheTier(t *testing.T) {
	for _, tier := range art.Tiers {
		want := colorprofile.TrueColor
		if tier == art.Colorless {
			want = colorprofile.ASCII
		}
		if got := colorProfile(tier); got != want {
			t.Errorf("the %s tier is served as %s, want %s", tier, got, want)
		}
	}
}
