package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// failingPingStore is a store whose database cannot be reached.
//
// The store behind it works, only the check of the connection fails, so the
// test can see the answer of the health check without a broken database.
type failingPingStore struct {
	*store.Store
	err error
}

// Ping fails with the error of the store.
func (s failingPingStore) Ping(ctx context.Context) error {
	return s.err
}

// TestHealthPage_answers_with_200 checks the answer of the check of the
// health of the application.
//
// The request carries no origin, the check of the origin only touches the
// writing requests.
func TestHealthPage_answers_with_200(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.get(t, "/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Errorf("the body is %q, want %q", body, "ok")
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("the answer has no content security policy")
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Errorf("the header X-Content-Type-Options is %q, want %q", nosniff, "nosniff")
	}
}

// TestHealthPage_answers_with_503_when_the_database_fails checks the answer
// of the check of the health of the application when the database cannot be
// reached.
func TestHealthPage_answers_with_503_when_the_database_fails(t *testing.T) {
	handler := newFailingPingApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if body := rec.Body.String(); body != "unhealthy" {
		t.Errorf("the body is %q, want %q", body, "unhealthy")
	}
}

// TestHealthPage_answers_with_404_for_a_wrong_method checks the answer of the
// check of the health of the application for a request that writes.
//
// The catch-all at the end of the routes answers every other request with
// the page for 404.
func TestHealthPage_answers_with_404_for_a_wrong_method(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postFormWithOrigin(t, "/health", url.Values{}, testOrigin)

	if rec.Code != http.StatusNotFound {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHealthPage_rejects_a_post_without_an_origin checks that a request that
// writes never reaches the check of the health when it has no origin.
//
// The check of the origin runs before the routes, so the answer is the page
// for 403 and not the answer of the check of the health.
func TestHealthPage_rejects_a_post_without_an_origin(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postFormWithOrigin(t, "/health", url.Values{}, "")

	if rec.Code != http.StatusForbidden {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if body := rec.Body.String(); body == "unhealthy" || body == "ok" {
		t.Errorf("the body is %q, want the page for the refused request", body)
	}
}

// TestHealthPage_answers_with_503_after_the_time_limit_of_the_server checks
// that the check of the health gives up when the database does not answer.
//
// The store behind it works, only the check of the connection waits for the
// end of the context. The server has to give up after its time limit of two
// seconds and to answer with the status 503.
func TestHealthPage_answers_with_503_after_the_time_limit_of_the_server(t *testing.T) {
	handler := newSlowPingApp(t)

	start := time.Now()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if body := rec.Body.String(); body != "unhealthy" {
		t.Errorf("the body is %q, want %q", body, "unhealthy")
	}
	// The server waits for the database, so the answer cannot come at once.
	if elapsed < 1900*time.Millisecond {
		t.Errorf("the answer came after %s, want at least the time limit of the server of two seconds", elapsed)
	}
	// The upper bound is three times the time limit of the server of two
	// seconds. It is far above the time the answer needs, so a slow runner
	// under the race detector does not cause a false failure.
	if elapsed > 6*time.Second {
		t.Errorf("the answer came after %s, want at most 6s", elapsed)
	}
}

// TestHealthPage_answers_without_an_origin checks that the check of the
// health does not need an origin.
//
// The check reads the database and writes nothing, so the check of the origin
// does not touch it.
func TestHealthPage_answers_without_an_origin(t *testing.T) {
	app := newTestApp(t, conceptToday)

	req := newRequest(http.MethodGet, "/health", nil)
	if origin := req.Header.Get("Origin"); origin != "" {
		t.Fatalf("the request has the origin %q, want none", origin)
	}

	rec := app.serve(t, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
}

// newFailingPingApp starts the application with a store whose database
// cannot be reached.
func newFailingPingApp(t *testing.T) http.Handler {
	t.Helper()

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})

	handler, err := New(Options{
		Store: failingPingStore{
			Store: st,
			err:   errors.New("the database is gone"),
		},
		Now:      func() time.Time { return conceptToday },
		Location: testLocation,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return handler
}

// logEntry is a message of the log with its level.
type logEntry struct {
	level   slog.Level
	message string
}

// logRecorder is a handler of slog that remembers the messages it receives.
//
// The handler accepts every level, so the test sees every message of the
// server.
type logRecorder struct {
	mu      sync.Mutex
	entries []logEntry
}

// Enabled accepts every level.
func (r *logRecorder) Enabled(context.Context, slog.Level) bool { return true }

// Handle remembers the message with its level.
func (r *logRecorder) Handle(_ context.Context, record slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, logEntry{level: record.Level, message: record.Message})
	return nil
}

// WithAttrs returns the recorder, the attributes are not needed.
func (r *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }

// WithGroup returns the recorder, the groups are not needed.
func (r *logRecorder) WithGroup(string) slog.Handler { return r }

// recorded returns the messages that the recorder received.
func (r *logRecorder) recorded() []logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]logEntry(nil), r.entries...)
}

// TestHealthPage_logs_the_failure_of_the_check_as_a_warning checks the level
// of the message in the log when the database cannot be reached.
//
// A container runtime runs the check again and again, every 30 seconds. A
// failure that lasts would be written into the log again and again, so it is
// a warning and not an error of the request.
func TestHealthPage_logs_the_failure_of_the_check_as_a_warning(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	recorder := &logRecorder{}

	handler, err := New(Options{
		Store: failingPingStore{
			Store: st,
			err:   errors.New("the database is gone"),
		},
		Now:      func() time.Time { return conceptToday },
		Location: testLocation,
		Logger:   slog.New(recorder),
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	entries := recorder.recorded()
	if len(entries) == 0 {
		t.Fatal("the log holds no message about the failure")
	}
	found := false
	for _, entry := range entries {
		if !strings.Contains(entry.message, "health check") {
			continue
		}
		found = true
		if entry.level != slog.LevelWarn {
			t.Errorf("the message %q has the level %q, want %q", entry.message, entry.level, slog.LevelWarn)
		}
	}
	if !found {
		t.Errorf("the log holds %v, want a message about the failed check", entries)
	}
}

// slowPingStore is a store whose database answers only when the check gives
// up.
//
// The store behind it works, only the check of the connection waits for the
// end of the context, so the test can see the time limit of the server.
type slowPingStore struct {
	*store.Store
}

// Ping waits for the end of the context and fails then.
func (s slowPingStore) Ping(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// newSlowPingApp starts the application with a store whose database answers
// only when the check gives up.
func newSlowPingApp(t *testing.T) http.Handler {
	t.Helper()

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})

	handler, err := New(Options{
		Store:    slowPingStore{Store: st},
		Now:      func() time.Time { return conceptToday },
		Location: testLocation,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return handler
}
