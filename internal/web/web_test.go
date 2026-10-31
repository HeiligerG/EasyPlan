package web

import (
	"context"
	"errors"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// TestNew_needs_a_store checks that the constructor refuses to work without a
// store.
func TestNew_needs_a_store(t *testing.T) {
	handler, err := New(Options{})
	if err == nil {
		t.Fatal("New() worked without a store, want an error")
	}
	if handler != nil {
		t.Error("New() returned a handler next to the error, want none")
	}
	if !strings.Contains(err.Error(), "no store given") {
		t.Errorf("New() failed with %q, want an error about the missing store", err)
	}
}

// TestNew_uses_the_defaults checks that the constructor works without a clock
// and without a time zone.
func TestNew_uses_the_defaults(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})

	handler, err := New(Options{Store: st})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestRender_answers_with_500_for_an_unknown_page checks the answer for a
// page that no template belongs to.
func TestRender_answers_with_500_for_an_unknown_page(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := httptest.NewRecorder()
	s := &server{
		store:     app.store,
		now:       func() time.Time { return conceptToday },
		location:  testLocation,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		templates: map[string]*template.Template{},
	}
	s.render(rec, http.StatusOK, "gibtsnicht", PageData{Title: "Fehler"})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestAmountMessage_names_every_error_of_the_parser checks the German
// messages of the errors of the amount parser.
func TestAmountMessage_names_every_error_of_the_parser(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{err: ErrAmountEmpty, want: "Der Betrag darf nicht leer sein."},
		{err: ErrAmountNegative, want: "Der Betrag darf nicht negativ sein."},
		{err: ErrAmountZero, want: "Der Betrag muss grösser als 0 sein."},
		{err: ErrAmountPrecision, want: "Höchstens 2 Nachkommastellen sind erlaubt."},
		{err: ErrAmountTooLarge, want: "Der Betrag ist zu gross."},
		{err: ErrAmountFormat, want: "Der Betrag muss eine Zahl sein."},
		{err: errors.New("something else"), want: "Der Betrag ist ungültig."},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := amountMessage(tc.err); got != tc.want {
				t.Errorf("amountMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEntriesPage_rejects_an_unknown_account checks the answer of a form with
// an account that does not exist.
func TestEntriesPage_rejects_an_unknown_account(t *testing.T) {
	app := newTestApp(t, conceptToday)

	values := newEntryFormValues(999, "Miete", "2000", "2026-01-01", "month", "1")
	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := rec.Body.String()

	if !contains(body, "Dieses Konto gibt es nicht.") {
		t.Errorf("the page does not show the message of the unknown account:\n%s", body)
	}
	if !contains(body, `id="account_id-error"`) {
		t.Errorf("the page does not show the error at the field:\n%s", body)
	}
	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
}

// TestEntriesPage_holds_the_end_date_in_the_form checks that the form of an
// entry holds its end date.
func TestEntriesPage_holds_the_end_date_in_the_form(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	entry := monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	entry.EndsOn = dayPtr(t, "2026-12-31")
	id := createEntry(t, app, entry)

	body := app.get(t, "/entries/"+strconv.FormatInt(id, 10)).Body.String()
	if !contains(body, `value="2026-12-31"`) {
		t.Errorf("the form does not hold the end date of the entry:\n%s", body)
	}
}

// TestEntriesPage_keeps_the_account_of_a_deactivated_entry_in_the_form
// checks that the account of an entry stays in the form, even when the
// account is deactivated.
func TestEntriesPage_keeps_the_account_of_a_deactivated_entry_in_the_form(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	if err := app.store.SetAccountActive(context.Background(), lifestyle.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	body := app.get(t, "/entries/"+strconv.FormatInt(id, 10)).Body.String()
	if !contains(body, ">Lifestyle (deaktiviert)<") {
		t.Errorf("the form does not offer the deactivated account of the entry:\n%s", body)
	}
}

// TestOriginCheck_rejects_an_origin_that_is_no_url checks the answer for an
// origin that the parser cannot read.
func TestOriginCheck_rejects_an_origin_that_is_no_url(t *testing.T) {
	app := newTestApp(t, conceptToday)

	req := newRequest(http.MethodPost, "/accounts", strings.NewReader(url.Values{"name": {"Fremd"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://[::1")

	rec := app.serve(t, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if len(app.accounts(t)) != 3 {
		t.Error("the database holds more accounts than before")
	}
}

// TestParseAmount_rejects_a_separator_without_a_fraction checks the amount
// with a separator and no digits behind it.
func TestParseAmount_rejects_a_separator_without_a_fraction(t *testing.T) {
	for _, value := range []string{"12.", "12,"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseAmount(value); !errors.Is(err, ErrAmountFormat) {
				t.Errorf("ParseAmount(%q) failed with %v, want %v", value, err, ErrAmountFormat)
			}
		})
	}
}
