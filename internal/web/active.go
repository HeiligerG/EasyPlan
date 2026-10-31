package web

import (
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// ActiveEntries returns the entries that count in the summaries.
//
// The entries of an inactive account are left out, and so are the entries
// that are not running on today any more. The result keeps the data of the
// account, so the caller can group the entries by account.
//
// The entries of an inactive account stay visible in the list of the
// entries, they are only left out of the dashboard, the reserve and the
// timeline.
func ActiveEntries(entries []store.Entry, today time.Time) []store.Entry {
	// Every entry is checked on its own: plan.ActiveOn decides whether the
	// entry is still running. The order of the entries is kept, and the
	// entries themselves are not changed.
	active := make([]store.Entry, 0, len(entries))
	for _, entry := range entries {
		if !entry.AccountActive {
			continue
		}
		if len(plan.ActiveOn([]plan.Entry{entry.Entry}, today)) != 1 {
			continue
		}
		active = append(active, entry)
	}
	return active
}

// planEntries returns the entries of the list as entries of package plan.
func planEntries(entries []store.Entry) []plan.Entry {
	list := make([]plan.Entry, len(entries))
	for i, entry := range entries {
		list[i] = entry.Entry
	}
	return list
}
