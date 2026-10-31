package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// entryFormValues returns the values of a form for the entry of an account.
func newEntryFormValues(accountID int64, name, amount, start, unit, interval string) url.Values {
	return url.Values{
		"account_id": {strconv.FormatInt(accountID, 10)},
		"name":       {name},
		"amount":     {amount},
		"start_date": {start},
		"unit":       {unit},
		"interval":   {interval},
	}
}

// TestEntriesPage_creates_an_entry checks that a valid form stores the entry
// and answers with the redirect to the list.
func TestEntriesPage_creates_an_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	values := newEntryFormValues(lifestyle.ID, "Miete", "2000", "2026-01-01", "month", "1")
	values.Set("notes", "eine Notiz")

	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/entries" {
		t.Errorf("the answer points to %q, want \"/entries\"", location)
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Fatalf("the database holds %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Name != "Miete" {
		t.Errorf("the name of the entry is %q, want \"Miete\"", entry.Name)
	}
	if entry.AccountID != lifestyle.ID {
		t.Errorf("the entry belongs to the account %d, want %d", entry.AccountID, lifestyle.ID)
	}
	if entry.AmountMinor != 200000 {
		t.Errorf("the entry costs %d Rappen, want 200000", entry.AmountMinor)
	}
	if got := plan.FormatDate(entry.StartDate); got != "2026-01-01" {
		t.Errorf("the entry starts on %q, want 2026-01-01", got)
	}
	if entry.Unit != plan.Month {
		t.Errorf("the unit of the entry is %q, want %q", entry.Unit, plan.Month)
	}
	if entry.Interval != 1 {
		t.Errorf("the interval of the entry is %d, want 1", entry.Interval)
	}
	if entry.Notes != "eine Notiz" {
		t.Errorf("the note of the entry is %q, want \"eine Notiz\"", entry.Notes)
	}
}

// TestEntriesPage_stores_the_interval_and_the_end_date_of_a_once_entry checks
// the rule of the unit "once".
func TestEntriesPage_stores_the_interval_and_the_end_date_of_a_once_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	values := newEntryFormValues(lifestyle.ID, "Zahnarzt", "300", "2026-10-15", "once", "7")
	values.Set("ends_on", "2026-12-31")

	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Fatalf("the database holds %d entries, want 1", len(entries))
	}
	if entries[0].Interval != 1 {
		t.Errorf("the interval of the entry is %d, want 1", entries[0].Interval)
	}
	if entries[0].EndsOn != nil {
		t.Errorf("the entry ends on %v, want no end date", entries[0].EndsOn)
	}
}

// TestEntriesPage_groups_the_entries_by_the_account checks the list of the
// entries.
func TestEntriesPage_groups_the_entries_by_the_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)

	body := app.get(t, "/entries").Body.String()

	titles := cardTitles(body)
	if len(titles) != 3 {
		t.Fatalf("the list holds %d groups, want 3: %v", len(titles), titles)
	}
	want := accountNames(accounts)
	for i, name := range want {
		if !strings.HasPrefix(titles[i], name) {
			t.Errorf("the group %d is %q, want the account %q", i, titles[i], name)
		}
	}

	// The three entries of the account Lifestyle are in its group, the
	// entries of the other accounts in theirs.
	sections := entrySections(body)
	if len(sections) != 3 {
		t.Fatalf("the list holds %d sections, want 3", len(sections))
	}
	if !contains(sections[0], "Miete") || !contains(sections[0], "Krankenkasse") || !contains(sections[0], "Internet") {
		t.Errorf("the group of the account Lifestyle does not hold its entries:\n%s", sections[0])
	}
	if contains(sections[0], "Autoversicherung") {
		t.Errorf("the group of the account Lifestyle holds the entry of another account:\n%s", sections[0])
	}
	if !contains(sections[1], "Autoversicherung") {
		t.Errorf("the group of the account Sparen does not hold its entry:\n%s", sections[1])
	}
	if !contains(sections[2], "Serafe") {
		t.Errorf("the group of the account Steuern does not hold its entry:\n%s", sections[2])
	}
}

// TestEntriesPage_shows_the_empty_group_of_an_account checks that an account
// without entries gets a group of its own.
func TestEntriesPage_shows_the_empty_group_of_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	body := app.get(t, "/entries").Body.String()

	titles := cardTitles(body)
	if len(titles) != 3 {
		t.Fatalf("the list holds %d groups, want 3: %v", len(titles), titles)
	}
	if !contains(body, "Dieses Konto hat noch keine Einträge.") {
		t.Errorf("the list does not show the empty group:\n%s", body)
	}
}

