package web

import (
	"net/http"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// zurichLocation returns the time zone of the application.
func zurichLocation(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatalf("time.LoadLocation(\"Europe/Zurich\") failed: %v", err)
	}
	return location
}

// TestTheApplication_uses_the_day_of_the_time_zone checks that the day of
// the application is the day of the time zone and not the day of UTC.
//
// The instant of the test is the 31st of october 2026 at 23:30 UTC. The
// summer time in Switzerland ended on the 25th of october 2026, so Zurich is
// one hour ahead of UTC and the instant lies on the 1st of november 2026 in
// Zurich.
//
// Every page of the application works with the day of the clock: the day of
// the dashboard, the day of the timeline and the state of the entries of the
// list.
func TestTheApplication_uses_the_day_of_the_time_zone(t *testing.T) {
	location := zurichLocation(t)

	instant := time.Date(2026, time.October, 31, 23, 30, 0, 0, time.UTC)
	if got := instant.UTC().Format("2006-01-02"); got != "2026-10-31" {
		t.Fatalf("the day of the instant in UTC is %q, want 2026-10-31", got)
	}
	if got := instant.In(location).Format("2006-01-02"); got != "2026-11-01" {
		t.Fatalf("the day of the instant in Zurich is %q, want 2026-11-01", got)
	}

	app := newTestAppIn(t, instant, location)
	lifestyle := accountByName(t, app, "Lifestyle")
	sparen := accountByName(t, app, "Sparen")

	// The one-off entry of the 31st of october lies in the past of the day
	// of Zurich, so it is done and it does not count any more.
	createEntry(t, app, plan.Entry{
		AccountID: lifestyle.ID, Name: "Zahnarzt", AmountMinor: 30000,
		StartDate: mustDay(t, "2026-10-31"), Unit: plan.Once, Interval: 1,
	})
	// The entry of the 15th runs on, so the pages are never empty.
	createEntry(t, app, plan.Entry{
		AccountID: sparen.ID, Name: "Autoversicherung", AmountMinor: 120000,
		StartDate: mustDay(t, "2026-01-15"), Unit: plan.Month, Interval: 1,
	})

	t.Run("the dashboard shows the day of the time zone", func(t *testing.T) {
		rec := app.get(t, "/")
		if rec.Code != http.StatusOK {
			t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()

		if !contains(body, "Stand 01.11.2026") {
			t.Errorf("the dashboard does not show the day of the time zone:\n%s", body)
		}
		if contains(body, "Stand 31.10.2026") {
			t.Errorf("the dashboard shows the day of UTC:\n%s", body)
		}

		rows := dashboardRows(body)
		if len(rows) != 3 {
			t.Fatalf("the dashboard holds %d rows, want 3:\n%s", len(rows), body)
		}
		// The one-off entry of the 31st of october lies in the past, so the
		// account has no next payment any more.
		if rows[0].name != "Lifestyle" {
			t.Fatalf("the first row is %q, want \"Lifestyle\"", rows[0].name)
		}
		if rows[0].next != "–" {
			t.Errorf("the next payment of the account Lifestyle is %q, want none", rows[0].next)
		}
		if rows[0].reserve != "0" {
			t.Errorf("the reserve of the account Lifestyle is %q, want \"0\"", rows[0].reserve)
		}

		// The reserve of the total holds only the entry of the 15th, the
		// one-off entry of the 31st of october lies in the past.
		total := dashboardTotal(t, body)
		if total.reserve != "1'200" {
			t.Errorf("the reserve of the total is %q, want \"1'200\"", total.reserve)
		}
	})

	t.Run("the timeline starts with the month of the time zone", func(t *testing.T) {
		rec := app.get(t, "/timeline")
		if rec.Code != http.StatusOK {
			t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()

		months := pageTimelineMonths(body)
		if len(months) == 0 {
			t.Fatalf("the page holds no month:\n%s", body)
		}
		if months[0] != "November 2026" {
			t.Errorf("the first month is %q, want \"November 2026\"", months[0])
		}
		if contains(body, "Stand 31.10.2026") {
			t.Errorf("the page shows the day of UTC:\n%s", body)
		}
	})

	t.Run("the list marks the entry of the past as done", func(t *testing.T) {
		rec := app.get(t, "/entries")
		if rec.Code != http.StatusOK {
			t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()

		row := entryRowByName(t, body, "Zahnarzt")
		if !contains(row.status, "erledigt") {
			t.Errorf("the state of the entry Zahnarzt is %q, want \"erledigt\"", row.status)
		}
	})
}

// TestTheApplication_uses_the_day_of_the_time_zone_in_the_summer_time checks
// the same rule with the summer time.
//
// In the summer time Zurich is two hours ahead of UTC, so the instant of the
// test lies in the day of the 1st of november in Zurich and in the day of the
// 31st of october in UTC.
func TestTheApplication_uses_the_day_of_the_time_zone_in_the_summer_time(t *testing.T) {
	location := zurichLocation(t)

	instant := time.Date(2026, time.July, 31, 22, 30, 0, 0, time.UTC)
	if got := instant.In(location).Format("2006-01-02"); got != "2026-08-01" {
		t.Fatalf("the day of the instant in Zurich is %q, want 2026-08-01", got)
	}

	app := newTestAppIn(t, instant, location)

	body := app.get(t, "/").Body.String()
	if !contains(body, "Stand 01.08.2026") {
		t.Errorf("the dashboard does not show the day of the time zone:\n%s", body)
	}
	if contains(body, "Stand 31.07.2026") {
		t.Errorf("the dashboard shows the day of UTC:\n%s", body)
	}
}

// TestTheApplication_uses_the_day_of_the_time_zone_of_the_entries checks that
// the state of the entries of the list uses the day of the time zone.
//
// The entry ends on the 31st of october 2026. On the 31st of october it still
// runs, on the 1st of november it is over. The day of the clock in Zurich is
// the 1st of november, so the entry is over.
func TestTheApplication_uses_the_day_of_the_time_zone_of_the_entries(t *testing.T) {
	location := zurichLocation(t)
	instant := time.Date(2026, time.October, 31, 23, 30, 0, 0, time.UTC)

	app := newTestAppIn(t, instant, location)
	lifestyle := accountByName(t, app, "Lifestyle")

	ended := monthlyEntry(t, lifestyle.ID, "Internet", 6000, "2026-01-15")
	ended.EndsOn = dayPtr(t, "2026-10-31")
	createEntry(t, app, ended)

	body := app.get(t, "/entries").Body.String()

	row := entryRowByName(t, body, "Internet")
	if !contains(row.status, "beendet") {
		t.Errorf("the state of the entry Internet is %q, want \"beendet\"", row.status)
	}
}
