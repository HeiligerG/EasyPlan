package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestHealthcheck_succeeds_with_the_answer_of_the_server checks the check
// against a server that answers with 200.
func TestHealthcheck_succeeds_with_the_answer_of_the_server(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("the request asks for %q, want %q", r.URL.Path, "/health")
		}
		// The address of the test server names the loopback interface, so
		// the check has to run against it.
		if !strings.HasPrefix(r.Host, "127.0.0.1:") {
			t.Errorf("the request asks for the host %q, want the loopback interface", r.Host)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	if err := healthcheck(context.Background(), serverAddress(server)); err != nil {
		t.Errorf("healthcheck() failed: %v", err)
	}
}

// TestHealthcheck_fails_with_the_status_of_the_answer checks the check
// against a server that answers with 503.
func TestHealthcheck_fails_with_the_status_of_the_answer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := healthcheck(context.Background(), serverAddress(server))
	if err == nil {
		t.Fatal("healthcheck() accepted the answer with the status 503, want an error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("healthcheck() failed with %q, want an error about the status", err)
	}
}

// TestHealthcheck_fails_with_the_status_500 checks the check against a
// server that answers with 500.
func TestHealthcheck_fails_with_the_status_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := healthcheck(context.Background(), serverAddress(server))
	if err == nil {
		t.Fatal("healthcheck() accepted the answer with the status 500, want an error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("healthcheck() failed with %q, want an error about the status", err)
	}
}

// TestHealthcheck_fails_when_the_server_answers_too_slowly checks that the
// check gives up when the server does not answer in time.
//
// The check may take three seconds, so the test waits for them. The server
// answers nothing at all, it only notices that the connection is closed.
func TestHealthcheck_fails_when_the_server_answers_too_slowly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	start := time.Now()
	err := healthcheck(context.Background(), serverAddress(server))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("healthcheck() accepted the answer of a server that does not answer, want an error")
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "timeout") && !strings.Contains(message, "deadline") {
		t.Errorf("healthcheck() failed with %q, want an error about the time limit", err)
	}
	// The check waits for its time limit of three seconds before it gives up.
	if elapsed < 2900*time.Millisecond {
		t.Errorf("the check gave up after %s, want at least the time limit of three seconds", elapsed)
	}
}

// TestHealthcheck_gives_up_with_a_context_that_ends_soon checks that the
// check respects the context of the caller.
//
// The check gets the time limit of three seconds, the context of the caller
// can be shorter.
func TestHealthcheck_gives_up_with_a_context_that_ends_soon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := healthcheck(ctx, serverAddress(server))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("healthcheck() accepted the answer of a server that does not answer, want an error")
	}
	// The context of the caller ends after 100 milliseconds, so the check has
	// to give up long before its own time limit of three seconds. The bound
	// is the time limit of the check itself: it is far above the 100
	// milliseconds of the expected answer, so a slow runner does not cause a
	// false failure, and a check that ignores the context of the caller is
	// still caught, because it would need at least the three seconds.
	if elapsed >= healthcheckTimeout {
		t.Errorf("the check gave up after %s, want less than its own time limit of %s", elapsed, healthcheckTimeout)
	}
}

// TestHealthcheck_fails_without_a_server checks the check against an address
// that nobody listens on.
func TestHealthcheck_fails_without_a_server(t *testing.T) {
	addr := freeAddress(t)

	if err := healthcheck(context.Background(), addr); err == nil {
		t.Fatal("healthcheck() worked without a server, want an error")
	}
}

// TestHealthcheck_fails_with_an_address_without_a_port checks the check
// against an address the check cannot use.
func TestHealthcheck_fails_with_an_address_without_a_port(t *testing.T) {
	if err := healthcheck(context.Background(), "localhost"); err == nil {
		t.Fatal("healthcheck() worked with an address without a port, want an error")
	}
}

// TestHealthURL_uses_the_host_of_the_address checks the derivation of the
// URL of the check from the listen address.
func TestHealthURL_uses_the_host_of_the_address(t *testing.T) {
	cases := []struct {
		addr string
		want string
	}{
		{addr: ":8080", want: "http://127.0.0.1:8080/health"},
		{addr: "0.0.0.0:8080", want: "http://127.0.0.1:8080/health"},
		{addr: "[::]:8080", want: "http://127.0.0.1:8080/health"},
		{addr: "127.0.0.1:9000", want: "http://127.0.0.1:9000/health"},
		{addr: "[::1]:8080", want: "http://[::1]:8080/health"},
		{addr: "192.168.1.10:8080", want: "http://192.168.1.10:8080/health"},
	}
	for _, tc := range cases {
		t.Run(tc.addr, func(t *testing.T) {
			got, err := healthURL(tc.addr)
			if err != nil {
				t.Fatalf("healthURL(%q) failed: %v", tc.addr, err)
			}
			if got != tc.want {
				t.Errorf("healthURL(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}

// TestHealthURL_uses_the_default_address checks the URL of the check for the
// default of the configuration.
func TestHealthURL_uses_the_default_address(t *testing.T) {
	t.Setenv("EASYPLAN_ADDR", "")

	got, err := healthURL(readAddr())
	if err != nil {
		t.Fatalf("healthURL() failed: %v", err)
	}
	if got != "http://127.0.0.1:8080/health" {
		t.Errorf("the URL is %q, want %q", got, "http://127.0.0.1:8080/health")
	}
}

// TestHealthURL_refuses_an_address_that_the_check_cannot_use checks the
// addresses without a usable port.
func TestHealthURL_refuses_an_address_that_the_check_cannot_use(t *testing.T) {
	for _, addr := range []string{"", "localhost", ":0", ":99999"} {
		t.Run(addr, func(t *testing.T) {
			if _, err := healthURL(addr); err == nil {
				t.Errorf("healthURL(%q) worked, want an error", addr)
			}
		})
	}
}

// TestHealthURL_escapes_the_zone_of_an_address checks the URL of the check
// for an address that names a zone.
//
// A link local address of IPv6 names its zone with a percent sign. The sign
// has to be escaped in the host of a URL, otherwise net/url refuses the URL.
func TestHealthURL_escapes_the_zone_of_an_address(t *testing.T) {
	got, err := healthURL("[fe80::1%lo]:8080")
	if err != nil {
		t.Fatalf("healthURL() failed: %v", err)
	}
	const want = "http://[fe80::1%25lo]:8080/health"
	if got != want {
		t.Errorf("healthURL() = %q, want %q", got, want)
	}
	if _, err := url.Parse(got); err != nil {
		t.Errorf("the URL %q cannot be parsed: %v", got, err)
	}
}

// TestHealthcheck_builds_the_request_for_an_address_with_a_zone checks that
// the check builds its request for an address that names a zone.
//
// The check finds no server on a link local address, but the request has to
// be built, otherwise the check reports a failure that it cannot explain.
func TestHealthcheck_builds_the_request_for_an_address_with_a_zone(t *testing.T) {
	_, port, err := net.SplitHostPort(freeAddress(t))
	if err != nil {
		t.Fatalf("net.SplitHostPort() failed: %v", err)
	}

	err = healthcheck(context.Background(), "[fe80::1%lo]:"+port)
	if err == nil {
		t.Fatal("healthcheck() succeeded without a server, want an error")
	}
	if strings.Contains(err.Error(), "cannot build the request") {
		t.Errorf("healthcheck() failed with %q, want a request that was built", err)
	}
}

// serverAddress returns the listen address of the test server.
func serverAddress(server *httptest.Server) string {
	return strings.TrimPrefix(server.URL, "http://")
}
