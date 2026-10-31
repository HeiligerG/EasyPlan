package web

import (
	"strings"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// validEntryForm returns a form with valid values, so a test can change the
// one value it wants to check.
func validEntryForm(accountID int64) EntryForm {
	return EntryForm{
		AccountID: "1",
		Name:      "Miete",
		Amount:    "2000",
		StartDate: "2026-01-01",
		Unit:      "month",
		Interval:  "1",
	}
}

// testAccounts returns the accounts of the tests.
//
// The account Steuern is deactivated, so it is not part of the accounts the
// form offers.
func testAccounts() []store.Account {
	return []store.Account{
		{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
		{ID: 2, Name: "Sparen", SortOrder: 2, IsActive: true},
		{ID: 3, Name: "Steuern", SortOrder: 3, IsActive: false},
	}
}

// activeTestAccounts returns the accounts that the form of a new entry
// offers: the active accounts of the tests.
func activeTestAccounts() []store.Account {
	accounts := testAccounts()
	offered := make([]store.Account, 0, len(accounts))
	for _, account := range accounts {
		if account.IsActive {
			offered = append(offered, account)
		}
	}
	return offered
}

// TestValidateEntryForm_accepts_a_valid_form checks that a form with valid
// values has no error and describes the entry.
func TestValidateEntryForm_accepts_a_valid_form(t *testing.T) {
	accounts := testAccounts()

	for _, unit := range []string{"once", "week", "month", "year"} {
		t.Run(unit, func(t *testing.T) {
			values := validEntryForm(1)
			values.Unit = unit

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			if len(result.Errors) != 0 {
				t.Fatalf("the form has the errors %v, want none", result.Errors)
			}
			if result.Error != "" {
				t.Errorf("the form has the message %q, want none", result.Error)
			}
			if result.Entry.AccountID != 1 {
				t.Errorf("the account of the entry is %d, want 1", result.Entry.AccountID)
			}
			if result.Entry.Name != "Miete" {
				t.Errorf("the name of the entry is %q, want \"Miete\"", result.Entry.Name)
			}
			if result.Entry.AmountMinor != 200000 {
				t.Errorf("the amount of the entry is %d, want 200000", result.Entry.AmountMinor)
			}
			if result.Entry.Unit != plan.Unit(unit) {
				t.Errorf("the unit of the entry is %q, want %q", result.Entry.Unit, unit)
			}
			if got := plan.FormatDate(result.Entry.StartDate); got != "2026-01-01" {
				t.Errorf("the entry starts on %q, want 2026-01-01", got)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_account checks the field of
// the account.
func TestValidateEntryForm_reports_the_error_of_the_account(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name      string
		accountID string
	}{
		{name: "empty", accountID: ""},
		{name: "blank", accountID: "   "},
		{name: "not a number", accountID: "abc"},
		{name: "zero", accountID: "0"},
		{name: "negative", accountID: "-1"},
		{name: "unknown", accountID: "99"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.AccountID = tc.accountID

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			if _, ok := result.Errors[fieldAccount]; !ok {
				t.Errorf("the form has the errors %v, want an error of the field %q", result.Errors, fieldAccount)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_name checks the field of the
// name.
func TestValidateEntryForm_reports_the_error_of_the_name(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "blank", value: "   ", wantErr: true},
		{name: "a hundred characters", value: strings.Repeat("a", 100), wantErr: false},
		{name: "a hundred and one characters", value: strings.Repeat("a", 101), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Name = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldName]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldName, tc.wantErr)
			}
			if !tc.wantErr && result.Entry.Name != tc.value {
				t.Errorf("the name of the entry is %q, want %q", result.Entry.Name, tc.value)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_amount checks the field of
// the amount.
func TestValidateEntryForm_reports_the_error_of_the_amount(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "zero with decimals", value: "0.00", wantErr: true},
		{name: "negative", value: "-5", wantErr: true},
		{name: "too many decimals", value: "12.505", wantErr: true},
		{name: "no number", value: "abc", wantErr: true},
		{name: "too large", value: "10000000000.01", wantErr: true},
		{name: "the maximum", value: "10000000000", wantErr: false},
		{name: "a small amount", value: "0.01", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Amount = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldAmount]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldAmount, tc.wantErr)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_start_date checks the field
// of the start date.
func TestValidateEntryForm_reports_the_error_of_the_start_date(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "no date", value: "abc", wantErr: true},
		{name: "the wrong format", value: "01.10.2026", wantErr: true},
		{name: "an incomplete date", value: "2026-1-1", wantErr: true},
		{name: "an impossible day", value: "2026-02-30", wantErr: true},
		{name: "an impossible month", value: "2026-13-01", wantErr: true},
		{name: "a valid date", value: "2026-10-01", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.StartDate = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldStart]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldStart, tc.wantErr)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_unit checks the field of the
// unit.
func TestValidateEntryForm_reports_the_error_of_the_unit(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "unknown", value: "day", wantErr: true},
		{name: "no unit", value: "fortnight", wantErr: true},
		{name: "once", value: "once", wantErr: false},
		{name: "week", value: "week", wantErr: false},
		{name: "month", value: "month", wantErr: false},
		{name: "year", value: "year", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Unit = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldUnit]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldUnit, tc.wantErr)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_interval checks the field of
// the interval.
func TestValidateEntryForm_reports_the_error_of_the_interval(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "no number", value: "abc", wantErr: true},
		{name: "above the maximum", value: "10001", wantErr: true},
		{name: "the maximum", value: "10000", wantErr: false},
		{name: "one", value: "1", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Unit = "month"
			values.Interval = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldInterval]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldInterval, tc.wantErr)
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_end_date checks the field of
// the end date.
func TestValidateEntryForm_reports_the_error_of_the_end_date(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: false},
		{name: "no date", value: "abc", wantErr: true},
		{name: "the wrong format", value: "01.10.2026", wantErr: true},
		{name: "an impossible day", value: "2026-02-30", wantErr: true},
		{name: "before the start date", value: "2025-12-31", wantErr: true},
		{name: "the start date", value: "2026-01-01", wantErr: false},
		{name: "after the start date", value: "2026-12-31", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Unit = "month"
			values.EndsOn = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldEndsOn]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldEndsOn, tc.wantErr)
			}
			if !tc.wantErr && tc.value != "" {
				if result.Entry.EndsOn == nil {
					t.Error("the entry has no end date, want one")
				} else if got := plan.FormatDate(*result.Entry.EndsOn); got != tc.value {
					t.Errorf("the entry ends on %q, want %q", got, tc.value)
				}
			}
		})
	}
}

