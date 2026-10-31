package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The patterns of the HTML of the pages.
var (
	// inlineScriptPattern matches a script tag with its content.
	inlineScriptPattern = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
	// inlineStylePattern matches an attribute of an inline style.
	inlineStylePattern = regexp.MustCompile(`(?i)\sstyle\s*=`)
	// styleTagPattern matches a style tag.
	styleTagPattern = regexp.MustCompile(`(?i)<style\b`)
)

// failingStore is a store that fails with a secret text.
//
// The secret stands for the details of a failure of the database, they belong
// into the log of the server and not into the answer to the visitor.
//
// The set of methods names the methods that fail. An empty set lets every
// method of the interface fail.
type failingStore struct {
	*store.Store
	err  error
	fail map[string]bool
}

// fails reports whether the named method of the interface fails.
func (s failingStore) fails(name string) bool {
	if len(s.fail) == 0 {
		return true
	}
	return s.fail[name]
}

// ListAccounts fails with the error of the store.
func (s failingStore) ListAccounts(ctx context.Context, includeInactive bool) ([]store.Account, error) {
	if s.fails("ListAccounts") {
		return nil, s.err
	}
	return s.Store.ListAccounts(ctx, includeInactive)
}

// ListEntries fails with the error of the store.
func (s failingStore) ListEntries(ctx context.Context) ([]store.Entry, error) {
	if s.fails("ListEntries") {
		return nil, s.err
	}
	return s.Store.ListEntries(ctx)
}

// ListEntriesByAccount fails with the error of the store.
func (s failingStore) ListEntriesByAccount(ctx context.Context, accountID int64) ([]store.Entry, error) {
	if s.fails("ListEntriesByAccount") {
		return nil, s.err
	}
	return s.Store.ListEntriesByAccount(ctx, accountID)
}

// GetAccount fails with the error of the store.
func (s failingStore) GetAccount(ctx context.Context, id int64) (store.Account, error) {
	if s.fails("GetAccount") {
		return store.Account{}, s.err
	}
	return s.Store.GetAccount(ctx, id)
}

// GetEntry fails with the error of the store.
func (s failingStore) GetEntry(ctx context.Context, id int64) (store.Entry, error) {
	if s.fails("GetEntry") {
		return store.Entry{}, s.err
	}
	return s.Store.GetEntry(ctx, id)
}

// CreateAccount fails with the error of the store.
func (s failingStore) CreateAccount(ctx context.Context, name string) (int64, error) {
	if s.fails("CreateAccount") {
		return 0, s.err
	}
	return s.Store.CreateAccount(ctx, name)
}

// RenameAccount fails with the error of the store.
func (s failingStore) RenameAccount(ctx context.Context, id int64, name string) error {
	if s.fails("RenameAccount") {
		return s.err
	}
	return s.Store.RenameAccount(ctx, id, name)
}

// SetAccountActive fails with the error of the store.
func (s failingStore) SetAccountActive(ctx context.Context, id int64, active bool) error {
	if s.fails("SetAccountActive") {
		return s.err
	}
	return s.Store.SetAccountActive(ctx, id, active)
}

// SetAccountOrder fails with the error of the store.
func (s failingStore) SetAccountOrder(ctx context.Context, id int64, order int) error {
	if s.fails("SetAccountOrder") {
		return s.err
	}
	return s.Store.SetAccountOrder(ctx, id, order)
}

// MoveAccountUp fails with the error of the store.
func (s failingStore) MoveAccountUp(ctx context.Context, id int64) error {
	if s.fails("MoveAccountUp") {
		return s.err
	}
	return s.Store.MoveAccountUp(ctx, id)
}

// MoveAccountDown fails with the error of the store.
func (s failingStore) MoveAccountDown(ctx context.Context, id int64) error {
	if s.fails("MoveAccountDown") {
		return s.err
	}
	return s.Store.MoveAccountDown(ctx, id)
}

