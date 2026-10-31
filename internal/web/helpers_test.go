package web

import (
	"context"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The host of the requests of the tests and the origin that belongs to it.
const (
	testHost   = "app.example:8080"
	testOrigin = "http://app.example:8080"
)

// conceptToday is the day of the example of the concept: the 1st of october
// 2026.
var conceptToday = time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)

// conceptAccounts returns the accounts of the example of the concept with the
// entries that belong to them.
//
// The example of Konzept.md §5 and of docs/features/berechnung.md:
//
//	Lifestyle | Miete         | 2'000 | monatlich, am 1.
//	Lifestyle | Krankenkasse  |   450 | monatlich, am 5.
//	Lifestyle | Internet      |    60 | monatlich, am 15.
//	Sparen    | Autoversicherung | 1'200 | jährlich, 15.2.
//	Steuern   | Serafe        |   335 | jährlich, 20.3.
func conceptAccounts(t *testing.T, a *testApp) []store.Account {
	t.Helper()

	lifestyle := accountByName(t, a, "Lifestyle")
	sparen := accountByName(t, a, "Sparen")
	steuern := accountByName(t, a, "Steuern")

	entries := []plan.Entry{
		{
			AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 200000,
			StartDate: mustDay(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: lifestyle.ID, Name: "Krankenkasse", AmountMinor: 45000,
			StartDate: mustDay(t, "2026-01-05"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: lifestyle.ID, Name: "Internet", AmountMinor: 6000,
			StartDate: mustDay(t, "2026-01-15"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: sparen.ID, Name: "Autoversicherung", AmountMinor: 120000,
			StartDate: mustDay(t, "2026-02-15"), Unit: plan.Year, Interval: 1,
		},
		{
			AccountID: steuern.ID, Name: "Serafe", AmountMinor: 33500,
			StartDate: mustDay(t, "2026-03-20"), Unit: plan.Year, Interval: 1,
		},
	}
	for _, entry := range entries {
		createEntry(t, a, entry)
	}

	return a.accounts(t)
}

// testLocation is the time zone of the tests. The dates of the plan package
// are calendar days in UTC, so the tests do not need another time zone.
var testLocation = time.UTC

// testApp is the application under test: the handler and the database behind
// it.
type testApp struct {
	handler http.Handler
	store   *store.Store
}

// newTestApp starts the application with a database in memory and a fixed
// clock. The clock is injected, so the tests never depend on the wall clock.
func newTestApp(t *testing.T, now time.Time) *testApp {
	t.Helper()
	return newTestAppIn(t, now, testLocation)
}

// newTestAppIn starts the application with a database in memory, a fixed
// clock and the given time zone.
//
// The clock returns the instant as it is, the server converts it into the time
// zone of the application. A test of the time zone needs this: only the
// server may decide in which time zone the day of the instant lies.
func newTestAppIn(t *testing.T, now time.Time, location *time.Location) *testApp {
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
		Store:    st,
		Now:      func() time.Time { return now },
		Location: location,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return &testApp{handler: handler, store: st}
}

// serve sends a request to the handler and returns the answer.
func (a *testApp) serve(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

// newRequest builds a request for the handler.
//
// The host of the request is the host that the origin has to match, so the
// writing requests can be sent with a matching origin.
func newRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Host = testHost
	return req
}

// get sends a reading request.
func (a *testApp) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	return a.serve(t, newRequest(http.MethodGet, target, nil))
}

// postForm sends a form with an origin that matches the host of the request.
func (a *testApp) postForm(t *testing.T, target string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return a.postFormWithOrigin(t, target, values, testOrigin)
}

// postFormWithOrigin sends a form with the given origin. An empty origin
// leaves the header away, so the request has none.
func (a *testApp) postFormWithOrigin(t *testing.T, target string, values url.Values, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := newRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return a.serve(t, req)
}

// accounts reads all accounts of the database.
func (a *testApp) accounts(t *testing.T) []store.Account {
	t.Helper()
	accounts, err := a.store.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	return accounts
}

// entries reads all entries of the database.
func (a *testApp) entries(t *testing.T) []store.Entry {
	t.Helper()
	entries, err := a.store.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	return entries
}

// accountNames returns the names of the accounts in the order of the list.
func accountNames(accounts []store.Account) []string {
	names := make([]string, 0, len(accounts))
	for _, account := range accounts {
		names = append(names, account.Name)
	}
	return names
}

// entryNames returns the names of the entries in the order of the list.
func entryNames(entries []store.Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	return names
}

// mustDay parses a calendar day of the tests and fails the test when the
// literal is wrong.
func mustDay(t *testing.T, value string) time.Time {
	t.Helper()
	day, err := plan.ParseDate(value)
	if err != nil {
		t.Fatalf("plan.ParseDate(%q) failed: %v", value, err)
	}
	return day
}

// dayPtr returns a pointer to a parsed calendar day.
func dayPtr(t *testing.T, value string) *time.Time {
	t.Helper()
	day := mustDay(t, value)
	return &day
}

// accountByName returns the account with the given name.
func accountByName(t *testing.T, a *testApp, name string) store.Account {
	t.Helper()
	for _, account := range a.accounts(t) {
		if account.Name == name {
			return account
		}
	}
	t.Fatalf("the account %q does not exist", name)
	return store.Account{}
}

// createAccount creates an account and fails the test when it does not work.
func createAccount(t *testing.T, a *testApp, name string) int64 {
	t.Helper()
	id, err := a.store.CreateAccount(context.Background(), name)
	if err != nil {
		t.Fatalf("CreateAccount(%q) failed: %v", name, err)
	}
	return id
}

// createEntry creates an entry and fails the test when it does not work.
func createEntry(t *testing.T, a *testApp, entry plan.Entry) int64 {
	t.Helper()
	id, err := a.store.CreateEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("CreateEntry(%q) failed: %v", entry.Name, err)
	}
	return id
}

// monthlyEntry builds a monthly entry of an account.
func monthlyEntry(t *testing.T, accountID int64, name string, amountMinor int64, start string) plan.Entry {
	t.Helper()
	return plan.Entry{
		AccountID:   accountID,
		Name:        name,
		AmountMinor: amountMinor,
		StartDate:   mustDay(t, start),
		Unit:        plan.Month,
		Interval:    1,
	}
}

// planEntry builds a monthly entry with the given identifier, account and
// name. The tests use it for the entries that only have to be counted.
func planEntry(id, accountID int64, name string) plan.Entry {
	return plan.Entry{
		ID:          id,
		AccountID:   accountID,
		Name:        name,
		AmountMinor: 10000,
		Unit:        plan.Month,
		Interval:    1,
	}
}

// storeEntry builds an entry of the store with the data of an active account.
// The tests use it for the entries that do not have to be in the database.
func storeEntry(t *testing.T, id, accountID int64, name string, amountMinor int64, start string, unit plan.Unit, interval int) store.Entry {
	t.Helper()
	return store.Entry{
		Entry: plan.Entry{
			ID:          id,
			AccountID:   accountID,
			Name:        name,
			AmountMinor: amountMinor,
			StartDate:   mustDay(t, start),
			Unit:        unit,
			Interval:    interval,
		},
		AccountName:   "Lifestyle",
		AccountActive: true,
	}
}

// The patterns of the HTML of the pages.
var (
	// cardTitlePattern matches the heading of a group of the entries.
	cardTitlePattern = regexp.MustCompile(`<h3 class="card__title">(.*?)</h3>`)
	// listTitlePattern matches the name of a row of the list of the accounts.
	listTitlePattern = regexp.MustCompile(`<p class="list__title">(.*?)</p>`)
	// tagPattern matches a tag of the list of the entries.
	tagPattern = regexp.MustCompile(`<span class="tag">(.*?)</span>`)
)

// cardTitles returns the headings of the groups of the entries.
func cardTitles(body string) []string {
	matches := cardTitlePattern.FindAllStringSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	for _, match := range matches {
		titles = append(titles, stripTags(match[1]))
	}
	return titles
}

// cardTitleHTML returns the headings of the groups of the entries with their
// markup, so a test can look for a tag inside a heading.
func cardTitleHTML(body string) []string {
	matches := cardTitlePattern.FindAllStringSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	for _, match := range matches {
		titles = append(titles, match[1])
	}
	return titles
}

// entrySections returns the parts of the page of the entries that belong to
// one account.
func entrySections(body string) []string {
	parts := strings.Split(body, `<section class="card">`)
	if len(parts) < 2 {
		return nil
	}
	return parts[1:]
}

// entryRowPattern matches one row of the list of the entries with the name
// of the entry and the state in the last column.
var entryRowPattern = regexp.MustCompile(`(?s)<tr>\s*<th scope="row"><a class="entry-link" href="[^"]*">(.*?)</a></th>.*?<td class="status">(.*?)</td>.*?</tr>`)

// entryRowData holds the name and the status of one row of the list of the
// entries.
type entryRowData struct {
	name   string
	status string
}

// entryRows returns the rows of the list of the entries.
func entryRows(body string) []entryRowData {
	matches := entryRowPattern.FindAllStringSubmatch(body, -1)
	rows := make([]entryRowData, 0, len(matches))
	for _, match := range matches {
		rows = append(rows, entryRowData{
			name:   stripTags(match[1]),
			status: stripTags(match[2]),
		})
	}
	return rows
}

// entryRowByName returns the row of the entry with the given name.
func entryRowByName(t *testing.T, body, name string) entryRowData {
	t.Helper()
	for _, row := range entryRows(body) {
		if row.name == name {
			return row
		}
	}
	t.Fatalf("the list does not hold the entry %q", name)
	return entryRowData{}
}

// listTitles returns the names of the rows of the list of the accounts.
func listTitles(body string) []string {
	matches := listTitlePattern.FindAllStringSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	for _, match := range matches {
		titles = append(titles, stripTags(match[1]))
	}
	return titles
}

// tags returns the tags of a page, for example "deaktiviert" or "erledigt".
func tags(body string) []string {
	matches := tagPattern.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, stripTags(match[1]))
	}
	return out
}

// stripTags removes the HTML tags of a text.
//
// The entities of the text are undone as well, so the result is the text the
// browser shows to the visitor.
func stripTags(text string) string {
	without := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
	return strings.TrimSpace(html.UnescapeString(without))
}

// contains reports whether the text holds the needle.
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// countOccurrences returns how often the needle is in the haystack.
func countOccurrences(haystack, needle string) int {
	return strings.Count(haystack, needle)
}
