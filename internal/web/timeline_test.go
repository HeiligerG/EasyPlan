package web

import (
	"context"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The patterns of the page of the timeline.
var (
	// timelineMonthPattern matches the heading of a month of the timeline.
	timelineMonthPattern = regexp.MustCompile(`<h3 class="card__title">(.*?)</h3>`)
	// timelineSumPattern matches the sum of a month of the timeline.
	timelineSumPattern = regexp.MustCompile(`<p class="timeline-sum">Summe (.*?)</p>`)
	// timelineItemPattern matches one payment of the timeline.
	timelineItemPattern = regexp.MustCompile(`(?s)<li class="list__item timeline-item">(.*?)</li>`)
	// timelineDatePattern matches the day of a payment of the timeline.
	timelineDatePattern = regexp.MustCompile(`datetime="([0-9-]+)"`)
)

// timelineMonths returns the headings of the months of the timeline.
func pageTimelineMonths(body string) []string {
	matches := timelineMonthPattern.FindAllStringSubmatch(body, -1)
	months := make([]string, 0, len(matches))
	for _, match := range matches {
		months = append(months, stripTags(match[1]))
	}
	return months
}

// timelineSums returns the sums of the months of the timeline.
//
// The entities of the sums are undone, so the values are the ones the browser
// shows to the visitor.
func timelineSums(body string) []string {
	matches := timelineSumPattern.FindAllStringSubmatch(body, -1)
	sums := make([]string, 0, len(matches))
	for _, match := range matches {
		sums = append(sums, strings.TrimSpace(html.UnescapeString(match[1])))
	}
	return sums
}

// timelineEvents returns the payments of the timeline with their day and
// their name.
func timelineEvents(body string) []string {
	matches := timelineItemPattern.FindAllStringSubmatch(body, -1)
	events := make([]string, 0, len(matches))
	for _, match := range matches {
		day := timelineDatePattern.FindStringSubmatch(match[1])
		name := stripTags(match[1])
		events = append(events, day[1]+" "+name)
	}
	return events
}

// buildTimeline builds the data of the timeline and fails the test when the
// build fails.
//
// The build of the timeline only fails when a payment of the period lies
// outside of the months of the display, which cannot happen. The tests use
// the helper so that a failure of the build does not pass unnoticed.
func buildTimeline(t *testing.T, accounts []store.Account, entries []store.Entry, today time.Time, accountID int64) Timeline {
	t.Helper()

	timeline, err := BuildTimeline(accounts, entries, today, accountID)
	if err != nil {
		t.Fatalf("BuildTimeline() failed: %v", err)
	}
	return timeline
}

// TestBuildTimeline_matches_the_example_of_the_concept checks the timeline
// against the example of the concept.
func TestBuildTimeline_matches_the_example_of_the_concept(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)

	if len(timeline.Months) != 12 {
		t.Fatalf("the timeline holds %d months, want 12", len(timeline.Months))
	}
	want := []string{
		"Oktober 2026", "November 2026", "Dezember 2026",
		"Januar 2027", "Februar 2027", "März 2027",
		"April 2027", "Mai 2027", "Juni 2027",
		"Juli 2027", "August 2027", "September 2027",
	}
	for i, title := range want {
		if timeline.Months[i].Title != title {
			t.Errorf("the month %d is %q, want %q", i, timeline.Months[i].Title, title)
		}
	}

	var total int64
	var count int
	for _, month := range timeline.Months {
		count += len(month.Events)
		total += month.Sum
	}
	if count != 38 {
		t.Errorf("the timeline holds %d payments, want 38", count)
	}
	if total != 3165500 {
		t.Errorf("the timeline sums to %d Rappen, want 3165500", total)
	}

	sums := map[string]int64{}
	for _, month := range timeline.Months {
		sums[month.Title] = month.Sum
	}
	if sums["Oktober 2026"] != 251000 {
		t.Errorf("the sum of the october 2026 is %d Rappen, want 251000", sums["Oktober 2026"])
	}
	if sums["Februar 2027"] != 371000 {
		t.Errorf("the sum of the february 2027 is %d Rappen, want 371000", sums["Februar 2027"])
	}
	if sums["März 2027"] != 284500 {
		t.Errorf("the sum of the march 2027 is %d Rappen, want 284500", sums["März 2027"])
	}
}

