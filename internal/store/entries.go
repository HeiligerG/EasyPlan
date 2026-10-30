package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// Entry is a recurring expense of an account.
//
// It embeds plan.Entry, so the calculations of package plan can be applied
// directly, and adds the data of the owning account.
type Entry struct {
	plan.Entry
	// AccountName is the name of the owning account.
	AccountName string
	// AccountActive reports whether the owning account is active. The
	// entries of an inactive account stay readable, whether they are counted
	// in the summaries is up to the caller.
	AccountActive bool
}

// entryColumns are the columns of the entry queries, in the order in which
// scanEntry expects them.
const entryColumns = `e.id, e.account_id, e.name, e.amount_minor, e.start_date, e.unit, e.interval, e.ends_on, e.notes, a.name, a.is_active`

// entryQuery reads the entries together with the data of their account.
const entryQuery = `SELECT ` + entryColumns + ` FROM entries e JOIN accounts a ON a.id = e.account_id`

// ListEntries returns all entries with the data of their account.
//
// The entries are sorted by the order of the account, then by the start date
// and the identifier, so the list is stable and grouped by account.
func (s *Store) ListEntries(ctx context.Context) ([]Entry, error) {
	return queryEntries(ctx, s.db, entryQuery+` ORDER BY a.sort_order, a.id, e.start_date, e.id`)
}

// ListEntriesByAccount returns the entries of one account.
//
// The entries are sorted by the start date and the identifier. An account
// without entries returns an empty list, not an error.
func (s *Store) ListEntriesByAccount(ctx context.Context, accountID int64) ([]Entry, error) {
	return queryEntries(ctx, s.db, entryQuery+` WHERE e.account_id = ? ORDER BY e.start_date, e.id`, accountID)
}

// GetEntry returns one entry with the data of its account.
//
// An unknown entry is reported with ErrNotFound.
func (s *Store) GetEntry(ctx context.Context, id int64) (Entry, error) {
	entries, err := queryEntries(ctx, s.db, entryQuery+` WHERE e.id = ?`, id)
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, ErrNotFound
	}
	return entries[0], nil
}

// CreateEntry stores a new entry and returns its identifier.
//
// The entry is checked with plan.Entry.Validate before it is written, so the
// store never holds an entry that the calculations could not use. An account
// that does not exist is reported with ErrUnknownAccount.
func (s *Store) CreateEntry(ctx context.Context, e plan.Entry) (int64, error) {
	if err := e.Validate(); err != nil {
		return 0, fmt.Errorf("invalid entry: %w", err)
	}
	const insert = `INSERT INTO entries (account_id, name, amount_minor, start_date, unit, interval, ends_on, notes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := s.db.ExecContext(ctx, insert,
		e.AccountID,
		strings.TrimSpace(e.Name),
		e.AmountMinor,
		plan.FormatDate(e.StartDate),
		string(e.Unit),
		e.Interval,
		formatEndsOn(e.EndsOn),
		e.Notes,
	)
	if err != nil {
		return 0, entryWriteError(err, e.AccountID, "insert")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cannot read new entry id: %w", err)
	}
	return id, nil
}

// UpdateEntry overwrites an existing entry.
//
// The identifier of the entry decides which entry is written, the account may
// be changed as well. The entry is checked with plan.Entry.Validate before it
// is written. An unknown entry is reported with ErrNotFound, an unknown
// account with ErrUnknownAccount.
func (s *Store) UpdateEntry(ctx context.Context, e plan.Entry) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("invalid entry: %w", err)
	}
	const update = `UPDATE entries
SET account_id = ?, name = ?, amount_minor = ?, start_date = ?, unit = ?, interval = ?, ends_on = ?, notes = ?
WHERE id = ?`
	res, err := s.db.ExecContext(ctx, update,
		e.AccountID,
		strings.TrimSpace(e.Name),
		e.AmountMinor,
		plan.FormatDate(e.StartDate),
		string(e.Unit),
		e.Interval,
		formatEndsOn(e.EndsOn),
		e.Notes,
		e.ID,
	)
	if err != nil {
		return entryWriteError(err, e.AccountID, "update")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("cannot update entry: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteEntry deletes an entry.
//
// An unknown entry is reported with ErrNotFound.
func (s *Store) DeleteEntry(ctx context.Context, id int64) error {
	const delete = "DELETE FROM entries WHERE id = ?"
	res, err := s.db.ExecContext(ctx, delete, id)
	if err != nil {
		return fmt.Errorf("cannot delete entry: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("cannot delete entry: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// queryEntries runs an entry query and returns its entries.
func queryEntries(ctx context.Context, q queryer, query string, args ...any) ([]Entry, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cannot read entries: %w", err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

// scanEntries reads the entries of an open query.
func scanEntries(rows *sql.Rows) ([]Entry, error) {
	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot read entries: %w", err)
	}
	return entries, nil
}

// scanEntry reads one entry of an open query.
func scanEntry(rows *sql.Rows) (Entry, error) {
	var (
		entry     Entry
		startDate string
		unit      string
		endsOn    sql.NullString
		active    int
	)
	err := rows.Scan(
		&entry.ID,
		&entry.AccountID,
		&entry.Name,
		&entry.AmountMinor,
		&startDate,
		&unit,
		&entry.Interval,
		&endsOn,
		&entry.Notes,
		&entry.AccountName,
		&active,
	)
	if err != nil {
		return Entry{}, fmt.Errorf("cannot read entry: %w", err)
	}

	day, err := plan.ParseDate(startDate)
	if err != nil {
		return Entry{}, fmt.Errorf("cannot read entry %d: %w", entry.ID, err)
	}
	entry.StartDate = day
	entry.Unit = plan.Unit(unit)
	if endsOn.Valid {
		day, err := plan.ParseDate(endsOn.String)
		if err != nil {
			return Entry{}, fmt.Errorf("cannot read entry %d: %w", entry.ID, err)
		}
		entry.EndsOn = &day
	}
	entry.AccountActive = active != 0
	return entry, nil
}

// formatEndsOn formats the end date of an entry. A missing end date is
// stored as NULL.
func formatEndsOn(endsOn *time.Time) any {
	if endsOn == nil {
		return nil
	}
	return plan.FormatDate(*endsOn)
}

// entryWriteError turns the constraint violations of a write into the
// errors of the store.
func entryWriteError(err error, accountID int64, action string) error {
	switch {
	case isForeignKeyViolation(err):
		return fmt.Errorf("%w: id %d", ErrUnknownAccount, accountID)
	case isCheckViolation(err):
		return fmt.Errorf("invalid entry: %w", err)
	default:
		return fmt.Errorf("cannot %s entry: %w", action, err)
	}
}
