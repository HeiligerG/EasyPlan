package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// TestValidateAccountName_trims_the_name checks the trimming of the name.
func TestValidateAccountName_trims_the_name(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "a name", raw: "Lifestyle", want: "Lifestyle"},
		{name: "blank spaces around", raw: "  Lifestyle  ", want: "Lifestyle"},
		{name: "empty", raw: "", wantErr: true},
		{name: "blank", raw: "   ", wantErr: true},
		{name: "a hundred characters", raw: strings.Repeat("a", 100), want: strings.Repeat("a", 100)},
		{name: "a hundred and one characters", raw: strings.Repeat("a", 101), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, message := ValidateAccountName(tc.raw)
			if tc.wantErr {
				if message == "" {
					t.Errorf("ValidateAccountName(%q) accepted the name, want an error", tc.raw)
				}
				return
			}
			if message != "" {
				t.Errorf("ValidateAccountName(%q) failed with %q, want no message", tc.raw, message)
			}
			if got != tc.want {
				t.Errorf("ValidateAccountName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestAccountRows_counts_the_entries_and_marks_the_border checks the rows of
// the list of the accounts.
func TestAccountRows_counts_the_entries_and_marks_the_border(t *testing.T) {
	accounts := []store.Account{
		{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
		{ID: 2, Name: "Sparen", SortOrder: 2, IsActive: false},
		{ID: 3, Name: "Steuern", SortOrder: 3, IsActive: true},
	}
	entries := []store.Entry{
		{Entry: planEntry(1, 1, "Miete")},
		{Entry: planEntry(2, 1, "Krankenkasse")},
		{Entry: planEntry(3, 3, "Serafe")},
	}

	rows := AccountRows(accounts, entries)
	if len(rows) != 3 {
		t.Fatalf("the list holds %d rows, want 3", len(rows))
	}

	if rows[0].Name != "Lifestyle" || rows[0].EntryCount != 2 {
		t.Errorf("the row %q counts %d entries, want 2", rows[0].Name, rows[0].EntryCount)
	}
	if rows[1].Name != "Sparen" || rows[1].EntryCount != 0 {
		t.Errorf("the row %q counts %d entries, want 0", rows[1].Name, rows[1].EntryCount)
	}
	if rows[2].Name != "Steuern" || rows[2].EntryCount != 1 {
		t.Errorf("the row %q counts %d entries, want 1", rows[2].Name, rows[2].EntryCount)
	}

	if !rows[0].IsFirst || rows[0].IsLast {
		t.Errorf("the first row is first: %t, last: %t, want true and false", rows[0].IsFirst, rows[0].IsLast)
	}
	if rows[1].IsFirst || rows[1].IsLast {
		t.Errorf("the middle row is first: %t, last: %t, want false and false", rows[1].IsFirst, rows[1].IsLast)
	}
	if rows[2].IsFirst || !rows[2].IsLast {
		t.Errorf("the last row is first: %t, last: %t, want false and true", rows[2].IsFirst, rows[2].IsLast)
	}

	if rows[0].Status != "aktiv" || rows[2].Status != "aktiv" {
		t.Errorf("the states are %q and %q, want \"aktiv\"", rows[0].Status, rows[2].Status)
	}
	if rows[1].Status != "deaktiviert" {
		t.Errorf("the state of the deactivated account is %q, want \"deaktiviert\"", rows[1].Status)
	}
	if !rows[0].IsActive || rows[1].IsActive {
		t.Errorf("the flags of the accounts are %t and %t, want true and false", rows[0].IsActive, rows[1].IsActive)
	}
}

// TestAccountsPage_creates_an_account checks the creation of an account.
func TestAccountsPage_creates_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {" Ferien "}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/accounts" {
		t.Errorf("the answer points to %q, want \"/accounts\"", location)
	}

	accounts := app.accounts(t)
	if len(accounts) != 4 {
		t.Fatalf("the database holds %d accounts, want 4", len(accounts))
	}
	if accounts[3].Name != "Ferien" {
		t.Errorf("the new account is %q, want \"Ferien\"", accounts[3].Name)
	}
	if !accounts[3].IsActive {
		t.Error("the new account is not active, want an active account")
	}
}

// TestAccountsPage_rejects_an_empty_name checks the answer of a form with an
// empty name.
func TestAccountsPage_rejects_an_empty_name(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {"   "}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := rec.Body.String()

	if !contains(body, "Der Name darf nicht leer sein.") {
		t.Errorf("the page does not show the message of the empty name:\n%s", body)
	}
	if !contains(body, `id="new-name-error"`) {
		t.Errorf("the page does not show the error at the field:\n%s", body)
	}
	// The list of the accounts stays visible.
	if !contains(body, "Lifestyle") {
		t.Errorf("the page does not show the list of the accounts:\n%s", body)
	}
	if len(app.accounts(t)) != 3 {
		t.Error("the database holds more accounts than before")
	}
}

// TestAccountsPage_rejects_a_duplicate_name checks the answer of a form with a
// name that is already taken.
func TestAccountsPage_rejects_a_duplicate_name(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {"Lifestyle"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := rec.Body.String()

	if !contains(body, "Dieser Name ist bereits vergeben.") {
		t.Errorf("the page does not show the message of the taken name:\n%s", body)
	}
	if len(app.accounts(t)) != 3 {
		t.Error("the database holds more accounts than before")
	}
}

// TestAccountsPage_rejects_a_too_long_name checks the answer of a form with a
// name that is too long.
func TestAccountsPage_rejects_a_too_long_name(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {strings.Repeat("a", 101)}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if len(app.accounts(t)) != 3 {
		t.Error("the database holds more accounts than before")
	}
}

// TestAccountsPage_treats_the_case_of_the_name checks that two names that
// differ in their case are two different names.
func TestAccountsPage_treats_the_case_of_the_name(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {"lifestyle"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}

	accounts := app.accounts(t)
	if len(accounts) != 4 {
		t.Fatalf("the database holds %d accounts, want 4", len(accounts))
	}
	if accounts[3].Name != "lifestyle" {
		t.Errorf("the new account is %q, want \"lifestyle\"", accounts[3].Name)
	}
}

// TestAccountsPage_renames_an_account checks the renaming.
func TestAccountsPage_renames_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	rec := app.get(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10))
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !contains(body, `value="Lifestyle"`) {
		t.Errorf("the form does not hold the name of the account:\n%s", body)
	}
	if !contains(body, `action="/accounts/`+strconv.FormatInt(lifestyle.ID, 10)+`"`) {
		t.Errorf("the form does not send to the account:\n%s", body)
	}

	rec = app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10), url.Values{"name": {"Alltag"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/accounts" {
		t.Errorf("the answer points to %q, want \"/accounts\"", location)
	}

	accounts := app.accounts(t)
	if accounts[0].Name != "Alltag" {
		t.Errorf("the account is %q, want \"Alltag\"", accounts[0].Name)
	}
}

// TestAccountsPage_rejects_a_duplicate_name_on_the_rename checks the answer
// of a rename with a name that is already taken.
func TestAccountsPage_rejects_a_duplicate_name_on_the_rename(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	rec := app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10), url.Values{"name": {"Sparen"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := rec.Body.String()
	if !contains(body, "Dieser Name ist bereits vergeben.") {
		t.Errorf("the page does not show the message of the taken name:\n%s", body)
	}
	if !contains(body, `value="Sparen"`) {
		t.Errorf("the page does not hold the name of the visitor:\n%s", body)
	}
	if accounts := app.accounts(t); accounts[0].Name != "Lifestyle" {
		t.Errorf("the account is %q, want \"Lifestyle\"", accounts[0].Name)
	}
}

// TestAccountsPage_moves_an_account checks the order of the accounts.
func TestAccountsPage_moves_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	sparen := accountByName(t, app, "Sparen")

	rec := app.postForm(t, "/accounts/"+strconv.FormatInt(sparen.ID, 10)+"/up", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/accounts" {
		t.Errorf("the answer points to %q, want \"/accounts\"", location)
	}

	accounts := app.accounts(t)
	if accounts[0].Name != "Sparen" || accounts[1].Name != "Lifestyle" {
		t.Errorf("the accounts are %v, want the account Sparen in front", accountNames(accounts))
	}

	rec = app.postForm(t, "/accounts/"+strconv.FormatInt(sparen.ID, 10)+"/down", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	accounts = app.accounts(t)
	if accounts[0].Name != "Lifestyle" || accounts[1].Name != "Sparen" {
		t.Errorf("the accounts are %v, want the account Sparen behind the account Lifestyle", accountNames(accounts))
	}
}

// TestAccountsPage_keeps_the_first_and_the_last_account_in_place checks that
// moving the first account up and the last account down changes nothing.
func TestAccountsPage_keeps_the_first_and_the_last_account_in_place(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	steuern := accountByName(t, app, "Steuern")
	before := accountNames(app.accounts(t))

	rec := app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10)+"/up", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	rec = app.postForm(t, "/accounts/"+strconv.FormatInt(steuern.ID, 10)+"/down", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}

	after := accountNames(app.accounts(t))
	if len(after) != len(before) {
		t.Fatalf("the database holds %d accounts, want %d", len(after), len(before))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("the account %d is %q, want %q", i, after[i], before[i])
		}
	}
}