// TestBuildTimeline_holds_the_period_of_twelve_months checks that the period
// begins with today and ends with the day before the same day of the next
// year.
func TestBuildTimeline_holds_the_period_of_twelve_months(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Miete", 200000, "2026-10-01", plan.Month, 1),
	}

	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)

	if len(timeline.Months[0].Events) != 1 {
		t.Fatalf("the month of today holds %d payments, want 1", len(timeline.Months[0].Events))
	}
	if got := plan.FormatDate(timeline.Months[0].Events[0].Date); got != "2026-10-01" {
		t.Errorf("the payment is on %q, want 2026-10-01", got)
	}

	// The payment of the 1st of october 2027 lies outside the period, the
	// payment of the 1st of september 2027 lies inside.
	var days []string
	for _, month := range timeline.Months {
		for _, event := range month.Events {
			days = append(days, plan.FormatDate(event.Date))
		}
	}
	if len(days) != 12 {
		t.Fatalf("the timeline holds %d payments, want 12: %v", len(days), days)
	}
	if days[0] != "2026-10-01" {
		t.Errorf("the first payment is on %q, want 2026-10-01", days[0])
	}
	if days[len(days)-1] != "2027-09-01" {
		t.Errorf("the last payment is on %q, want 2027-09-01", days[len(days)-1])
	}
	for _, day := range days {
		if day == "2027-10-01" {
			t.Error("the payment of the 1st of october 2027 is inside the period, want it outside")
		}
	}
}

// TestBuildTimeline_leaves_out_the_deactivated_accounts checks that the
// payments of a deactivated account do not show up.
func TestBuildTimeline_leaves_out_the_deactivated_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	accounts := app.accounts(t)
	entries := app.entries(t)
	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)

	for _, month := range timeline.Months {
		for _, event := range month.Events {
			if event.Account == "Sparen" {
				t.Errorf("the timeline shows the payment %q of the deactivated account", event.Name)
			}
		}
	}
	if len(timeline.Accounts) != 2 {
		t.Errorf("the filter offers %d accounts, want 2", len(timeline.Accounts))
	}
}

// TestBuildTimeline_leaves_out_the_ended_entries checks that the payments of
// an entry whose end date lies before today do not show up.
func TestBuildTimeline_leaves_out_the_ended_entries(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}
	ended := storeEntry(t, 1, 1, "Miete", 200000, "2026-01-01", plan.Month, 1)
	ended.EndsOn = dayPtr(t, "2026-09-30")

	timeline := buildTimeline(t, accounts, []store.Entry{ended}, conceptToday, 0)

	if timeline.HasPayments {
		t.Error("the timeline shows payments, want none")
	}
	for _, month := range timeline.Months {
		if len(month.Events) != 0 {
			t.Errorf("the month %q holds %d payments, want none", month.Title, len(month.Events))
		}
	}
}

// TestBuildTimeline_orders_the_payments_of_a_day_by_the_order_of_the_accounts
// checks the order of the payments of the same day.
func TestBuildTimeline_orders_the_payments_of_a_day_by_the_order_of_the_accounts(t *testing.T) {
	accounts := []store.Account{
		{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
		{ID: 2, Name: "Sparen", SortOrder: 2, IsActive: true},
	}
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Miete", 200000, "2026-10-01", plan.Month, 1),
		storeEntry(t, 2, 2, "Serafe", 33500, "2026-10-01", plan.Month, 1),
	}

	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)
	events := timeline.Months[0].Events
	if len(events) != 2 {
		t.Fatalf("the month holds %d payments, want 2", len(events))
	}
	if events[0].Account != "Lifestyle" || events[1].Account != "Sparen" {
		t.Errorf("the payments are %q and %q, want the account Lifestyle first",
			events[0].Account, events[1].Account)
	}

	// The account Sparen moves in front, so its payment comes first.
	accounts[0].SortOrder = 2
	accounts[1].SortOrder = 1
	timeline = buildTimeline(t, accounts, entries, conceptToday, 0)
	events = timeline.Months[0].Events
	if events[0].Account != "Sparen" || events[1].Account != "Lifestyle" {
		t.Errorf("the payments are %q and %q, want the account Sparen first",
			events[0].Account, events[1].Account)
	}
}

