package web

import (
	"net/http"
	"strconv"
	"time"
)

// reserveDays are the numbers of days the reserve can be shown for.
var reserveDays = []int{7, 30, 60, 90}

// defaultReserveDays is the number of days that is used when the query
// parameter does not name one of the allowed values.
const defaultReserveDays = 30

// dashboardPage serves the dashboard.
//
// The dashboard shows the demand per account and in total, the reserve for
// the selected number of days and the next payment of every account. The
// number of days comes from the query parameter n, an unknown value falls
// back to the default.
//
// The values are calculated live, nothing is stored. A failure of the store
// leads to the error page.
func (s *server) dashboardPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := s.today()

	accounts, err := s.store.ListAccounts(ctx, false)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	entries, err := s.store.ListEntries(ctx)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	days := parseReserveDays(r.URL.Query().Get("n"))

	s.render(w, http.StatusOK, "dashboard", PageData{
		Title:  "Dashboard",
		Active: "dashboard",
		Data: dashboardData{
			Today:   today,
			Days:    days,
			Options: reserveDays,
			Summary: BuildSummary(accounts, entries, today, days),
		},
	})
}

// dashboardData holds the data of the dashboard page.
type dashboardData struct {
	// Today is the current calendar day in the time zone of the application.
	Today time.Time
	// Days is the number of days the reserve is shown for.
	Days int
	// Options are the numbers of days that can be chosen.
	Options []int
	// Summary holds the values of the accounts and the total.
	Summary Summary
}

// parseReserveDays returns the number of days of the query parameter.
//
// A missing value and a value that is not one of the allowed numbers fall
// back to the default of 30 days. The page answers with the standard
// selection instead of an error.
func parseReserveDays(value string) int {
	if value == "" {
		return defaultReserveDays
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		return defaultReserveDays
	}
	for _, option := range reserveDays {
		if option == days {
			return days
		}
	}
	return defaultReserveDays
}
