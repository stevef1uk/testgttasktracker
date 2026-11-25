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
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("taskboard: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := openStore(cfg)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = st.Close() }()

	app := web.NewApp(st)
	registerRoutes(app)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("taskboard listening on %s", srv.Addr)
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
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Print("taskboard stopped gracefully")
	return nil
}

// registerRoutes delegates every request to the app router, which owns the
// index page and the /api/* routes per the SPEC HTTP table.
func registerRoutes(app *web.App) {
	h := app.Handler()
	http.HandleFunc("/", h.ServeHTTP)
}

// openStore selects the persistence driver from config and returns a ready Store.
func openStore(cfg config.Config) (store.Store, error) {
	switch cfg.DBDriver {
	case "sqlite":
		return store.NewSQLiteStore(cfg.DBPath)
	case "postgres":
		return store.NewPostgresStore(cfg.DatabaseURL)
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.DBDriver)
	}
}