// CreateEntry fails with the error of the store.
func (s failingStore) CreateEntry(ctx context.Context, entry plan.Entry) (int64, error) {
	if s.fails("CreateEntry") {
		return 0, s.err
	}
	return s.Store.CreateEntry(ctx, entry)
}

// UpdateEntry fails with the error of the store.
func (s failingStore) UpdateEntry(ctx context.Context, entry plan.Entry) error {
	if s.fails("UpdateEntry") {
		return s.err
	}
	return s.Store.UpdateEntry(ctx, entry)
}

// DeleteEntry fails with the error of the store.
func (s failingStore) DeleteEntry(ctx context.Context, id int64) error {
	if s.fails("DeleteEntry") {
		return s.err
	}
	return s.Store.DeleteEntry(ctx, id)
}

// newFailingApp starts the application with a store that fails on every
// method.
func newFailingApp(t *testing.T, err error) http.Handler {
	t.Helper()
	return newFailingAppWith(t, err, nil)
}

// newFailingAppWith starts the application with a store that fails on the
// named methods.
func newFailingAppWith(t *testing.T, err error, fail map[string]bool) http.Handler {
	t.Helper()

	st, storeErr := store.OpenMemory()
	if storeErr != nil {
		t.Fatalf("store.OpenMemory() failed: %v", storeErr)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})

	handler, err := New(Options{
		Store:    failingStore{Store: st, err: err, fail: fail},
		Now:      func() time.Time { return conceptToday },
		Location: testLocation,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return handler
}

// TestNotFoundPage_shows_the_page_of_the_application checks the answer for an
// address that no route matches.
func TestNotFoundPage_shows_the_page_of_the_application(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.get(t, "/gibtsnicht")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusNotFound)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("the content type is %q, want text/html", contentType)
	}
	body := rec.Body.String()
	if !contains(body, "Seite nicht gefunden") {
		t.Errorf("the page does not name the reason:\n%s", body)
	}
	if contains(body, "404 page not found") {
		t.Errorf("the page shows the text of the standard library:\n%s", body)
	}
}

// TestTheWrongMethod_answers_with_404 documents the behaviour of the catch-all
// route.
//
// The patterns of the routes carry the method, so a request with the wrong
// method does not match any of them. The catch-all route at the end of the
// router answers with the page for 404 and not with the status 405. This is
// wanted: the page for 404 tells the visitor that the address does not exist,
// which is true for the method as well.
func TestTheWrongMethod_answers_with_404(t *testing.T) {
	app := newTestApp(t, conceptToday)

	cases := []struct {
		method string
		target string
	}{
		{method: http.MethodDelete, target: "/entries"},
		{method: http.MethodPut, target: "/entries"},
		{method: http.MethodPatch, target: "/accounts"},
		{method: http.MethodPost, target: "/timeline"},
		{method: http.MethodPost, target: "/"},
		{method: http.MethodGet, target: "/accounts/1/up"},
		{method: http.MethodGet, target: "/entries/1/delete"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			req := newRequest(tc.method, tc.target, nil)
			req.Header.Set("Origin", testOrigin)

			rec := app.serve(t, req)
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotFound)
			}
			if !contains(rec.Body.String(), "Seite nicht gefunden") {
				t.Errorf("the page does not show the page for 404:\n%s", rec.Body.String())
			}
		})
	}
}

