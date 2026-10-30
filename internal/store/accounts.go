package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Account is a group of entries that is summarised on its own.
type Account struct {
	// ID is the database identifier of the account.
	ID int64
	// Name is the display name of the account.
	Name string
	// SortOrder is the position of the account in the account list.
	SortOrder int
	// IsActive reports whether the account is counted in the summaries. The
	// entries of an inactive account stay readable.
	IsActive bool
}

// accountColumns are the columns of the account queries.
const accountColumns = "id, name, sort_order, is_active"

// ListAccounts returns the accounts, sorted by their order and then by their
// identifier.
//
// The inactive accounts are only part of the list if includeInactive is true.
func (s *Store) ListAccounts(ctx context.Context, includeInactive bool) ([]Account, error) {
	query := "SELECT " + accountColumns + " FROM accounts"
	if !includeInactive {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY sort_order, id"
	return listAccounts(ctx, s.db, query)
}

// GetAccount returns one account.
func (s *Store) GetAccount(ctx context.Context, id int64) (Account, error) {
	query := "SELECT " + accountColumns + " FROM accounts WHERE id = ?"
	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return Account{}, fmt.Errorf("cannot read account: %w", err)
	}
	defer rows.Close()

	accounts, err := scanAccounts(rows)
	if err != nil {
		return Account{}, err
	}
	if len(accounts) == 0 {
		return Account{}, ErrNotFound
	}
	return accounts[0], nil
}

// CreateAccount creates an account and returns its identifier.
//
// The name is trimmed. An empty name is rejected with ErrInvalidName, a name
// that is already taken with ErrDuplicateName. The new account goes to the
// end of the order.
func (s *Store) CreateAccount(ctx context.Context, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrInvalidName
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const nextOrder = "SELECT COALESCE(MAX(sort_order), 0) + 1 FROM accounts"
	var order int
	if err := tx.QueryRowContext(ctx, nextOrder).Scan(&order); err != nil {
		return 0, fmt.Errorf("cannot read next sort order: %w", err)
	}

	const insert = "INSERT INTO accounts (name, sort_order, is_active) VALUES (?, ?, 1)"
	res, err := tx.ExecContext(ctx, insert, name, order)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, fmt.Errorf("%w: %q", ErrDuplicateName, name)
		}
		return 0, fmt.Errorf("cannot insert account: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cannot read new account id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("cannot commit account: %w", err)
	}
	return id, nil
}

// RenameAccount gives the account a new name.
//
// The name is trimmed. An empty name is rejected with ErrInvalidName, a name
// that is already taken with ErrDuplicateName. An unknown account is
// reported with ErrNotFound.
func (s *Store) RenameAccount(ctx context.Context, id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}

	const update = "UPDATE accounts SET name = ? WHERE id = ?"
	res, err := s.db.ExecContext(ctx, update, name, id)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %q", ErrDuplicateName, name)
		}
		return fmt.Errorf("cannot rename account: %w", err)
	}
	return accountChanged(res, "rename")
}

// SetAccountActive activates or deactivates the account.
//
// The entries of a deactivated account stay readable, they are only left out
// of the summaries. An unknown account is reported with ErrNotFound.
func (s *Store) SetAccountActive(ctx context.Context, id int64, active bool) error {
	value := 0
	if active {
		value = 1
	}
	const update = "UPDATE accounts SET is_active = ? WHERE id = ?"
	res, err := s.db.ExecContext(ctx, update, value, id)
	if err != nil {
		return fmt.Errorf("cannot change account: %w", err)
	}
	return accountChanged(res, "change")
}

// SetAccountOrder sets the position of the account.
//
// The accounts are sorted by their order, ties are broken by the identifier.
// Use MoveAccountUp and MoveAccountDown to move an account relative to its
// neighbours. An unknown account is reported with ErrNotFound.
func (s *Store) SetAccountOrder(ctx context.Context, id int64, order int) error {
	const update = "UPDATE accounts SET sort_order = ? WHERE id = ?"
	res, err := s.db.ExecContext(ctx, update, order, id)
	if err != nil {
		return fmt.Errorf("cannot set sort order: %w", err)
	}
	return accountChanged(res, "sort")
}

