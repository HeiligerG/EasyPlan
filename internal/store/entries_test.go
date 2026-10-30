package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

func TestCreateEntry_stores_the_entry_and_reads_it_back(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	endsOn := mustDate(t, "2026-12-31")
	entry := plan.Entry{
		AccountID:   lifestyle.ID,
		Name:        "Miete",
		AmountMinor: 200000,
		StartDate:   mustDate(t, "2026-01-01"),
		Unit:        plan.Month,
		Interval:    1,
		EndsOn:      &endsOn,
		Notes:       "Wohnung mit Balkon",
	}

	id := createEntry(t, s, entry)

	stored, err := s.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEntry() failed: %v", err)
	}
	if stored.ID != id {
		t.Errorf("the id is %d, want %d", stored.ID, id)
	}
	if stored.AccountID != lifestyle.ID {
		t.Errorf("the account id is %d, want %d", stored.AccountID, lifestyle.ID)
	}
	if stored.Name != "Miete" {
		t.Errorf("the name is %q, want %q", stored.Name, "Miete")
	}
	if stored.AmountMinor != 200000 {
		t.Errorf("the amount is %d Rappen, want 200000", stored.AmountMinor)
	}
	if got := plan.FormatDate(stored.StartDate); got != "2026-01-01" {
		t.Errorf("the start date is %q, want %q", got, "2026-01-01")
	}
	if stored.Unit != plan.Month {
		t.Errorf("the unit is %q, want %q", stored.Unit, plan.Month)
	}
	if stored.Interval != 1 {
		t.Errorf("the interval is %d, want 1", stored.Interval)
	}
	if stored.EndsOn == nil {
		t.Fatal("the end date is missing")
	}
	if got := plan.FormatDate(*stored.EndsOn); got != "2026-12-31" {
		t.Errorf("the end date is %q, want %q", got, "2026-12-31")
	}
	if stored.Notes != "Wohnung mit Balkon" {
		t.Errorf("the notes are %q, want %q", stored.Notes, "Wohnung mit Balkon")
	}
	if stored.AccountName != "Lifestyle" {
		t.Errorf("the account name is %q, want %q", stored.AccountName, "Lifestyle")
	}
	if !stored.AccountActive {
		t.Error("the account is reported as inactive")
	}
}

func TestCreateEntry_stores_a_missing_end_date_as_null(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	entry := planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01")

	id := createEntry(t, s, entry)

	if endsOn := rawEndsOn(t, s, id); endsOn.Valid {
		t.Errorf("the stored end date is %q, want NULL", endsOn.String)
	}
	stored, err := s.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEntry() failed: %v", err)
	}
	if stored.EndsOn != nil {
		t.Errorf("the end date is %q, want none", plan.FormatDate(*stored.EndsOn))
	}
}

func TestCreateEntry_stores_the_end_date(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	endsOn := mustDate(t, "2026-12-31")
	entry := planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	entry.EndsOn = &endsOn

	id := createEntry(t, s, entry)

	stored := rawEndsOn(t, s, id)
	if !stored.Valid {
		t.Fatal("the end date is NULL, want 2026-12-31")
	}
	if stored.String != "2026-12-31" {
		t.Errorf("the stored end date is %q, want %q", stored.String, "2026-12-31")
	}
}

func TestCreateEntry_rejects_invalid_entries(t *testing.T) {
	lifestyleID := int64(1)
	cases := []struct {
		name  string
		entry plan.Entry
	}{
		{
			name: "amount zero",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "Miete", AmountMinor: 0,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
			},
		},
		{
			name: "amount negative",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "Miete", AmountMinor: -100,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
			},
		},
		{
			name: "interval zero",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "Miete", AmountMinor: 200000,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 0,
			},
		},
		{
			name: "unknown unit",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "Miete", AmountMinor: 200000,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Unit("day"), Interval: 1,
			},
		},
		{
			name: "empty name",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "  ", AmountMinor: 200000,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
			},
		},
		{
			name: "end date before the start date",
			entry: plan.Entry{
				AccountID: lifestyleID, Name: "Miete", AmountMinor: 200000,
				StartDate: mustDate(t, "2026-01-01"), Unit: plan.Month, Interval: 1,
				EndsOn: datePointer(mustDate(t, "2025-12-31")),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newMemoryStore(t)
			lifestyle := accountByName(t, s, "Lifestyle")
			entry := tc.entry
			entry.AccountID = lifestyle.ID

			_, err := s.CreateEntry(context.Background(), entry)
			if err == nil {
				t.Fatal("CreateEntry() succeeded, want an error")
			}
			if got := countEntries(t, s); got != 0 {
				t.Errorf("the database holds %d entries, want none", got)
			}
		})
	}
}

