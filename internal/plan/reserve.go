package plan

import "time"

// Reserve returns the amount in Rappen that is due in the half-open range
// [today, today+days).
//
// Every occurrence counts, also when the same entry occurs several times in
// the range. The calculation uses the same occurrences as the timeline.
func Reserve(entries []Entry, today time.Time, days int) int64 {
	if days <= 0 {
		return 0
	}
	from := dayStart(today)
	if days > maxDayOffset {
		days = maxDayOffset
	}
	to := from.AddDate(0, 0, days)
	var total int64
	for _, e := range entries {
		for range Occurrences(e, from, to) {
			total += e.AmountMinor
		}
	}
	return total
}
