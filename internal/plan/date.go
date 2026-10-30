package plan

import (
	"fmt"
	"time"
)

// dateLayout is the layout of the calendar days stored in the database.
const dateLayout = "2006-01-02"

// ParseDate parses a calendar day in YYYY-MM-DD format and returns it as a
// time.Time at 00:00:00 UTC.
//
// The value must be a complete and valid date; incomplete values such as
// "2026-1-1" and impossible dates such as "2026-02-30" are rejected.
func ParseDate(value string) (time.Time, error) {
	t, err := time.ParseInLocation(dateLayout, value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", value)
	}
	return t, nil
}

// FormatDate formats the calendar day of t as YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return dayStart(t).Format(dateLayout)
}

// dayStart returns the calendar day of t as a time.Time at 00:00:00 UTC.
//
// The day is taken from the location of t, so a timestamp created in the
// local time zone keeps its calendar day.
func dayStart(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// dayIndex returns the number of days between the calendar day of t and the
// Unix epoch. It is exact for every supported date.
func dayIndex(t time.Time) int64 {
	return dayStart(t).Unix() / 86400
}

// daysBetween returns the number of calendar days between a and b.
func daysBetween(a, b time.Time) int64 {
	return dayIndex(a) - dayIndex(b)
}

// monthsBetween returns the number of calendar months between the month of a
// and the month of b.
func monthsBetween(a, b time.Time) int64 {
	return (int64(b.Year())-int64(a.Year()))*12 + int64(b.Month()) - int64(a.Month())
}

// normalizeEntry returns a copy of the entry whose dates are calendar days at
// 00:00:00 UTC.
func normalizeEntry(e Entry) Entry {
	e.StartDate = dayStart(e.StartDate)
	if e.EndsOn != nil {
		endsOn := dayStart(*e.EndsOn)
		e.EndsOn = &endsOn
	}
	return e
}
