package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// entriesPage serves the list of the entries.
//
// The list shows all entries, grouped by account, and the link to the form of
// a new entry. The entries of a deactivated account stay visible, the account
// is marked as deactivated.
func (s *server) entriesPage(w http.ResponseWriter, r *http.Request) {
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

	s.render(w, http.StatusOK, "entries", PageData{
		Title:  "Einträge",
		Active: "entries",
		Data: entriesData{
			Today:  today,
			Groups: entryGroups(accounts, entries, today),
		},
	})
}

// entryDeletePage asks for the confirmation of the deletion.
//
// The page is the second step of the deletion, so a click on the button of
// the list cannot delete an entry by accident. An unknown entry gets the page
// for 404.
func (s *server) entryDeletePage(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	entry, err := s.store.GetEntry(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	s.render(w, http.StatusOK, "entrydelete", PageData{
		Title:  "Eintrag löschen",
		Active: "entries",
		Data: entryDeleteData{
			ID:   entry.ID,
			Name: entry.Name,
		},
	})
}

// entryDelete deletes an entry.
//
// The deletion is a writing request, so the origin middleware checks it. The
// answer is the status 303 with the address of the list.
func (s *server) entryDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	if err := s.store.DeleteEntry(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	http.Redirect(w, r, "/entries", http.StatusSeeOther)
}

// entriesData holds the data of the list of the entries.
type entriesData struct {
	// Today is the current calendar day in the time zone of the application.
	Today time.Time
	// Groups holds one group per account, in the order of the accounts.
	Groups []entryGroup
}

// entryGroup holds the entries of one account.
type entryGroup struct {
	// Account is the account of the group.
	Account store.Account
	// Entries are the entries of the account.
	Entries []entryRow
}

// entryRow holds one entry of the list with the texts of the display.
type entryRow struct {
	// ID is the identifier of the entry.
	ID int64
	// Name is the label of the entry.
	Name string
	// Amount is the amount of one occurrence in Rappen.
	Amount int64
	// Rhythm describes the recurrence of the entry in German.
	Rhythm string
	// Start is the first day of the entry, as DD.MM.YYYY.
	Start string
	// End is the last day of the entry, as DD.MM.YYYY. It is empty when the
	// entry has no end date.
	End string
	// Done reports whether the entry is finished, as plan.IsDone says.
	Done bool
	// Ended reports whether the entry does not occur any more.
	Ended bool
}

// entryDeleteData holds the data of the page that confirms the deletion.
type entryDeleteData struct {
	// ID is the identifier of the entry.
	ID int64
	// Name is the label of the entry.
	Name string
}

// entryGroups groups the entries by their account.
//
// The accounts keep their order, so the groups appear in the order of the
// account list. An account without entries gets an empty group.
func entryGroups(accounts []store.Account, entries []store.Entry, today time.Time) []entryGroup {
	grouped := make(map[int64][]store.Entry, len(accounts))
	for _, entry := range entries {
		grouped[entry.AccountID] = append(grouped[entry.AccountID], entry)
	}

	groups := make([]entryGroup, 0, len(accounts))
	for _, account := range accounts {
		group := entryGroup{Account: account}
		for _, entry := range grouped[account.ID] {
			group.Entries = append(group.Entries, newEntryRow(entry, today))
		}
		groups = append(groups, group)
	}
	return groups
}

// newEntryRow builds the display texts of an entry.
//
// The state of the entry comes from the plan package: plan.IsDone knows
// whether the entry is finished, plan.ActiveOn knows whether it occurs any
// more. The web layer decides nothing about the terms itself.
func newEntryRow(entry store.Entry, today time.Time) entryRow {
	row := entryRow{
		ID:     entry.ID,
		Name:   entry.Name,
		Amount: entry.AmountMinor,
		Rhythm: RhythmText(entry.Entry),
		Start:  FormatDate(entry.StartDate),
		Done:   plan.IsDone(entry.Entry, today),
	}
	if entry.EndsOn != nil {
		row.End = FormatDate(*entry.EndsOn)
	}
	// plan.ActiveOn leaves out the entries whose end date lies before today,
	// so an empty result means that the entry is over.
	row.Ended = len(plan.ActiveOn([]plan.Entry{entry.Entry}, today)) == 0
	return row
}

// RhythmText describes the recurrence of an entry in German.
//
// A once entry is named with its day, the other units are named with their
// interval.
func RhythmText(entry plan.Entry) string {
	switch entry.Unit {
	case plan.Once:
		return "einmalig am " + FormatDate(entry.StartDate)
	case plan.Week:
		if entry.Interval == 1 {
			return "wöchentlich"
		}
		return "alle " + strconv.Itoa(entry.Interval) + " Wochen"
	case plan.Month:
		if entry.Interval == 1 {
			return "monatlich"
		}
		return "alle " + strconv.Itoa(entry.Interval) + " Monate"
	case plan.Year:
		if entry.Interval == 1 {
			return "jährlich"
		}
		return "alle " + strconv.Itoa(entry.Interval) + " Jahre"
	}
	return "unbekannt"
}

// FormatAmountInput formats an amount in Rappen for a form field.
//
// The amount is written in francs with at most two decimals and without a
// thousands separator, so the parser of the web layer reads it back. The
// amount of an entry is always positive, the sign is only kept so a strange
// value does not become a wrong one.
func FormatAmountInput(minor int64) string {
	negative := minor < 0
	value := absUint64(minor)

	text := strconv.FormatUint(value/100, 10)
	if centimes := value % 100; centimes > 0 {
		text += "." + fmt.Sprintf("%02d", centimes)
	}
	if negative {
		return "-" + text
	}
	return text
}

// entryID reads the identifier of the entry from the path.
//
// A malformed identifier is reported with false, the caller answers with the
// page for 404.
func entryID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// entryPath returns the address of the form of an entry.
func entryPath(id int64) string {
	return "/entries/" + strconv.FormatInt(id, 10)
}
