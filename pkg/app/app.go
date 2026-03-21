package app

import (
	"context"
	"cpamgt/pkg/server"
	"errors"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type App struct {
	name    string
	servers []server.Server
}

type Option func(a *App)

func NewApp(opts ...Option) *App {
	a := &App{}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func WithServer(servers ...server.Server) Option {
	return func(a *App) {
		a.servers = servers
	}
}

func WithName(name string) Option {
	return func(a *App) {
		a.name = name
	}
}

func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// create a channel to collect errors from all servers
	errCh := make(chan error, len(a.servers))
	// wait all server goroutines to exit
	var wg sync.WaitGroup
	for _, srv := range a.servers {
		wg.Add(1)
		go func(srv server.Server) {
			defer wg.Done()
			// Start the server
			if err := srv.Start(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				errCh <- err
			}
		}(srv)
	}

	var runErr error
	select {
	case <-runCtx.Done(): // runCtx.Done() is the signal that the user wants to stop the app
		slog.Info("run context done", "err", runCtx.Err())
	case err := <-errCh: // Server start error,shutdown all servers
		runErr = err
		slog.Error("server start err", "err", err)
		cancel()
	}

	// shutdown all servers
	// wait all server goroutines to exit
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	for _, srv := range a.servers {
		if err := srv.Stop(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("server stop err", "err", err)
			if runErr == nil {
				runErr = err
			}
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait() // wait all server goroutines to exit
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(11 * time.Second): // some goroutines are not exit in time, log a warning
		slog.Warn("timeout waiting for server goroutines to exit")
	}

	return runErr
}