// TestEntriesPage_marks_a_deactivated_account checks that the entries of a
// deactivated account stay visible and that the account is marked.
func TestEntriesPage_marks_a_deactivated_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	body := app.get(t, "/entries").Body.String()

	sections := entrySections(body)
	if len(sections) != 3 {
		t.Fatalf("the list holds %d groups, want 3", len(sections))
	}
	if !contains(sections[1], "Autoversicherung") {
		t.Errorf("the group of the deactivated account does not hold its entry:\n%s", sections[1])
	}
	if !contains(sections[1], "deaktiviert") {
		t.Errorf("the group of the deactivated account is not marked:\n%s", sections[1])
	}
	for _, title := range cardTitleHTML(body) {
		if strings.Contains(title, "deaktiviert") && !strings.Contains(title, "Sparen") {
			t.Errorf("another account is marked as deactivated: %q", title)
		}
	}
}

// TestEntriesPage_follows_the_order_of_the_accounts checks that the groups
// follow the order of the accounts.
func TestEntriesPage_follows_the_order_of_the_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.MoveAccountUp(context.Background(), sparen.ID); err != nil {
		t.Fatalf("MoveAccountUp() failed: %v", err)
	}

	body := app.get(t, "/entries").Body.String()

	titles := cardTitles(body)
	if len(titles) != 3 {
		t.Fatalf("the list holds %d groups, want 3: %v", len(titles), titles)
	}
	// The account Sparen was the second one, moving it up puts it in front
	// of the account Lifestyle.
	if !strings.HasPrefix(titles[0], "Sparen") || !strings.HasPrefix(titles[1], "Lifestyle") {
		t.Errorf("the groups are %v, want the account Sparen in front", titles)
	}
}

// TestEntriesPage_fills_the_form_of_an_entry checks the form of the edit of
// an entry.
func TestEntriesPage_fills_the_form_of_an_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	id := createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 1250,
		StartDate: mustDay(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
		Notes: "eine Notiz",
	})

	rec := app.get(t, "/entries/"+strconv.FormatInt(id, 10))
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, `value="Miete"`) {
		t.Errorf("the form does not hold the name of the entry:\n%s", body)
	}
	// The amount is written in francs without a thousands separator, so the
	// parser of the application reads it back.
	if !contains(body, `value="12.50"`) {
		t.Errorf("the form does not hold the amount of the entry:\n%s", body)
	}
	if !contains(body, `value="2026-01-01"`) {
		t.Errorf("the form does not hold the start date of the entry:\n%s", body)
	}
	if !contains(body, `value="1"`) {
		t.Errorf("the form does not hold the interval of the entry:\n%s", body)
	}
	if !contains(body, "eine Notiz") {
		t.Errorf("the form does not hold the note of the entry:\n%s", body)
	}
	if !contains(body, `action="/entries/`+strconv.FormatInt(id, 10)+`"`) {
		t.Errorf("the form does not send to the entry:\n%s", body)
	}
}

// TestEntriesPage_writes_the_amount_without_a_thousand_separator checks the
// format of the amount in the form.
func TestEntriesPage_writes_the_amount_without_a_thousand_separator(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	id := createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 120000,
		StartDate: mustDay(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
	})

	body := app.get(t, "/entries/"+strconv.FormatInt(id, 10)).Body.String()
	if !contains(body, `value="1200"`) {
		t.Errorf("the form does not hold the amount 1200:\n%s", body)
	}
}

// TestEntriesPage_changes_an_entry checks the update of an entry.
func TestEntriesPage_changes_an_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	sparen := accountByName(t, app, "Sparen")

	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	values := newEntryFormValues(sparen.ID, "Miete neu", "250.50", "2026-02-01", "month", "2")
	rec := app.postForm(t, "/entries/"+strconv.FormatInt(id, 10), values)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/entries" {
		t.Errorf("the answer points to %q, want \"/entries\"", location)
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Fatalf("the database holds %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Name != "Miete neu" {
		t.Errorf("the name of the entry is %q, want \"Miete neu\"", entry.Name)
	}
	if entry.AccountID != sparen.ID {
		t.Errorf("the entry belongs to the account %d, want %d", entry.AccountID, sparen.ID)
	}
	if entry.AmountMinor != 25050 {
		t.Errorf("the entry costs %d Rappen, want 25050", entry.AmountMinor)
	}
	if got := plan.FormatDate(entry.StartDate); got != "2026-02-01" {
		t.Errorf("the entry starts on %q, want 2026-02-01", got)
	}
	if entry.Interval != 2 {
		t.Errorf("the interval of the entry is %d, want 2", entry.Interval)
	}
}

