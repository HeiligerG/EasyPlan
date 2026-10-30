package plan

import (
	"math"
	"time"
)

const (
	// minYear is the first supported year of a calendar day.
	minYear = 1
	// maxYear is the last supported year of a calendar day.
	maxYear = 9999
	// maxDayOffset is the number of days between 0001-01-01 and 9999-12-31.
	maxDayOffset = 3652058
	// maxMonthOffset is the number of months between 0001-01-01 and
	// 9999-12-31, rounded up.
	maxMonthOffset = 120000
)

// Occurrences returns all occurrence dates of the entry in the half-open
// range [from, to), in ascending order.
//
// The bounds are interpreted as calendar days. An occurrence is never before
// the start date of the entry and never after its end date. The unit "once"
// yields exactly the start date; its interval and its end date are ignored.
//
// The function returns no dates for an empty range or for an entry without an
// occurrence in it. Occurrences and Next share the core helper nth, and it is
// the only place where the occurrence dates are created.
func Occurrences(e Entry, from, to time.Time) []time.Time {
	e = normalizeEntry(e)
	from = dayStart(from)
	to = dayStart(to)
	if !from.Before(to) {
		return nil
	}
	switch e.Unit {
	case Once:
		if e.StartDate.Before(from) || !e.StartDate.Before(to) {
			return nil
		}
		return []time.Time{e.StartDate}
	case Week, Month, Year:
	default:
		return nil
	}
	if e.Interval <= 0 {
		return nil
	}
	n := firstIndex(e, from)
	if n < 0 {
		return nil
	}
	var dates []time.Time
	for {
		t, ok := nth(e, n)
		if !ok {
			break
		}
		if t.Before(from) {
			// Cannot happen: firstIndex returns the first occurrence that is
			// on or after from.
			break
		}
		if !t.Before(to) {
			break
		}
		if e.EndsOn != nil && t.After(*e.EndsOn) {
			break
		}
		dates = append(dates, t)
		if n == math.MaxInt64 {
			break
		}
		n++
	}
	return dates
}

// Next returns the first occurrence of the entry that is on or after from.
//
// The second return value is false when the entry has no further occurrence,
// for example because its end date lies before from. The date is derived with
// the same formula as Occurrences, so long intervals such as every three
// years are supported as well.
func Next(e Entry, from time.Time) (time.Time, bool) {
	e = normalizeEntry(e)
	from = dayStart(from)
	switch e.Unit {
	case Once:
		if e.StartDate.Before(from) {
			return time.Time{}, false
		}
		return e.StartDate, true
	case Week, Month, Year:
	default:
		return time.Time{}, false
	}
	if e.Interval <= 0 {
		return time.Time{}, false
	}
	n := firstIndex(e, from)
	if n < 0 {
		return time.Time{}, false
	}
	t, ok := nth(e, n)
	if !ok {
		return time.Time{}, false
	}
	if e.EndsOn != nil && t.After(*e.EndsOn) {
		return time.Time{}, false
	}
	return t, true
}

// firstIndex returns the index of the first occurrence of the entry that is
// on or after from, or -1 when the entry has no such occurrence.
func firstIndex(e Entry, from time.Time) int64 {
	n := estimateIndex(e, from)
	// The n-th occurrence grows strictly with n, so the estimate is corrected
	// by moving in both directions until it is exact.
	for n > 0 {
		prev, ok := nth(e, n-1)
		if !ok {
			break
		}
		if prev.Before(from) {
			break
		}
		n--
	}
	for {
		t, ok := nth(e, n)
		if !ok {
			return -1
		}
		if !t.Before(from) {
			return n
		}
		if n == math.MaxInt64 {
			return -1
		}
		n++
	}
}

// estimateIndex returns a first guess for the index of the first occurrence
// of the entry that is on or after from.
//
// The guess is off by at most one occurrence, because the only difference
// between the guess and the exact index comes from the day clamping of the
// end of a month.
func estimateIndex(e Entry, from time.Time) int64 {
	switch e.Unit {
	case Week:
		days := daysBetween(from, e.StartDate)
		if days <= 0 {
			return 0
		}
		step, ok := mulInt64(7, int64(e.Interval))
		if !ok {
			// The step between two terms does not fit into an int64, so the
			// entry has no term after the start date. The index 1 lets nth
			// fail cleanly instead of losing the start date.
			return 1
		}
		return ceilDiv(days, step)
	case Month:
		months := monthsBetween(e.StartDate, from)
		if months <= 0 {
			return 0
		}
		return ceilDiv(months, int64(e.Interval))
	case Year:
		months := monthsBetween(e.StartDate, from)
		if months <= 0 {
			return 0
		}
		step, ok := mulInt64(12, int64(e.Interval))
		if !ok {
			return math.MaxInt64
		}
		return ceilDiv(months, step)
	}
	return 0
}

// nth returns the n-th occurrence of the entry, where n == 0 is the start
// date. The second return value is false when the date is not representable.
//
// The n-th occurrence is always derived from the start date and never from the
// previous occurrence. A monthly entry that starts on the 31st therefore
// recurs on the last day of the shorter months and returns to the 31st
// afterwards: 31.01. -> 28.02. -> 31.03.
func nth(e Entry, n int64) (time.Time, bool) {
	if n < 0 {
		return time.Time{}, false
	}
	switch e.Unit {
	case Once:
		return e.StartDate, true
	case Week:
		days, ok := mulInt64(n, int64(e.Interval))
		if !ok {
			return time.Time{}, false
		}
		days, ok = mulInt64(days, 7)
		if !ok {
			return time.Time{}, false
		}
		if days > maxDayOffset {
			return time.Time{}, false
		}
		t := e.StartDate.AddDate(0, 0, int(days))
		if !supported(t) {
			return time.Time{}, false
		}
		return t, true
	case Month:
		months, ok := mulInt64(n, int64(e.Interval))
		if !ok {
			return time.Time{}, false
		}
		if months > maxMonthOffset {
			return time.Time{}, false
		}
		return addMonths(e.StartDate, months)
	case Year:
		months, ok := mulInt64(n, int64(e.Interval))
		if !ok {
			return time.Time{}, false
		}
		months, ok = mulInt64(months, 12)
		if !ok {
			return time.Time{}, false
		}
		if months > maxMonthOffset {
			return time.Time{}, false
		}
		return addMonths(e.StartDate, months)
	}
	return time.Time{}, false
}

// addMonths adds the given number of calendar months to start and clamps the
// day of the start to the last day of the resulting month.
func addMonths(start time.Time, months int64) (time.Time, bool) {
	total := int64(start.Year())*12 + int64(start.Month()) - 1 + months
	year := total / 12
	month := time.Month(total%12 + 1)
	if year < minYear || year > maxYear {
		return time.Time{}, false
	}
	day := start.Day()
	if last := daysInMonth(int(year), month); day > last {
		day = last
	}
	return time.Date(int(year), month, day, 0, 0, 0, 0, time.UTC), true
}

// daysInMonth returns the number of days of the given month.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// supported reports whether the date lies inside the supported year range.
func supported(t time.Time) bool {
	year := t.UTC().Year()
	return year >= minYear && year <= maxYear
}

// mulInt64 multiplies two non-negative int64 values and reports whether the
// result is representable.
func mulInt64(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a != 0 && b > math.MaxInt64/a {
		return 0, false
	}
	return a * b, true
}

// ceilDiv divides a by b and rounds the result up.
func ceilDiv(a, b int64) int64 {
	if b <= 0 {
		return 0
	}
	quotient := a / b
	if a%b != 0 {
		quotient++
	}
	return quotient
}
