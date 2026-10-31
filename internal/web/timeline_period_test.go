package web

import (
	"net/http"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The days the tests of the period of the timeline use.
//
// The period begins with the day of today and ends with the day before the
// same day of the next year. The months of the display follow the period, so
// the first day of a month gives twelve groups and every other day gives
// thirteen groups.
var (
	// timelineFirstOfOctober is the day of the example of the concept.
	timelineFirstOfOctober = mustDayOf("2026-10-01")
	// timelineMiddleOfOctober is a day in the middle of a month.
	timelineMiddleOfOctober = mustDayOf("2026-10-15")
	// timelineLastOfOctober is the last day of a month.
	timelineLastOfOctober = mustDayOf("2026-10-31")
	// timelineLastOfFebruary is the last day of the month of a leap year.
	timelineLastOfFebruary = mustDayOf("2028-02-29")
)

// mustDayOf parses a calendar day of the tests.
//
// The days are constants of the tests, so a wrong literal is a mistake in the
// test itself and stops the whole package.
func mustDayOf(value string) time.Time {
	day, err := plan.ParseDate(value)
	if err != nil {
		panic("plan.ParseDate(" + value + ") failed: " + err.Error())
	}
	return day
}

// TestBuildTimeline_holds_the_months_of_the_period checks that the months of
// the display reach from the month of today to the month of the last day of
// the period.
//
// The period is the half-open range of the twelve months that begin with the
// day of today. The days of the period are not the same as the months of the
// display: a day that is not the first of a month needs a thirteenth group,
// otherwise the payments of the last month would fall out of the display.
func TestBuildTimeline_holds_the_months_of_the_period(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}

	cases := []struct {
		name      string
		today     time.Time
		wantCount int
		wantFirst string
		wantLast  string
	}{
		{
			name:      "the first of october 2026",
			today:     timelineFirstOfOctober,
			wantCount: 12,
			wantFirst: "Oktober 2026",
			wantLast:  "September 2027",
		},
		{
			name:      "the 15th of october 2026",
			today:     timelineMiddleOfOctober,
			wantCount: 13,
			wantFirst: "Oktober 2026",
			wantLast:  "Oktober 2027",
		},
		{
			name:      "the 31st of october 2026",
			today:     timelineLastOfOctober,
			wantCount: 13,
			wantFirst: "Oktober 2026",
			wantLast:  "Oktober 2027",
		},
		{
			name:      "the 29th of february 2028",
			today:     timelineLastOfFebruary,
			wantCount: 13,
			wantFirst: "Februar 2028",
			wantLast:  "Februar 2029",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			timeline := buildTimeline(t, accounts, nil, tc.today, 0)

			if len(timeline.Months) != tc.wantCount {
				t.Fatalf("the timeline holds %d months, want %d", len(timeline.Months), tc.wantCount)
			}
			if got := timeline.Months[0].Title; got != tc.wantFirst {
				t.Errorf("the first month is %q, want %q", got, tc.wantFirst)
			}
			if got := timeline.Months[len(timeline.Months)-1].Title; got != tc.wantLast {
				t.Errorf("the last month is %q, want %q", got, tc.wantLast)
			}
		})
	}
}

// TestBuildTimeline_holds_the_payments_of_the_whole_period checks that the
// period ends with the day before the same day of the next year.
//
// A payment on the last day of the period belongs to the timeline, a payment
// on the day of the end of the period does not. No payment may fall out of
// the display without a word.
func TestBuildTimeline_holds_the_payments_of_the_whole_period(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}

	cases := []struct {
		name     string
		today    time.Time
		lastDay  string
		afterDay string
	}{
		{
			name:     "the 15th of october 2026",
			today:    timelineMiddleOfOctober,
			lastDay:  "2027-10-14",
			afterDay: "2027-10-15",
		},
		{
			name:     "the 31st of october 2026",
			today:    timelineLastOfOctober,
			lastDay:  "2027-10-30",
			afterDay: "2027-10-31",
		},
		{
			name:     "the 29th of february 2028",
			today:    timelineLastOfFebruary,
			lastDay:  "2029-02-28",
			afterDay: "2029-03-01",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries := []store.Entry{
				storeEntry(t, 1, 1, "Letzter Tag", 10000, tc.lastDay, plan.Once, 1),
				storeEntry(t, 2, 1, "Tag danach", 10000, tc.afterDay, plan.Once, 1),
			}

			timeline := buildTimeline(t, accounts, entries, tc.today, 0)

			var days []string
			for _, month := range timeline.Months {
				for _, event := range month.Events {
					days = append(days, plan.FormatDate(event.Date))
				}
			}
			if len(days) != 1 {
				t.Fatalf("the timeline holds the payments %v, want only the payment of the last day of the period", days)
			}
			if days[0] != tc.lastDay {
				t.Errorf("the payment is on %q, want %q", days[0], tc.lastDay)
			}
		})
	}
}

