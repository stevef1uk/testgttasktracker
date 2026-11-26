package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"taskboard/internal/config"
	"taskboard/internal/store"
	"taskboard/internal/web"
	webassets "taskboard/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("taskboard: %v", err)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Printf("config: %s", cfg.Redacted())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := s.Close(); cerr != nil {
			log.Printf("close store: %v", cerr)
		}
	}()

	app, err := web.NewApp(s, webassets.Templates)
	if err != nil {
		return fmt.Errorf("build app: %w", err)
	}

	// Top-level routes: the health probe and the embedded UI/API application.
	mux := http.NewServeMux()
	mux.Handle("/api/health", app)
	mux.Handle("/api/", app)
	mux.Handle("/", app)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
	case <-ctx.Done():
		log.Printf("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
	}
	return nil
}

// openStore selects the storage backend named by the configuration.
func openStore(ctx context.Context, cfg config.Config) (store.Store, error) {
	switch cfg.Driver {
	case "sqlite":
		s, err := store.OpenSQLite(ctx, cfg.SQLitePath)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		return s, nil
	case "postgres":
		s, err := store.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}
