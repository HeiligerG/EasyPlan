package plan

import (
	"math"
	"slices"
	"testing"
)

func TestOccurrences(t *testing.T) {
	cases := []struct {
		name  string
		entry Entry
		from  string
		to    string
		want  []string
	}{
		{
			// 31.01. -> 28.02. -> 31.03. -> 30.04.
			// The 31st of march is computed directly from the start date, it is
			// not carried over from the 28th of february.
			name:  "monthly on the 31st clamps to the end of february and returns to the 31st",
			entry: recurring(t, 1, 1, "Miete", 200000, "2026-01-31", Month, 1),
			from:  "2026-01-01",
			to:    "2026-05-01",
			want:  []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"},
		},
		{
			// 31.01.2028 -> 29.02.2028 (leap year) -> 31.03.2028 -> 30.04.2028
			name:  "monthly on the 31st uses the 29th of february in a leap year",
			entry: recurring(t, 1, 1, "Miete", 200000, "2028-01-31", Month, 1),
			from:  "2028-01-01",
			to:    "2028-05-01",
			want:  []string{"2028-01-31", "2028-02-29", "2028-03-31", "2028-04-30"},
		},
		{
			// 30.01. -> 28.02. -> 30.03. -> 30.04.
			name:  "monthly on the 30th clamps to the end of february",
			entry: recurring(t, 2, 1, "Miete", 200000, "2026-01-30", Month, 1),
			from:  "2026-01-01",
			to:    "2026-05-01",
			want:  []string{"2026-01-30", "2026-02-28", "2026-03-30", "2026-04-30"},
		},
		{
			// 29.02.2028 -> 28.02.2029 -> 28.02.2030 -> 28.02.2031 -> 29.02.2032
			// The 29th of february 2032 exists again, because the day is derived
			// from the start date and clamped month by month.
			name:  "yearly starting on a leap day clamps to the 28th of february",
			entry: recurring(t, 3, 2, "Sparen", 120000, "2028-02-29", Year, 1),
			from:  "2028-01-01",
			to:    "2033-01-01",
			want: []string{
				"2028-02-29",
				"2029-02-28",
				"2030-02-28",
				"2031-02-28",
				"2032-02-29",
			},
		},
		{
			// every three months: 10.01., 10.04., 10.07., 10.10.
			name:  "quarterly entry",
			entry: recurring(t, 4, 3, "Quartalszahlung", 50000, "2026-01-10", Month, 3),
			from:  "2026-01-01",
			to:    "2027-01-01",
			want:  []string{"2026-01-10", "2026-04-10", "2026-07-10", "2026-10-10"},
		},
		{
			// every two weeks: 05.01., 19.01., 02.02., 16.02. (14 days apart)
			name:  "entry every two weeks",
			entry: recurring(t, 5, 4, "Putzfrau", 5000, "2026-01-05", Week, 2),
			from:  "2026-01-01",
			to:    "2026-03-01",
			want:  []string{"2026-01-05", "2026-01-19", "2026-02-02", "2026-02-16"},
		},
		{
			// every two years: 01.06.2026, 01.06.2028, 01.06.2030
			name:  "entry every two years",
			entry: recurring(t, 6, 5, "Grossreinemacher", 25000, "2026-06-01", Year, 2),
			from:  "2026-01-01",
			to:    "2031-01-01",
			want:  []string{"2026-06-01", "2028-06-01", "2030-06-01"},
		},
		{
			// "once" ignores interval and end date, it occurs exactly once.
			name: "once entry ignores interval and end date",
			entry: func() Entry {
				e := recurring(t, 7, 6, "Zahnarzt", 15000, "2026-05-01", Once, 3)
				return withEndsOn(t, e, "2026-05-01")
			}(),
			from: "2026-01-01",
			to:   "2027-01-01",
			want: []string{"2026-05-01"},
		},
		{
			// "once" with an end date far in the future still occurs only once.
			name: "once entry with a distant end date occurs only once",
			entry: func() Entry {
				e := recurring(t, 8, 6, "Zahnarzt", 15000, "2026-05-01", Once, 7)
				return withEndsOn(t, e, "2030-12-31")
			}(),
			from: "2026-01-01",
			to:   "2031-01-01",
			want: []string{"2026-05-01"},
		},
		{
			// The end date is inclusive: 15.03. is still a valid term.
			name: "end date on the term itself is inclusive",
			entry: func() Entry {
				e := recurring(t, 9, 7, "Abonnement", 4500, "2026-01-15", Month, 1)
				return withEndsOn(t, e, "2026-03-15")
			}(),
			from: "2026-01-01",
			to:   "2026-12-31",
			want: []string{"2026-01-15", "2026-02-15", "2026-03-15"},
		},
		{
			// The end date lies one day before the term: the term is dropped.
			name: "end date one day before the term drops the term",
			entry: func() Entry {
				e := recurring(t, 9, 7, "Abonnement", 4500, "2026-01-15", Month, 1)
				return withEndsOn(t, e, "2026-03-14")
			}(),
			from: "2026-01-01",
			to:   "2026-12-31",
			want: []string{"2026-01-15", "2026-02-15"},
		},
		{
			// The end date lies one day after the term: the term stays.
			name: "end date one day after the term keeps the term",
			entry: func() Entry {
				e := recurring(t, 9, 7, "Abonnement", 4500, "2026-01-15", Month, 1)
				return withEndsOn(t, e, "2026-03-16")
			}(),
			from: "2026-01-01",
			to:   "2026-12-31",
			want: []string{"2026-01-15", "2026-02-15", "2026-03-15"},
		},
		{
			// The range is half open: the term on "from" belongs to the range.
			name:  "term exactly on the start of the range is included",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2026-02-01",
			to:    "2026-03-01",
			want:  []string{"2026-02-01"},
		},
		{
			// The range is half open: the term on "to" belongs to the next range.
			name:  "term exactly on the end of the range is excluded",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2026-02-01",
			to:    "2026-04-01",
			want:  []string{"2026-02-01", "2026-03-01"},
		},
		{
			name:  "empty range with equal bounds",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2026-02-01",
			to:    "2026-02-01",
			want:  nil,
		},
		{
			name:  "empty range with inverted bounds",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2026-04-01",
			to:    "2026-02-01",
			want:  nil,
		},
		{
			// No term ever lies before the start date of the entry.
			name:  "range before the start date of the entry",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2025-01-01",
			to:    "2026-01-01",
			want:  nil,
		},
		{
			// The range ends before the start date of the entry.
			name:  "range that ends before the start date of the entry",
			entry: recurring(t, 10, 8, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2025-01-01",
			to:    "2025-12-01",
			want:  nil,
		},
		{
			// An open ended entry produces terms over the whole range.
			name:  "open ended entry over a full year",
			entry: recurring(t, 11, 9, "Miete", 200000, "2026-01-01", Month, 1),
			from:  "2026-01-01",
			to:    "2027-01-01",
			want: []string{
				"2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01",
				"2026-05-01", "2026-06-01", "2026-07-01", "2026-08-01",
				"2026-09-01", "2026-10-01", "2026-11-01", "2026-12-01",
			},
		},
		{
			// An entry with an interval of zero produces no terms at all.
			name:  "entry with an interval of zero produces no terms",
			entry: recurring(t, 12, 10, "Kaputt", 1000, "2026-01-01", Month, 0),
			from:  "2026-01-01",
			to:    "2027-01-01",
			want:  nil,
		},
		{
			// An unknown unit produces no terms at all.
			name:  "entry with an unknown unit produces no terms",
			entry: recurring(t, 13, 10, "Kaputt", 1000, "2026-01-01", Unit("day"), 1),
			from:  "2026-01-01",
			to:    "2027-01-01",
			want:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Occurrences(tc.entry, day(t, tc.from), day(t, tc.to))
			if !slices.Equal(dateStrings(got), tc.want) {
				t.Errorf("Occurrences(%s .. %s) = %v, want %v", tc.from, tc.to, dateStrings(got), tc.want)
			}
		})
	}
}

