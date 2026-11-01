package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// VacuumInto writes a copy of the database into the file path.
//
// The copy is made with the statement VACUUM INTO. It holds the data of the
// moment of the call, also when the database is written while the copy is
// made, and the copy does not need the write lock of the database, so a
// backup does not disturb the visitors of the application.
//
// The file must not exist yet. SQLite refuses to overwrite a file, so a
// backup never destroys the data of a file that is already there. The caller
// removes the copy when it is not needed anymore.
//
// The path must not come from the input of a visitor: the copy holds the
// whole database.
func (s *Store) VacuumInto(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("store: empty path for the backup")
	}
	const statement = "VACUUM INTO ?"
	if _, err := s.db.ExecContext(ctx, statement, path); err != nil {
		return fmt.Errorf("store: cannot write the backup to %q: %w", path, err)
	}
	return nil
}
