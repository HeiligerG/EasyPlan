package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// TestOriginCheck_rejects_the_writing_requests_with_a_foreign_origin checks
// the rule of Konzept.md §9: a writing request has to come from the same
// origin the server was addressed to.
func TestOriginCheck_rejects_the_writing_requests_with_a_foreign_origin(t *testing.T) {
	cases := []struct {
		name   string
		origin string
	}{
		{name: "another host", origin: "http://evil.example"},
		{name: "another host with a port", origin: "http://evil.example:8080"},
		{name: "another scheme", origin: "https://evil.example:8080"},
		{name: "the host as a subdomain", origin: "http://app.evil.example:8080"},
		{name: "no origin", origin: ""},
		{name: "the origin of a sandbox", origin: "null"},
		{name: "another port", origin: "http://app.example:9090"},
		{name: "the host without the port", origin: "http://app.example"},
		{name: "an origin without a scheme", origin: "app.example:8080"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApp(t, conceptToday)
			rec := app.postFormWithOrigin(t, "/accounts", url.Values{"name": {"Fremd"}}, tc.origin)
			if rec.Code != http.StatusForbidden {
				t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
			}
			if names := accountNames(app.accounts(t)); len(names) != 3 {
				t.Errorf("the database holds the accounts %v, want only the three default accounts", names)
			}
		})
	}
}

// TestOriginCheck_lets_the_writing_requests_with_a_matching_origin_pass
// checks that the requests of the interface itself are not blocked.
func TestOriginCheck_lets_the_writing_requests_with_a_matching_origin_pass(t *testing.T) {
	cases := []struct {
		name   string
		origin string
	}{
		{name: "the same origin", origin: "http://app.example:8080"},
		{name: "another scheme", origin: "https://app.example:8080"},
		{name: "in a different case", origin: "http://APP.Example:8080"},
		{name: "the host in a different case", origin: "http://app.EXAMPLE:8080"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApp(t, conceptToday)
			rec := app.postFormWithOrigin(t, "/accounts", url.Values{"name": {"Eigen"}}, tc.origin)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
			}
			names := accountNames(app.accounts(t))
			if len(names) != 4 || names[3] != "Eigen" {
				t.Errorf("the database holds the accounts %v, want the new account \"Eigen\" at the end", names)
			}
		})
	}
}

// TestOriginCheck_ignores_the_forwarded_host checks that the host of the
// proxy does not open the door: the origin is compared with the host of the
// request and not with the header of the proxy.
func TestOriginCheck_ignores_the_forwarded_host(t *testing.T) {
	app := newTestApp(t, conceptToday)
	req := newRequest(http.MethodPost, "/accounts", strings.NewReader(url.Values{"name": {"Fremd"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://forwarded.example")
	req.Header.Set("X-Forwarded-Host", "forwarded.example")

	rec := app.serve(t, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if names := accountNames(app.accounts(t)); len(names) != 3 {
		t.Errorf("the database holds the accounts %v, want only the three default accounts", names)
	}
}

// TestOriginCheck_does_not_block_the_reading_requests checks that a request
// without an origin is answered as long as it does not change data.
func TestOriginCheck_does_not_block_the_reading_requests(t *testing.T) {
	app := newTestApp(t, conceptToday)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			req := newRequest(method, "/", nil)
			rec := app.serve(t, req)
			if rec.Code != http.StatusOK {
				t.Errorf("the status is %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

// TestOriginCheck_covers_every_writing_method checks that every method that
// changes data is checked.
//
// The application has no route for PUT, PATCH and DELETE, so the request
// that passes the check answers with the page for 404. The middleware runs
// before the router, so a request with a foreign origin is rejected with 403
// before the router sees it.
func TestOriginCheck_covers_every_writing_method(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			app := newTestApp(t, conceptToday)

			req := newRequest(method, "/entries", strings.NewReader(""))
			req.Header.Set("Origin", "http://evil.example")
			rec := app.serve(t, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("with a foreign origin the status is %d, want %d", rec.Code, http.StatusForbidden)
			}

			req = newRequest(method, "/entries", strings.NewReader(""))
			req.Header.Set("Origin", testOrigin)
			rec = app.serve(t, req)
			if rec.Code == http.StatusForbidden {
				t.Errorf("with a matching origin the status is %d, want a status that is not %d", rec.Code, http.StatusForbidden)
			}
		})
	}
}

// TestOriginCheck_writes_nothing_on_a_rejected_entry checks that a rejected
// request does not reach the database.
func TestOriginCheck_writes_nothing_on_a_rejected_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	form := url.Values{
		"account_id": {strconv.FormatInt(lifestyle.ID, 10)},
		"name":       {"Miete"},
		"amount":     {"2000"},
		"start_date": {"2026-01-01"},
		"unit":       {"month"},
		"interval":   {"1"},
	}
	rec := app.postFormWithOrigin(t, "/entries", form, "http://evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
}

// TestOriginCheck_writes_nothing_on_a_rejected_account checks that a rejected
// request does not create an account.
func TestOriginCheck_writes_nothing_on_a_rejected_account(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postFormWithOrigin(t, "/accounts", url.Values{"name": {"Fremd"}}, "http://evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if names := accountNames(app.accounts(t)); len(names) != 3 {
		t.Errorf("the database holds the accounts %v, want only the three default accounts", names)
	}
}

// TestOriginCheck_writes_nothing_on_a_rejected_deletion checks that a
// rejected request does not delete an entry.
func TestOriginCheck_writes_nothing_on_a_rejected_deletion(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	rec := app.postFormWithOrigin(t, "/entries/"+strconv.FormatInt(id, 10)+"/delete", url.Values{}, "http://evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	entries := app.entries(t)
	if len(entries) != 1 || entries[0].Name != "Miete" {
		t.Errorf("the database holds the entries %v, want the entry \"Miete\"", entryNames(entries))
	}
}

// TestOriginCheck_shows_the_error_page checks that a rejected request
// answers with the page of the application and not with a bare text.
func TestOriginCheck_shows_the_error_page(t *testing.T) {
	app := newTestApp(t, conceptToday)
	rec := app.postFormWithOrigin(t, "/accounts", url.Values{"name": {"Fremd"}}, "http://evil.example")

	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("the content type is %q, want text/html", contentType)
	}
	body := rec.Body.String()
	if !contains(body, "Zugriff verweigert") {
		t.Errorf("the page does not name the reason of the rejection:\n%s", body)
	}
}