// TestTimelinePage_follows_the_order_of_the_accounts checks that the payments
// of a day follow the order of the accounts of the page of the accounts.
//
// The order of the accounts is the order of the visitor, so the account that
// the visitor moves up comes first in the timeline as well.
func TestTimelinePage_follows_the_order_of_the_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")
	sparen := accountByName(t, app, "Sparen")

	// Both entries are due on the same day.
	createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 200000,
		StartDate: mustDay(t, "2026-10-01"), Unit: plan.Month, Interval: 1,
	})
	createEntry(t, app, plan.Entry{
		AccountID: sparen.ID, Name: "Serafe", AmountMinor: 33500,
		StartDate: mustDay(t, "2026-10-01"), Unit: plan.Month, Interval: 1,
	})

	events := timelineEvents(app.get(t, "/timeline").Body.String())
	if len(events) != 24 {
		t.Fatalf("the page holds %d payments, want 24", len(events))
	}
	// The two first payments are due on the same day.
	if !contains(events[0], "Miete") || !contains(events[1], "Serafe") {
		t.Errorf("the payments of the first day are %q and %q, want the miete first",
			events[0], events[1])
	}

	// The account Sparen moves in front of the account Lifestyle.
	if err := app.store.MoveAccountUp(context.Background(), sparen.ID); err != nil {
		t.Fatalf("MoveAccountUp() failed: %v", err)
	}

	events = timelineEvents(app.get(t, "/timeline").Body.String())
	if len(events) != 24 {
		t.Fatalf("the page holds %d payments, want 24", len(events))
	}
	if !contains(events[0], "Serafe") || !contains(events[1], "Miete") {
		t.Errorf("the payments of the first day are %q and %q, want the payment of the account Sparen first",
			events[0], events[1])
	}
}

// TestBuildTimeline_orders_the_payments_of_a_day_by_the_name_and_the_id
// checks the remaining ties of the order.
func TestBuildTimeline_orders_the_payments_of_a_day_by_the_name_and_the_id(t *testing.T) {
	accounts := []store.Account{{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true}}
	entries := []store.Entry{
		storeEntry(t, 2, 1, "Zebra", 10000, "2026-10-01", plan.Month, 1),
		storeEntry(t, 1, 1, "Alpha", 10000, "2026-10-01", plan.Month, 1),
	}

	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)
	events := timeline.Months[0].Events
	if len(events) != 2 {
		t.Fatalf("the month holds %d payments, want 2", len(events))
	}
	if events[0].Name != "Alpha" || events[1].Name != "Zebra" {
		t.Errorf("the payments are %q and %q, want \"Alpha\" first", events[0].Name, events[1].Name)
	}

	// Two entries with the same name are ordered by their identifier.
	entries = []store.Entry{
		storeEntry(t, 5, 1, "Alpha", 10000, "2026-10-01", plan.Month, 1),
		storeEntry(t, 3, 1, "Alpha", 10000, "2026-10-01", plan.Month, 1),
	}
	timeline = buildTimeline(t, accounts, entries, conceptToday, 0)
	events = timeline.Months[0].Events
	if events[0].Amount != events[1].Amount {
		t.Error("the payments do not hold the same amount")
	}
}