// TestBuildTimeline_sums_the_payments_of_the_period checks that the timeline
// holds the payments of the period and that the sum of the groups is the sum
// of the period.
//
// The expected values are derived by hand from the entries of the case and from
// the half-open period [today, today + 12 months). They do not come from the
// code under test:
//
//   - The miete of 200'000 Rappen falls on the first of every month, so the
//     period of twelve months holds twelve of them:
//     12 x 200'000 = 2'400'000 Rappen.
//   - The internet of 6'000 Rappen falls on the 15th of every month, so the
//     period of twelve months holds twelve of them:
//     12 x 6'000 = 72'000 Rappen.
//   - The payment of the last day of the period lies inside of the period, so
//     it adds one payment and its amount to the sum.
//   - The payment of the day after the period lies outside of the period, so
//     it counts neither as a payment nor in the sum.
//
// The period of a case therefore holds 12 + 12 + 1 = 25 payments, and the sum
// is 2'400'000 + 72'000 + the amount of the last day of the period.
func TestBuildTimeline_sums_the_payments_of_the_period(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}

	cases := []struct {
		name        string
		today       time.Time
		lastDay     string
		afterDay    string
		lastAmount  int64
		afterAmount int64
		wantEvents  int
		wantSum     int64
	}{
		{
			// The period is [2026-10-01, 2027-10-01): the miete runs from
			// 2026-10-01 to 2027-09-01, the internet from 2026-10-15 to
			// 2027-09-15. The last day of the period is 2027-09-30. The day
			// after the period is 2027-10-01, on which the miete falls as
			// well; that miete lies outside of the period too.
			name:        "the first of october 2026",
			today:       timelineFirstOfOctober,
			lastDay:     "2027-09-30",
			afterDay:    "2027-10-01",
			lastAmount:  7000,
			afterAmount: 11000,
			wantEvents:  25,
			wantSum:     2479000,
		},
		{
			// The period is [2026-10-15, 2027-10-15): the miete runs from
			// 2026-11-01 to 2027-10-01, the internet from 2026-10-15 to
			// 2027-09-15. The last day of the period is 2027-10-14. The day
			// after the period is 2027-10-15, on which the internet falls as
			// well; that internet lies outside of the period too.
			name:        "the 15th of october 2026",
			today:       timelineMiddleOfOctober,
			lastDay:     "2027-10-14",
			afterDay:    "2027-10-15",
			lastAmount:  8000,
			afterAmount: 12000,
			wantEvents:  25,
			wantSum:     2480000,
		},
		{
			// The period is [2026-10-31, 2027-10-31): the miete runs from
			// 2026-11-01 to 2027-10-01, the internet from 2026-11-15 to
			// 2027-10-15. The last day of the period is 2027-10-30, the day
			// after the period is 2027-10-31.
			name:        "the 31st of october 2026",
			today:       timelineLastOfOctober,
			lastDay:     "2027-10-30",
			afterDay:    "2027-10-31",
			lastAmount:  9000,
			afterAmount: 13000,
			wantEvents:  25,
			wantSum:     2481000,
		},
		{
			// The period is [2028-02-29, 2029-03-01): twelve months after the
			// 29th of february 2028 is the 1st of march 2029, so the last day
			// of the period is 2029-02-28. The miete runs from 2028-03-01 to
			// 2029-02-01, the internet from 2028-03-15 to 2029-02-15. The day
			// after the period is 2029-03-01.
			name:        "the 29th of february 2028",
			today:       timelineLastOfFebruary,
			lastDay:     "2029-02-28",
			afterDay:    "2029-03-01",
			lastAmount:  10000,
			afterAmount: 14000,
			wantEvents:  25,
			wantSum:     2482000,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries := []store.Entry{
				storeEntry(t, 1, 1, "Miete", 200000, "2026-10-01", plan.Month, 1),
				storeEntry(t, 2, 1, "Internet", 6000, "2026-10-15", plan.Month, 1),
				storeEntry(t, 3, 1, "Letzter Tag", tc.lastAmount, tc.lastDay, plan.Once, 1),
				storeEntry(t, 4, 1, "Tag danach", tc.afterAmount, tc.afterDay, plan.Once, 1),
			}

			timeline := buildTimeline(t, accounts, entries, tc.today, 0)

			var (
				total  int64
				events int
			)
			for _, month := range timeline.Months {
				var sum int64
				for _, event := range month.Events {
					sum += event.Amount
					events++
				}
				if sum != month.Sum {
					t.Errorf("the sum of the month %q is %d, want %d", month.Title, sum, month.Sum)
				}
				total += month.Sum
			}
			if events != tc.wantEvents {
				t.Errorf("the timeline holds %d payments, want %d", events, tc.wantEvents)
			}
			if total != tc.wantSum {
				t.Errorf("the timeline sums to %d Rappen, want %d", total, tc.wantSum)
			}
		})
	}
}