// MoveAccountUp moves the account one position towards the front.
//
// The order contains all accounts, active and inactive, as they are listed in
// the account list. Moving the first account does nothing. An unknown
// account is reported with ErrNotFound.
func (s *Store) MoveAccountUp(ctx context.Context, id int64) error {
	return s.moveAccount(ctx, id, -1)
}

// MoveAccountDown moves the account one position towards the back.
//
// The order contains all accounts, active and inactive, as they are listed in
// the account list. Moving the last account does nothing. An unknown account
// is reported with ErrNotFound.
func (s *Store) MoveAccountDown(ctx context.Context, id int64) error {
	return s.moveAccount(ctx, id, 1)
}

// moveAccount moves an account by delta positions in the order.
//
// All accounts are numbered again afterwards, so the order stays consecutive
// and free of gaps.
func (s *Store) moveAccount(ctx context.Context, id int64, delta int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	accounts, err := listAccounts(ctx, tx, "SELECT "+accountColumns+" FROM accounts ORDER BY sort_order, id")
	if err != nil {
		return err
	}

	position := -1
	for i, account := range accounts {
		if account.ID == id {
			position = i
			break
		}
	}
	if position < 0 {
		return ErrNotFound
	}
	target := position + delta
	if target < 0 || target >= len(accounts) {
		// The account is already the first or the last one of the order.
		return nil
	}
	accounts[position], accounts[target] = accounts[target], accounts[position]

	const update = "UPDATE accounts SET sort_order = ? WHERE id = ?"
	for i, account := range accounts {
		if _, err := tx.ExecContext(ctx, update, i+1, account.ID); err != nil {
			return fmt.Errorf("cannot set sort order: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit order: %w", err)
	}
	return nil
}

// DeleteAccount deletes an account that has no entries.
//
// Konzept.md §3 forbids the deletion of an account that still has entries.
// Such an account is reported with ErrAccountInUse and can be deactivated
// instead. An unknown account is reported with ErrNotFound.
func (s *Store) DeleteAccount(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, "SELECT name FROM accounts WHERE id = ?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("cannot read account: %w", err)
	}

	const count = "SELECT COUNT(*) FROM entries WHERE account_id = ?"
	var entries int
	if err := tx.QueryRowContext(ctx, count, id).Scan(&entries); err != nil {
		return fmt.Errorf("cannot count entries: %w", err)
	}
	if entries > 0 {
		return fmt.Errorf("%w: %q", ErrAccountInUse, name)
	}

	const delete = "DELETE FROM accounts WHERE id = ?"
	if _, err := tx.ExecContext(ctx, delete, id); err != nil {
		if isForeignKeyViolation(err) {
			return fmt.Errorf("%w: %q", ErrAccountInUse, name)
		}
		return fmt.Errorf("cannot delete account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit deletion: %w", err)
	}
	return nil
}

// listAccounts runs an account query and returns its accounts.
func listAccounts(ctx context.Context, q queryer, query string, args ...any) ([]Account, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("cannot read accounts: %w", err)
	}
	defer rows.Close()
	return scanAccounts(rows)
}

// scanAccounts reads the accounts of an open query.
func scanAccounts(rows *sql.Rows) ([]Account, error) {
	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		var active int
		if err := rows.Scan(&account.ID, &account.Name, &account.SortOrder, &active); err != nil {
			return nil, fmt.Errorf("cannot read account: %w", err)
		}
		account.IsActive = active != 0
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot read accounts: %w", err)
	}
	return accounts, nil
}

// accountChanged reports whether an update touched an account. An update that
// touches nothing means that the account does not exist.
func accountChanged(res sql.Result, action string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("cannot %s account: %w", action, err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