// TestValidateEntryForm_reports_the_error_of_the_note checks the field of the
// note.
func TestValidateEntryForm_reports_the_error_of_the_note(t *testing.T) {
	accounts := testAccounts()

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: false},
		{name: "a thousand characters", value: strings.Repeat("a", 1000), wantErr: false},
		{name: "a thousand and one characters", value: strings.Repeat("a", 1001), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := validEntryForm(1)
			values.Notes = tc.value

			result := ValidateEntryForm(values, activeTestAccounts(), accounts)
			_, ok := result.Errors[fieldNotes]
			if ok != tc.wantErr {
				t.Errorf("the form has the errors %v, want an error of the field %q: %t", result.Errors, fieldNotes, tc.wantErr)
			}
		})
	}
}

// TestValidateEntryForm_ignores_the_interval_and_the_end_date_of_a_once_entry
// checks the rule of the unit "once": the interval is 1 and the entry has no
// end date.
func TestValidateEntryForm_ignores_the_interval_and_the_end_date_of_a_once_entry(t *testing.T) {
	accounts := testAccounts()

	values := validEntryForm(1)
	values.Unit = "once"
	values.Interval = "abc"
	values.EndsOn = "2020-01-01"

	result := ValidateEntryForm(values, activeTestAccounts(), accounts)
	if len(result.Errors) != 0 {
		t.Fatalf("the form has the errors %v, want none", result.Errors)
	}
	if result.Entry.Interval != 1 {
		t.Errorf("the interval of the entry is %d, want 1", result.Entry.Interval)
	}
	if result.Entry.EndsOn != nil {
		t.Errorf("the entry ends on %v, want no end date", result.Entry.EndsOn)
	}
}

// TestValidateEntryForm_trims_the_values checks that the name and the note
// are trimmed.
func TestValidateEntryForm_trims_the_values(t *testing.T) {
	accounts := testAccounts()

	values := validEntryForm(1)
	values.Name = "  Miete  "
	values.Notes = "  eine Notiz  "

	result := ValidateEntryForm(values, activeTestAccounts(), accounts)
	if len(result.Errors) != 0 {
		t.Fatalf("the form has the errors %v, want none", result.Errors)
	}
	if result.Entry.Name != "Miete" {
		t.Errorf("the name of the entry is %q, want \"Miete\"", result.Entry.Name)
	}
	if result.Entry.Notes != "eine Notiz" {
		t.Errorf("the note of the entry is %q, want \"eine Notiz\"", result.Entry.Notes)
	}
}

// TestValidateEntryForm_reports_every_field_at_once checks that the form
// reports the errors of all fields and not only of the first one.
func TestValidateEntryForm_reports_every_field_at_once(t *testing.T) {
	accounts := testAccounts()

	values := EntryForm{
		AccountID: "abc",
		Name:      "",
		Amount:    "0",
		StartDate: "2026-13-01",
		Unit:      "day",
		Interval:  "0",
		EndsOn:    "abc",
		Notes:     strings.Repeat("a", 1001),
	}
	result := ValidateEntryForm(values, activeTestAccounts(), accounts)

	for _, field := range []string{
		fieldAccount, fieldName, fieldAmount, fieldStart,
		fieldUnit, fieldInterval, fieldEndsOn, fieldNotes,
	} {
		if _, ok := result.Errors[field]; !ok {
			t.Errorf("the form has the errors %v, want an error of the field %q", result.Errors, field)
		}
	}
}

// TestValidateEntryForm_is_pure checks that the validation does not change
// its arguments.
func TestValidateEntryForm_is_pure(t *testing.T) {
	accounts := testAccounts()
	values := validEntryForm(1)

	first := ValidateEntryForm(values, activeTestAccounts(), accounts)
	second := ValidateEntryForm(values, activeTestAccounts(), accounts)

	if len(first.Errors) != len(second.Errors) {
		t.Fatalf("the two runs have %d and %d errors, want the same number", len(first.Errors), len(second.Errors))
	}
	if first.Entry != second.Entry {
		t.Error("the two runs describe different entries")
	}
	if values.Name != "Miete" || values.Amount != "2000" {
		t.Errorf("the form changed: %+v", values)
	}
}
