// Command easyplan starts the EasyPlan web application.
//
// The configuration is read from the environment, the SQLite database is
// opened in the data directory and the HTTP server of package web serves the
// interface. The subcommand "healthcheck" is a placeholder and arrives with
// Phase 3.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/store"
	"github.com/HeiligerG/EasyPlan/internal/web"

	// The distroless image ships without a time zone database, so TZ would
	// not resolve inside the container. Embedding tzdata keeps
	// TZ=Europe/Zurich working.
	_ "time/tzdata"
)

const (
	defaultAddr = ":8080"
	defaultData = "/data"
	defaultTZ   = "Europe/Zurich"

	// databaseName is the name of the SQLite file in the data directory.
	databaseName = "easyplan.db"

	// shutdownTimeout is the time the server gets to finish the requests
	// that are still running before it is closed.
	shutdownTimeout = 10 * time.Second
)

// config holds the runtime configuration read from the environment.
type config struct {
	addr string
	data string
	tz   string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "easyplan:", err)
		os.Exit(1)
	}
}

// run dispatches the optional subcommand and starts the server.
//
// The plain invocation serves the interface until the process is asked to
// stop.
func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "healthcheck":
			// Placeholder: the real check against /health follows in Phase 3.
			return fmt.Errorf("subcommand %q is not implemented yet", args[0])
		default:
			return fmt.Errorf("unknown subcommand %q", args[0])
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return serve(ctx, readConfig())
}

// serve starts the HTTP server and blocks until the context is done or the
// server fails.
func serve(ctx context.Context, cfg config) error {
	location, err := time.LoadLocation(cfg.tz)
	if err != nil {
		return fmt.Errorf("cannot load the time zone %q: %w", cfg.tz, err)
	}
	time.Local = location

	if err := os.MkdirAll(cfg.data, 0o750); err != nil {
		return fmt.Errorf("cannot create the data directory %q: %w", cfg.data, err)
	}

	database := filepath.Join(cfg.data, databaseName)
	st, err := store.Open(database)
	if err != nil {
		return err
	}
	defer func() {
		if err := st.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "easyplan: cannot close the database:", err)
		}
	}()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	handler, err := web.New(web.Options{
		Store:    st,
		Now:      time.Now,
		Location: location,
		Logger:   logger,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		logger.Info("easyplan started",
			"addr", cfg.addr,
			"data", cfg.data,
			"database", database,
			"tz", cfg.tz,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- err
			return
		}
		failed <- nil
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
		logger.Info("easyplan stops")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("cannot stop the server: %w", err)
	}
	return <-failed
}

// readConfig returns the configuration from the environment, applying the
// documented defaults for unset values.
func readConfig() config {
	return config{
		addr: envOrDefault("EASYPLAN_ADDR", defaultAddr),
		data: envOrDefault("EASYPLAN_DATA", defaultData),
		tz:   envOrDefault("TZ", defaultTZ),
	}
}

// envOrDefault returns the value of the environment variable name, or
// fallback if the variable is unset or empty.
func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
