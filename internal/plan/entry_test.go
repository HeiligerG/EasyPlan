package plan

import (
	"strings"
	"testing"
	"time"
)

func TestUnit_Valid(t *testing.T) {
	cases := []struct {
		unit Unit
		want bool
	}{
		{unit: Once, want: true},
		{unit: Week, want: true},
		{unit: Month, want: true},
		{unit: Year, want: true},
		{unit: Unit("day"), want: false},
		{unit: Unit(""), want: false},
		{unit: Unit("MONTH"), want: false},
		{unit: Unit(" months"), want: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.unit), func(t *testing.T) {
			if got := tc.unit.Valid(); got != tc.want {
				t.Errorf("Unit(%q).Valid() = %v, want %v", tc.unit, got, tc.want)
			}
		})
	}
}

func TestEntry_Validate_valid(t *testing.T) {
	cases := []struct {
		name  string
		entry Entry
	}{
		{
			name: "monthly entry",
			entry: Entry{
				Name:        "Miete",
				AmountMinor: 200000,
				StartDate:   day(t, "2026-01-01"),
				Unit:        Month,
				Interval:    1,
			},
		},
		{
			name: "weekly entry with an interval of two",
			entry: Entry{
				Name:        "Putzfrau",
				AmountMinor: 5000,
				StartDate:   day(t, "2026-01-05"),
				Unit:        Week,
				Interval:    2,
			},
		},
		{
			name: "yearly entry with an end date after the start date",
			entry: Entry{
				Name:        "Autoversicherung",
				AmountMinor: 120000,
				StartDate:   day(t, "2026-02-15"),
				Unit:        Year,
				Interval:    1,
				EndsOn:      dayPtr(t, "2030-02-15"),
			},
		},
		{
			name: "end date equal to the start date is allowed",
			entry: Entry{
				Name:        "Zahnarzt",
				AmountMinor: 15000,
				StartDate:   day(t, "2026-11-12"),
				Unit:        Once,
				Interval:    1,
				EndsOn:      dayPtr(t, "2026-11-12"),
			},
		},
		{
			name: "name with a leading and trailing space",
			entry: Entry{
				Name:        "  Internet  ",
				AmountMinor: 6000,
				StartDate:   day(t, "2026-01-15"),
				Unit:        Month,
				Interval:    1,
			},
		},
		{
			name: "amount of one Rappen",
			entry: Entry{
				Name:        "Restsparen",
				AmountMinor: 1,
				StartDate:   day(t, "2026-01-01"),
				Unit:        Year,
				Interval:    1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.entry.Validate(); err != nil {
				t.Errorf("Validate() returned an error: %v", err)
			}
		})
	}
}