func TestCreateEntry_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)
	entry := planEntryForAccount(t, 4242, "Miete", 200000, "2026-01-01")

	_, err := s.CreateEntry(context.Background(), entry)
	if !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("CreateEntry() returned %v, want ErrUnknownAccount", err)
	}
	if got := countEntries(t, s); got != 0 {
		t.Errorf("the database holds %d entries, want none", got)
	}
}

func TestGetEntry_reports_an_unknown_entry(t *testing.T) {
	s := newMemoryStore(t)

	_, err := s.GetEntry(context.Background(), 4242)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetEntry() returned %v, want ErrNotFound", err)
	}
}

func TestUpdateEntry_changes_the_entry(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	sparen := accountByName(t, s, "Sparen")
	id := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	endsOn := mustDate(t, "2027-06-30")
	update := plan.Entry{
		ID:          id,
		AccountID:   sparen.ID,
		Name:        "Miete alt",
		AmountMinor: 250000,
		StartDate:   mustDate(t, "2026-02-01"),
		Unit:        plan.Week,
		Interval:    2,
		EndsOn:      &endsOn,
		Notes:       "Umzug",
	}
	if err := s.UpdateEntry(context.Background(), update); err != nil {
		t.Fatalf("UpdateEntry() failed: %v", err)
	}

	stored, err := s.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEntry() failed: %v", err)
	}
	if stored.AccountID != sparen.ID {
		t.Errorf("the account id is %d, want %d", stored.AccountID, sparen.ID)
	}
	if stored.AccountName != "Sparen" {
		t.Errorf("the account name is %q, want %q", stored.AccountName, "Sparen")
	}
	if stored.Name != "Miete alt" {
		t.Errorf("the name is %q, want %q", stored.Name, "Miete alt")
	}
	if stored.AmountMinor != 250000 {
		t.Errorf("the amount is %d Rappen, want 250000", stored.AmountMinor)
	}
	if got := plan.FormatDate(stored.StartDate); got != "2026-02-01" {
		t.Errorf("the start date is %q, want %q", got, "2026-02-01")
	}
	if stored.Unit != plan.Week {
		t.Errorf("the unit is %q, want %q", stored.Unit, plan.Week)
	}
	if stored.Interval != 2 {
		t.Errorf("the interval is %d, want 2", stored.Interval)
	}
	if stored.EndsOn == nil {
		t.Fatal("the end date is missing")
	}
	if got := plan.FormatDate(*stored.EndsOn); got != "2027-06-30" {
		t.Errorf("the end date is %q, want %q", got, "2027-06-30")
	}
	if stored.Notes != "Umzug" {
		t.Errorf("the notes are %q, want %q", stored.Notes, "Umzug")
	}
}

func TestUpdateEntry_reports_an_unknown_entry(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	update := planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	update.ID = 4242

	err := s.UpdateEntry(context.Background(), update)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateEntry() returned %v, want ErrNotFound", err)
	}
	if got := countEntries(t, s); got != 0 {
		t.Errorf("the database holds %d entries, want none", got)
	}
}

func TestUpdateEntry_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	id := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	update := planEntryForAccount(t, 4242, "Miete", 300000, "2026-01-01")
	update.ID = id
	err := s.UpdateEntry(context.Background(), update)
	if !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("UpdateEntry() returned %v, want ErrUnknownAccount", err)
	}
	stored, err := s.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEntry() failed: %v", err)
	}
	if stored.AccountID != lifestyle.ID || stored.AmountMinor != 200000 {
		t.Errorf("the entry was changed although the update failed: %+v", stored)
	}
}