// TestSortTimelineEvents_breaks_the_tie_by_the_identifier checks the last
// tie of the order of the payments.
//
// Two payments of the same day of the same account with the same name can
// only be told apart by their identifier. The identifier is not shown on the
// page, so the rule is checked with the sorting function itself.
func TestSortTimelineEvents_breaks_the_tie_by_the_identifier(t *testing.T) {
	accounts := map[int64]store.Account{
		1: {ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
	}
	day := mustDay(t, "2026-10-01")

	events := []plan.Event{
		{Date: day, Entry: plan.Entry{ID: 7, AccountID: 1, Name: "Alpha", AmountMinor: 10000, Unit: plan.Month, Interval: 1}},
		{Date: day, Entry: plan.Entry{ID: 3, AccountID: 1, Name: "Alpha", AmountMinor: 10000, Unit: plan.Month, Interval: 1}},
	}
	sortTimelineEvents(events, accounts)

	if events[0].Entry.ID != 3 || events[1].Entry.ID != 7 {
		t.Errorf("the payments have the identifiers %d and %d, want 3 and 7",
			events[0].Entry.ID, events[1].Entry.ID)
	}
}

// TestSortTimelineEvents_puts_the_order_of_the_account_before_the_name
// checks that the order of the account wins against the name of the entry.
func TestSortTimelineEvents_puts_the_order_of_the_account_before_the_name(t *testing.T) {
	accounts := map[int64]store.Account{
		1: {ID: 1, Name: "Lifestyle", SortOrder: 2, IsActive: true},
		2: {ID: 2, Name: "Sparen", SortOrder: 1, IsActive: true},
	}
	day := mustDay(t, "2026-10-01")

	// The name of the first payment is the smaller one, but its account
	// stands behind the account of the second payment.
	events := []plan.Event{
		{Date: day, Entry: plan.Entry{ID: 1, AccountID: 1, Name: "Alpha", AmountMinor: 10000, Unit: plan.Month, Interval: 1}},
		{Date: day, Entry: plan.Entry{ID: 2, AccountID: 2, Name: "Zebra", AmountMinor: 10000, Unit: plan.Month, Interval: 1}},
	}
	sortTimelineEvents(events, accounts)

	if events[0].Entry.AccountID != 2 {
		t.Errorf("the first payment belongs to the account %d, want 2", events[0].Entry.AccountID)
	}
}

// TestBuildTimeline_shows_a_payment_of_an_unknown_account checks the case
// that a database should not hold: an entry whose account is not in the list
// of the accounts. The payment shows up without the name of the account and
// stands at the end of the order of the day.
func TestBuildTimeline_shows_a_payment_of_an_unknown_account(t *testing.T) {
	accounts := []store.Account{
		{ID: 1, Name: "Lifestyle", SortOrder: 1, IsActive: true},
		{ID: 2, Name: "Sparen", SortOrder: 2, IsActive: true},
		{ID: 3, Name: "Steuern", SortOrder: 3, IsActive: true},
	}
	entries := []store.Entry{
		storeEntry(t, 1, 1, "Miete", 200000, "2026-10-01", plan.Month, 1),
		storeEntry(t, 2, 99, "Serafe", 33500, "2026-10-01", plan.Month, 1),
	}

	timeline := buildTimeline(t, accounts, entries, conceptToday, 0)

	events := timeline.Months[0].Events
	if len(events) != 2 {
		t.Fatalf("the month holds %d payments, want 2", len(events))
	}
	if events[0].Name != "Miete" {
		t.Errorf("the first payment is %q, want \"Miete\"", events[0].Name)
	}
	if events[1].Account != "" {
		t.Errorf("the payment of the unknown account names the account %q, want none", events[1].Account)
	}
}

// TestBuildTimeline_filters_by_the_account checks the filter of the page.
func TestBuildTimeline_filters_by_the_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)
	lifestyle := accountByName(t, app, "Lifestyle")

	timeline := buildTimeline(t, accounts, entries, conceptToday, lifestyle.ID)

	if timeline.AccountID != lifestyle.ID {
		t.Errorf("the timeline shows the account %d, want %d", timeline.AccountID, lifestyle.ID)
	}
	if timeline.AccountName != "Lifestyle" {
		t.Errorf("the timeline shows the account %q, want \"Lifestyle\"", timeline.AccountName)
	}

	var total int64
	var count int
	for _, month := range timeline.Months {
		count += len(month.Events)
		total += month.Sum
	}
	// The account Lifestyle holds three entries, each of them with twelve
	// payments in the period.
	if count != 36 {
		t.Errorf("the timeline holds %d payments, want 36", count)
	}
	if total != 3012000 {
		t.Errorf("the timeline sums to %d Rappen, want 3012000", total)
	}

	// The sum of a month is the sum of the payments that are shown.
	for _, month := range timeline.Months {
		var sum int64
		for _, event := range month.Events {
			sum += event.Amount
		}
		if sum != month.Sum {
			t.Errorf("the sum of the month %q is %d, want %d", month.Title, month.Sum, sum)
		}
	}
}

