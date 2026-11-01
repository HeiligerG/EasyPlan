// Command easyplan starts the EasyPlan web application.
//
// The configuration is read from the environment, the SQLite database is
// opened in the data directory and the HTTP server of package web serves the
// interface. The subcommand "healthcheck" asks the running application
// whether it is healthy, it is meant for the health check of the container.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	// defaultAddr is the listen address of the application outside of a
	// container. The service listens on the loopback interface only, the
	// access is regulated by a reverse proxy. The image of the container
	// sets EASYPLAN_ADDR=:8080, there the server listens on every
	// interface.
	defaultAddr = "127.0.0.1:8080"

	defaultData = "/data"
	defaultTZ   = "Europe/Zurich"

	// databaseName is the name of the SQLite file in the data directory.
	databaseName = "easyplan.db"

	// shutdownTimeout is the time the server gets to finish the requests
	// that are still running before it is closed.
	shutdownTimeout = 10 * time.Second

	// healthcheckTimeout is the time the check of the health of the
	// application may take. The check has to be fast, a container runtime
	// waits a few seconds for the answer.
	healthcheckTimeout = 3 * time.Second
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
			return healthcheck(context.Background(), readAddr())
		default:
			return fmt.Errorf("unknown subcommand %q", args[0])
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return serve(ctx, readConfig())
}

// healthcheck asks the running application whether it is healthy.
//
// The check runs against the address of the configuration, so it reaches the
// server inside the container as well as outside of it. An address without a
// host means the loopback interface.
//
// The function returns nil when the application answers with 200 and an error
// in every other case. The error ends the process with the exit code 1.
func healthcheck(ctx context.Context, addr string) error {
	url, err := healthURL(addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot build the request for %s: %w", url, err)
	}

	client := &http.Client{Timeout: healthcheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("the healthcheck failed: %w", err)
	}
	defer resp.Body.Close()

	// The body is read and dropped, so the connection can be reused.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("the healthcheck failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the healthcheck failed: the answer has the status %d", resp.StatusCode)
	}
	return nil
}

// healthURL returns the URL of the health check for the listen address of
// the configuration.
//
// The address is the one of the variable EASYPLAN_ADDR, it looks like
// ":8080", "127.0.0.1:8080" or "[::1]:8080". The host of the address is
// used, an address without a host or with the unspecified host listens on
// every interface, the check runs against the loopback interface in that
// case.
//
// A link local address of IPv6 names its zone with a percent sign, for
// example "[fe80::1%eth0]:8080". The sign is escaped in the host of the URL,
// otherwise net/url refuses the URL.
//
// The port 0 is refused: it means that the server takes any free port, so
// the check cannot know where to look.
func healthURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("cannot read the port of the address %q: %w", addr, err)
	}
	number, err := net.LookupPort("tcp", port)
	if err != nil {
		return "", fmt.Errorf("the port %q of the address %q is unknown: %w", port, addr, err)
	}
	if number < 1 || number > 65535 {
		return "", fmt.Errorf("the port %d of the address %q is out of range", number, addr)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(escapeZoneID(host), strconv.Itoa(number)) + "/health", nil
}

// escapeZoneID escapes the percent sign of the zone of an address.
//
// The zone of a link local address of IPv6 is separated from the address by
// a percent sign, for example "fe80::1%eth0". The sign has to be escaped in
// the host of a URL, otherwise net/url refuses the URL.
func escapeZoneID(host string) string {
	return strings.ReplaceAll(host, "%", "%25")
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

	// A backup that was killed hard leaves its temporary directory behind,
	// because the program cannot clean it up when it dies. The next start
	// removes the leftovers.
	if err := web.CleanupBackups(cfg.data); err != nil {
		logger.Warn("cannot remove the temporary directories of the backups", "dir", cfg.data, "error", err)
	}

	handler, err := web.New(web.Options{
		Store:    st,
		Now:      time.Now,
		Location: location,
		Logger:   logger,
		// The copy of the database is written into the data directory, so
		// the backup works even when the file system of the container is
		// read-only.
		BackupDir: cfg.data,
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
		addr: readAddr(),
		data: envOrDefault("EASYPLAN_DATA", defaultData),
		tz:   envOrDefault("TZ", defaultTZ),
	}
}

// readAddr returns the listen address from the environment, applying the
// documented default for an unset value.
func readAddr() string {
	return envOrDefault("EASYPLAN_ADDR", defaultAddr)
}

// envOrDefault returns the value of the environment variable name, or
// fallback if the variable is unset or empty.
func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
