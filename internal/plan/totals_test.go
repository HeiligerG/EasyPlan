package plan

import (
	"math"
	"testing"
)

func TestTotals_single_entries(t *testing.T) {
	// The demands are derived from the yearly costs:
	//   week:  amount * 365.25 / 7 / interval
	//   month: amount * 12 / interval
	//   year:  amount / interval
	//   once:  not contained
	// Month = annual / 12, Week = annual * 7 / 365.25, Day = annual / 365.25.
	// The names Month, Week and Day are the columns of the concept.
	cases := []struct {
		name  string
		entry Entry
		// The yearly costs in Rappen, derived by hand.
		annual    float64
		monthWant float64
		weekWant  float64
		dayWant   float64
	}{
		{
			// 700 Rappen per week -> 700 * 365.25 / 7 = 36525 Rappen per year.
			// Month = 36525 / 12 = 3043.75, Week = 36525 * 7 / 365.25 = 700,
			// Day = 36525 / 365.25 = 100.
			name:      "weekly entry",
			entry:     recurring(t, 1, 1, "Waesche", 700, "2026-01-01", Week, 1),
			annual:    36525,
			monthWant: 3043.75,
			weekWant:  700,
			dayWant:   100,
		},
		{
			// 1000 Rappen every two weeks -> 1000 * 365.25 / 7 / 2 = 26089.2857
			// Rappen per year. Month = 26089.2857 / 12 = 2174.1071,
			// Week = 182625 / 365.25 = 500, Day = 71.4286.
			name:      "weekly entry every two weeks",
			entry:     recurring(t, 2, 1, "Putzfrau", 1000, "2026-01-05", Week, 2),
			annual:    26089.29,
			monthWant: 2174.11,
			weekWant:  500,
			dayWant:   71.43,
		},
		{
			// 10000 Rappen per month -> 10000 * 12 = 120000 Rappen per year.
			// Month = 10000, Week = 840000 / 365.25 = 2299.7947,
			// Day = 120000 / 365.25 = 328.5421.
			name:      "monthly entry",
			entry:     recurring(t, 3, 1, "Miete", 10000, "2026-01-01", Month, 1),
			annual:    120000,
			monthWant: 10000,
			weekWant:  2299.79,
			dayWant:   328.54,
		},
		{
			// 33333 Rappen every three months -> 33333 * 12 / 3 = 133332 Rappen
			// per year. Month = 11111, Week = 933324 / 365.25 = 2555.3019,
			// Day = 133332 / 365.25 = 365.0431.
			name:      "quarterly entry",
			entry:     recurring(t, 4, 1, "Quartalszahlung", 33333, "2026-01-10", Month, 3),
			annual:    133332,
			monthWant: 11111,
			weekWant:  2555.30,
			dayWant:   365.04,
		},
		{
			// 120000 Rappen per year -> 120000 Rappen per year.
			// Month = 10000, Week = 2299.7947, Day = 328.5421.
			name:      "yearly entry",
			entry:     recurring(t, 5, 1, "Autoversicherung", 120000, "2026-02-15", Year, 1),
			annual:    120000,
			monthWant: 10000,
			weekWant:  2299.79,
			dayWant:   328.54,
		},
		{
			// 120000 Rappen every two years -> 120000 / 2 = 60000 Rappen per year.
			// Month = 5000, Week = 420000 / 365.25 = 1149.8973,
			// Day = 60000 / 365.25 = 164.2711.
			name:      "yearly entry every two years",
			entry:     recurring(t, 6, 1, "Grossreinemacher", 120000, "2026-06-01", Year, 2),
			annual:    60000,
			monthWant: 5000,
			weekWant:  1149.90,
			dayWant:   164.27,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			need := Totals([]Entry{tc.entry})
			assertClose(t, "Month", need.Month, tc.monthWant)
			assertClose(t, "Week", need.Week, tc.weekWant)
			assertClose(t, "Day", need.Day, tc.dayWant)
		})
	}
}

func TestTotals_once_entries_are_not_contained(t *testing.T) {
	// A one time expense is no running demand, it must not change any of the
	// three values.
	base := recurring(t, 1, 1, "Miete", 10000, "2026-01-01", Month, 1)
	once := recurring(t, 2, 1, "Zahnarzt", 500000, "2026-11-12", Once, 1)

	without := Totals([]Entry{base})
	with := Totals([]Entry{base, once})

	if with != without {
		t.Errorf("Totals with a once entry = %+v, want %+v", with, without)
	}

	onlyOnce := Totals([]Entry{once})
	if onlyOnce.Month != 0 || onlyOnce.Week != 0 || onlyOnce.Day != 0 {
		t.Errorf("Totals of a single once entry = %+v, want all values to be zero", onlyOnce)
	}
}