// TestBuildTimeline_shows_all_accounts_for_an_unknown_filter checks that a
// filter that names no active account shows the payments of all accounts.
func TestBuildTimeline_shows_all_accounts_for_an_unknown_filter(t *testing.T) {
	app := newTestApp(t, conceptToday)
	accounts := conceptAccounts(t, app)
	entries := app.entries(t)

	for _, accountID := range []int64{999, 0} {
		timeline := buildTimeline(t, accounts, entries, conceptToday, accountID)
		if timeline.AccountID != 0 {
			t.Errorf("the timeline shows the account %d, want all accounts", timeline.AccountID)
		}
		var count int
		for _, month := range timeline.Months {
			count += len(month.Events)
		}
		if count != 38 {
			t.Errorf("the timeline holds %d payments, want 38", count)
		}
	}
}

// TestBuildTimeline_shows_all_accounts_for_a_deactivated_account checks that
// a deactivated account cannot be chosen in the filter.
func TestBuildTimeline_shows_all_accounts_for_a_deactivated_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	accounts := app.accounts(t)
	entries := app.entries(t)
	timeline := buildTimeline(t, accounts, entries, conceptToday, sparen.ID)

	if timeline.AccountID != 0 {
		t.Errorf("the timeline shows the account %d, want all accounts", timeline.AccountID)
	}
	if timeline.AccountName != "" {
		t.Errorf("the timeline shows the account %q, want none", timeline.AccountName)
	}
}