func TestDeleteEntry_removes_the_entry(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	id := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	if err := s.DeleteEntry(context.Background(), id); err != nil {
		t.Fatalf("DeleteEntry() failed: %v", err)
	}
	if _, err := s.GetEntry(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetEntry() returned %v, want ErrNotFound", err)
	}
	if got := countEntries(t, s); got != 0 {
		t.Errorf("the database holds %d entries, want none", got)
	}
}

func TestDeleteEntry_reports_an_unknown_entry(t *testing.T) {
	s := newMemoryStore(t)

	err := s.DeleteEntry(context.Background(), 4242)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteEntry() returned %v, want ErrNotFound", err)
	}
}

func TestListEntries_sorts_the_entries_deterministically(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	sparen := accountByName(t, s, "Sparen")

	createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Internet", 6000, "2026-01-15"))
	createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))
	createEntry(t, s, planEntryForAccount(t, sparen.ID, "Autoversicherung", 120000, "2026-02-15"))

	entries, err := s.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if got := entryNames(entries); !equalStrings(got, []string{"Miete", "Internet", "Autoversicherung"}) {
		t.Errorf("the entries are %v, want [Miete Internet Autoversicherung]", got)
	}

	// The order of the accounts comes first, the start date and the
	// identifier break the ties inside an account.
	again, err := s.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if got := entryNames(again); !equalStrings(got, []string{"Miete", "Internet", "Autoversicherung"}) {
		t.Errorf("the second call returned %v, want the same order", got)
	}
}

func TestListEntriesByAccount_sorts_the_entries_by_start_date_and_identifier(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")

	third := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Dritte", 100, "2026-03-01"))
	first := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Erste", 100, "2026-01-01"))
	secondA := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Zweite A", 100, "2026-02-01"))
	secondB := createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Zweite B", 100, "2026-02-01"))

	entries, err := s.ListEntriesByAccount(context.Background(), lifestyle.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount() failed: %v", err)
	}
	want := []int64{first, secondA, secondB, third}
	if len(entries) != len(want) {
		t.Fatalf("the list holds %d entries, want %d: %v", len(entries), len(want), entryNames(entries))
	}
	for i, id := range want {
		if entries[i].ID != id {
			t.Errorf("the entry %d is %q, want the id %d", i, entries[i].Name, id)
		}
	}
}

func TestListEntriesByAccount_returns_an_empty_list_for_an_account_without_entries(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")

	entries, err := s.ListEntriesByAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("ListEntriesByAccount() failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the list holds %d entries, want none", len(entries))
	}

	unknown, err := s.ListEntriesByAccount(context.Background(), 4242)
	if err != nil {
		t.Fatalf("ListEntriesByAccount() of an unknown account failed: %v", err)
	}
	if len(unknown) != 0 {
		t.Errorf("the list of an unknown account holds %d entries, want none", len(unknown))
	}
}

func TestListEntries_reports_the_name_and_the_state_of_the_account(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))
	if err := s.SetAccountActive(context.Background(), lifestyle.ID, false); err != nil {
		t.Fatalf("SetAccountActive(false) failed: %v", err)
	}

	entries, err := s.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("the list holds %d entries, want 1", len(entries))
	}
	if entries[0].AccountName != "Lifestyle" {
		t.Errorf("the account name is %q, want %q", entries[0].AccountName, "Lifestyle")
	}
	if entries[0].AccountActive {
		t.Error("the account is reported as active")
	}

	byAccount, err := s.ListEntriesByAccount(context.Background(), lifestyle.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount() failed: %v", err)
	}
	if len(byAccount) != 1 {
		t.Fatalf("the list holds %d entries, want 1", len(byAccount))
	}
	if byAccount[0].AccountActive {
		t.Error("the account is reported as active")
	}
}

// datePointer returns a pointer to a day.
func datePointer(day time.Time) *time.Time {
	return &day
}
