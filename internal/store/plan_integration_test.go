package store

import (
	"context"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// The example of the concept (section 5, state of the 1st of october 2026).
//
// The amounts of the concept are given in CHF, the database stores them in
// Rappen, therefore the rent of 2000 CHF carries 200000 Rappen.
func TestConceptExample_stored_entries_produce_the_totals_of_the_concept(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	sparen := accountByName(t, s, "Sparen")
	steuern := accountByName(t, s, "Steuern")

	entries := []plan.Entry{
		{
			AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 200000,
			StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: lifestyle.ID, Name: "Krankenkasse", AmountMinor: 45000,
			StartDate: mustDate(t, "2026-01-05"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: lifestyle.ID, Name: "Internet", AmountMinor: 6000,
			StartDate: mustDate(t, "2026-01-15"), Unit: plan.Month, Interval: 1,
		},
		{
			AccountID: sparen.ID, Name: "Autoversicherung", AmountMinor: 120000,
			StartDate: mustDate(t, "2026-02-15"), Unit: plan.Year, Interval: 1,
		},
		{
			AccountID: steuern.ID, Name: "Serafe", AmountMinor: 33500,
			StartDate: mustDate(t, "2026-03-20"), Unit: plan.Year, Interval: 1,
		},
	}
	for _, entry := range entries {
		createEntry(t, s, entry)
	}

	stored, err := s.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if len(stored) != len(entries) {
		t.Fatalf("the database holds %d entries, want %d", len(stored), len(entries))
	}
	for i, entry := range entries {
		got := stored[i]
		if got.Name != entry.Name {
			t.Errorf("the entry %d is %q, want %q", i, got.Name, entry.Name)
		}
		if got.AccountID != entry.AccountID {
			t.Errorf("the entry %q belongs to the account %d, want %d", got.Name, got.AccountID, entry.AccountID)
		}
		if got.AmountMinor != entry.AmountMinor {
			t.Errorf("the entry %q costs %d Rappen, want %d", got.Name, got.AmountMinor, entry.AmountMinor)
		}
		if gotDate, wantDate := plan.FormatDate(got.StartDate), plan.FormatDate(entry.StartDate); gotDate != wantDate {
			t.Errorf("the entry %q starts on %q, want %q", got.Name, gotDate, wantDate)
		}
		if got.Unit != entry.Unit {
			t.Errorf("the entry %q has the unit %q, want %q", got.Name, got.Unit, entry.Unit)
		}
		if got.Interval != entry.Interval {
			t.Errorf("the entry %q has the interval %d, want %d", got.Name, got.Interval, entry.Interval)
		}
	}

	// The totals of the concept (section 5):
	//
	//   Account   | Month | Week | Day
	//   Lifestyle |  2510 |  577 |  82
	//   Sparen    |   100 |   23 |   3
	//   Steuern   |    28 |    6 |   1
	//   Total     |  2638 |  607 |  87
	byAccount := map[string][]plan.Entry{}
	for _, entry := range stored {
		byAccount[entry.AccountName] = append(byAccount[entry.AccountName], entry.Entry)
	}

	cases := []struct {
		account string
		month   int64
		week    int64
		day     int64
	}{
		{account: "Lifestyle", month: 2510, week: 577, day: 82},
		{account: "Sparen", month: 100, week: 23, day: 3},
		{account: "Steuern", month: 28, week: 6, day: 1},
	}
	for _, tc := range cases {
		t.Run(tc.account, func(t *testing.T) {
			need := plan.Totals(byAccount[tc.account])
			if got := plan.RoundCHF(need.Month); got != tc.month {
				t.Errorf("Month = %d CHF, want %d CHF", got, tc.month)
			}
			if got := plan.RoundCHF(need.Week); got != tc.week {
				t.Errorf("Week = %d CHF, want %d CHF", got, tc.week)
			}
			if got := plan.RoundCHF(need.Day); got != tc.day {
				t.Errorf("Day = %d CHF, want %d CHF", got, tc.day)
			}
		})
	}

	all := make([]plan.Entry, 0, len(stored))
	for _, entry := range stored {
		all = append(all, entry.Entry)
	}
	need := plan.Totals(all)
	if got := plan.RoundCHF(need.Month); got != 2638 {
		t.Errorf("Month total = %d CHF, want 2638 CHF", got)
	}
	if got := plan.RoundCHF(need.Week); got != 607 {
		t.Errorf("Week total = %d CHF, want 607 CHF", got)
	}
	if got := plan.RoundCHF(need.Day); got != 87 {
		t.Errorf("Day total = %d CHF, want 87 CHF", got)
	}
}
