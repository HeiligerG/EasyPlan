package web

import (
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// Summary holds the values of the dashboard.
//
// The rows hold one row per active account, the total holds the values over
// all active accounts. The values are never rounded, the rounding happens
// when the page is displayed.
type Summary struct {
	// Rows holds one row per active account, in the order of the accounts.
	Rows []SummaryRow
	// Total holds the values over all active accounts.
	Total SummaryRow
	// Entries is the number of the entries that count in the summary.
	Entries int
}

// SummaryRow holds the values of one account or of the total.
type SummaryRow struct {
	// Name is the name of the account.
	Name string
	// Need holds the demand per month, week and day. The values are
	// unrounded amounts in Rappen.
	Need plan.Need
	// Reserve is the amount that is due in the next days, in Rappen.
	Reserve int64
	// NextName is the name of the entry of the next payment.
	NextName string
	// NextDate is the day of the next payment.
	NextDate time.Time
	// HasNext reports whether the account has a next payment.
	HasNext bool
}

// BuildSummary builds the values of the dashboard.
//
// The accounts are the accounts of the application, the entries are the
// entries of all accounts. Only the active accounts and the entries that
// still run on today count in the summary: the entries of a deactivated
// account and the entries that have ended are left out.
//
// The rows keep the order of the accounts. The total is calculated over all
// active entries and not over the rounded values of the rows, so the total
// can differ from the sum of the rows by one franc.
//
// The days are the number of days the reserve is calculated for.
func BuildSummary(accounts []store.Account, entries []store.Entry, today time.Time, days int) Summary {
	active := ActiveEntries(entries, today)

	grouped := make(map[int64][]store.Entry, len(accounts))
	for _, entry := range active {
		grouped[entry.AccountID] = append(grouped[entry.AccountID], entry)
	}

	summary := Summary{Entries: len(active)}
	for _, account := range accounts {
		if !account.IsActive {
			continue
		}
		accountEntries := grouped[account.ID]
		row := SummaryRow{
			Name:    account.Name,
			Need:    plan.Totals(planEntries(accountEntries)),
			Reserve: plan.Reserve(planEntries(accountEntries), today, days),
		}
		row.NextName, row.NextDate, row.HasNext = nextPayment(accountEntries, today)
		summary.Rows = append(summary.Rows, row)
	}

	summary.Total = SummaryRow{
		Name:    "Total",
		Need:    plan.Totals(planEntries(active)),
		Reserve: plan.Reserve(planEntries(active), today, days),
	}
	return summary
}

// nextPayment returns the name and the day of the next payment of the given
// entries.
//
// The next payment is the earliest day of all entries that is on or after
// today, the day of today included. When several entries are due on the same
// day, the entry with the smaller name wins, and the entry with the smaller
// identifier when the names are equal. The result is therefore deterministic.
//
// The third return value is false when none of the entries has a further
// payment.
func nextPayment(entries []store.Entry, today time.Time) (name string, date time.Time, ok bool) {
	var (
		best     store.Entry
		bestDate time.Time
		found    bool
	)
	for _, entry := range entries {
		next, hasNext := plan.Next(entry.Entry, today)
		if !hasNext {
			continue
		}
		if !found || next.Before(bestDate) || (next.Equal(bestDate) && winsTie(entry, best)) {
			best, bestDate, found = entry, next, true
		}
	}
	if !found {
		return "", time.Time{}, false
	}
	return best.Name, bestDate, true
}

// winsTie reports whether the entry wins the tie against the entry that is
// already the next payment.
//
// The entry with the smaller name wins, and the entry with the smaller
// identifier when the names are equal.
func winsTie(entry, best store.Entry) bool {
	if entry.Name != best.Name {
		return entry.Name < best.Name
	}
	return entry.ID < best.ID
}
