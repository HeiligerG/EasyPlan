package plan

import (
	"sort"
	"time"
)

// Event is a single occurrence of an entry on the timeline.
type Event struct {
	// Date is the date of the occurrence.
	Date time.Time
	// Entry is the entry the occurrence belongs to.
	Entry Entry
}

// Timeline returns the occurrences of all entries in the half-open range
// [from, to).
//
// The events are sorted by date. Occurrences on the same date are ordered by
// account, then by name and finally by the identifier of the entry, so the
// order is stable and deterministic.
func Timeline(entries []Entry, from, to time.Time) []Event {
	var events []Event
	for _, e := range entries {
		for _, date := range Occurrences(e, from, to) {
			events = append(events, Event{Date: date, Entry: e})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		a, b := events[i], events[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.Before(b.Date)
		}
		if a.Entry.AccountID != b.Entry.AccountID {
			return a.Entry.AccountID < b.Entry.AccountID
		}
		if a.Entry.Name != b.Entry.Name {
			return a.Entry.Name < b.Entry.Name
		}
		return a.Entry.ID < b.Entry.ID
	})
	return events
}

// IsDone reports whether the entry is finished on today.
//
// A once entry is done when its date lies before today. A recurring entry is
// done when its end date is set and lies before today.
func IsDone(e Entry, today time.Time) bool {
	day := dayStart(today)
	if e.Unit == Once {
		return dayStart(e.StartDate).Before(day)
	}
	if e.EndsOn != nil {
		return dayStart(*e.EndsOn).Before(day)
	}
	return false
}
