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

	teaHandler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, _ := sess.Pty()
		m := ui.New(pack, pty.Window.Width, pty.Window.Height)
		return m, []tea.ProgramOption{}
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
				bubbletea.Middleware(teaHandler),
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
