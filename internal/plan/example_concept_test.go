package plan_test

import (
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// The example of the concept (section 5, state of the 1st of october 2026).
//
// The amounts of the concept are given in CHF, the database stores them in
// Rappen, therefore the entries below carry 200000 Rappen for the rent of
// 2000 CHF. The account and entry names are taken from the concept, the
// column names Monat, Woche and Tag are called Month, Week and Day here.
const (
	accountLifestyle = 1
	accountSparen    = 2
	accountSteuern   = 3
)

// mustDate parses a calendar day of the example and fails the test when the
// literal is wrong.
func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := plan.ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) failed: %v", value, err)
	}
	return parsed
}

// exampleEntries returns the five entries of the concept example.
func exampleEntries(t *testing.T) []plan.Entry {
	t.Helper()
	return []plan.Entry{
		{
			ID: 1, AccountID: accountLifestyle, Name: "Miete",
			AmountMinor: 200000, StartDate: mustDate(t, "2026-01-01"),
			Unit: plan.Month, Interval: 1,
		},
		{
			ID: 2, AccountID: accountLifestyle, Name: "Krankenkasse",
			AmountMinor: 45000, StartDate: mustDate(t, "2026-01-05"),
			Unit: plan.Month, Interval: 1,
		},
		{
			ID: 3, AccountID: accountLifestyle, Name: "Internet",
			AmountMinor: 6000, StartDate: mustDate(t, "2026-01-15"),
			Unit: plan.Month, Interval: 1,
		},
		{
			ID: 4, AccountID: accountSparen, Name: "Autoversicherung",
			AmountMinor: 120000, StartDate: mustDate(t, "2026-02-15"),
			Unit: plan.Year, Interval: 1,
		},
		{
			ID: 5, AccountID: accountSteuern, Name: "Serafe",
			AmountMinor: 33500, StartDate: mustDate(t, "2026-03-20"),
			Unit: plan.Year, Interval: 1,
		},
	}
}

// byAccount groups the entries by their account id.
func byAccount(entries []plan.Entry) map[int64][]plan.Entry {
	grouped := make(map[int64][]plan.Entry)
	for _, e := range entries {
		grouped[e.AccountID] = append(grouped[e.AccountID], e)
	}
	return grouped
}

func TestConceptExample_demands_per_account(t *testing.T) {
	// The table of the concept (section 5):
	//
	//   Account   | Month | Week | Day | Reserve 30 days
	//   Lifestyle |  2510 |  577 |  82 |           2510
	//   Sparen    |   100 |   23 |   3 |              0
	//   Steuern   |    28 |    6 |   1 |              0
	//   Total     |  2638 |  607 |  87 |           2510
	//
	// Lifestyle: 200000 + 45000 + 6000 = 251000 Rappen per month
	//   -> 3012000 Rappen per year
	//   -> Month  251000 Rappen = 2510.00 CHF -> 2510
	//   -> Week   21084000 / 365.25 = 57724.85 Rappen = 577.25 CHF -> 577
	//   -> Day     3012000 / 365.25 =  8247.23 Rappen =  82.47 CHF ->   82
	// Sparen: 120000 Rappen per year
	//   -> Month   10000 Rappen =  100.00 CHF ->  100
	//   -> Week    840000 / 365.25 =  2299.79 Rappen =  23.00 CHF ->   23
	//   -> Day     120000 / 365.25 =   328.54 Rappen =   3.29 CHF ->    3
	// Steuern: 33500 Rappen per year
	//   -> Month   2791.67 Rappen =   27.92 CHF ->   28
	//   -> Week    234500 / 365.25 =   642.03 Rappen =   6.42 CHF ->    6
	//   -> Day      33500 / 365.25 =    91.72 Rappen =   0.92 CHF ->    1
	cases := []struct {
		accountID int64
		name      string
		month     int64
		week      int64
		day       int64
	}{
		{accountID: accountLifestyle, name: "Lifestyle", month: 2510, week: 577, day: 82},
		{accountID: accountSparen, name: "Sparen", month: 100, week: 23, day: 3},
		{accountID: accountSteuern, name: "Steuern", month: 28, week: 6, day: 1},
	}

	grouped := byAccount(exampleEntries(t))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			need := plan.Totals(grouped[tc.accountID])
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
}

