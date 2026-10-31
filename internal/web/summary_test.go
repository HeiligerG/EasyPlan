package web

import (
	"context"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// TestBuildSummary_matches_the_example_of_the_concept checks the values of
// the dashboard against the example of Konzept.md §5 and of
// docs/features/berechnung.md.
func TestBuildSummary_matches_the_example_of_the_concept(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	summary := BuildSummary(accounts, entries, conceptToday, 30)

	if summary.Entries != 5 {
		t.Errorf("the summary counts %d entries, want 5", summary.Entries)
	}
	if len(summary.Rows) != 3 {
		t.Fatalf("the summary holds %d rows, want 3", len(summary.Rows))
	}

	cases := []struct {
		account  string
		month    int64
		week     int64
		day      int64
		reserve  int64
		nextName string
		nextDay  string
	}{
		{account: "Lifestyle", month: 2510, week: 577, day: 82, reserve: 251000, nextName: "Miete", nextDay: "2026-10-01"},
		{account: "Sparen", month: 100, week: 23, day: 3, reserve: 0, nextName: "Autoversicherung", nextDay: "2027-02-15"},
		{account: "Steuern", month: 28, week: 6, day: 1, reserve: 0, nextName: "Serafe", nextDay: "2027-03-20"},
	}
	for i, tc := range cases {
		t.Run(tc.account, func(t *testing.T) {
			row := summary.Rows[i]
			if row.Name != tc.account {
				t.Fatalf("the row %d is %q, want %q", i, row.Name, tc.account)
			}
			if got := plan.RoundCHF(row.Need.Month); got != tc.month {
				t.Errorf("the demand per month is %d, want %d", got, tc.month)
			}
			if got := plan.RoundCHF(row.Need.Week); got != tc.week {
				t.Errorf("the demand per week is %d, want %d", got, tc.week)
			}
			if got := plan.RoundCHF(row.Need.Day); got != tc.day {
				t.Errorf("the demand per day is %d, want %d", got, tc.day)
			}
			if row.Reserve != tc.reserve {
				t.Errorf("the reserve is %d Rappen, want %d", row.Reserve, tc.reserve)
			}
			if !row.HasNext {
				t.Fatalf("the account %q has no next payment", tc.account)
			}
			if row.NextName != tc.nextName {
				t.Errorf("the next payment is %q, want %q", row.NextName, tc.nextName)
			}
			if got := plan.FormatDate(row.NextDate); got != tc.nextDay {
				t.Errorf("the next payment is on %q, want %q", got, tc.nextDay)
			}
		})
	}
}

// TestBuildSummary_calculates_the_total_over_the_unrounded_values checks the
// rounding difference of the concept: the total of the daily values is 87,
// the sum of the rows is 86.
func TestBuildSummary_calculates_the_total_over_the_unrounded_values(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	summary := BuildSummary(accounts, entries, conceptToday, 30)

	if got := plan.RoundCHF(summary.Total.Need.Month); got != 2638 {
		t.Errorf("the demand per month is %d, want 2638", got)
	}
	if got := plan.RoundCHF(summary.Total.Need.Week); got != 607 {
		t.Errorf("the demand per week is %d, want 607", got)
	}
	if got := plan.RoundCHF(summary.Total.Need.Day); got != 87 {
		t.Errorf("the demand per day is %d, want 87", got)
	}
	if summary.Total.Reserve != 251000 {
		t.Errorf("the reserve is %d Rappen, want 251000", summary.Total.Reserve)
	}

	// The sum of the rounded rows is 86, the rounded total is 87. The
	// difference is wanted, the total is not the sum of the shown rows.
	var sum int64
	for _, row := range summary.Rows {
		sum += plan.RoundCHF(row.Need.Day)
	}
	if sum != 86 {
		t.Errorf("the sum of the daily values of the rows is %d, want 86", sum)
	}
}

// TestBuildSummary_shows_the_next_payment_of_the_day_after checks the rule of
// ADR-0004: on the 2nd of october the next payment of the account Lifestyle
// is the Krankenkasse of the 5th.
func TestBuildSummary_shows_the_next_payment_of_the_day_after(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	summary := BuildSummary(accounts, entries, mustDay(t, "2026-10-02"), 30)

	row := summary.Rows[0]
	if row.Name != "Lifestyle" {
		t.Fatalf("the first row is %q, want \"Lifestyle\"", row.Name)
	}
	if row.NextName != "Krankenkasse" {
		t.Errorf("the next payment is %q, want \"Krankenkasse\"", row.NextName)
	}
	if got := plan.FormatDate(row.NextDate); got != "2026-10-05" {
		t.Errorf("the next payment is on %q, want 2026-10-05", got)
	}
}

// TestBuildSummary_leaves_out_the_account_without_entries checks that an
// account without entries gets a row with zeros and no payment.
func TestBuildSummary_leaves_out_the_account_without_entries(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	createEntry(t, app, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	accounts := app.accounts(t)
	entries := app.entries(t)
	summary := BuildSummary(accounts, entries, conceptToday, 30)

	if len(summary.Rows) != 3 {
		t.Fatalf("the summary holds %d rows, want 3", len(summary.Rows))
	}
	for _, row := range summary.Rows[1:] {
		if row.Need.Month != 0 || row.Need.Week != 0 || row.Need.Day != 0 {
			t.Errorf("the account %q has the demand %v, want zeros", row.Name, row.Need)
		}
		if row.Reserve != 0 {
			t.Errorf("the account %q has the reserve %d, want 0", row.Name, row.Reserve)
		}
		if row.HasNext {
			t.Errorf("the account %q has the next payment %q, want none", row.Name, row.NextName)
		}
	}
}

// TestBuildSummary_leaves_out_the_deactivated_account checks that a
// deactivated account and its entries are not part of the rows, the total and
// the reserve.
func TestBuildSummary_leaves_out_the_deactivated_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	accounts = app.accounts(t)
	entries = app.entries(t)
	summary := BuildSummary(accounts, entries, conceptToday, 30)

	for _, row := range summary.Rows {
		if row.Name == "Sparen" {
			t.Errorf("the summary holds the deactivated account %q", row.Name)
		}
	}
	if len(summary.Rows) != 2 {
		t.Errorf("the summary holds %d rows, want 2", len(summary.Rows))
	}
	// The total is the total of the two remaining accounts.
	if got := plan.RoundCHF(summary.Total.Need.Month); got != 2538 {
		t.Errorf("the demand per month is %d, want 2538", got)
	}
	if summary.Total.Reserve != 251000 {
		t.Errorf("the reserve is %d Rappen, want 251000", summary.Total.Reserve)
	}
}

// TestBuildSummary_leaves_out_the_ended_entries checks that an entry whose
// end date lies before today does not count any more.
func TestBuildSummary_leaves_out_the_ended_entries(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	ended := monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	ended.EndsOn = dayPtr(t, "2026-09-30")
	createEntry(t, app, ended)

	accounts := app.accounts(t)
	entries := app.entries(t)
	summary := BuildSummary(accounts, entries, conceptToday, 30)

	if summary.Entries != 0 {
		t.Errorf("the summary counts %d entries, want 0", summary.Entries)
	}
	for _, row := range summary.Rows {
		if row.Need.Month != 0 || row.Reserve != 0 {
			t.Errorf("the account %q counts the ended entry: %v, reserve %d", row.Name, row.Need, row.Reserve)
		}
	}
}

// TestBuildSummary_ignores_a_once_entry_in_the_past checks that a one-off
// entry of the past does not count in the demand of the period.
func TestBuildSummary_ignores_a_once_entry_in_the_past(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	past := plan.Entry{
		AccountID: lifestyle.ID, Name: "Zahnarzt", AmountMinor: 30000,
		StartDate: mustDay(t, "2026-09-15"), Unit: plan.Once, Interval: 1,
	}
	createEntry(t, app, past)

	accounts := app.accounts(t)
	entries := app.entries(t)
	summary := BuildSummary(accounts, entries, conceptToday, 30)

	row := summary.Rows[0]
	if row.Need.Month != 0 || row.Need.Week != 0 || row.Need.Day != 0 {
		t.Errorf("the demand of the account %q is %v, want zeros", row.Name, row.Need)
	}
	if row.Reserve != 0 {
		t.Errorf("the reserve is %d Rappen, want 0", row.Reserve)
	}
	if row.HasNext {
		t.Errorf("the account %q has the next payment %q, want none", row.Name, row.NextName)
	}
}

// TestBuildSummary_counts_a_once_entry_of_the_future_in_the_reserve checks
// that a one-off entry of the future counts in the reserve but not in the
// demand of the period.
func TestBuildSummary_counts_a_once_entry_of_the_future_in_the_reserve(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	future := plan.Entry{
		AccountID: lifestyle.ID, Name: "Zahnarzt", AmountMinor: 30000,
		StartDate: mustDay(t, "2026-10-15"), Unit: plan.Once, Interval: 1,
	}
	createEntry(t, app, future)

	accounts := app.accounts(t)
	entries := app.entries(t)
	summary := BuildSummary(accounts, entries, conceptToday, 30)

	row := summary.Rows[0]
	if row.Need.Month != 0 || row.Need.Week != 0 || row.Need.Day != 0 {
		t.Errorf("the demand of the account %q is %v, want zeros", row.Name, row.Need)
	}
	if row.Reserve != 30000 {
		t.Errorf("the reserve is %d Rappen, want 30000", row.Reserve)
	}
	if !row.HasNext || row.NextName != "Zahnarzt" {
		t.Errorf("the next payment is %q, want \"Zahnarzt\"", row.NextName)
	}
}

// TestTheNextPayment_breaks_the_tie_by_the_name_and_the_identifier checks that
// the next payment is deterministic when two entries are due on the same day.
func TestTheNextPayment_breaks_the_tie_by_the_name_and_the_identifier(t *testing.T) {
	start := "2026-10-01"

	t.Run("the smaller name wins", func(t *testing.T) {
		entries := []store.Entry{
			storeEntry(t, 1, 1, "Zebra", 10000, start, plan.Month, 1),
			storeEntry(t, 2, 1, "Alpha", 10000, start, plan.Month, 1),
		}
		name, date, ok := nextPayment(entries, conceptToday)
		if !ok {
			t.Fatal("there is no next payment, want one")
		}
		if name != "Alpha" {
			t.Errorf("the next payment is %q, want \"Alpha\"", name)
		}
		if got := plan.FormatDate(date); got != "2026-10-01" {
			t.Errorf("the next payment is on %q, want 2026-10-01", got)
		}
	})

	t.Run("the smaller identifier wins", func(t *testing.T) {
		entries := []store.Entry{
			storeEntry(t, 7, 1, "Alpha", 10000, start, plan.Month, 1),
			storeEntry(t, 3, 1, "Alpha", 10000, start, plan.Month, 1),
		}
		name, _, ok := nextPayment(entries, conceptToday)
		if !ok {
			t.Fatal("there is no next payment, want one")
		}
		if name != "Alpha" {
			t.Errorf("the next payment is %q, want \"Alpha\"", name)
		}
	})

	t.Run("the earlier day wins", func(t *testing.T) {
		entries := []store.Entry{
			storeEntry(t, 1, 1, "Zebra", 10000, "2026-10-05", plan.Month, 1),
			storeEntry(t, 2, 1, "Alpha", 10000, start, plan.Month, 1),
		}
		name, _, ok := nextPayment(entries, conceptToday)
		if !ok {
			t.Fatal("there is no next payment, want one")
		}
		if name != "Alpha" {
			t.Errorf("the next payment is %q, want \"Alpha\"", name)
		}
	})
}

// TestTheNextPayment_reports_the_missing_payment checks the answer for a list
// without a further payment.
func TestTheNextPayment_reports_the_missing_payment(t *testing.T) {
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Zahnarzt", 10000, "2026-09-15", plan.Once, 1),
	}
	name, date, ok := nextPayment(entries, conceptToday)
	if ok {
		t.Errorf("there is a next payment %q on %v, want none", name, date)
	}
	if !date.IsZero() {
		t.Errorf("the day of the missing payment is %v, want the zero time", date)
	}
}