// TestAccountsPage_deactivates_and_activates_an_account checks the state of
// an account.
func TestAccountsPage_deactivates_and_activates_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	rec := app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10)+"/deactivate", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}

	accounts := app.accounts(t)
	if accounts[0].IsActive {
		t.Error("the account is still active, want a deactivated account")
	}
	body := app.get(t, "/accounts").Body.String()
	if !contains(body, "deaktiviert") {
		t.Errorf("the list does not show the state of the account:\n%s", body)
	}
	if !contains(body, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10)+"/activate") {
		t.Errorf("the list does not offer the activation:\n%s", body)
	}

	rec = app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10)+"/activate", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if accounts = app.accounts(t); !accounts[0].IsActive {
		t.Error("the account is still deactivated, want an active account")
	}
}

// TestAccountsPage_leaves_a_deactivated_account_out_of_the_summaries checks
// that a deactivated account does not count in the dashboard and does not
// show up in the form of a new entry.
func TestAccountsPage_leaves_a_deactivated_account_out_of_the_summaries(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	lifestyle := accountByName(t, app, "Lifestyle")
	if err := app.store.SetAccountActive(context.Background(), lifestyle.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	dashboard := app.get(t, "/").Body.String()
	for _, row := range dashboardRows(dashboard) {
		if row.name == "Lifestyle" {
			t.Errorf("the dashboard shows the deactivated account %q", row.name)
		}
	}

	form := app.get(t, "/entries/new").Body.String()
	if contains(form, ">Lifestyle<") {
		t.Errorf("the form offers the deactivated account:\n%s", form)
	}

	entries := app.get(t, "/entries").Body.String()
	if !contains(entries, "Miete") {
		t.Errorf("the list of the entries does not show the entries of the deactivated account:\n%s", entries)
	}
}

// TestAccountsPage_answers_with_404_for_an_unknown_account checks the answer
// for the identifiers that name no account.
func TestAccountsPage_answers_with_404_for_an_unknown_account(t *testing.T) {
	identifiers := []string{"999", "abc", "0", "-1"}

	for _, identifier := range identifiers {
		t.Run(identifier, func(t *testing.T) {
			app := newTestApp(t, conceptToday)

			rec := app.get(t, "/accounts/"+identifier)
			if rec.Code != http.StatusNotFound {
				t.Errorf("the status of the page is %d, want %d", rec.Code, http.StatusNotFound)
			}

			for _, action := range []string{"", "/up", "/down", "/activate", "/deactivate"} {
				rec = app.postForm(t, "/accounts/"+identifier+action, url.Values{})
				if rec.Code != http.StatusNotFound {
					t.Errorf("the status of the action %q is %d, want %d", action, rec.Code, http.StatusNotFound)
				}
			}
		})
	}
}

// TestAccountsPage_answers_with_413_for_a_big_body checks the limit of the
// body of a form.
func TestAccountsPage_answers_with_413_for_a_big_body(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/accounts", url.Values{"name": {strings.Repeat("a", maxFormBytes+1)}})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if len(app.accounts(t)) != 3 {
		t.Error("the database holds more accounts than before")
	}
}

