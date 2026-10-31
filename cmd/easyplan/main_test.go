package main

import (
	"context"
	"strings"
	"testing"
)

// TestRun_rejects_an_unknown_subcommand checks the dispatch of the optional
// subcommand.
func TestRun_rejects_an_unknown_subcommand(t *testing.T) {
	err := run([]string{"gibtsnicht"})
	if err == nil {
		t.Fatal("run() accepted the subcommand, want an error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("run() failed with %q, want an error about an unknown subcommand", err)
	}
}

// TestRun_reports_the_subcommand_that_is_not_implemented checks the
// placeholder of the healthcheck, which arrives with Phase 3.
func TestRun_reports_the_subcommand_that_is_not_implemented(t *testing.T) {
	err := run([]string{"healthcheck"})
	if err == nil {
		t.Fatal("run() accepted the subcommand, want an error")
	}
	if !strings.Contains(err.Error(), "healthcheck") {
		t.Errorf("run() failed with %q, want an error about the subcommand healthcheck", err)
	}
}

// TestReadConfig_uses_the_defaults checks the configuration without any
// environment variable.
func TestReadConfig_uses_the_defaults(t *testing.T) {
	t.Setenv("EASYPLAN_ADDR", "")
	t.Setenv("EASYPLAN_DATA", "")
	t.Setenv("TZ", "")

	cfg := readConfig()
	if cfg.addr != defaultAddr {
		t.Errorf("the address is %q, want %q", cfg.addr, defaultAddr)
	}
	if cfg.data != defaultData {
		t.Errorf("the data directory is %q, want %q", cfg.data, defaultData)
	}
	if cfg.tz != defaultTZ {
		t.Errorf("the time zone is %q, want %q", cfg.tz, defaultTZ)
	}
}

// TestReadConfig_reads_the_environment checks the configuration from the
// environment.
func TestReadConfig_reads_the_environment(t *testing.T) {
	t.Setenv("EASYPLAN_ADDR", "127.0.0.1:9999")
	t.Setenv("EASYPLAN_DATA", "/tmp/easyplan-test")
	t.Setenv("TZ", "Europe/Berlin")

	cfg := readConfig()
	if cfg.addr != "127.0.0.1:9999" {
		t.Errorf("the address is %q, want \"127.0.0.1:9999\"", cfg.addr)
	}
	if cfg.data != "/tmp/easyplan-test" {
		t.Errorf("the data directory is %q, want \"/tmp/easyplan-test\"", cfg.data)
	}
	if cfg.tz != "Europe/Berlin" {
		t.Errorf("the time zone is %q, want \"Europe/Berlin\"", cfg.tz)
	}
}

// TestEnvOrDefault_falls_back_to_the_default checks the helper of the
// configuration.
func TestEnvOrDefault_falls_back_to_the_default(t *testing.T) {
	t.Setenv("EASYPLAN_TEST_VALUE", "gesetzt")
	if got := envOrDefault("EASYPLAN_TEST_VALUE", "fallback"); got != "gesetzt" {
		t.Errorf("envOrDefault() = %q, want \"gesetzt\"", got)
	}

	t.Setenv("EASYPLAN_TEST_VALUE", "")
	if got := envOrDefault("EASYPLAN_TEST_VALUE", "fallback"); got != "fallback" {
		t.Errorf("envOrDefault() = %q, want \"fallback\"", got)
	}

	t.Setenv("EASYPLAN_TEST_VALUE", "  ")
	if got := envOrDefault("EASYPLAN_TEST_VALUE", "fallback"); got != "  " {
		t.Errorf("envOrDefault() = %q, want the value of the environment", got)
	}
}

// TestServe_rejects_an_unknown_time_zone checks that the server does not
// start with a time zone that does not exist.
func TestServe_rejects_an_unknown_time_zone(t *testing.T) {
	err := serve(context.Background(), config{
		addr: "127.0.0.1:0",
		data: t.TempDir(),
		tz:   "Gibtsnicht/Zone",
	})
	if err == nil {
		t.Fatal("serve() accepted the time zone, want an error")
	}
	if !strings.Contains(err.Error(), "cannot load the time zone") {
		t.Errorf("serve() failed with %q, want an error about the time zone", err)
	}
}