// TestEntriesPage_asks_before_it_deletes_an_entry checks the page of the
// confirmation of the deletion.
func TestEntriesPage_asks_before_it_deletes_an_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	rec := app.get(t, "/entries/"+strconv.FormatInt(id, 10)+"/delete")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !contains(body, "Soll der Eintrag") {
		t.Errorf("the page does not ask for the confirmation:\n%s", body)
	}
	if !contains(body, `action="/entries/`+strconv.FormatInt(id, 10)+`/delete"`) {
		t.Errorf("the page does not send to the deletion:\n%s", body)
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Errorf("the database holds %d entries, want the entry untouched", len(entries))
	}
}

// TestEntriesPage_deletes_an_entry checks the deletion.
func TestEntriesPage_deletes_an_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	rec := app.postForm(t, "/entries/"+strconv.FormatInt(id, 10)+"/delete", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/entries" {
		t.Errorf("the answer points to %q, want \"/entries\"", location)
	}
	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
}

// TestEntriesPage_shows_the_error_of_the_form checks the answer of a form
// with an error.
func TestEntriesPage_shows_the_error_of_the_form(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	values := newEntryFormValues(lifestyle.ID, "", "0", "2026-13-01", "day", "0")
	values.Set("notes", "eine Notiz")

	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := rec.Body.String()

	for _, field := range []string{"name", "amount", "start_date", "unit", "interval"} {
		if !contains(body, `id="`+field+`-error"`) {
			t.Errorf("the page does not show the error of the field %q:\n%s", field, body)
		}
	}
	if !contains(body, "Der Name darf nicht leer sein.") {
		t.Errorf("the page does not show the message of the name:\n%s", body)
	}
	if !contains(body, "Der Betrag muss grösser als 0 sein.") {
		t.Errorf("the page does not show the message of the amount:\n%s", body)
	}
	// The values of the visitor stay in the form.
	if !contains(body, "eine Notiz") {
		t.Errorf("the page does not hold the note of the visitor:\n%s", body)
	}
	if !contains(body, `value="2026-13-01"`) {
		t.Errorf("the page does not hold the start date of the visitor:\n%s", body)
	}

	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
}

// TestEntriesPage_answers_with_404_for_an_unknown_entry checks the answer
// for the identifiers that name no entry.
func TestEntriesPage_answers_with_404_for_an_unknown_entry(t *testing.T) {
	identifiers := []string{"999", "abc", "0", "-1"}

	for _, identifier := range identifiers {
		t.Run(identifier, func(t *testing.T) {
			app := newTestApp(t, conceptToday)

			rec := app.get(t, "/entries/"+identifier)
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status of the page is %d, want %d", rec.Code, http.StatusNotFound)
			}

			rec = app.postForm(t, "/entries/"+identifier, url.Values{})
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status of the update is %d, want %d", rec.Code, http.StatusNotFound)
			}

			rec = app.get(t, "/entries/"+identifier+"/delete")
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status of the confirmation is %d, want %d", rec.Code, http.StatusNotFound)
			}

			rec = app.postForm(t, "/entries/"+identifier+"/delete", url.Values{})
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status of the deletion is %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

// TestEntriesPage_answers_with_413_for_a_big_body checks the limit of the
// body of a form.
func TestEntriesPage_answers_with_413_for_a_big_body(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	values := newEntryFormValues(lifestyle.ID, "Miete", "2000", "2026-01-01", "month", "1")
	values.Set("notes", strings.Repeat("a", maxFormBytes+1))

	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
}

// TestEntriesPage_accepts_a_body_at_the_limit checks that a body of the
// allowed size is not rejected.
func TestEntriesPage_accepts_a_body_at_the_limit(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	values := newEntryFormValues(lifestyle.ID, "Miete", "2000", "2026-01-01", "month", "1")
	// The note is limited to 1000 characters, so a body of the size of the
	// limit needs another field.
	values.Set("notes", strings.Repeat("a", maxFormBytes-200))

	rec := app.postForm(t, "/entries", values)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("the status is %d, want %d for the too long note", rec.Code, http.StatusUnprocessableEntity)
	}
}