func TestOccurrences_never_returns_a_day_before_the_start_date(t *testing.T) {
	// The first term of the entry is the start date, no matter how the range
	// is chosen. The 28th of february is the first term after the 31st of
	// january, the 31st of march is the next one.
	e := recurring(t, 1, 1, "Miete", 200000, "2026-01-31", Month, 1)
	got := Occurrences(e, day(t, "2020-01-01"), day(t, "2030-01-01"))
	if len(got) == 0 {
		t.Fatalf("Occurrences() returned no terms")
	}
	for _, d := range got {
		if d.Before(day(t, "2026-01-31")) {
			t.Errorf("Occurrences() contains %s, want no day before the start date", FormatDate(d))
		}
	}
	if first := FormatDate(got[0]); first != "2026-01-31" {
		t.Errorf("first term = %s, want %s", first, "2026-01-31")
	}
}

func TestOccurrences_extreme_values(t *testing.T) {
	// The calculation must not panic and must not loop forever on absurd
	// values, it returns no term or only the start date.
	t.Run("interval of a week that cannot be represented", func(t *testing.T) {
		// 7 * MaxInt64 does not fit into an int64, so the second term of the
		// entry cannot be computed. The start date is still a valid term,
		// exactly like for the monthly and the yearly entry: the estimate of
		// the first term returns the index 0 as soon as the range starts
		// before or on the start date, so the start date is not lost.
		e := recurring(t, 1, 1, "Kaputt", 1000, "2026-01-01", Week, math.MaxInt64)
		got := Occurrences(e, day(t, "2026-01-01"), day(t, "2027-01-01"))
		want := []string{"2026-01-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("interval of a week that cannot be represented after the start date", func(t *testing.T) {
		// The range starts after the start date. The next term would need an
		// interval of 7 * MaxInt64 days, which cannot be represented, so the
		// entry has no term left in the range.
		e := recurring(t, 1, 1, "Kaputt", 1000, "2026-01-01", Week, math.MaxInt64)
		got := Occurrences(e, day(t, "2026-01-02"), day(t, "2027-01-01"))
		if len(got) != 0 {
			t.Errorf("Occurrences() = %v, want no term", dateStrings(got))
		}
	})

	t.Run("interval of a month that cannot be represented", func(t *testing.T) {
		// The start date is still a valid term, the next one is out of range.
		e := recurring(t, 2, 1, "Kaputt", 1000, "2026-01-01", Month, math.MaxInt64)
		got := Occurrences(e, day(t, "2026-01-01"), day(t, "2027-01-01"))
		want := []string{"2026-01-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("interval of a year that cannot be represented", func(t *testing.T) {
		// The start date is still a valid term, the next one is out of range.
		e := recurring(t, 3, 1, "Kaputt", 1000, "2026-01-01", Year, math.MaxInt64)
		got := Occurrences(e, day(t, "2026-01-01"), day(t, "2027-01-01"))
		want := []string{"2026-01-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("start date at the end of the supported year range", func(t *testing.T) {
		// The next month would be in the year 10000 and is not supported.
		e := recurring(t, 4, 1, "Am Limit", 1000, "9999-12-01", Month, 1)
		got := Occurrences(e, day(t, "9999-12-01"), day(t, "9999-12-31"))
		want := []string{"9999-12-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("weekly entry at the end of the supported year range", func(t *testing.T) {
		// The terms of december 9999 exist, the next one would be in the year
		// 10000 and is not supported.
		e := recurring(t, 5, 1, "Am Limit", 1000, "9999-12-01", Week, 1)
		got := Occurrences(e, day(t, "9999-12-01"), day(t, "9999-12-31"))
		want := []string{
			"9999-12-01", "9999-12-08", "9999-12-15", "9999-12-22", "9999-12-29",
		}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("weekly interval beyond the supported number of days", func(t *testing.T) {
		// 600000 weeks are more than the supported number of days.
		e := recurring(t, 6, 1, "Kaputt", 1000, "2026-01-01", Week, 600000)
		got := Occurrences(e, day(t, "2026-01-01"), day(t, "2027-01-01"))
		want := []string{"2026-01-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})

	t.Run("yearly interval beyond the supported number of months", func(t *testing.T) {
		// 20000 years are more than the supported number of months.
		e := recurring(t, 7, 1, "Kaputt", 1000, "2026-01-01", Year, 20000)
		got := Occurrences(e, day(t, "2026-01-01"), day(t, "2027-01-01"))
		want := []string{"2026-01-01"}
		if !slices.Equal(dateStrings(got), want) {
			t.Errorf("Occurrences() = %v, want %v", dateStrings(got), want)
		}
	})
}

func TestNext(t *testing.T) {
	cases := []struct {
		name   string
		entry  Entry
		from   string
		want   string
		wantOK bool
	}{
		{
			// The term lies exactly on "from" and is therefore returned.
			name:   "term exactly on the reference day",
			entry:  recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1),
			from:   "2026-03-01",
			want:   "2026-03-01",
			wantOK: true,
		},
		{
			// The next term after the reference day is returned.
			name:   "term after the reference day",
			entry:  recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1),
			from:   "2026-03-02",
			want:   "2026-04-01",
			wantOK: true,
		},
		{
			// The reference day is before the start date, the start date is the
			// first term.
			name:   "reference day before the start date",
			entry:  recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1),
			from:   "2025-12-31",
			want:   "2026-01-01",
			wantOK: true,
		},
		{
			// The end date is inclusive, the term on the end date is returned.
			name: "term on the end date is returned",
			entry: func() Entry {
				e := recurring(t, 2, 2, "Abonnement", 4500, "2026-01-15", Month, 1)
				return withEndsOn(t, e, "2026-03-15")
			}(),
			from:   "2026-03-15",
			want:   "2026-03-15",
			wantOK: true,
		},
		{
			// The reference day is after the end date, there is no further term.
			name: "reference day after the end date",
			entry: func() Entry {
				e := recurring(t, 2, 2, "Abonnement", 4500, "2026-01-15", Month, 1)
				return withEndsOn(t, e, "2026-03-15")
			}(),
			from:   "2026-03-16",
			wantOK: false,
		},
		{
			// A one time entry in the past has no further term.
			name:   "once entry in the past",
			entry:  recurring(t, 3, 3, "Zahnarzt", 15000, "2026-11-12", Once, 1),
			from:   "2026-12-01",
			wantOK: false,
		},
		{
			// A one time entry today is still pending.
			name:   "once entry on the reference day",
			entry:  recurring(t, 3, 3, "Zahnarzt", 15000, "2026-11-12", Once, 1),
			from:   "2026-11-12",
			want:   "2026-11-12",
			wantOK: true,
		},
		{
			// Every three years: 15.01.2026, 15.01.2029, 15.01.2032
			name:   "entry every three years",
			entry:  recurring(t, 4, 4, "Sparen", 120000, "2026-01-15", Year, 3),
			from:   "2027-01-01",
			want:   "2029-01-15",
			wantOK: true,
		},
		{
			// The clamping of the month end does not shift the next term.
			name:   "monthly entry on the 31st",
			entry:  recurring(t, 5, 5, "Miete", 200000, "2026-01-31", Month, 1),
			from:   "2026-02-01",
			want:   "2026-02-28",
			wantOK: true,
		},
		{
			// After the clamped term the entry is back on the 31st.
			name:   "monthly entry on the 31st after february",
			entry:  recurring(t, 5, 5, "Miete", 200000, "2026-01-31", Month, 1),
			from:   "2026-03-01",
			want:   "2026-03-31",
			wantOK: true,
		},
		{
			// Every two weeks: 05.01., 19.01., 02.02.
			name:   "entry every two weeks",
			entry:  recurring(t, 6, 6, "Putzfrau", 5000, "2026-01-05", Week, 2),
			from:   "2026-01-06",
			want:   "2026-01-19",
			wantOK: true,
		},
		{
			// A broken entry without a usable interval has no further term.
			name:   "entry with an interval of zero",
			entry:  recurring(t, 7, 7, "Kaputt", 1000, "2026-01-01", Month, 0),
			from:   "2026-01-01",
			wantOK: false,
		},
		{
			// A broken entry with an unknown unit has no further term.
			name:   "entry with an unknown unit",
			entry:  recurring(t, 8, 8, "Kaputt", 1000, "2026-01-01", Unit("day"), 1),
			from:   "2026-01-01",
			wantOK: false,
		},
		{
			// A weekly interval that cannot be represented has only the start
			// date as a term, exactly like the monthly and the yearly entry:
			// the estimate of the first term returns the index 0 as long as
			// the reference day is before or on the start date.
			name:   "weekly interval that cannot be represented",
			entry:  recurring(t, 9, 9, "Kaputt", 1000, "2026-01-01", Week, math.MaxInt64),
			from:   "2026-01-01",
			want:   "2026-01-01",
			wantOK: true,
		},
		{
			// After the start date the next term needs an interval of
			// 7 * MaxInt64 days, which cannot be represented, so there is no
			// further term.
			name:   "weekly interval that cannot be represented after the start date",
			entry:  recurring(t, 9, 9, "Kaputt", 1000, "2026-01-01", Week, math.MaxInt64),
			from:   "2026-01-02",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Next(tc.entry, day(t, tc.from))
			if ok != tc.wantOK {
				t.Fatalf("Next(from %s) ok = %v, want %v", tc.from, ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if formatted := FormatDate(got); formatted != tc.want {
				t.Errorf("Next(from %s) = %s, want %s", tc.from, formatted, tc.want)
			}
		})
	}
}

func TestNext_agrees_with_Occurrences(t *testing.T) {
	// Next must return exactly the first term that Occurrences reports for
	// the same range. This protects the two functions against drifting apart.
	entries := []Entry{
		recurring(t, 1, 1, "Miete am 31.", 200000, "2026-01-31", Month, 1),
		recurring(t, 2, 1, "Quartal", 50000, "2026-01-10", Month, 3),
		recurring(t, 3, 1, "Alle zwei Wochen", 5000, "2026-01-05", Week, 2),
		recurring(t, 4, 1, "Alle zwei Jahre", 25000, "2026-06-01", Year, 2),
		recurring(t, 5, 1, "Alle drei Jahre", 120000, "2026-01-15", Year, 3),
		recurring(t, 6, 1, "Einmalig", 15000, "2026-05-01", Once, 1),
	}

	references := []string{
		"2026-01-01", "2026-02-01", "2026-02-28", "2026-03-01", "2026-06-01",
		"2027-01-01", "2028-06-01", "2029-02-01",
	}

	for _, e := range entries {
		for _, reference := range references {
			from := day(t, reference)
			got, ok := Next(e, from)
			occurrences := Occurrences(e, from, from.AddDate(0, 0, 365*5))
			if !ok {
				if len(occurrences) != 0 {
					t.Errorf("Next(%s, %s) = false, but Occurrences found %d terms",
						e.Name, reference, len(occurrences))
				}
				continue
			}
			if len(occurrences) == 0 {
				t.Errorf("Next(%s, %s) = %s, but Occurrences found no term",
					e.Name, reference, FormatDate(got))
				continue
			}
			if first := FormatDate(occurrences[0]); first != FormatDate(got) {
				t.Errorf("Next(%s, %s) = %s, Occurrences starts with %s",
					e.Name, reference, FormatDate(got), first)
			}
		}
	}
}
