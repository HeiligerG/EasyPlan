package plan

import (
	"fmt"
	"slices"
	"testing"
)

// renderEvents turns the events into comparable strings. The order of the
// fields follows the sort order of the timeline: date, account, name, id.
func renderEvents(events []Event) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, fmt.Sprintf("%s|%d|%s|%d",
			FormatDate(event.Date), event.Entry.AccountID, event.Entry.Name, event.Entry.ID))
	}
	return out
}

func TestTimeline(t *testing.T) {
	// All entries of this table have a term on the 1st of october 2026, one
	// entry has a term on the 5th. The expected order is:
	//   date, then account, then name, then id.
	entries := []Entry{
		recurring(t, 10, 5, "Miete", 200000, "2026-10-01", Month, 1),
		recurring(t, 2, 5, "Abos", 4500, "2026-10-01", Month, 1),
		recurring(t, 1, 3, "Miete", 15000, "2026-10-01", Month, 1),
		recurring(t, 9, 5, "Miete", 6000, "2026-10-05", Month, 1),
	}

	want := []string{
		"2026-10-01|3|Miete|1",
		"2026-10-01|5|Abos|2",
		"2026-10-01|5|Miete|10",
		"2026-10-05|5|Miete|9",
	}

	got := renderEvents(Timeline(entries, day(t, "2026-10-01"), day(t, "2026-10-10")))
	if !slices.Equal(got, want) {
		t.Errorf("Timeline() =\n%v\nwant\n%v", got, want)
	}
}

func TestTimeline_sorts_by_the_id_when_account_and_name_are_equal(t *testing.T) {
	// Two entries of the same account with the same name are ordered by their
	// id, the smaller id comes first.
	entries := []Entry{
		recurring(t, 7, 2, "Gleicher Name", 1000, "2026-10-01", Month, 1),
		recurring(t, 3, 2, "Gleicher Name", 2000, "2026-10-01", Month, 1),
	}

	want := []string{
		"2026-10-01|2|Gleicher Name|3",
		"2026-10-01|2|Gleicher Name|7",
	}

	got := renderEvents(Timeline(entries, day(t, "2026-10-01"), day(t, "2026-11-01")))
	if !slices.Equal(got, want) {
		t.Errorf("Timeline() =\n%v\nwant\n%v", got, want)
	}
}

func TestTimeline_range_is_half_open(t *testing.T) {
	entries := []Entry{
		recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1),
	}

	cases := []struct {
		name string
		from string
		to   string
		want []string
	}{
		{
			name: "term on the start of the range is included",
			from: "2026-10-01",
			to:   "2026-11-01",
			want: []string{"2026-10-01|1|Miete|1"},
		},
		{
			name: "term on the end of the range is excluded",
			from: "2026-09-01",
			to:   "2026-10-01",
			want: []string{"2026-09-01|1|Miete|1"},
		},
		{
			name: "empty range with equal bounds",
			from: "2026-10-01",
			to:   "2026-10-01",
			want: []string{},
		},
		{
			name: "empty range with inverted bounds",
			from: "2026-11-01",
			to:   "2026-10-01",
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderEvents(Timeline(entries, day(t, tc.from), day(t, tc.to)))
			if !slices.Equal(got, tc.want) {
				t.Errorf("Timeline(%s .. %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestTimeline_empty_input(t *testing.T) {
	events := Timeline(nil, day(t, "2026-10-01"), day(t, "2026-11-01"))
	if len(events) != 0 {
		t.Errorf("Timeline(nil) returned %d events, want 0", len(events))
	}
}

func TestTimeline_is_deterministic(t *testing.T) {
	entries := []Entry{
		recurring(t, 10, 5, "Miete", 200000, "2026-10-01", Month, 1),
		recurring(t, 2, 5, "Abos", 4500, "2026-10-01", Month, 1),
		recurring(t, 1, 3, "Miete", 15000, "2026-10-01", Month, 1),
		recurring(t, 9, 5, "Miete", 6000, "2026-10-05", Month, 1),
		recurring(t, 4, 4, "Zahnarzt", 15000, "2026-10-20", Once, 1),
	}

	from := day(t, "2026-10-01")
	to := day(t, "2026-11-01")

	first := renderEvents(Timeline(entries, from, to))
	second := renderEvents(Timeline(entries, from, to))
	if !slices.Equal(first, second) {
		t.Errorf("two calls returned different results:\n%v\n%v", first, second)
	}

	// The order of the input must not change the result.
	shuffled := []Entry{entries[3], entries[0], entries[4], entries[2], entries[1]}
	third := renderEvents(Timeline(shuffled, from, to))
	if !slices.Equal(first, third) {
		t.Errorf("the order of the input changed the result:\n%v\n%v", first, third)
	}
}

func TestTimeline_event_carries_the_entry(t *testing.T) {
	entry := recurring(t, 42, 7, "Sparen", 120000, "2026-10-01", Month, 1)
	events := Timeline([]Entry{entry}, day(t, "2026-10-01"), day(t, "2026-11-01"))
	if len(events) != 1 {
		t.Fatalf("Timeline() returned %d events, want 1", len(events))
	}
	if events[0].Entry.ID != entry.ID {
		t.Errorf("event entry id = %d, want %d", events[0].Entry.ID, entry.ID)
	}
	if events[0].Entry.Name != entry.Name {
		t.Errorf("event entry name = %q, want %q", events[0].Entry.Name, entry.Name)
	}
	if events[0].Entry.AmountMinor != entry.AmountMinor {
		t.Errorf("event entry amount = %d, want %d", events[0].Entry.AmountMinor, entry.AmountMinor)
	}
	if got := FormatDate(events[0].Date); got != "2026-10-01" {
		t.Errorf("event date = %s, want %s", got, "2026-10-01")
	}
}

func TestIsDone(t *testing.T) {
	// Today is the 1st of october 2026.
	today := "2026-10-01"

	cases := []struct {
		name  string
		entry Entry
		want  bool
	}{
		{
			name:  "once entry in the past is done",
			entry: recurring(t, 1, 1, "Zahnarzt", 15000, "2026-09-01", Once, 1),
			want:  true,
		},
		{
			name:  "once entry today is not done yet",
			entry: recurring(t, 2, 1, "Zahnarzt", 15000, "2026-10-01", Once, 1),
			want:  false,
		},
		{
			name:  "once entry in the future is not done yet",
			entry: recurring(t, 3, 1, "Zahnarzt", 15000, "2026-11-01", Once, 1),
			want:  false,
		},
		{
			name: "recurring entry with an end date in the past is done",
			entry: withEndsOn(t,
				recurring(t, 4, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-09-30"),
			want: true,
		},
		{
			name: "recurring entry with an end date today is not done yet",
			entry: withEndsOn(t,
				recurring(t, 5, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-10-01"),
			want: false,
		},
		{
			name: "recurring entry with an end date in the future is not done yet",
			entry: withEndsOn(t,
				recurring(t, 6, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-10-02"),
			want: false,
		},
		{
			name:  "recurring entry without an end date never is done",
			entry: recurring(t, 7, 1, "Miete", 200000, "2026-01-01", Month, 1),
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDone(tc.entry, day(t, today)); got != tc.want {
				t.Errorf("IsDone(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