// TestEntriesPage_answers_with_400_for_an_unreadable_body checks the answer
// for a body that cannot be read as a form.
//
// The body holds an escape that is not valid, so the form cannot be parsed.
// The answer is the status 400 and not the status 422 of a form with an
// error, because the server did not see the values of the visitor at all.
func TestEntriesPage_answers_with_400_for_an_unreadable_body(t *testing.T) {
	app := newTestApp(t, conceptToday)

	for _, target := range []string{"/entries", "/accounts"} {
		t.Run(target, func(t *testing.T) {
			req := newRequest(http.MethodPost, target, strings.NewReader("name=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", testOrigin)

			rec := app.serve(t, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("the status is %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}

	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds the entries %v, want none", entryNames(entries))
	}
	if accounts := app.accounts(t); len(accounts) != 3 {
		t.Errorf("the database holds the accounts %v, want only the three default accounts", accountNames(accounts))
	}
}

// TestRhythmText_describes_the_recurrence checks the text of the rhythmus of
// the list of the entries.
func TestRhythmText_describes_the_recurrence(t *testing.T) {
	cases := []struct {
		name     string
		unit     plan.Unit
		interval int
		start    string
		want     string
	}{
		{name: "once", unit: plan.Once, interval: 1, start: "2026-10-15", want: "einmalig am 15.10.2026"},
		{name: "week", unit: plan.Week, interval: 1, start: "2026-10-01", want: "wöchentlich"},
		{name: "every two weeks", unit: plan.Week, interval: 2, start: "2026-10-01", want: "alle 2 Wochen"},
		{name: "month", unit: plan.Month, interval: 1, start: "2026-10-01", want: "monatlich"},
		{name: "every three months", unit: plan.Month, interval: 3, start: "2026-10-01", want: "alle 3 Monate"},
		{name: "year", unit: plan.Year, interval: 1, start: "2026-10-01", want: "jährlich"},
		{name: "every two years", unit: plan.Year, interval: 2, start: "2026-10-01", want: "alle 2 Jahre"},
		{name: "an unknown unit", unit: plan.Unit("day"), interval: 1, start: "2026-10-01", want: "unbekannt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := plan.Entry{
				Name:        "Eintrag",
				AmountMinor: 10000,
				StartDate:   mustDay(t, tc.start),
				Unit:        tc.unit,
				Interval:    tc.interval,
			}
			if got := RhythmText(entry); got != tc.want {
				t.Errorf("RhythmText() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEntriesPage_shows_the_state_of_the_entries checks the tags of the list
// of the entries.
func TestEntriesPage_shows_the_state_of_the_entries(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	done := plan.Entry{
		AccountID: lifestyle.ID, Name: "Zahnarzt", AmountMinor: 30000,
		StartDate: mustDay(t, "2026-09-15"), Unit: plan.Once, Interval: 1,
	}
	createEntry(t, app, done)

	ended := monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	ended.EndsOn = dayPtr(t, "2026-09-30")
	createEntry(t, app, ended)

	running := monthlyEntry(t, lifestyle.ID, "Internet", 6000, "2026-01-15")
	createEntry(t, app, running)

	body := app.get(t, "/entries").Body.String()

	// The once entry of the past is done, the recurring entry whose end date
	// lies before today is done and over, the entry that still runs is
	// neither of them.
	once := entryRowByName(t, body, "Zahnarzt")
	if !contains(once.status, "erledigt") {
		t.Errorf("the entry of the past is not marked as done: %q", once.status)
	}
	if contains(once.status, "beendet") {
		t.Errorf("the entry of the past is marked as over: %q", once.status)
	}

	endedRow := entryRowByName(t, body, "Miete")
	if !contains(endedRow.status, "erledigt") || !contains(endedRow.status, "beendet") {
		t.Errorf("the ended entry is not marked as done and over: %q", endedRow.status)
	}

	runningRow := entryRowByName(t, body, "Internet")
	if runningRow.status != "" {
		t.Errorf("the running entry carries the tags %q, want none", runningRow.status)
	}
}

// TestEntriesPage_shows_the_new_form checks the form of a new entry.
func TestEntriesPage_shows_the_new_form(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.get(t, "/entries/new")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, `action="/entries"`) {
		t.Errorf("the form does not send to the list:\n%s", body)
	}
	if !contains(body, `name="account_id"`) {
		t.Errorf("the form has no field for the account:\n%s", body)
	}
	if !contains(body, `value="2026-10-01"`) {
		t.Errorf("the form does not hold the day of the clock:\n%s", body)
	}
	for _, unit := range []string{"once", "week", "month", "year"} {
		if !contains(body, `value="`+unit+`"`) {
			t.Errorf("the form does not offer the unit %q:\n%s", unit, body)
		}
	}
}

// TestEntriesPage_offers_only_the_active_accounts checks the selection of the
// account in the form of a new entry.
func TestEntriesPage_offers_only_the_active_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	body := app.get(t, "/entries/new").Body.String()

	if !contains(body, "Lifestyle") {
		t.Errorf("the form does not offer the active account:\n%s", body)
	}
	if contains(body, ">Sparen<") {
		t.Errorf("the form offers the deactivated account:\n%s", body)
	}
}