func TestConceptExample_total_demands(t *testing.T) {
	// Total: 3012000 + 120000 + 33500 = 3165500 Rappen per year
	//   -> Month  263791.67 Rappen = 2637.92 CHF -> 2638
	//   -> Week 22158500 / 365.25 = 60666.58 Rappen = 606.67 CHF -> 607
	//   -> Day   3165500 / 365.25 =   8666.67 Rappen =  86.67 CHF ->   87
	need := plan.Totals(exampleEntries(t))

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

func TestConceptExample_rounding_deviation_of_the_daily_values(t *testing.T) {
	// The concept states: the total of the daily values is 87, the sum of the
	// rows is 86. This deviation is intended and must not be corrected.
	entries := exampleEntries(t)
	grouped := byAccount(entries)

	var sumOfRows int64
	for _, accountID := range []int64{accountLifestyle, accountSparen, accountSteuern} {
		sumOfRows += plan.RoundCHF(plan.Totals(grouped[accountID]).Day)
	}
	total := plan.RoundCHF(plan.Totals(entries).Day)

	if sumOfRows != 86 {
		t.Errorf("sum of the rounded daily values = %d CHF, want 86 CHF", sumOfRows)
	}
	if total != 87 {
		t.Errorf("rounded daily total = %d CHF, want 87 CHF", total)
	}
	if sumOfRows == total {
		t.Errorf("the rounded rows (%d) and the rounded total (%d) are equal, the test no longer covers the deviation",
			sumOfRows, total)
	}
}

func TestConceptExample_reserve_of_30_days(t *testing.T) {
	// The reserve covers [01.10.2026, 31.10.2026).
	// Lifestyle: the rent on the 1st (200000), the health insurance on the 5th
	// (45000) and the internet on the 15th (6000) are due, in total 251000
	// Rappen = 2510 CHF.
	// Sparen: the car insurance is due on the 15th of february, nothing in october.
	// Steuern: the tax is due on the 20th of march, nothing in october.
	today := mustDate(t, "2026-10-01")
	entries := exampleEntries(t)
	grouped := byAccount(entries)

	cases := []struct {
		accountID int64
		name      string
		want      int64
	}{
		{accountID: accountLifestyle, name: "Lifestyle", want: 251000},
		{accountID: accountSparen, name: "Sparen", want: 0},
		{accountID: accountSteuern, name: "Steuern", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reserve := plan.Reserve(grouped[tc.accountID], today, 30)
			if reserve != tc.want {
				t.Errorf("Reserve(%s) = %d Rappen, want %d Rappen", tc.name, reserve, tc.want)
			}
			if got := plan.RoundCHF(float64(reserve)); got != tc.want/100 {
				t.Errorf("Reserve(%s) = %d CHF, want %d CHF", tc.name, got, tc.want/100)
			}
		})
	}

	total := plan.Reserve(entries, today, 30)
	if total != 251000 {
		t.Errorf("Reserve total = %d Rappen, want 251000 Rappen", total)
	}
	if got := plan.RoundCHF(float64(total)); got != 2510 {
		t.Errorf("Reserve total = %d CHF, want 2510 CHF", got)
	}
}

func TestConceptExample_next_payments(t *testing.T) {
	// The next payment is searched per entry with Next(entry, today+1). A
	// payment of today still belongs to the reserve but is not the next one.
	// The concept lists the rent on the 1st of november for the lifestyle
	// account, the car insurance on the 15th of february 2027 and the tax on
	// the 20th of march 2027.
	from := mustDate(t, "2026-10-02")
	entries := exampleEntries(t)

	cases := []struct {
		name string
		want string
	}{
		{name: "Miete", want: "2026-11-01"},
		{name: "Autoversicherung", want: "2027-02-15"},
		{name: "Serafe", want: "2027-03-20"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var found bool
			for _, e := range entries {
				if e.Name != tc.name {
					continue
				}
				got, ok := plan.Next(e, from)
				if !ok {
					t.Fatalf("Next(%s) found no payment", tc.name)
				}
				if formatted := plan.FormatDate(got); formatted != tc.want {
					t.Errorf("Next(%s) = %s, want %s", tc.name, formatted, tc.want)
				}
				found = true
			}
			if !found {
				t.Fatalf("the entry %q is not part of the example", tc.name)
			}
		})
	}
}

func TestConceptExample_next_payments_of_the_remaining_entries(t *testing.T) {
	// The two remaining entries of the lifestyle account are due inside the
	// reserve window of 30 days, therefore their next payment is in october
	// and not in november.
	from := mustDate(t, "2026-10-02")
	entries := exampleEntries(t)

	cases := []struct {
		name string
		want string
	}{
		{name: "Krankenkasse", want: "2026-10-05"},
		{name: "Internet", want: "2026-10-15"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var found bool
			for _, e := range entries {
				if e.Name != tc.name {
					continue
				}
				got, ok := plan.Next(e, from)
				if !ok {
					t.Fatalf("Next(%s) found no payment", tc.name)
				}
				if formatted := plan.FormatDate(got); formatted != tc.want {
					t.Errorf("Next(%s) = %s, want %s", tc.name, formatted, tc.want)
				}
				found = true
			}
			if !found {
				t.Fatalf("the entry %q is not part of the example", tc.name)
			}
		})
	}
}