func TestEntry_Validate_errors(t *testing.T) {
	// Every case changes exactly one property of an otherwise valid entry, so
	// each error is triggered on its own.
	valid := Entry{
		Name:        "Miete",
		AmountMinor: 200000,
		StartDate:   day(t, "2026-01-01"),
		Unit:        Month,
		Interval:    1,
	}

	cases := []struct {
		name    string
		mutate  func(e *Entry)
		wantMsg string
	}{
		{
			name:    "amount of zero",
			mutate:  func(e *Entry) { e.AmountMinor = 0 },
			wantMsg: "amount must be greater than zero",
		},
		{
			name:    "negative amount",
			mutate:  func(e *Entry) { e.AmountMinor = -1 },
			wantMsg: "amount must be greater than zero",
		},
		{
			name:    "interval of zero",
			mutate:  func(e *Entry) { e.Interval = 0 },
			wantMsg: "interval must be greater than zero",
		},
		{
			name:    "negative interval",
			mutate:  func(e *Entry) { e.Interval = -3 },
			wantMsg: "interval must be greater than zero",
		},
		{
			name:    "unknown unit",
			mutate:  func(e *Entry) { e.Unit = Unit("day") },
			wantMsg: `unknown unit "day"`,
		},
		{
			name:    "empty unit",
			mutate:  func(e *Entry) { e.Unit = Unit("") },
			wantMsg: "unknown unit",
		},
		{
			name:    "empty name",
			mutate:  func(e *Entry) { e.Name = "" },
			wantMsg: "name must not be empty",
		},
		{
			name:    "name of spaces only",
			mutate:  func(e *Entry) { e.Name = "   " },
			wantMsg: "name must not be empty",
		},
		{
			name:    "name of a tab and a newline only",
			mutate:  func(e *Entry) { e.Name = "\t\n" },
			wantMsg: "name must not be empty",
		},
		{
			name:    "end date one day before the start date",
			mutate:  func(e *Entry) { e.EndsOn = dayPtr(t, "2025-12-31") },
			wantMsg: "is before start_date",
		},
		{
			name:    "end date far before the start date",
			mutate:  func(e *Entry) { e.EndsOn = dayPtr(t, "2020-01-01") },
			wantMsg: "is before start_date",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := valid
			tc.mutate(&e)

			err := e.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestEntry_Validate_once_ignores_interval_and_end_date_for_the_calculation(t *testing.T) {
	// The unit "once" ignores interval and end date in the calculations, but
	// the values still have to be valid.
	e := Entry{
		Name:        "Zahnarzt",
		AmountMinor: 15000,
		StartDate:   day(t, "2026-11-12"),
		Unit:        Once,
		Interval:    4,
		EndsOn:      dayPtr(t, "2026-11-12"),
	}
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() returned an error: %v", err)
	}

	e.Interval = 0
	if err := e.Validate(); err == nil {
		t.Errorf("Validate() = nil for a once entry with interval 0, want an error")
	}
}

// TestEntry_Validate_amount_boundaries checks the upper limit of the amount.
// The limit is the point where the sums of the calculations stay inside the
// range of an int64.
func TestEntry_Validate_amount_boundaries(t *testing.T) {
	base := Entry{
		Name:      "Miete",
		StartDate: day(t, "2026-01-01"),
		Unit:      Month,
		Interval:  1,
	}

	t.Run("the largest allowed amount is valid", func(t *testing.T) {
		e := base
		e.AmountMinor = MaxAmountMinor
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with the amount %d returned an error: %v", MaxAmountMinor, err)
		}
	})

	t.Run("one Rappen above the limit is rejected", func(t *testing.T) {
		e := base
		e.AmountMinor = MaxAmountMinor + 1
		err := e.Validate()
		if err == nil {
			t.Fatalf("Validate() with the amount %d = nil, want an error", MaxAmountMinor+1)
		}
		if !strings.Contains(err.Error(), "amount must not be greater than") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "amount must not be greater than")
		}
	})
}

// TestEntry_Validate_interval_boundaries checks the upper limit of the
// interval. A "once" entry does not use the interval, so the upper limit
// does not apply to it, but the interval still has to be positive.
func TestEntry_Validate_interval_boundaries(t *testing.T) {
	base := Entry{
		Name:        "Miete",
		AmountMinor: 200000,
		StartDate:   day(t, "2026-01-01"),
	}

	t.Run("the largest allowed interval is valid", func(t *testing.T) {
		e := base
		e.Unit = Month
		e.Interval = MaxInterval
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with the interval %d returned an error: %v", MaxInterval, err)
		}
	})

	t.Run("one above the limit is rejected", func(t *testing.T) {
		e := base
		e.Unit = Month
		e.Interval = MaxInterval + 1
		err := e.Validate()
		if err == nil {
			t.Fatalf("Validate() with the interval %d = nil, want an error", MaxInterval+1)
		}
		if !strings.Contains(err.Error(), "interval must not be greater than") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "interval must not be greater than")
		}
	})

	t.Run("a once entry is not limited by the maximum interval", func(t *testing.T) {
		// The interval of a "once" entry is not used, so the upper limit
		// does not apply to it.
		e := base
		e.Unit = Once
		e.Interval = MaxInterval + 1
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with the interval %d returned an error: %v", MaxInterval+1, err)
		}
	})

	t.Run("a once entry still needs a positive interval", func(t *testing.T) {
		e := base
		e.Unit = Once
		e.Interval = 0
		err := e.Validate()
		if err == nil {
			t.Fatal("Validate() with the interval 0 = nil, want an error")
		}
		if !strings.Contains(err.Error(), "interval must be greater than zero") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "interval must be greater than zero")
		}
	})
}