// TestSecurityHeaders_are_on_every_answer checks the headers of the security
// on the pages, on the static files and on the answers with an error.
func TestSecurityHeaders_are_on_every_answer(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	answers := map[string]*struct {
		method string
		target string
		origin string
	}{
		"the dashboard":        {method: http.MethodGet, target: "/"},
		"the list of entries":  {method: http.MethodGet, target: "/entries"},
		"the form of an entry": {method: http.MethodGet, target: "/entries/new"},
		"the edit of an entry": {method: http.MethodGet, target: "/entries/" + strconv.FormatInt(id, 10)},
		"the timeline":         {method: http.MethodGet, target: "/timeline"},
		"the accounts":         {method: http.MethodGet, target: "/accounts"},
		"a static file":        {method: http.MethodGet, target: "/static/style.css"},
		"an unknown address":   {method: http.MethodGet, target: "/gibtsnicht"},
		"a rejected request":   {method: http.MethodPost, target: "/accounts", origin: "http://evil.example"},
	}

	for name, answer := range answers {
		t.Run(name, func(t *testing.T) {
			req := newRequest(answer.method, answer.target, strings.NewReader(url.Values{"name": {"Fremd"}}.Encode()))
			if answer.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if answer.origin != "" {
				req.Header.Set("Origin", answer.origin)
			}
			rec := app.serve(t, req)

			policy := rec.Header().Get("Content-Security-Policy")
			if !contains(policy, "default-src 'self'") {
				t.Errorf("the content security policy is %q, want a policy with \"default-src 'self'\"", policy)
			}
			if options := rec.Header().Get("X-Content-Type-Options"); options != "nosniff" {
				t.Errorf("the header of the content type is %q, want \"nosniff\"", options)
			}
			if policy := rec.Header().Get("Referrer-Policy"); policy != "same-origin" {
				t.Errorf("the policy of the referrer is %q, want \"same-origin\"", policy)
			}
		})
	}
}

// TestStaticFiles_are_served_with_an_entity_tag checks the static files of the
// application.
func TestStaticFiles_are_served_with_an_entity_tag(t *testing.T) {
	app := newTestApp(t, conceptToday)

	cases := []struct {
		name        string
		target      string
		contentType string
	}{
		{name: "the style sheet", target: "/static/style.css", contentType: "text/css"},
		{name: "the script", target: "/static/htmx.min.js", contentType: "javascript"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := app.get(t, tc.target)
			if rec.Code != http.StatusOK {
				t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
			}
			if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, tc.contentType) {
				t.Errorf("the content type is %q, want a type with %q", contentType, tc.contentType)
			}
			etag := rec.Header().Get("ETag")
			if etag == "" {
				t.Fatal("the answer has no entity tag")
			}
			if cache := rec.Header().Get("Cache-Control"); cache != cacheControl {
				t.Errorf("the cache header is %q, want %q", cache, cacheControl)
			}
			if rec.Body.Len() == 0 {
				t.Error("the file is empty")
			}

			// A browser that has the file already gets the answer 304.
			req := newRequest(http.MethodGet, tc.target, nil)
			req.Header.Set("If-None-Match", etag)
			rec = app.serve(t, req)
			if rec.Code != http.StatusNotModified {
				t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotModified)
			}
			if rec.Body.Len() != 0 {
				t.Error("the answer holds a body, want none")
			}
		})
	}
}

// TestStaticFiles_have_a_different_entity_tag checks that the entity tags of
// the files differ.
func TestStaticFiles_have_a_different_entity_tag(t *testing.T) {
	app := newTestApp(t, conceptToday)

	css := app.get(t, "/static/style.css").Header().Get("ETag")
	js := app.get(t, "/static/htmx.min.js").Header().Get("ETag")
	if css == "" || js == "" {
		t.Fatalf("the entity tags are %q and %q, want both", css, js)
	}
	if css == js {
		t.Errorf("both files have the entity tag %q, want two different tags", css)
	}
}

// TestStaticFiles_answer_with_404_for_an_unknown_file checks the answers for
// the addresses that name no file.
func TestStaticFiles_answer_with_404_for_an_unknown_file(t *testing.T) {
	app := newTestApp(t, conceptToday)

	for _, target := range []string{"/static/", "/static/gibtsnicht.css", "/static/unterordner/style.css"} {
		t.Run(target, func(t *testing.T) {
			rec := app.get(t, target)
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotFound)
			}
			if !contains(rec.Body.String(), "Seite nicht gefunden") {
				t.Errorf("the page does not show the page for 404:\n%s", rec.Body.String())
			}
		})
	}
}

