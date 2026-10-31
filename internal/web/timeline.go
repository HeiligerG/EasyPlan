package web

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// timelineMonths is the number of months the timeline shows.
const timelineMonths = 12

// monthNames are the German names of the months.
//
// The names stand in a table of their own, so the display does not depend on
// the locale of the server.
var monthNames = [...]string{
	"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember",
}

// weekdayNames are the German abbreviations of the weekdays.
//
// The first name belongs to the Sunday, as time.Weekday counts.
var weekdayNames = [...]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}

// Timeline holds the data of the timeline page.
type Timeline struct {
	// Today is the current calendar day in the time zone of the application.
	Today time.Time
	// Months holds one group per month of the period, from the month of
	// today to the month of the last day of the period.
	Months []TimelineMonth
	// Accounts are the accounts the filter offers. They are the active
	// accounts, in the order of the accounts.
	Accounts []TimelineAccount
	// AccountID is the identifier of the chosen account. It is 0 when the
	// page shows the payments of all accounts.
	AccountID int64
	// AccountName is the name of the chosen account. It is empty when no
	// account is chosen.
	AccountName string
	// HasPayments reports whether the period holds a payment at all.
	HasPayments bool
}

// TimelineMonth holds the payments of one month.
type TimelineMonth struct {
	// Title is the heading of the month, as "Oktober 2026".
	Title string
	// Events are the payments of the month, in the order of the display.
	Events []TimelineEvent
	// Sum is the sum of the payments of the month in Rappen. The value is
	// not rounded, the rounding happens when the page is displayed.
	Sum int64
}

// TimelineEvent holds one payment of the timeline.
type TimelineEvent struct {
	// Date is the day of the payment.
	Date time.Time
	// DateText is the day of the payment for the display, with the weekday
	// and without the year, as "Do 01.10.".
	DateText string
	// Name is the name of the entry.
	Name string
	// Account is the name of the account of the entry.
	Account string
	// Amount is the amount of the payment in Rappen.
	Amount int64
}

// TimelineAccount is an account in the filter of the timeline.
type TimelineAccount struct {
	// ID is the identifier of the account.
	ID int64
	// Name is the name of the account.
	Name string
	// Selected reports whether the account is the chosen one.
	Selected bool
}

// timelinePage serves the timeline.
//
// The page shows the payments of the next twelve months, grouped by month.
// The account of the query parameter filters the payments, a value that names
// no active account shows all accounts.
//
// The values are calculated live, nothing is stored. A failure of the store
// leads to the error page.
func (s *server) timelinePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := s.today()

	accounts, err := s.store.ListAccounts(ctx, true)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	entries, err := s.store.ListEntries(ctx)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	accountID := ParseAccountFilter(r.URL.Query().Get("account"))

	timeline, err := BuildTimeline(accounts, entries, today, accountID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	s.render(w, http.StatusOK, "timeline", PageData{
		Title:  "Timeline",
		Active: "timeline",
		Data:   timeline,
	})
}

// BuildTimeline builds the data of the timeline page.
//
// The function is pure: it reads nothing but its arguments, so a test can
// check it with a fixed date and without a server.
//
// The period is the half-open range of the twelve months that begin with the
// day of today: [today, today + 12 months). Only the entries that still run
// on today count, the entries of a deactivated account are left out.
//
// The months of the display follow the period: the first group belongs to the
// month of today, the last one to the month of the last day of the period.
// The period holds twelve months plus the rest of the month of today, so the
// display holds twelve groups on the first day of a month and thirteen groups
// on every other day.
//
// The payments of the period come from plan.Timeline, the web layer does not
// calculate a single date. The payments of a day follow the order of the
// accounts, the order the visitor sees on the page of the accounts. The name
// of the entry and its identifier break the remaining ties, so the order of
// the payments is always the same.
//
// The account of the filter only counts when it names an active account. Any
// other value shows the payments of all accounts.
//
// The error is only returned when a payment of the period lies outside of the
// months of the display, which cannot happen. The payment is reported then
// and never dropped without a word.
func BuildTimeline(accounts []store.Account, entries []store.Entry, today time.Time, accountID int64) (Timeline, error) {
	active := activeAccounts(accounts)

	var (
		selected     int64
		selectedName string
	)
	for _, account := range active {
		if account.ID == accountID {
			selected, selectedName = account.ID, account.Name
			break
		}
	}

	running := ActiveEntries(entries, today)
	shown := make([]store.Entry, 0, len(running))
	for _, entry := range running {
		if selected == 0 || entry.AccountID == selected {
			shown = append(shown, entry)
		}
	}

	known := make(map[int64]store.Account, len(accounts))
	for _, account := range accounts {
		known[account.ID] = account
	}

	// The period is the half-open range [from, to). The months of the display
	// reach from the month of from to the month of the last day of the
	// period.
	from := dayStart(today)
	to := from.AddDate(0, timelineMonths, 0)
	last := to.AddDate(0, 0, -1)

	events := plan.Timeline(planEntries(shown), from, to)
	sortTimelineEvents(events, known)

	months := timelineMonthGroups(from, last)
	fromYear, fromMonth, _ := from.Date()
	grouped := 0
	for _, event := range events {
		year, month, _ := event.Date.Date()
		index := (year-fromYear)*12 + int(month-fromMonth)
		if index < 0 || index >= len(months) {
			return Timeline{}, fmt.Errorf(
				"web: the payment %q of the %s lies outside of the months of the timeline",
				event.Entry.Name, plan.FormatDate(event.Date))
		}
		group := &months[index]
		group.Events = append(group.Events, newTimelineEvent(event, known))
		group.Sum += event.Entry.AmountMinor
		grouped++
	}

	options := make([]TimelineAccount, 0, len(active))
	for _, account := range active {
		options = append(options, TimelineAccount{
			ID:       account.ID,
			Name:     account.Name,
			Selected: account.ID == selected,
		})
	}

	timeline := Timeline{
		Today:       today,
		Months:      months,
		Accounts:    options,
		AccountID:   selected,
		AccountName: selectedName,
		HasPayments: grouped > 0,
	}
	return timeline, nil
}