// TestParseReserveDays_falls_back_to_the_default checks the values of the
// query parameter of the dashboard.
func TestParseReserveDays_falls_back_to_the_default(t *testing.T) {
	cases := []struct {
		value string
		want  int
	}{
		{value: "", want: 30},
		{value: "7", want: 7},
		{value: "30", want: 30},
		{value: "60", want: 60},
		{value: "90", want: 90},
		{value: "abc", want: 30},
		{value: "15", want: 30},
		{value: "0", want: 30},
		{value: "-7", want: 30},
		{value: " 7 ", want: 30},
		{value: "7.0", want: 30},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := parseReserveDays(tc.value); got != tc.want {
				t.Errorf("parseReserveDays(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

// TestBuildSummary_uses_the_day_of_the_clock checks that the summary is
// calculated for the given day and not for another one.
func TestBuildSummary_uses_the_day_of_the_clock(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	summary := BuildSummary(accounts, entries, mustDay(t, "2026-10-20"), 30)
	row := summary.Rows[0]
	// On the 20th of october the miete of the 1st is paid, so the next
	// payment of the account Lifestyle is the miete of the 1st of november.
	if row.NextName != "Miete" {
		t.Errorf("the next payment is %q, want \"Miete\"", row.NextName)
	}
	if got := plan.FormatDate(row.NextDate); got != "2026-11-01" {
		t.Errorf("the next payment is on %q, want 2026-11-01", got)
	}
}

// TestBuildSummary_holds_the_day_of_the_next_payment checks that the day of
// the next payment is a calendar day.
func TestBuildSummary_holds_the_day_of_the_next_payment(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	summary := BuildSummary(accounts, entries, conceptToday, 30)
	for _, row := range summary.Rows {
		if row.NextDate.Hour() != 0 || row.NextDate.Minute() != 0 || row.NextDate.Second() != 0 {
			t.Errorf("the next payment of %q is on %v, want a calendar day", row.Name, row.NextDate)
		}
		if row.NextDate.Location() != time.UTC {
			t.Errorf("the next payment of %q is in %v, want UTC", row.Name, row.NextDate.Location())
		}
	}
}
