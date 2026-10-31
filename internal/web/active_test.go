package web

import (
	"reflect"
	"testing"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// TestActiveEntries_leaves_out_the_entries_of_a_deactivated_account checks
// the filter of the summaries.
func TestActiveEntries_leaves_out_the_entries_of_a_deactivated_account(t *testing.T) {
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Miete", 200000, "2026-01-01", plan.Month, 1),
		storeEntry(t, 2, 2, "Serafe", 33500, "2026-03-20", plan.Year, 1),
	}
	entries[1].AccountActive = false

	active := ActiveEntries(entries, conceptToday)
	if len(active) != 1 {
		t.Fatalf("the filter keeps %d entries, want 1", len(active))
	}
	if active[0].Name != "Miete" {
		t.Errorf("the filter keeps the entry %q, want \"Miete\"", active[0].Name)
	}
}

// TestActiveEntries_leaves_out_the_ended_entries checks that the filter uses
// the end date of the entry.
func TestActiveEntries_leaves_out_the_ended_entries(t *testing.T) {
	ended := storeEntry(t, 1, 1, "Miete", 200000, "2026-01-01", plan.Month, 1)
	ended.EndsOn = dayPtr(t, "2026-09-30")

	running := storeEntry(t, 2, 1, "Internet", 6000, "2026-01-15", plan.Month, 1)
	running.EndsOn = dayPtr(t, "2026-10-15")

	active := ActiveEntries([]store.Entry{ended, running}, conceptToday)
	if len(active) != 1 {
		t.Fatalf("the filter keeps %d entries, want 1", len(active))
	}
	if active[0].Name != "Internet" {
		t.Errorf("the filter keeps the entry %q, want \"Internet\"", active[0].Name)
	}
}

// TestActiveEntries_keeps_an_entry_that_ends_today checks the border of the
// end date.
//
// The entry still occurs on the last day, so it counts until the day after it.
func TestActiveEntries_keeps_an_entry_that_ends_today(t *testing.T) {
	entry := storeEntry(t, 1, 1, "Internet", 6000, "2026-01-15", plan.Month, 1)
	entry.EndsOn = dayPtr(t, "2026-10-01")

	active := ActiveEntries([]store.Entry{entry}, conceptToday)
	if len(active) != 1 {
		t.Fatalf("the filter keeps %d entries, want 1", len(active))
	}
}

// TestActiveEntries_keeps_a_once_entry_of_the_past checks the rule of the
// unit "once".
//
// A one-off entry has no end date, so it is not left out of the summaries.
// The calculations of the plan package decide on their own that a one-off
// entry of the past does not occur any more.
func TestActiveEntries_keeps_a_once_entry_of_the_past(t *testing.T) {
	entry := storeEntry(t, 1, 1, "Zahnarzt", 30000, "2026-09-15", plan.Once, 1)

	active := ActiveEntries([]store.Entry{entry}, conceptToday)
	if len(active) != 1 {
		t.Fatalf("the filter keeps %d entries, want 1", len(active))
	}
	if active[0].Name != "Zahnarzt" {
		t.Errorf("the filter keeps the entry %q, want \"Zahnarzt\"", active[0].Name)
	}
}

// TestActiveEntries_keeps_the_order_of_the_entries checks that the filter
// does not mix up the entries.
func TestActiveEntries_keeps_the_order_of_the_entries(t *testing.T) {
	entries := []store.Entry{
		storeEntry(t, 3, 1, "C", 10000, "2026-01-01", plan.Month, 1),
		storeEntry(t, 1, 1, "A", 10000, "2026-01-01", plan.Month, 1),
		storeEntry(t, 2, 1, "B", 10000, "2026-01-01", plan.Month, 1),
	}
	active := ActiveEntries(entries, conceptToday)
	if len(active) != 3 {
		t.Fatalf("the filter keeps %d entries, want 3", len(active))
	}
	for i, want := range []string{"C", "A", "B"} {
		if active[i].Name != want {
			t.Errorf("the entry %d is %q, want %q", i, active[i].Name, want)
		}
	}
}

// TestActiveEntries_does_not_change_the_entries checks that the filter does
// not change the entries it gets.
//
// The filter returns its own list, so the list of the caller keeps its
// entries and its order.
func TestActiveEntries_does_not_change_the_entries(t *testing.T) {
	ended := storeEntry(t, 1, 1, "Miete", 200000, "2026-01-01", plan.Month, 1)
	ended.EndsOn = dayPtr(t, "2026-09-30")
	deactivated := storeEntry(t, 2, 2, "Serafe", 33500, "2026-03-20", plan.Year, 1)
	deactivated.AccountActive = false

	entries := []store.Entry{ended, deactivated, storeEntry(t, 3, 1, "Internet", 6000, "2026-01-15", plan.Month, 1)}
	before := cloneEntries(entries)

	active := ActiveEntries(entries, conceptToday)

	if !reflect.DeepEqual(entries, before) {
		t.Errorf("the filter changed the entries:\n%+v\nwant:\n%+v", entries, before)
	}
	if len(entries) != len(before) {
		t.Errorf("the filter holds %d entries, want %d", len(entries), len(before))
	}
	if len(active) != 1 {
		t.Fatalf("the filter keeps %d entries, want 1", len(active))
	}
}

// TestActiveEntries_works_without_entries checks the empty list.
func TestActiveEntries_works_without_entries(t *testing.T) {
	if active := ActiveEntries(nil, conceptToday); len(active) != 0 {
		t.Errorf("the filter keeps %d entries, want none", len(active))
	}
}

// cloneEntries returns a copy of the entries with its own end dates, so a
// test can compare the entries before and after a call.
func cloneEntries(entries []store.Entry) []store.Entry {
	clone := make([]store.Entry, len(entries))
	for i, entry := range entries {
		clone[i] = entry
		if entry.EndsOn != nil {
			endsOn := *entry.EndsOn
			clone[i].EndsOn = &endsOn
		}
	}
	return clone
}