// TestBuildTimeline_reports_no_payment_for_a_period_without_one checks that
// the timeline knows that the period holds no payment at all.
func TestBuildTimeline_reports_no_payment_for_a_period_without_one(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Früher", 10000, "2026-01-01", plan.Once, 1),
		storeEntry(t, 2, 1, "Später", 10000, "2030-01-01", plan.Once, 1),
	}

	timeline := buildTimeline(t, accounts, entries, timelineLastOfOctober, 0)

	if timeline.HasPayments {
		t.Error("the timeline reports payments, want none")
	}
	for _, month := range timeline.Months {
		if len(month.Events) != 0 {
			t.Errorf("the month %q holds %d payments, want none", month.Title, len(month.Events))
		}
	}
}

// TestBuildTimeline_fails_only_with_a_payment_outside_of_the_period checks
// that the build of the timeline does not fail in the normal case.
//
// The build fails when a payment of the period lies outside of the months of
// the display. That cannot happen, so the error is only reported and never
// returned to the visitor of a normal database.
func TestBuildTimeline_fails_only_with_a_payment_outside_of_the_period(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	for _, today := range []time.Time{
		timelineFirstOfOctober, timelineMiddleOfOctober,
		timelineLastOfOctober, timelineLastOfFebruary,
	} {
		t.Run(today.Format("2006-01-02"), func(t *testing.T) {
			timeline, err := BuildTimeline(accounts, entries, today, 0)
			if err != nil {
				t.Fatalf("BuildTimeline() failed: %v", err)
			}
			if len(timeline.Months) == 0 {
				t.Error("the timeline holds no month, want some")
			}
		})
	}
}

// TestTimelinePage_shows_the_payment_of_the_thirteenth_month checks the
// blocker of the review: a payment of the thirteenth month must show up on
// the page.
//
// The period of the timeline is the range of the twelve months that begin
// with the day of today. A day that is not the first of a month needs a
// thirteenth group of months, otherwise the payments of the last month fall
// out of the display without a word.
func TestTimelinePage_shows_the_payment_of_the_thirteenth_month(t *testing.T) {
	app := newTestApp(t, timelineLastOfOctober)
	conceptAccounts(t, app)

	rec := app.get(t, "/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	months := pageTimelineMonths(body)
	if len(months) != 13 {
		t.Fatalf("the page holds %d months, want 13: %v", len(months), months)
	}
	if months[0] != "Oktober 2026" || months[12] != "Oktober 2027" {
		t.Errorf("the page holds the months %v, want the october 2026 until the october 2027", months)
	}

	// The miete of the 1st of october 2027 is the payment of the thirteenth
	// month, it was lost before the correction.
	if !contains(body, `datetime="2027-10-01"`) {
		t.Errorf("the page does not show the payment of the 1st of october 2027:\n%s", body)
	}
}

// TestTimelinePage_shows_the_months_of_the_period checks the months of the
// page for a day that is not the first of a month.
func TestTimelinePage_shows_the_months_of_the_period(t *testing.T) {
	app := newTestApp(t, timelineMiddleOfOctober)
	conceptAccounts(t, app)

	body := app.get(t, "/timeline").Body.String()

	months := pageTimelineMonths(body)
	if len(months) != 13 {
		t.Fatalf("the page holds %d months, want 13: %v", len(months), months)
	}
	if months[0] != "Oktober 2026" || months[12] != "Oktober 2027" {
		t.Errorf("the page holds the months %v, want the october 2026 until the october 2027", months)
	}
}
