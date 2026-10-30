// Package plan contains the pure calculation logic of EasyPlan.
//
// The package performs no I/O and never reads the current time: callers pass
// "today" explicitly. It is the single place where recurring dates are
// derived, so the dashboard, the reserve and the timeline always agree.
//
// # Date convention
//
// Calendar days are time.Time values at 00:00:00 UTC without a time zone
// association, so day arithmetic is immune to daylight saving transitions.
// The store parses the YYYY-MM-DD strings of the database with ParseDate,
// callers pass "today" and the other range bounds explicitly. Every date
// argument is interpreted as the calendar day of its timestamp, normalised to
// 00:00:00 UTC.
//
// # Occurrences
//
// Occurrences and Next share the core helper nth(e, n), which is the only
// place where the occurrence dates are created. Reserve and Timeline are
// built on top of them, so an entry always ends on the same day of the month,
// no matter which function asks for its dates.
//
// # Rounding
//
// The package never rounds. Totals returns unrounded demands in Rappen and
// the caller rounds for display with RoundCHF, which rounds half away from
// zero to whole CHF.
package plan
