package store

import (
	"errors"

	"modernc.org/sqlite"
)

var (
	// ErrNotFound is returned when an account or an entry does not exist.
	ErrNotFound = errors.New("store: not found")
	// ErrDuplicateName is returned when an account name is already taken.
	ErrDuplicateName = errors.New("store: name already exists")
	// ErrInvalidName is returned when a name is empty.
	ErrInvalidName = errors.New("store: name must not be empty")
	// ErrAccountInUse is returned when an account still has entries and
	// therefore must not be deleted.
	ErrAccountInUse = errors.New("store: account still has entries")
	// ErrUnknownAccount is returned when an entry refers to an account that
	// does not exist.
	ErrUnknownAccount = errors.New("store: unknown account")
)

// The result codes of the SQLite constraint violations. The driver reports
// the extended result codes, the primary code of all of them is
// SQLITE_CONSTRAINT.
const (
	sqliteConstraint        = 19
	sqliteConstraintCheck   = 275
	sqliteConstraintForeign = 787
	sqliteConstraintUnique  = 2067
)

// constraintCode returns the SQLite result code of err. It returns 0 if err
// did not come from the SQLite driver.
func constraintCode(err error) int {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return 0
	}
	return sqliteErr.Code()
}

// isUniqueViolation reports whether err is a violation of a unique
// constraint.
func isUniqueViolation(err error) bool {
	return constraintCode(err) == sqliteConstraintUnique
}

// isForeignKeyViolation reports whether err is a violation of a foreign key
// constraint.
func isForeignKeyViolation(err error) bool {
	return constraintCode(err) == sqliteConstraintForeign
}

// isCheckViolation reports whether err is a violation of a check constraint.
func isCheckViolation(err error) bool {
	return constraintCode(err) == sqliteConstraintCheck
}
