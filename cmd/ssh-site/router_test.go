package main

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/ssh"
	"charm.land/wish/v2/testsession"
	gossh "golang.org/x/crypto/ssh"

	"github.com/SnehanshnC/ssh-site/internal/analytics"
	"github.com/SnehanshnC/ssh-site/internal/content"
	"github.com/SnehanshnC/ssh-site/internal/ui"
)

// testStats is a fresh analytics.Store backed by a throwaway file, for tests
// that need one to construct documentRouter but do not care about analytics
// themselves.
func testStats(t *testing.T) *analytics.Store {
	t.Helper()
	s, err := analytics.Open(filepath.Join(t.TempDir(), "analytics.json"))
	if err != nil {
		t.Fatalf("open analytics store: %v", err)
	}
	return s
}

// realPack loads the actual fetched content pack, the way cmd/ssh-site loads
// it at startup - not a fixture, so these tests prove the piped path against
// what a visitor would actually see.
func realPack(t *testing.T) *content.Pack {
	t.Helper()
	pack, err := content.Load()
	if err != nil {
		t.Fatalf("load content pack: %v", err)
	}
	return pack
}

// TestNonPTYSessionGetsThePlainTextDocument is D2 proved end to end: `ssh
// host | cat` never allocates a PTY, so a session that never calls RequestPty
// is exactly that visitor, and testsession is the tool the ecosystem already
// uses to drive a session without a live TCP listener or a real client.
//
// The router's own next never runs - the interactive program is never
// reached - which is the acceptance criterion "activeterm is gone, replaced
// by a PTY router" proved from the outside: the thing that used to reject
// this session now serves it instead of forwarding it.
func TestNonPTYSessionGetsThePlainTextDocument(t *testing.T) {
	pack := realPack(t)
	reachedProgram := false
	srv := &ssh.Server{
		Handler: documentRouter(pack, testStats(t))(func(ssh.Session) { reachedProgram = true }),
	}

	out, err := testsession.New(t, srv, nil).Output("")
	if err != nil {
		t.Fatalf("a non-PTY session did not exit cleanly: %v", err)
	}
	if reachedProgram {
		t.Error("a non-PTY session reached the interactive program instead of the document")
	}

	got := string(out)
	if want := ui.Document(pack); got != want {
		t.Errorf("the piped session printed:\n%s\nwant the document:\n%s", got, want)
	}
	if strings.ContainsRune(got, '\x1b') {
		t.Error("the piped session's output carries an ANSI escape")
	}
}

// TestPTYSessionReachesTheProgramUnchanged is the acceptance criterion "a PTY
// session's behavior is unchanged": the router is a fork in front of the
// interactive program, not a replacement for it, so a session that does
// allocate a PTY - every real visitor without `| cat` on the end of their
// command - still reaches exactly what activeterm used to hand off to,
// unmodified and with the document never written.
func TestPTYSessionReachesTheProgramUnchanged(t *testing.T) {
	pack := realPack(t)
	const marker = "the interactive program ran"
	srv := &ssh.Server{
		Handler: documentRouter(pack, testStats(t))(func(s ssh.Session) {
			_, _ = s.Write([]byte(marker))
		}),
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 24, 80, gossh.TerminalModes{}); err != nil {
		t.Fatalf("could not request a pty: %v", err)
	}
	out, err := sess.Output("")
	if err != nil {
		t.Fatalf("a PTY session did not exit cleanly: %v", err)
	}
	if got := string(out); got != marker {
		t.Errorf("a PTY session saw %q, want the router to hand it to the program unchanged", got)
	}
}

// TestNonPTYSessionIsRecordedAsASessionWithNoTier is D7's other counting
// site: the router itself, not programHandler, is what sees a piped session,
// so it has to be the one that bumps the session counter - and it must leave
// the tier histogram alone, since a piped session never had a terminal to
// place on the ladder.
func TestNonPTYSessionIsRecordedAsASessionWithNoTier(t *testing.T) {
	pack := realPack(t)
	stats := testStats(t)
	srv := &ssh.Server{
		Handler: documentRouter(pack, stats)(func(ssh.Session) {}),
	}

	if _, err := testsession.New(t, srv, nil).Output(""); err != nil {
		t.Fatalf("a non-PTY session did not exit cleanly: %v", err)
	}

	got := stats.Snapshot()
	if got.Sessions != 1 {
		t.Errorf("Sessions = %d, want 1", got.Sessions)
	}
	for tier, count := range got.Tiers {
		if count != 0 {
			t.Errorf("Tiers[%s] = %d, want 0 for a piped session", tier, count)
		}
	}
}

// TestPTYSessionIsNotRecordedByTheRouter: the router hands a PTY session
// straight to its next unmodified, so any analytics recording for it is
// programHandler's job, not the router's - the router itself must not double
// count it.
func TestPTYSessionIsNotRecordedByTheRouter(t *testing.T) {
	pack := realPack(t)
	stats := testStats(t)
	srv := &ssh.Server{
		Handler: documentRouter(pack, stats)(func(ssh.Session) {}),
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 24, 80, gossh.TerminalModes{}); err != nil {
		t.Fatalf("could not request a pty: %v", err)
	}
	if _, err := sess.Output(""); err != nil {
		t.Fatalf("a PTY session did not exit cleanly: %v", err)
	}

	if got := stats.Snapshot().Sessions; got != 0 {
		t.Errorf("Sessions = %d, want 0 - the router should leave a PTY session's recording to programHandler", got)
	}
}