// ParseAccountFilter returns the identifier of the account of the query
// parameter.
//
// A missing value, a value that is not a number and a number that is not
// positive give 0, that means all accounts. An unknown account and a
// deactivated account are left out by BuildTimeline, so the page answers with
// the payments of all accounts.
func ParseAccountFilter(value string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// activeAccounts returns the active accounts, in the order they come in.
func activeAccounts(accounts []store.Account) []store.Account {
	active := make([]store.Account, 0, len(accounts))
	for _, account := range accounts {
		if account.IsActive {
			active = append(active, account)
		}
	}
	return active
}

// sortTimelineEvents sorts the payments of the period by day and then by the
// order of the account.
//
// The plan package sorts the payments of the same day by the identifier of
// the account. The order of the accounts is the order of the page of the
// accounts, so the payments of a day follow that order. The name of the entry
// and its identifier break the remaining ties.
func sortTimelineEvents(events []plan.Event, accounts map[int64]store.Account) {
	sort.SliceStable(events, func(i, j int) bool {
		a, b := events[i], events[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.Before(b.Date)
		}
		orderA := accountOrder(accounts, a.Entry.AccountID)
		orderB := accountOrder(accounts, b.Entry.AccountID)
		if orderA != orderB {
			return orderA < orderB
		}
		if a.Entry.Name != b.Entry.Name {
			return a.Entry.Name < b.Entry.Name
		}
		return a.Entry.ID < b.Entry.ID
	})
}

// accountOrder returns the order of an account in the list of the accounts.
//
// An account that is not in the map stands at the end of the list.
func accountOrder(accounts map[int64]store.Account, id int64) int {
	account, ok := accounts[id]
	if !ok {
		return len(accounts)
	}
	return account.SortOrder
}

// timelineMonthGroups returns the groups of the months of the period.
//
// The first group belongs to the month of the first day, the last one to the
// month of the last day. The period holds every month exactly once, so the
// groups stand in the order of the months.
func timelineMonthGroups(first, last time.Time) []TimelineMonth {
	fromYear, fromMonth, _ := first.Date()
	toYear, toMonth, _ := last.Date()
	count := (toYear-fromYear)*12 + int(toMonth-fromMonth) + 1

	months := make([]TimelineMonth, count)
	for i := range months {
		total := int(fromYear)*12 + int(fromMonth) - 1 + i
		months[i] = TimelineMonth{Title: monthTitle(total/12, time.Month(total%12+1))}
	}
	return months
}

// dayStart returns the calendar day of t at the start of the day.
//
// The time zone of t is kept, so the calendar day does not change.
func dayStart(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// newTimelineEvent builds the display texts of a payment.
func newTimelineEvent(event plan.Event, accounts map[int64]store.Account) TimelineEvent {
	payment := TimelineEvent{
		Date:     event.Date,
		DateText: dateText(event.Date),
		Name:     event.Entry.Name,
		Amount:   event.Entry.AmountMinor,
	}
	if account, ok := accounts[event.Entry.AccountID]; ok {
		payment.Account = account.Name
	}
	return payment
}

// monthTitle returns the heading of a month, as "Oktober 2026".
func monthTitle(year int, month time.Month) string {
	return fmt.Sprintf("%s %d", monthNames[int(month)-1], year)
}

// dateText returns the day of a payment for the display, as "Do 01.10.".
//
// The year stands in the heading of the month, so it is left out here.
func dateText(t time.Time) string {
	_, month, day := t.Date()
	return fmt.Sprintf("%s %02d.%02d.", weekdayNames[int(t.Weekday())], day, int(month))
}
