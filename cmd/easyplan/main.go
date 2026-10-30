// Command easyplan starts the EasyPlan web application.
//
// This is the Phase 1 scaffold: the configuration is read from the
// environment, but the HTTP server is not implemented yet. The SQLite store
// exists in package store, the web server arrives with Phase 2. The time
// zone is only reported so far, it is applied with the server. The
// subcommand "healthcheck" is a placeholder and arrives with Phase 3.
package main

import (
	"fmt"
	"os"

	// The distroless image ships without a time zone database, so TZ would
	// not resolve inside the container. Embedding tzdata keeps
	// TZ=Europe/Zurich working.
	_ "time/tzdata"
)

const (
	defaultAddr = ":8080"
	defaultData = "/data"
	defaultTZ   = "Europe/Zurich"
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

// run dispatches the optional subcommand. The plain invocation only reports
// the effective configuration; serving HTTP is implemented in a later phase.
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

	cfg := readConfig()
	fmt.Printf("easyplan scaffold: listen=%s data=%s tz=%s\n", cfg.addr, cfg.data, cfg.tz)
	return nil
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