// TestEntry_Validate_date_boundaries checks the supported year range of the
// start date and of the end date.
func TestEntry_Validate_date_boundaries(t *testing.T) {
	base := Entry{
		Name:        "Miete",
		AmountMinor: 200000,
		Unit:        Month,
		Interval:    1,
	}

	cases := []struct {
		name      string
		startDate time.Time
		wantValid bool
	}{
		{name: "the first year is valid", startDate: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), wantValid: true},
		{name: "the last year is valid", startDate: time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC), wantValid: true},
		{name: "the year before the first year is rejected", startDate: time.Date(0, time.December, 31, 0, 0, 0, 0, time.UTC), wantValid: false},
		{name: "the year after the last year is rejected", startDate: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC), wantValid: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := base
			e.StartDate = tc.startDate
			err := e.Validate()
			if tc.wantValid {
				if err != nil {
					t.Errorf("Validate() with the start date %s returned an error: %v", FormatDate(tc.startDate), err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() with the start date %s = nil, want an error", FormatDate(tc.startDate))
			}
			if !strings.Contains(err.Error(), "outside the supported year range") {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "outside the supported year range")
			}
		})
	}

	t.Run("the end date uses the same year range", func(t *testing.T) {
		for _, endsOn := range []time.Time{
			time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC),
		} {
			e := base
			e.StartDate = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
			end := endsOn
			e.EndsOn = &end
			if err := e.Validate(); err != nil {
				t.Errorf("Validate() with the end date %s returned an error: %v", FormatDate(endsOn), err)
			}
		}
		for _, endsOn := range []time.Time{
			time.Date(0, time.December, 31, 0, 0, 0, 0, time.UTC),
			time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC),
		} {
			e := base
			e.StartDate = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
			end := endsOn
			e.EndsOn = &end
			err := e.Validate()
			if err == nil {
				t.Fatalf("Validate() with the end date %s = nil, want an error", FormatDate(endsOn))
			}
			if !strings.Contains(err.Error(), "outside the supported year range") {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "outside the supported year range")
			}
		}
	})
}

// TestEntry_Validate_date_boundaries_in_a_non_utc_time_zone checks that the
// start date and the end date are judged by the calendar day of their own
// time zone and not by the year of the same instant in UTC.
//
// The expectations are derived by hand. The zone X is five hours behind UTC,
// the zone Y five hours ahead of UTC:
//
//	X  9999-12-31 23:30  ->  10000-01-01 04:30 UTC  ->  calendar year 9999
//	X  10000-01-01 00:00 ->  10000-01-01 05:00 UTC  ->  calendar year 10000
//	Y  0001-01-01 00:30  ->  0000-12-31 19:30 UTC  ->  calendar year 1
//	X  0000-12-31 23:30  ->  0001-01-01 04:30 UTC  ->  calendar year 0
//
// The supported year range is 1 to 9999. The first and the third timestamp
// are therefore valid although their instant lies outside the range in UTC,
// the second and the fourth one are rejected although their instant lies
// inside the range in UTC.
func TestEntry_Validate_date_boundaries_in_a_non_utc_time_zone(t *testing.T) {
	zoneWest := time.FixedZone("X", -5*3600)
	zoneEast := time.FixedZone("Y", 5*3600)

	base := Entry{
		Name:        "Miete",
		AmountMinor: 200000,
		Unit:        Month,
		Interval:    1,
	}

	t.Run("the start date uses the calendar day of its own time zone", func(t *testing.T) {
		cases := []struct {
			name      string
			startDate time.Time
			wantValid bool
		}{
			{
				name:      "the last year is valid although the instant lies in the year 10000 in UTC",
				startDate: time.Date(9999, time.December, 31, 23, 30, 0, 0, zoneWest),
				wantValid: true,
			},
			{
				name:      "the year after the last year is rejected",
				startDate: time.Date(10000, time.January, 1, 0, 0, 0, 0, zoneWest),
				wantValid: false,
			},
			{
				name:      "the first year is valid although the instant lies in the year 0 in UTC",
				startDate: time.Date(1, time.January, 1, 0, 30, 0, 0, zoneEast),
				wantValid: true,
			},
			{
				name:      "the year before the first year is rejected although the instant lies in the year 1 in UTC",
				startDate: time.Date(0, time.December, 31, 23, 30, 0, 0, zoneWest),
				wantValid: false,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				e := base
				e.StartDate = tc.startDate
				err := e.Validate()
				if tc.wantValid {
					if err != nil {
						t.Errorf("Validate() with the start date %s returned an error: %v", FormatDate(tc.startDate), err)
					}
					return
				}
				if err == nil {
					t.Fatalf("Validate() with the start date %s = nil, want an error", FormatDate(tc.startDate))
				}
				if !strings.Contains(err.Error(), "outside the supported year range") {
					t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "outside the supported year range")
				}
			})
		}
	})

	t.Run("the end date uses the calendar day of its own time zone", func(t *testing.T) {
		// The start date lies one calendar day before the end date of the
		// first case, so the end date is not rejected for being before the
		// start date.
		startBefore := time.Date(9999, time.December, 30, 0, 0, 0, 0, zoneWest)

		cases := []struct {
			name      string
			endsOn    time.Time
			wantValid bool
		}{
			{
				name:      "the last year is valid although the instant lies in the year 10000 in UTC",
				endsOn:    time.Date(9999, time.December, 31, 23, 30, 0, 0, zoneWest),
				wantValid: true,
			},
			{
				name:      "the year after the last year is rejected",
				endsOn:    time.Date(10000, time.January, 1, 0, 0, 0, 0, zoneWest),
				wantValid: false,
			},
			{
				name:      "the year before the first year is rejected although the instant lies in the year 1 in UTC",
				endsOn:    time.Date(0, time.December, 31, 23, 30, 0, 0, zoneWest),
				wantValid: false,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				e := base
				e.StartDate = startBefore
				end := tc.endsOn
				e.EndsOn = &end
				err := e.Validate()
				if tc.wantValid {
					if err != nil {
						t.Errorf("Validate() with the end date %s returned an error: %v", FormatDate(tc.endsOn), err)
					}
					return
				}
				if err == nil {
					t.Fatalf("Validate() with the end date %s = nil, want an error", FormatDate(tc.endsOn))
				}
				if !strings.Contains(err.Error(), "outside the supported year range") {
					t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "outside the supported year range")
				}
			})
		}
	})

	t.Run("the end date is compared by calendar day and not by the instant", func(t *testing.T) {
		e := base
		// 08:30 in the zone X is 13:30 UTC.
		e.StartDate = time.Date(2026, time.November, 12, 8, 30, 0, 0, zoneWest)

		// The end date lies on the same calendar day as the start date but
		// at an earlier time of the day. As an instant it would be before
		// the start date (11:00 UTC against 13:30 UTC), as a calendar day
		// it is the same day, so the entry is valid.
		sameDay := time.Date(2026, time.November, 12, 6, 0, 0, 0, zoneWest)
		e.EndsOn = &sameDay
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with the start date %s and the end date %s returned an error: %v",
				FormatDate(e.StartDate), FormatDate(sameDay), err)
		}

		// The end date lies on the same calendar day as the start date and
		// carries a time of the day in another zone. The zones do not
		// matter for the comparison.
		otherZone := time.Date(2026, time.November, 12, 6, 0, 0, 0, zoneEast)
		e.EndsOn = &otherZone
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with the start date %s and the end date %s returned an error: %v",
				FormatDate(e.StartDate), FormatDate(otherZone), err)
		}

		// One calendar day earlier is before the start date and is
		// rejected.
		earlier := time.Date(2026, time.November, 11, 23, 59, 59, 0, zoneWest)
		e.EndsOn = &earlier
		err := e.Validate()
		if err == nil {
			t.Fatal("Validate() with an end date one calendar day before the start date = nil, want an error")
		}
		if !strings.Contains(err.Error(), "is before start_date") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "is before start_date")
		}
	})
}