func TestTotals_sums_over_several_accounts(t *testing.T) {
	// Three accounts with 10000 Rappen per month each: 30000 Rappen per month
	// -> 360000 Rappen per year.
	// Month = 30000, Week = 2520000 / 365.25 = 6899.3833,
	// Day = 360000 / 365.25 = 985.6263.
	entries := []Entry{
		recurring(t, 1, 1, "Miete", 10000, "2026-01-01", Month, 1),
		recurring(t, 2, 2, "Sparen", 10000, "2026-01-01", Month, 1),
		recurring(t, 3, 3, "Steuern", 10000, "2026-01-01", Month, 1),
	}

	need := Totals(entries)
	assertClose(t, "Month", need.Month, 30000)
	assertClose(t, "Week", need.Week, 6899.38)
	assertClose(t, "Day", need.Day, 985.63)
}

func TestTotals_returns_unrounded_values(t *testing.T) {
	// Totals must not round, the rounding happens only in the display with
	// RoundCHF.
	// 10000 Rappen per month -> 120000 Rappen per year ->
	// Week = 840000 / 365.25 = 2299.7946618754284... Rappen.
	need := Totals([]Entry{recurring(t, 1, 1, "Miete", 10000, "2026-01-01", Month, 1)})

	assertClose(t, "Week", need.Week, 2299.7946618754284)
	if need.Week == 2300 {
		t.Errorf("Week = %v, want the unrounded value", need.Week)
	}
	if got := RoundCHF(need.Week); got != 23 {
		t.Errorf("RoundCHF(Week) = %d, want 23", got)
	}
}

func TestTotals_rounds_only_for_the_display(t *testing.T) {
	// Two accounts with 10000 Rappen per year each:
	// per account Month = 10000 / 12 = 833.33 Rappen = 8.33 CHF -> 8 CHF
	// total       Month = 20000 / 12 = 1666.67 Rappen = 16.67 CHF -> 17 CHF
	// The sum of the rounded rows (8 + 8 = 16) differs from the rounded total
	// (17). This deviation is intended and must not be corrected.
	entry := func(id int64) Entry {
		return recurring(t, id, id, "Ruecklage", 10000, "2026-01-01", Year, 1)
	}

	first := Totals([]Entry{entry(1)})
	second := Totals([]Entry{entry(2)})
	total := Totals([]Entry{entry(1), entry(2)})

	assertClose(t, "Month of account 1", first.Month, 833.33)
	assertClose(t, "Month of account 2", second.Month, 833.33)
	assertClose(t, "Month of the total", total.Month, 1666.67)

	if got := RoundCHF(first.Month); got != 8 {
		t.Errorf("RoundCHF(Month of account 1) = %d, want 8", got)
	}
	if got := RoundCHF(second.Month); got != 8 {
		t.Errorf("RoundCHF(Month of account 2) = %d, want 8", got)
	}
	if got := RoundCHF(total.Month); got != 17 {
		t.Errorf("RoundCHF(Month of the total) = %d, want 17", got)
	}

	sumOfRoundedRows := RoundCHF(first.Month) + RoundCHF(second.Month)
	if sumOfRoundedRows == RoundCHF(total.Month) {
		t.Errorf("the rounded rows (%d) and the rounded total (%d) are equal, the test no longer covers the rounding deviation",
			sumOfRoundedRows, RoundCHF(total.Month))
	}
}

func TestTotals_ignores_broken_entries(t *testing.T) {
	// An entry without a usable interval or with an unknown unit contributes
	// nothing to the demands, it must not make the calculation fail.
	broken := []Entry{
		recurring(t, 1, 1, "Kaputt", 100000, "2026-01-01", Month, 0),
		recurring(t, 2, 1, "Kaputt", 100000, "2026-01-01", Unit("day"), 1),
		recurring(t, 3, 1, "Kaputt", 100000, "2026-01-01", Month, -1),
	}

	need := Totals(broken)
	if need.Month != 0 || need.Week != 0 || need.Day != 0 {
		t.Errorf("Totals of broken entries = %+v, want all values to be zero", need)
	}
}