// TestTheStoreFailure_shows_the_error_page_without_the_details checks that a
// failure of the database does not reach the visitor.
func TestTheStoreFailure_shows_the_error_page_without_the_details(t *testing.T) {
	const secret = "SECRET-DATABASE-FAILURE-4711"
	handler := newFailingApp(t, errors.New(secret))

	req := newRequest(http.MethodGet, "/", nil)
	rec := httptestRecorder(handler, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if contains(body, secret) {
		t.Errorf("the page shows the details of the failure:\n%s", body)
	}
	if !contains(body, "Die Seite konnte nicht geladen werden.") {
		t.Errorf("the page does not show the message of the failure:\n%s", body)
	}
}

// httptestRecorder sends a request to a handler and returns the answer.
func httptestRecorder(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestTheStoreFailure_shows_the_error_page_on_every_page checks that a
// failure of the database leads to the error page on every page of the
// application and that the details of the failure stay in the log.
func TestTheStoreFailure_shows_the_error_page_on_every_page(t *testing.T) {
	const secret = "SECRET-DATABASE-FAILURE-4711"
	handler := newFailingApp(t, errors.New(secret))

	pages := []string{
		"/", "/entries", "/entries/new", "/entries/1", "/entries/1/delete",
		"/timeline", "/accounts", "/accounts/1",
	}
	for _, target := range pages {
		t.Run("GET "+target, func(t *testing.T) {
			rec := httptestRecorder(handler, newRequest(http.MethodGet, target, nil))
			checkErrorPage(t, rec, secret)
		})
	}

	writes := []string{
		"/entries", "/entries/1", "/entries/1/delete",
		"/accounts", "/accounts/1", "/accounts/1/up", "/accounts/1/down",
		"/accounts/1/activate", "/accounts/1/deactivate",
	}
	for _, target := range writes {
		t.Run("POST "+target, func(t *testing.T) {
			req := newRequest(http.MethodPost, target, strings.NewReader(url.Values{"name": {"X"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", testOrigin)

			rec := httptestRecorder(handler, req)
			checkErrorPage(t, rec, secret)
		})
	}
}

// TestTheStoreFailure_of_the_entries_shows_the_error_page checks the pages
// that read the accounts first and the entries afterwards.
func TestTheStoreFailure_of_the_entries_shows_the_error_page(t *testing.T) {
	const secret = "SECRET-DATABASE-FAILURE-4711"
	handler := newFailingAppWith(t, errors.New(secret), map[string]bool{"ListEntries": true})

	for _, target := range []string{"/", "/entries", "/timeline", "/accounts"} {
		t.Run(target, func(t *testing.T) {
			rec := httptestRecorder(handler, newRequest(http.MethodGet, target, nil))
			checkErrorPage(t, rec, secret)
		})
	}
}

// checkErrorPage checks the answer of a failure of the store.
func checkErrorPage(t *testing.T, rec *httptest.ResponseRecorder, secret string) {
	t.Helper()

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if contains(body, secret) {
		t.Errorf("the page shows the details of the failure:\n%s", body)
	}
	if !contains(body, "Die Seite konnte nicht geladen werden.") {
		t.Errorf("the page does not show the message of the failure:\n%s", body)
	}
}

// TestThePages_hold_no_inline_script_and_no_inline_style checks the content
// security policy of the application.
//
// The policy forbids every inline script and every inline style, so a page
// with one of them would be broken in the browser without an error in the
// server.
func TestThePages_hold_no_inline_script_and_no_inline_style(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	pages := map[string]string{
		"the dashboard":            "/",
		"the list of entries":      "/entries",
		"the form of an entry":     "/entries/new",
		"the edit of an entry":     "/entries/" + strconv.FormatInt(id, 10),
		"the deletion of an entry": "/entries/" + strconv.FormatInt(id, 10) + "/delete",
		"the timeline":             "/timeline",
		"the accounts":             "/accounts",
		"the rename of an account": "/accounts/" + strconv.FormatInt(lifestyle.ID, 10),
		"an unknown address":       "/gibtsnicht",
	}
	for name, target := range pages {
		t.Run(name, func(t *testing.T) {
			body := app.get(t, target).Body.String()
			checkNoInlineScript(t, body)
			checkNoInlineStyle(t, body)
		})
	}

	// The rejected request and the form with an error are pages as well.
	rejected := app.postFormWithOrigin(t, "/accounts", url.Values{"name": {"Fremd"}}, "http://evil.example")
	checkNoInlineScript(t, rejected.Body.String())
	checkNoInlineStyle(t, rejected.Body.String())
}

// checkNoInlineScript checks that the page holds no script with a content.
func checkNoInlineScript(t *testing.T, body string) {
	t.Helper()
	for _, match := range inlineScriptPattern.FindAllStringSubmatch(body, -1) {
		if strings.TrimSpace(match[2]) != "" {
			t.Errorf("the page holds an inline script: %q", match[0])
		}
		if !strings.Contains(match[1], "src=") {
			t.Errorf("the page holds a script without a source: %q", match[0])
		}
	}
}

// TestTheFormWithAnError_holds_no_inline_script_and_no_inline_style checks
// the pages that answer with a rejected form.
//
// The pages of the errors use the same layout as the other pages, so they
// have to obey the same rules of the content security policy.
func TestTheFormWithAnError_holds_no_inline_script_and_no_inline_style(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	// The form of an entry with an error.
	values := newEntryFormValues(lifestyle.ID, "", "0", "2026-13-01", "day", "0")
	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	checkNoInlineScript(t, rec.Body.String())
	checkNoInlineStyle(t, rec.Body.String())

	// The list of the accounts with the error of a new account.
	rec = app.postForm(t, "/accounts", url.Values{"name": {"   "}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	checkNoInlineScript(t, rec.Body.String())
	checkNoInlineStyle(t, rec.Body.String())

	// The form of the rename of an account with an error.
	rec = app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10), url.Values{"name": {"   "}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	checkNoInlineScript(t, rec.Body.String())
	checkNoInlineStyle(t, rec.Body.String())
}

// checkNoInlineStyle checks that the page holds no inline style.
func checkNoInlineStyle(t *testing.T, body string) {
	t.Helper()
	if match := inlineStylePattern.FindString(body); match != "" {
		t.Errorf("the page holds an inline style: %q", match)
	}
	if match := styleTagPattern.FindString(body); match != "" {
		t.Errorf("the page holds a style tag: %q", match)
	}
}

// TestThePages_escape_the_names_of_the_entries_and_the_accounts checks that
// a name with markup in it cannot break the page.
func TestThePages_escape_the_names_of_the_entries_and_the_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	const dangerous = `<script>alert(1)</script>`
	const quote = `Miete "gross"`

	id := createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: dangerous + " " + quote, AmountMinor: 200000,
		StartDate: mustDay(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
	})
	if err := app.store.RenameAccount(context.Background(), lifestyle.ID, dangerous+" "+quote); err != nil {
		t.Fatalf("RenameAccount() failed: %v", err)
	}

	pages := map[string]string{
		"the list of entries":      "/entries",
		"the edit of an entry":     "/entries/" + strconv.FormatInt(id, 10),
		"the accounts":             "/accounts",
		"the rename of an account": "/accounts/" + strconv.FormatInt(lifestyle.ID, 10),
		"the dashboard":            "/",
		"the timeline":             "/timeline",
	}
	for name, target := range pages {
		t.Run(name, func(t *testing.T) {
			body := app.get(t, target).Body.String()
			if contains(body, dangerous) {
				t.Errorf("the page holds the name without escaping:\n%s", body)
			}
			if !contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
				t.Errorf("the page does not show the escaped name:\n%s", body)
			}
			if !contains(body, "Miete &#34;gross&#34;") && !contains(body, "Miete &quot;gross&quot;") {
				t.Errorf("the page does not show the escaped quotation marks:\n%s", body)
			}
		})
	}
}
