package web

import (
	"context"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The patterns of the table of the dashboard.
var (
	// dashboardBodyPattern matches the rows of the accounts of the dashboard.
	dashboardBodyPattern = regexp.MustCompile(`(?s)<tbody>(.*?)</tbody>`)
	// dashboardFootPattern matches the rows of the total of the dashboard.
	dashboardFootPattern = regexp.MustCompile(`(?s)<tfoot>(.*?)</tfoot>`)
	// dashboardRowPattern matches one row of the table of the dashboard.
	dashboardRowPattern = regexp.MustCompile(`(?s)<tr>\s*<th scope="row">(.*?)</th>\s*` +
		`<td class="amount">(.*?)</td>\s*` +
		`<td class="amount">(.*?)</td>\s*` +
		`<td class="amount">(.*?)</td>\s*` +
		`<td class="amount">(.*?)</td>\s*` +
		`<td>(.*?)</td>\s*</tr>`)
	// currentPeriodPattern matches the chosen number of days of the reserve.
	currentPeriodPattern = regexp.MustCompile(`<a class="period is-active" href="/\?n=(\d+)" aria-current="true">`)
)

// dashboardRow holds thevalues of one row of the dashboard.
type dashboardRow struct {
	name    string
	month   string
	week    string
	day     string
	reserve string
	next    string
}

// parseDashboardRows reads the rows of a block of the table of the dashboard.
//
// The entities of the cells are undone, so the values are the ones the browser
// shows to the visitor.
func parseDashboardRows(block string) []dashboardRow {
	matches := dashboardRowPattern.FindAllStringSubmatch(block, -1)
	rows := make([]dashboardRow, 0, len(matches))
	for _, match := range matches {
		rows = append(rows, dashboardRow{
			name:    strings.TrimSpace(html.UnescapeString(match[1])),
			month:   strings.TrimSpace(html.UnescapeString(match[2])),
			week:    strings.TrimSpace(html.UnescapeString(match[3])),
			day:     strings.TrimSpace(html.UnescapeString(match[4])),
			reserve: strings.TrimSpace(html.UnescapeString(match[5])),
			next:    strings.TrimSpace(html.UnescapeString(match[6])),
		})
	}
	return rows
}

// dashboardRows returns the rows of the accounts of the dashboard.
func dashboardRows(body string) []dashboardRow {
	block := dashboardBodyPattern.FindStringSubmatch(body)
	if block == nil {
		return nil
	}
	return parseDashboardRows(block[1])
}

// dashboardTotal returns the row of the total of the dashboard.
func dashboardTotal(t *testing.T, body string) dashboardRow {
	t.Helper()
	block := dashboardFootPattern.FindStringSubmatch(body)
	if block == nil {
		t.Fatalf("the dashboard has no row of the total:\n%s", body)
	}
	rows := parseDashboardRows(block[1])
	if len(rows) != 1 {
		t.Fatalf("the dashboard holds %d rows of the total, want 1:\n%s", len(rows), body)
	}
	return rows[0]
}

// TestDashboardPage_shows_the_values_of_the_concept checks the page of the
// dashboard against the example of the concept.
func TestDashboardPage_shows_the_values_of_the_concept(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	rec := app.get(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	rows := dashboardRows(body)
	if len(rows) != 3 {
		t.Fatalf("the table holds %d rows, want 3:\n%s", len(rows), body)
	}
	want := []dashboardRow{
		{name: "Lifestyle", month: "2'510", week: "577", day: "82", reserve: "2'510", next: "Miete 01.10.2026"},
		{name: "Sparen", month: "100", week: "23", day: "3", reserve: "0", next: "Autoversicherung 15.02.2027"},
		{name: "Steuern", month: "28", week: "6", day: "1", reserve: "0", next: "Serafe 20.03.2027"},
	}
	for i, w := range want {
		got := rows[i]
		if got.name != w.name {
			t.Errorf("the row %d is %q, want %q", i, got.name, w.name)
		}
		if got.month != w.month {
			t.Errorf("the demand per month of %q is %q, want %q", w.name, got.month, w.month)
		}
		if got.week != w.week {
			t.Errorf("the demand per week of %q is %q, want %q", w.name, got.week, w.week)
		}
		if got.day != w.day {
			t.Errorf("the demand per day of %q is %q, want %q", w.name, got.day, w.day)
		}
		if got.reserve != w.reserve {
			t.Errorf("the reserve of %q is %q, want %q", w.name, got.reserve, w.reserve)
		}
		if got.next != w.next {
			t.Errorf("the next payment of %q is %q, want %q", w.name, got.next, w.next)
		}
	}

	total := dashboardTotal(t, body)
	if total.name != "Total" {
		t.Errorf("the total is %q, want \"Total\"", total.name)
	}
	if total.month != "2'638" || total.week != "607" || total.day != "87" {
		t.Errorf("the total is %q, %q, %q, want \"2'638\", \"607\", \"87\"", total.month, total.week, total.day)
	}
	if total.reserve != "2'510" {
		t.Errorf("the reserve of the total is %q, want \"2'510\"", total.reserve)
	}
}

// TestDashboardPage_renders_the_amounts_so_the_browser_shows_the_apostrophe
// checks that the thousands separator of the amounts reaches the reader.
//
// The apostrophe is a special character in the text of an HTML page, so the
// escaper of the template writes it as an entity. The browser shows the
// apostrophe again, so the amount stays readable. The test reads the page the
// way the browser does: it undoes the escaping of the entities.
func TestDashboardPage_renders_the_amounts_so_the_browser_shows_the_apostrophe(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	body := app.get(t, "/").Body.String()

	if !contains(html.UnescapeString(body), "2'510") {
		t.Errorf("the page does not show the amount 2'510:\n%s", body)
	}
	// The raw page holds the entity of the apostrophe.
	if !contains(body, "2&#39;510") {
		t.Errorf("the page does not write the amount as an entity:\n%s", body)
	}
}

// TestDashboardPage_offers_the_numbers_of_the_reserve checks the selection of
// the number of days.
func TestDashboardPage_offers_the_numbers_of_the_reserve(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	body := app.get(t, "/").Body.String()
	for _, days := range []string{"7", "30", "60", "90"} {
		if !contains(body, `href="/?n=`+days+`"`) {
			t.Errorf("the page does not offer the number of days %s:\n%s", days, body)
		}
	}
}

// TestDashboardPage_marks_the_chosen_number_of_days checks the selection of
// the number of days and the reserve that belongs to it.
func TestDashboardPage_marks_the_chosen_number_of_days(t *testing.T) {
	cases := []struct {
		query   string
		days    string
		reserve string
	}{
		{query: "?n=7", days: "7", reserve: "2'450"},
		{query: "?n=30", days: "30", reserve: "2'510"},
		{query: "?n=60", days: "60", reserve: "5'020"},
		{query: "?n=90", days: "90", reserve: "7'530"},
		{query: "", days: "30", reserve: "2'510"},
		{query: "?n=abc", days: "30", reserve: "2'510"},
		{query: "?n=15", days: "30", reserve: "2'510"},
		{query: "?n=0", days: "30", reserve: "2'510"},
		{query: "?n=-7", days: "30", reserve: "2'510"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			app := newTestApp(t, conceptToday)
			conceptAccounts(t, app)

			rec := app.get(t, "/"+tc.query)
			if rec.Code != http.StatusOK {
				t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
			}
			body := rec.Body.String()

			match := currentPeriodPattern.FindStringSubmatch(body)
			if match == nil {
				t.Fatalf("the page marks no number of days:\n%s", body)
			}
			if match[1] != tc.days {
				t.Errorf("the page marks the number of days %q, want %q", match[1], tc.days)
			}
			if countOccurrences(body, `aria-current="true"`) != 1 {
				t.Errorf("the page marks %d numbers of days, want 1", countOccurrences(body, `aria-current="true"`))
			}

			total := dashboardTotal(t, body)
			if total.reserve != tc.reserve {
				t.Errorf("the reserve is %q, want %q", total.reserve, tc.reserve)
			}
			if !contains(body, "Reserve für "+tc.days+" Tage") {
				t.Errorf("the page does not name the number of days %q:\n%s", tc.days, body)
			}
		})
	}
}

// TestDashboardPage_shows_the_empty_state checks the page of a database
// without entries.
func TestDashboardPage_shows_the_empty_state(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.get(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	if !contains(body, "Es sind noch keine aktiven Einträge vorhanden") {
		t.Errorf("the page does not show the empty state:\n%s", body)
	}
	if !contains(body, `href="/entries"`) {
		t.Errorf("the page does not link to the entries:\n%s", body)
	}
	if len(dashboardRows(body)) != 0 {
		t.Errorf("the page shows a table without entries:\n%s", body)
	}
}

// TestDashboardPage_leaves_out_the_deactivated_account checks that a
// deactivated account does not show up on the dashboard.
func TestDashboardPage_leaves_out_the_deactivated_account(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	sparen := accountByName(t, app, "Sparen")
	if err := app.store.SetAccountActive(context.Background(), sparen.ID, false); err != nil {
		t.Fatalf("SetAccountActive() failed: %v", err)
	}

	body := app.get(t, "/").Body.String()
	for _, row := range dashboardRows(body) {
		if row.name == "Sparen" {
			t.Errorf("the dashboard shows the deactivated account %q:\n%s", row.name, body)
		}
	}
}

// TestDashboardPage_shows_the_day_of_the_clock checks that the page uses the
// injected clock.
func TestDashboardPage_shows_the_day_of_the_clock(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	body := app.get(t, "/").Body.String()
	if !contains(body, "Stand 01.10.2026") {
		t.Errorf("the page does not show the day of the clock:\n%s", body)
	}
}

// TestDashboardPage_shows_the_next_payment_of_the_day_after checks the rule
// of ADR-0004 on the page.
func TestDashboardPage_shows_the_next_payment_of_the_day_after(t *testing.T) {
	app := newTestApp(t, mustDay(t, "2026-10-02"))
	conceptAccounts(t, app)

	body := app.get(t, "/").Body.String()
	if !contains(body, "Krankenkasse 05.10.2026") {
		t.Errorf("the page does not show the next payment of the day after:\n%s", body)
	}
}

// TestDashboardPage_uses_the_number_of_days_of_the_query_parameter checks
// that the parameter is read and not ignored.
func TestDashboardPage_uses_the_number_of_days_of_the_query_parameter(t *testing.T) {
	app := newTestApp(t, conceptToday)
	conceptAccounts(t, app)

	rec := app.get(t, "/?n=90")
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	match := currentPeriodPattern.FindStringSubmatch(rec.Body.String())
	if match == nil || match[1] != "90" {
		t.Errorf("the page marks the number of days %v, want 90", match)
	}
	if !contains(rec.Body.String(), "Reserve für 90 Tage") {
		t.Error("the page does not name the number of days 90")
	}
}

// TestParseReserveDays_holds_the_allowed_numbers checks the list of the
// numbers of days of the dashboard.
func TestParseReserveDays_holds_the_allowed_numbers(t *testing.T) {
	if len(reserveDays) != 4 {
		t.Fatalf("the application offers %d numbers of days, want 4", len(reserveDays))
	}
	for i, want := range []int{7, 30, 60, 90} {
		if reserveDays[i] != want {
			t.Errorf("the number of days %d is %d, want %d", i, reserveDays[i], want)
		}
	}
	if defaultReserveDays != 30 {
		t.Errorf("the default is %d days, want 30", defaultReserveDays)
	}
	for _, days := range reserveDays {
		if got := parseReserveDays(strconv.Itoa(days)); got != days {
			t.Errorf("parseReserveDays(%d) = %d, want %d", days, got, days)
		}
	}
}