// TestParseAccountFilter_reads_the_identifier checks the query parameter of
// the filter.
func TestParseAccountFilter_reads_the_identifier(t *testing.T) {
	cases := []struct {
		value string
		want  int64
	}{
		{value: "", want: 0},
		{value: "1", want: 1},
		{value: "42", want: 42},
		{value: " 42 ", want: 42},
		{value: "abc", want: 0},
		{value: "0", want: 0},
		{value: "-1", want: 0},
		{value: "1.5", want: 0},
		{value: "1a", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := ParseAccountFilter(tc.value); got != tc.want {
				t.Errorf("ParseAccountFilter(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

// TestTimelinePage_shows_the_payments_of_the_accounts checks the page of the
// timeline.
func TestTimelinePage_shows_the_payments_of_the_accounts(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	rec := app.get(t, "/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	months := pageTimelineMonths(body)
	if len(months) != 12 {
		t.Fatalf("the page holds %d months, want 12: %v", len(months), months)
	}
	if months[0] != "Oktober 2026" || months[11] != "September 2027" {
		t.Errorf("the page holds the months %v, want the october 2026 until the september 2027", months)
	}

	events := timelineEvents(body)
	if len(events) != 38 {
		t.Errorf("the page holds %d payments, want 38", len(events))
	}
	if !contains(events[0], "2026-10-01") {
		t.Errorf("the first payment is %q, want the 1st of october 2026", events[0])
	}
	if !contains(events[0], "Miete") {
		t.Errorf("the first payment is %q, want the miete", events[0])
	}

	sums := timelineSums(body)
	if len(sums) != 12 {
		t.Fatalf("the page holds %d sums, want 12", len(sums))
	}
	if sums[0] != "2'510" {
		t.Errorf("the sum of the october 2026 is %q, want \"2'510\"", sums[0])
	}
}

// TestTimelinePage_writes_the_names_in_german checks the names of the months
// and of the weekdays.
func TestTimelinePage_writes_the_names_in_german(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	body := app.get(t, "/timeline").Body.String()

	for _, name := range []string{"Oktober", "Februar", "März", "September"} {
		if !contains(body, name) {
			t.Errorf("the page does not show the month %q", name)
		}
	}
	// The 1st of october 2026 is a Thursday.
	if !contains(body, "Do 01.10.") {
		t.Errorf("the page does not show the weekday of the payment:\n%s", body)
	}
}

// TestTimelinePage_filters_by_the_account checks the filter of the page.
func TestTimelinePage_filters_by_the_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)
	lifestyle := accountByName(t, app, "Lifestyle")

	rec := app.get(t, "/timeline?account="+strconv.FormatInt(lifestyle.ID, 10))
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, "nur vom Konto Lifestyle") {
		t.Errorf("the page does not name the account of the filter:\n%s", body)
	}
	events := timelineEvents(body)
	if len(events) != 36 {
		t.Errorf("the page holds %d payments, want 36", len(events))
	}
	if contains(body, "Autoversicherung") {
		t.Errorf("the page shows a payment of another account:\n%s", body)
	}
}

// TestTimelinePage_shows_all_accounts_for_an_unknown_filter checks that a
// filter that names no account shows all payments.
func TestTimelinePage_shows_all_accounts_for_an_unknown_filter(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	for _, query := range []string{"?account=999", "?account=abc", "?account=", "?account=-1"} {
		t.Run(query, func(t *testing.T) {
			rec := app.get(t, "/timeline"+query)
			if rec.Code != http.StatusOK {
				t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
			}
			if events := timelineEvents(rec.Body.String()); len(events) != 38 {
				t.Errorf("the page holds %d payments, want 38", len(events))
			}
		})
	}
}

// TestTimelinePage_leaves_out_the_payment_at_the_end_of_the_period checks
// that the period ends with the day before the same day of the next year.
//
// A payment that lies exactly on the day of today plus twelve months is
// outside of the period, so the page shows the empty state.
func TestTimelinePage_leaves_out_the_payment_at_the_end_of_the_period(t *testing.T) {
	app := newTestApp(t, conceptToday)
	lifestyle := accountByName(t, app, "Lifestyle")

	// The payment of the 1st of october 2027 is exactly twelve months after
	// the 1st of october 2026.
	createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: "Miete", AmountMinor: 200000,
		StartDate: mustDay(t, "2027-10-01"), Unit: plan.Once, Interval: 1,
	})

	rec := app.get(t, "/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, "In den nächsten zwölf Monaten ist keine Zahlung fällig") {
		t.Errorf("the page shows a payment outside of the period:\n%s", body)
	}
	if len(timelineEvents(body)) != 0 {
		t.Errorf("the page holds %d payments, want none", len(timelineEvents(body)))
	}
}

// TestTimelinePage_shows_the_empty_state checks the page of a database
// without entries.
func TestTimelinePage_shows_the_empty_state(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.get(t, "/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, "In den nächsten zwölf Monaten ist keine Zahlung fällig") {
		t.Errorf("the page does not show the empty state:\n%s", body)
	}
	if !contains(body, `href="/entries"`) {
		t.Errorf("the page does not link to the entries:\n%s", body)
	}
	if len(pageTimelineMonths(body)) != 0 {
		t.Errorf("the page shows the months of a timeline without payments:\n%s", body)
	}
}
