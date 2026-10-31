package web

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// TestEntriesPage_refuses_a_deactivated_account checks that a form cannot
// hang a new entry on a deactivated account.
//
// The browser only offers the active accounts, so a prepared request is the
// only way to reach a deactivated account. The server checks the account
// again, so the database does not hold an entry that no summary shows.
func TestEntriesPage_refuses_a_deactivated_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	steuern := accountByName(t, app, "Steuern")
	if err := app.store.SetAccountActive(context.Background(), steuern.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	values := newEntryFormValues(steuern.ID, "Serafe", "335", "2026-03-20", "year", "1")
	rec := app.postForm(t, "/entries", values)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if !contains(rec.Body.String(), "Dieses Konto ist deaktiviert.") {
		t.Errorf("the answer does not name the deactivated account:\n%s", rec.Body.String())
	}

	if entries := app.entries(t); len(entries) != 0 {
		t.Errorf("the database holds %d entries, want none", len(entries))
	}
}

// TestEntriesPage_refuses_another_deactivated_account_on_the_update checks
// that an entry cannot be moved to another deactivated account.
func TestEntriesPage_refuses_another_deactivated_account_on_the_update(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	steuern := accountByName(t, app, "Steuern")
	if err := app.store.SetAccountActive(context.Background(), steuern.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	id := createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	values := newEntryFormValues(steuern.ID, "Miete", "2000", "2026-01-01", "month", "1")
	rec := app.postForm(t, "/entries/"+strconv.FormatInt(id, 10), values)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if !contains(rec.Body.String(), "Dieses Konto ist deaktiviert.") {
		t.Errorf("the answer does not name the deactivated account:\n%s", rec.Body.String())
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Fatalf("the database holds %d entries, want 1", len(entries))
	}
	if entries[0].AccountID != lifestyle.ID {
		t.Errorf("the entry belongs to the account %d, want %d", entries[0].AccountID, lifestyle.ID)
	}
}

// TestEntriesPage_keeps_the_deactivated_account_of_the_entry checks that an
// entry may stay on the deactivated account of the entry.
//
// The account of the edited entry is offered in the form, so the visitor can
// change the other values of the entry without moving it to another account.
func TestEntriesPage_keeps_the_deactivated_account_of_the_entry(t *testing.T) {
	app := newTestApp(t, conceptToday)
	steuern := accountByName(t, app, "Steuern")

	id := createEntry(t, app, monthlyEntry(t, steuern.ID, "Serafe", 33500, "2026-03-20"))

	if err := app.store.SetAccountActive(context.Background(), steuern.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	values := newEntryFormValues(steuern.ID, "Serafe", "400", "2026-03-20", "year", "1")
	rec := app.postForm(t, "/entries/"+strconv.FormatInt(id, 10), values)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusSeeOther)
	}

	entries := app.entries(t)
	if len(entries) != 1 {
		t.Fatalf("the database holds %d entries, want 1", len(entries))
	}
	if entries[0].AccountID != steuern.ID {
		t.Errorf("the entry belongs to the account %d, want %d", entries[0].AccountID, steuern.ID)
	}
	if entries[0].AmountMinor != 40000 {
		t.Errorf("the entry costs %d Rappen, want 40000", entries[0].AmountMinor)
	}
}

// TestValidateEntryForm_reports_the_state_of_the_account checks the account
// of the form as a pure function.
//
// The offered accounts are the accounts the visitor may choose: the active
// accounts and, when an entry is edited, the account of that entry. An
// account that is not in the list of all accounts does not exist, an account
// that exists but is not offered is deactivated.
func TestValidateEntryForm_reports_the_state_of_the_account(t *testing.T) {
	accounts := []store.Account{
		{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
		{ID: 2, Name: "Steuern", SortOrder: 2, IsActive: false},
	}
	offered := []store.Account{accounts[0]}

	cases := []struct {
		name      string
		accountID string
		offered   []store.Account
		want      string
	}{
		{
			name:      "the deactivated account",
			accountID: "2",
			offered:   offered,
			want:      "Dieses Konto ist deaktiviert.",
		},
		{
			name:      "the unknown account",
			accountID: "99",
			offered:   offered,
			want:      "Dieses Konto gibt es nicht.",
		},
		{
			name:      "the deactivated account of the edited entry",
			accountID: "2",
			offered:   accounts,
			want:      "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.AccountID = tc.accountID

			result := ValidateEntryForm(values, tc.offered, accounts)

			message, ok := result.Errors[fieldAccount]
			if tc.want == "" {
				if ok {
					t.Errorf("the form has the error %q of the account, want none", message)
				}
				return
			}
			if !ok {
				t.Fatalf("the form has the errors %v, want the error of the field %q", result.Errors, fieldAccount)
			}
			if message != tc.want {
				t.Errorf("the error of the account is %q, want %q", message, tc.want)
			}
		})
	}
}
