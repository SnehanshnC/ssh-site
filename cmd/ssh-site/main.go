// Command ssh-site runs the Wish SSH server that serves Snehanshn's
// portfolio TUI to visitors.
package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	recovermw "charm.land/wish/v2/recover"
	"github.com/charmbracelet/colorprofile"
	"golang.org/x/time/rate"

	"github.com/SnehanshnC/ssh-site/internal/art"
	"github.com/SnehanshnC/ssh-site/internal/capability"
	"github.com/SnehanshnC/ssh-site/internal/content"
	"github.com/SnehanshnC/ssh-site/internal/ui"
)

const (
	defaultHost = "localhost"
	defaultPort = "2222"

	// The box's own copy overrides this via SSH_SITE_HOST_KEY_PATH to
	// /var/lib/ssh-site/host_ed25519 (D5): generated once, outside the build
	// tree, so a redeploy never regenerates it. This default stays a
	// build-tree-relative path because it only ever runs against a working
	// copy of the repo in local dev.
	defaultHostKeyPath = ".ssh/ssh_site_ed25519"

	idleTimeout   = 15 * time.Minute
	maxTimeout    = 1 * time.Hour
	shutdownGrace = 30 * time.Second

	// Session-level backstop only - nftables on the box is what actually
	// stops a handshake flood, since this middleware only runs after a
	// session channel opens. Values from research/09-server-hardening.md's
	// recommended baseline: ~1 new session/sec sustained per source IP, a
	// burst of 5, bounding the LRU to 10k tracked IPs.
	rateLimit      = rate.Limit(1)
	rateLimitBurst = 5
	rateLimitLRU   = 10_000
)

func main() {
	pack, err := content.Load()
	if err != nil {
		log.Error("could not load content pack", "error", err)
		os.Exit(1)
	}

	host := envOrDefault("SSH_SITE_HOST", defaultHost)
	port := envOrDefault("SSH_SITE_PORT", defaultPort)
	hostKeyPath := envOrDefault("SSH_SITE_HOST_KEY_PATH", defaultHostKeyPath)

	// Every session is placed on a rung of the render ladder before its first
	// frame, from the environment it arrived with: TERM out of the pty-req,
	// and whatever the client chose to forward. internal/capability owns that
	// decision and the reasoning behind it; nothing here guesses.
	//
	// The colour profile follows the tier rather than being detected again.
	// Left to itself Bubble Tea would pick the profile out of the same
	// environment and reach a different answer, because OpenSSH forwards TERM
	// and not COLORTERM, so a truecolor terminal arrives as plain
	// `xterm-256color` and the portrait would be quantised to 256 colours -
	// the one path ticket 04 measured and rejected outright, because 256
	// colours on a colour master paints the jaw and the neck a saturated red.
	// Truecolor or no colour, never the middle, which is exactly the split the
	// tier already encodes: the three cell tiers are truecolor renders, and
	// the bottom rung is a drawing that names no colour at all and is served
	// under a profile that would strip one if it did.
	programHandler := func(sess ssh.Session) *tea.Program {
		pty, _, _ := sess.Pty()
		tier := sessionTier(sess)
		log.Info("session tier", "tier", tier, "term", pty.Term,
			"remote", sess.RemoteAddr().String())

		m := ui.New(pack, tier, pty.Window.Width, pty.Window.Height)
		// Appended after MakeOptions so it is the last writer of the profile.
		opts := append(bubbletea.MakeOptions(sess),
			tea.WithColorProfile(colorProfile(tier)))
		return tea.NewProgram(m, opts...)
	}

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithIdleTimeout(idleTimeout),
		wish.WithMaxTimeout(maxTimeout),
		wish.WithMiddleware(
			// Middlewares compose from first to last, so the last one here
			// (logging) runs first, calling into the rate limiter, then the
			// recover-guarded chain (the PTY router, then bubbletea) as its
			// "next".
			recovermw.Middleware(
				bubbletea.MiddlewareWithProgramHandler(programHandler),
				documentRouter(pack), // D2: routes a session with no active PTY to the document instead of rejecting it.
			),
			// A session-scoped backstop, not the real defense: it only fires
			// after a session channel is already open, so it throttles repeat
			// sessions from one IP rather than the handshake cost of a raw
			// connection flood. nftables on the box (deploy/nftables.conf) is
			// what actually stops that.
			ratelimiter.Middleware(ratelimiter.NewRateLimiter(rateLimit, rateLimitBurst, rateLimitLRU)),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("could not configure server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting SSH server", "host", host, "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done

	log.Info("stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server gracefully", "error", err)
	}
}

// sessionTier reads the two places a visitor's terminal shows up in an SSH
// session and hands them to the ladder.
//
// They arrive by different routes and are not equally trustworthy. TERM rides
// in the pty-req, which OpenSSH always sends; everything else has to be an
// explicit `env` request from a client configured to forward it, which almost
// none are. capability.Detect knows that, and weighs them accordingly.
func sessionTier(sess ssh.Session) art.Tier {
	pty, _, _ := sess.Pty()
	return capability.Detect(pty.Term, sess.Environ())
}

// colorProfile is the writer profile a tier is served under.
//
// It is deliberately not a second detection pass. The bottom rung is reached
// precisely because nothing in the session vouched for truecolor, so it is
// served under a profile that strips colour out of everything - not only the
// portrait, which names none anyway, but the wordmark's gradient and the whole
// runtime-composed copy column beside it. Attributes survive: the nav row's
// live item and a list's cursor are bold under this profile rather than
// coloured, so the visitor can still see what they are on.
func colorProfile(tier art.Tier) colorprofile.Profile {
	if tier == art.Colorless {
		return colorprofile.ASCII
	}
	return colorprofile.TrueColor
}

// documentRouter is D2: a session with an active PTY is sent on to the
// interactive program - this middleware's own next, wired up in main to the
// bubbletea handler - while a session with none, `ssh ... | cat` chief among
// them, gets the whole portfolio as ui.Document's plain-text document and
// exits 0. It replaces the scaffold's activeterm middleware, which rejected
// exactly the sessions this one now serves.
//
// Serving the document is cheap enough to do inline - render once, write,
// close - and it runs inside the same recover-guarded, logged middleware
// chain as everything else, so the server's IdleTimeout and MaxTimeout still
// apply to it exactly as they do to an interactive session.
func documentRouter(pack *content.Pack) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			if _, _, active := sess.Pty(); active {
				next(sess)
				return
			}
			_, _ = wish.WriteString(sess, ui.Document(pack))
			_ = sess.Exit(0)
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
