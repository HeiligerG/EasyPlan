package web

import (
	"context"
	"io"
	"net/http"
	"time"
)

// The answers of the check of the health of the application.
const (
	// healthOK is the body of the answer of a healthy application.
	healthOK = "ok"
	// healthUnhealthy is the body of the answer of an application whose
	// database cannot be reached.
	healthUnhealthy = "unhealthy"
)

// healthTimeout is the time the database may need for the check.
//
// The answer of the health check has to be fast: a container runtime waits a
// few seconds for it.
const healthTimeout = 2 * time.Second

// healthPage answers the question whether the application is healthy.
//
// The answer is short and always the same, so a container runtime can read
// it. The check asks the driver of the database, it answers with a select of
// a constant (select 1). The check reads no data of the database, it does
// not look at the file of the database and it does not try to write, so a
// health check does not change the state of the application.
//
// The answer is 200 when the database answers, 503 when it does not.
func (s *server) healthPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		// A container runtime runs the check again and again, every 30
		// seconds. A failure that lasts is written into the log again and
		// again, so it is a warning and not an error of the request.
		s.logger.Warn("the health check failed", "error", err)
		s.writeHealth(w, http.StatusServiceUnavailable, healthUnhealthy)
		return
	}

	s.writeHealth(w, http.StatusOK, healthOK)
}

// writeHealth writes the answer of the health check.
//
// The body is plain text, it is not a page of the interface. The details of a
// failure stay in the log of the server.
func (s *server) writeHealth(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		s.logger.Error("cannot write the health answer", "error", err)
	}
}