// TestEntry_Validate_end_date_on_the_same_day_with_a_time_is_valid checks
// that the end date is compared as a calendar day. A time of the day never
// decides whether an entry is valid.
func TestEntry_Validate_end_date_on_the_same_day_with_a_time_is_valid(t *testing.T) {
	e := Entry{
		Name:        "Zahnarzt",
		AmountMinor: 15000,
		StartDate:   day(t, "2026-11-12"),
		Unit:        Once,
		Interval:    1,
		EndsOn:      dayPtr(t, "2026-11-12"),
	}
	// The end date lies on the same calendar day as the start date, but it
	// carries a time of the day.
	sameDay := time.Date(2026, time.November, 12, 23, 59, 59, 0, time.UTC)
	e.EndsOn = &sameDay
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() with the end date %s returned an error: %v", FormatDate(sameDay), err)
	}

	// A start date with a time of the day and an end date on the same day
	// is valid as well.
	withTime := time.Date(2026, time.November, 12, 8, 30, 0, 0, time.UTC)
	e.StartDate = withTime
	e.EndsOn = &sameDay
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() with the start date %s and the end date %s returned an error: %v",
			FormatDate(withTime), FormatDate(sameDay), err)
	}

	// The end date lies on the same calendar day as the start date and
	// carries an earlier time of the day. The dates are compared as calendar
	// days, so the entry is valid.
	earlierTime := time.Date(2026, time.November, 12, 6, 0, 0, 0, time.UTC)
	e.StartDate = withTime
	e.EndsOn = &earlierTime
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() with the start date %s and the end date %s returned an error: %v",
			FormatDate(withTime), FormatDate(earlierTime), err)
	}

	// One day earlier is still before the start date and is rejected.
	earlier := time.Date(2026, time.November, 11, 23, 59, 59, 0, time.UTC)
	e.StartDate = day(t, "2026-11-12")
	e.EndsOn = &earlier
	err := e.Validate()
	if err == nil {
		t.Fatal("Validate() with an end date one day before the start date = nil, want an error")
	}
	if !strings.Contains(err.Error(), "is before start_date") {
		t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "is before start_date")
	}
}
