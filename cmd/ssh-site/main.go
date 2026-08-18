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
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	recovermw "charm.land/wish/v2/recover"
	"github.com/charmbracelet/colorprofile"

	"github.com/SnehanshnC/ssh-site/internal/content"
	"github.com/SnehanshnC/ssh-site/internal/ui"
)

const (
	defaultHost = "localhost"
	defaultPort = "2222"

	hostKeyPath = ".ssh/ssh_site_ed25519"

	idleTimeout   = 15 * time.Minute
	maxTimeout    = 1 * time.Hour
	shutdownGrace = 30 * time.Second
)

func main() {
	pack, err := content.Load()
	if err != nil {
		log.Error("could not load content pack", "error", err)
		os.Exit(1)
	}

	host := envOrDefault("SSH_SITE_HOST", defaultHost)
	port := envOrDefault("SSH_SITE_PORT", defaultPort)

	// The card's art is a truecolor cell render: every cell paints both a
	// foreground and a background, which is what makes it proof against the
	// visitor's own terminal theme. Left to itself Bubble Tea picks the
	// profile out of the session environment, and OpenSSH forwards TERM but
	// not COLORTERM, so a truecolor terminal arrives as plain
	// `xterm-256color` and the portrait gets quantised to 256 colours - the
	// one path ticket 04 measured and rejected outright, because 256 colours
	// on a colour master paints the jaw and neck a saturated red. Truecolor or
	// grayscale, never 256, so the profile is forced here.
	//
	// Establishing the visitor's real capability, and rendering the lower
	// tiers for terminals that do not have this one, is the render-ladder
	// slice's job. Until then every session gets the tier the card was signed
	// off in.
	programHandler := func(sess ssh.Session) *tea.Program {
		pty, _, _ := sess.Pty()
		m := ui.New(pack, pty.Window.Width, pty.Window.Height)
		// Appended after MakeOptions so it is the last writer of the profile.
		opts := append(bubbletea.MakeOptions(sess),
			tea.WithColorProfile(colorprofile.TrueColor))
		return tea.NewProgram(m, opts...)
	}

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithIdleTimeout(idleTimeout),
		wish.WithMaxTimeout(maxTimeout),
		wish.WithMiddleware(
			// Middlewares compose from first to last, so the last one here
			// (logging) runs first, calling into the recover-guarded chain
			// (activeterm, then bubbletea) as its "next".
			recovermw.Middleware(
				bubbletea.MiddlewareWithProgramHandler(programHandler),
				activeterm.Middleware(), // rejects sessions with no active PTY.
			),
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

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
