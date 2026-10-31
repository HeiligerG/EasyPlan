package web

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// brokenTemplate returns a template that fails while it is rendered.
//
// The template writes a part of the page and then asks for a field that the
// data does not have. The failure happens in the middle of the page, so a
// server that writes the page directly would send a half-written page.
func brokenTemplate(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("broken").Parse(
		`{{define "layout.html"}}<html><body><h1>EasyPlan</h1>{{.Data.Missing}}</body></html>{{end}}`,
	)
	if err != nil {
		t.Fatalf("template.Parse() failed: %v", err)
	}
	return tmpl
}

// TestRender_answers_with_500_for_a_broken_template checks that a template
// that fails does not send a half-written page.
//
// The page is rendered into a buffer first. A template that fails halfway
// must not leave a part of the page in the answer: the visitor gets a plain
// error and the details of the failure stay in the log of the server.
func TestRender_answers_with_500_for_a_broken_template(t *testing.T) {
	app := newTestApp(t, conceptToday)

	var log bytes.Buffer
	s := &server{
		store:    app.store,
		now:      func() time.Time { return conceptToday },
		location: testLocation,
		logger:   slog.New(slog.NewTextHandler(&log, nil)),
		templates: map[string]*template.Template{
			"broken": brokenTemplate(t),
		},
	}

	rec := httptest.NewRecorder()
	s.render(rec, http.StatusOK, "broken", PageData{Title: "Fehler", Data: dashboardData{}})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if contains(body, "<html") || contains(body, "EasyPlan") {
		t.Errorf("the answer holds a half-written page:\n%s", body)
	}
	if contains(body, "Missing") {
		t.Errorf("the answer holds the details of the failure:\n%s", body)
	}
	if !contains(log.String(), "Missing") {
		t.Errorf("the log does not hold the details of the failure:\n%s", log.String())
	}
}

// TestThePage_answers_with_500_for_a_broken_template checks the same rule on
// the way through the server.
func TestThePage_answers_with_500_for_a_broken_template(t *testing.T) {
	app := newTestApp(t, conceptToday)

	var log bytes.Buffer
	s := &server{
		store:    app.store,
		now:      func() time.Time { return conceptToday },
		location: testLocation,
		logger:   slog.New(slog.NewTextHandler(&log, nil)),
		templates: map[string]*template.Template{
			"dashboard": brokenTemplate(t),
		},
	}

	mux := http.NewServeMux()
	static, err := s.newStaticHandler()
	if err != nil {
		t.Fatalf("newStaticHandler() failed: %v", err)
	}
	s.static = static
	s.routes(mux)
	handler := s.securityHeaders(s.originCheck(mux))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if contains(body, "<html") || contains(body, "EasyPlan") {
		t.Errorf("the answer holds a half-written page:\n%s", body)
	}
	if contains(body, "Missing") {
		t.Errorf("the answer holds the details of the failure:\n%s", body)
	}
	if !contains(log.String(), "Missing") {
		t.Errorf("the log does not hold the details of the failure:\n%s", log.String())
	}
}
