package plan

import (
	"math"
	"testing"
)

func TestReserve(t *testing.T) {
	// Today is the 1st of october 2026. The reserve covers the half open
	// range [today, today + days).
	today := "2026-10-01"

	cases := []struct {
		name    string
		entries []Entry
		today   string
		days    int
		want    int64
	}{
		{
			// Weekly entry of 1000 Rappen: 01., 08., 15., 22. and 29.10. lie
			// in the range [01.10., 31.10.), so five terms are counted.
			name:    "weekly entry occurs five times in 30 days",
			entries: []Entry{recurring(t, 1, 1, "Waesche", 1000, "2026-10-01", Week, 1)},
			today:   today,
			days:    30,
			want:    5000,
		},
		{
			// Weekly entry of 1000 Rappen: 01., 08. and 15.10. lie in the
			// range [01.10., 16.10.), the term on 22.10. is outside.
			name:    "weekly entry occurs three times in 15 days",
			entries: []Entry{recurring(t, 1, 1, "Waesche", 1000, "2026-10-01", Week, 1)},
			today:   today,
			days:    15,
			want:    3000,
		},
		{
			// The term on the first day of the range belongs to the reserve.
			name:    "term on the first day of the range is counted",
			entries: []Entry{recurring(t, 2, 1, "Miete", 200000, "2026-01-01", Month, 1)},
			today:   today,
			days:    30,
			want:    200000,
		},
		{
			// The term on the last day of the range belongs to the next range
			// and is not counted.
			name:    "term on the last day of the range is not counted",
			entries: []Entry{recurring(t, 3, 1, "Miete", 200000, "2026-10-31", Month, 1)},
			today:   today,
			days:    30,
			want:    0,
		},
		{
			// The term on the last day of the range is counted when the range
			// is one day longer.
			name:    "term on the last day of a longer range is counted",
			entries: []Entry{recurring(t, 4, 1, "Miete", 200000, "2026-10-31", Month, 1)},
			today:   today,
			days:    31,
			want:    200000,
		},
		{
			// The end date lies in the middle of the range: 01., 08. and
			// 15.10. are counted, the term on 22.10. no longer exists.
			name: "entry with an end date in the middle of the range",
			entries: []Entry{
				withEndsOn(t, recurring(t, 5, 1, "Waesche", 1000, "2026-10-01", Week, 1), "2026-10-15"),
			},
			today: today,
			days:  30,
			want:  3000,
		},
		{
			// Several entries are added up: five terms of 1000 Rappen and one
			// term of 200000 Rappen.
			name: "several entries are added up",
			entries: []Entry{
				recurring(t, 6, 1, "Waesche", 1000, "2026-10-01", Week, 1),
				recurring(t, 7, 2, "Miete", 200000, "2026-01-01", Month, 1),
			},
			today: today,
			days:  30,
			want:  205000,
		},
		{
			// A one time expense in the range is counted once.
			name:    "once entry in the range is counted once",
			entries: []Entry{recurring(t, 8, 1, "Zahnarzt", 15000, "2026-10-20", Once, 1)},
			today:   today,
			days:    30,
			want:    15000,
		},
		{
			// A one time expense before the range is not counted.
			name:    "once entry before the range is not counted",
			entries: []Entry{recurring(t, 9, 1, "Zahnarzt", 15000, "2026-09-20", Once, 1)},
			today:   today,
			days:    30,
			want:    0,
		},
		{
			name:    "no entries",
			entries: nil,
			today:   today,
			days:    30,
			want:    0,
		},
		{
			name:    "range of zero days",
			entries: []Entry{recurring(t, 10, 1, "Waesche", 1000, "2026-10-01", Week, 1)},
			today:   today,
			days:    0,
			want:    0,
		},
		{
			name:    "negative range",
			entries: []Entry{recurring(t, 11, 1, "Waesche", 1000, "2026-10-01", Week, 1)},
			today:   today,
			days:    -30,
			want:    0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Reserve(tc.entries, day(t, tc.today), tc.days)
			if got != tc.want {
				t.Errorf("Reserve(%s, %d days) = %d Rappen, want %d Rappen", tc.today, tc.days, got, tc.want)
			}
		})
	}
}

func TestReserve_clamps_a_huge_range(t *testing.T) {
	// A range of more than the supported number of days is limited, the
	// result is the sum of all terms of the entry.
	entry := withEndsOn(t,
		recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-12-31")

	// The entry has twelve terms in the year 2026.
	got := Reserve([]Entry{entry}, day(t, "2026-01-01"), math.MaxInt64)
	if got != 12*200000 {
		t.Errorf("Reserve() = %d Rappen, want %d Rappen", got, 12*200000)
	}
}

func TestReserve_uses_the_same_occurrences_as_the_timeline(t *testing.T) {
	// The reserve and the timeline must count exactly the same terms. Every
	// term of the timeline inside the range adds its amount to the reserve.
	today := day(t, "2026-10-01")
	days := 30

	entries := []Entry{
		recurring(t, 1, 1, "Miete", 200000, "2026-01-31", Month, 1),
		recurring(t, 2, 1, "Quartal", 50000, "2026-01-10", Month, 3),
		recurring(t, 3, 2, "Autoversicherung", 120000, "2026-02-15", Year, 1),
		recurring(t, 4, 3, "Zahnarzt", 15000, "2026-10-20", Once, 1),
	}

	events := Timeline(entries, today, today.AddDate(0, 0, days))
	var want int64
	for _, event := range events {
		want += event.Entry.AmountMinor
	}

	got := Reserve(entries, today, days)
	if got != want {
		t.Errorf("Reserve() = %d Rappen, want %d Rappen from the timeline", got, want)
	}
}