// TestAccountsPage_has_no_route_to_delete_an_account checks that the
// application does not delete an account.
func TestAccountsPage_has_no_route_to_delete_an_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	rec := app.postForm(t, "/accounts/"+strconv.FormatInt(lifestyle.ID, 10)+"/delete", url.Values{})
	if rec.Code != http.StatusNotFound {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotFound)
	}

	accounts := app.accounts(t)
	if len(accounts) != 3 {
		t.Fatalf("the database holds %d accounts, want 3", len(accounts))
	}
	if accounts[0].Name != "Lifestyle" {
		t.Errorf("the account is %q, want \"Lifestyle\"", accounts[0].Name)
	}
}

// TestAccountsPage_shows_the_list_with_the_states checks the list of the
// accounts.
func TestAccountsPage_shows_the_list_with_the_states(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	body := app.get(t, "/accounts").Body.String()

	titles := listTitles(body)
	if len(titles) != 3 {
		t.Fatalf("the list holds %d accounts, want 3: %v", len(titles), titles)
	}
	if titles[0] != "Lifestyle" || titles[1] != "Sparen" || titles[2] != "Steuern" {
		t.Errorf("the list holds %v, want the three default accounts", titles)
	}
	if !contains(body, "1 Eintrag") {
		t.Errorf("the list does not count the entries:\n%s", body)
	}
	if !contains(body, "0 Einträge") {
		t.Errorf("the list does not count the entries:\n%s", body)
	}
}