func TestRoundCHF(t *testing.T) {
	cases := []struct {
		name  string
		minor float64
		want  int64
	}{
		{name: "zero", minor: 0, want: 0},
		{name: "one Rappen", minor: 1, want: 0},
		{name: "49 Rappen", minor: 49, want: 0},
		{name: "50 Rappen rounds up", minor: 50, want: 1},
		{name: "51 Rappen", minor: 51, want: 1},
		{name: "99 Rappen", minor: 99, want: 1},
		{name: "100 Rappen", minor: 100, want: 1},
		{name: "101 Rappen", minor: 101, want: 1},
		{name: "149 Rappen", minor: 149, want: 1},
		{name: "150 Rappen rounds up", minor: 150, want: 2},
		{name: "249 Rappen", minor: 249, want: 2},
		{name: "250 Rappen rounds up", minor: 250, want: 3},
		{name: "251 Rappen", minor: 251, want: 3},
		{name: "999 Rappen", minor: 999, want: 10},
		{name: "1000 Rappen", minor: 1000, want: 10},
		{name: "negative 49 Rappen", minor: -49, want: 0},
		{name: "negative 50 Rappen rounds away from zero", minor: -50, want: -1},
		{name: "negative 51 Rappen", minor: -51, want: -1},
		{name: "negative 149 Rappen", minor: -149, want: -1},
		{name: "negative 150 Rappen rounds away from zero", minor: -150, want: -2},
		{name: "negative 250 Rappen rounds away from zero", minor: -250, want: -3},
		{name: "negative 251 Rappen", minor: -251, want: -3},
		{name: "negative 1000 Rappen", minor: -1000, want: -10},
		{name: "large value", minor: 1e17, want: 1e15},
		{name: "value above the int64 range is clamped", minor: 1e21, want: math.MaxInt64},
		{name: "negative value below the int64 range is clamped", minor: -1e21, want: math.MinInt64},
		{name: "positive infinity is clamped", minor: math.Inf(1), want: math.MaxInt64},
		{name: "negative infinity is clamped", minor: math.Inf(-1), want: math.MinInt64},
		{name: "NaN becomes zero", minor: math.NaN(), want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RoundCHF(tc.minor); got != tc.want {
				t.Errorf("RoundCHF(%v) = %d, want %d", tc.minor, got, tc.want)
			}
		})
	}
}

func TestActiveOn(t *testing.T) {
	// Today is the 1st of october 2026, the end dates are compared with it.
	today := "2026-10-01"

	cases := []struct {
		name   string
		entry  Entry
		wantIn bool
	}{
		{
			name:   "entry without an end date stays active",
			entry:  recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1),
			wantIn: true,
		},
		{
			name:   "entry with an end date after today stays active",
			entry:  withEndsOn(t, recurring(t, 2, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-10-02"),
			wantIn: true,
		},
		{
			name:   "entry with an end date on today stays active",
			entry:  withEndsOn(t, recurring(t, 3, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-10-01"),
			wantIn: true,
		},
		{
			name:   "entry with an end date yesterday is not active anymore",
			entry:  withEndsOn(t, recurring(t, 4, 1, "Miete", 200000, "2026-01-01", Month, 1), "2026-09-30"),
			wantIn: false,
		},
		{
			name:   "entry that ended long ago is not active anymore",
			entry:  withEndsOn(t, recurring(t, 5, 1, "Miete", 200000, "2020-01-01", Month, 1), "2020-12-31"),
			wantIn: false,
		},
		{
			// A one time entry keeps its end date, it is ignored by the
			// calculation. It stays in the list, the state is shown with
			// IsDone.
			name:   "once entry with an end date in the past stays in the list",
			entry:  withEndsOn(t, recurring(t, 6, 1, "Zahnarzt", 15000, "2020-05-01", Once, 1), "2020-05-01"),
			wantIn: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			active := ActiveOn([]Entry{tc.entry}, day(t, today))
			if tc.wantIn {
				if len(active) != 1 {
					t.Fatalf("ActiveOn() returned %d entries, want 1", len(active))
				}
				if active[0].ID != tc.entry.ID {
					t.Errorf("ActiveOn() returned the entry %d, want %d", active[0].ID, tc.entry.ID)
				}
				return
			}
			if len(active) != 0 {
				t.Errorf("ActiveOn() returned %d entries, want 0", len(active))
			}
		})
	}
}

func TestActiveOn_keeps_the_order_and_removes_only_the_finished_entries(t *testing.T) {
	today := day(t, "2026-10-01")
	first := recurring(t, 1, 1, "Miete", 200000, "2026-01-01", Month, 1)
	second := withEndsOn(t, recurring(t, 2, 2, "Alt", 10000, "2026-01-01", Month, 1), "2026-09-30")
	third := recurring(t, 3, 3, "Internet", 6000, "2026-01-15", Month, 1)

	active := ActiveOn([]Entry{first, second, third}, today)
	if len(active) != 2 {
		t.Fatalf("ActiveOn() returned %d entries, want 2", len(active))
	}
	if active[0].ID != first.ID || active[1].ID != third.ID {
		t.Errorf("ActiveOn() returned the entries %d and %d, want %d and %d",
			active[0].ID, active[1].ID, first.ID, third.ID)
	}
}

func TestActiveOn_empty_input(t *testing.T) {
	active := ActiveOn(nil, day(t, "2026-10-01"))
	if len(active) != 0 {
		t.Errorf("ActiveOn(nil) returned %d entries, want 0", len(active))
	}
}
