package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The magic number at the beginning of every SQLite file.
const sqliteHeader = "SQLite format 3\x00"

// TestVacuumInto_writes_a_copy_of_the_database checks that the copy is a
// valid SQLite file with the same accounts and entries as the source.
func TestVacuumInto_writes_a_copy_of_the_database(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "easyplan.db")
	s := newFileStore(t, path)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	lifestyle := accountByName(t, s, "Lifestyle")
	entry := planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01")
	entry.Notes = "Wohnung"
	createEntry(t, s, entry)

	copyPath := filepath.Join(dir, "copy.db")
	if err := s.VacuumInto(context.Background(), copyPath); err != nil {
		t.Fatalf("VacuumInto() failed: %v", err)
	}

	content, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("cannot read the copy: %v", err)
	}
	if !strings.HasPrefix(string(content), sqliteHeader) {
		t.Errorf("the copy does not begin with the magic number of SQLite: %q", content[:min(len(content), 16)])
	}

	copyDB := openCopy(t, copyPath)
	if got := pragmaOfCopy(t, copyDB, "integrity_check"); got != "ok" {
		t.Errorf("the integrity check of the copy is %q, want %q", got, "ok")
	}

	accounts := "SELECT id, name, sort_order, is_active FROM accounts ORDER BY id"
	if got, want := rowsOfCopy(t, copyDB, accounts), rowsOfStore(t, s, accounts); !equalStrings(got, want) {
		t.Errorf("the accounts of the copy are %v, want %v", got, want)
	}

	entries := "SELECT id, account_id, name, amount_minor, start_date, unit, interval, ends_on, notes FROM entries ORDER BY id"
	if got, want := rowsOfCopy(t, copyDB, entries), rowsOfStore(t, s, entries); !equalStrings(got, want) {
		t.Errorf("the entries of the copy are %v, want %v", got, want)
	}
}

// TestVacuumInto_fails_when_the_file_exists_already checks that a backup
// never overwrites a file that is already there.
func TestVacuumInto_fails_when_the_file_exists_already(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "copy.db")
	const content = "not a database"
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the file: %v", err)
	}

	s := newMemoryStore(t)

	err := s.VacuumInto(context.Background(), target)
	if err == nil {
		t.Fatal("VacuumInto() succeeded, want an error for the file that exists already")
	}
	if !strings.Contains(err.Error(), "store:") {
		t.Errorf("the error %q does not name the store", err)
	}

	// The file has to stay as it was, the backup may not destroy it.
	read, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("cannot read the file: %v", err)
	}
	if string(read) != content {
		t.Errorf("the file holds %q, want %q", read, content)
	}
}

// TestVacuumInto_fails_when_the_directory_cannot_be_written checks that a
// backup reports a directory it cannot write into.
func TestVacuumInto_fails_when_the_directory_cannot_be_written(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the test runs as root, the rights of a directory do not stop the write")
	}

	dir := t.TempDir()
	readOnly := filepath.Join(dir, "read-only")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("cannot create the directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(readOnly, 0o700); err != nil {
			t.Errorf("cannot change the rights of the directory: %v", err)
		}
	})

	s := newMemoryStore(t)

	err := s.VacuumInto(context.Background(), filepath.Join(readOnly, "copy.db"))
	if err == nil {
		t.Fatal("VacuumInto() succeeded, want an error for the directory that cannot be written")
	}
	if !strings.Contains(err.Error(), "store:") {
		t.Errorf("the error %q does not name the store", err)
	}
}

// TestVacuumInto_rejects_an_empty_path checks that the store refuses a path
// without any character.
func TestVacuumInto_rejects_an_empty_path(t *testing.T) {
	s := newMemoryStore(t)

	err := s.VacuumInto(context.Background(), "   ")
	if err == nil {
		t.Fatal("VacuumInto() succeeded, want an error for the empty path")
	}
	if !strings.Contains(err.Error(), "empty path") {
		t.Errorf("the error %q does not explain the problem", err)
	}
}

// TestVacuumInto_copies_the_data_of_a_database_in_wal_mode checks that the
// copy holds the entries of a database that keeps its write ahead log.
func TestVacuumInto_copies_the_data_of_a_database_in_wal_mode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")
	s := newFileStore(t, path)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	lifestyle := accountByName(t, s, "Lifestyle")
	createEntry(t, s, planEntryForAccount(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	copyPath := filepath.Join(t.TempDir(), "copy.db")
	if err := s.VacuumInto(context.Background(), copyPath); err != nil {
		t.Fatalf("VacuumInto() failed: %v", err)
	}

	copyDB := openCopy(t, copyPath)
	var count int
	const query = "SELECT COUNT(*) FROM entries"
	if err := copyDB.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("cannot count the entries of the copy: %v", err)
	}
	if count != 1 {
		t.Errorf("the copy holds %d entries, want 1", count)
	}
}

// TestVacuumInto_works_with_a_quote_in_the_path checks that the statement of
// the copy uses a parameter.
//
// The path of the copy is built by the application and not by a visitor, but
// a quote in the name of the directory must not break the statement.
func TestVacuumInto_works_with_a_quote_in_the_path(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.db")
	s := newFileStore(t, source)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	copyDir := filepath.Join(dir, "backup's directory")
	if err := os.Mkdir(copyDir, 0o700); err != nil {
		t.Fatalf("cannot create the directory: %v", err)
	}
	copyPath := filepath.Join(copyDir, "copy's.db")

	if err := s.VacuumInto(context.Background(), copyPath); err != nil {
		t.Fatalf("VacuumInto() failed: %v", err)
	}

	content, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("cannot read the copy: %v", err)
	}
	if !strings.HasPrefix(string(content), sqliteHeader) {
		t.Errorf("the copy does not begin with the magic number of SQLite: %q", content[:min(len(content), 16)])
	}
}

// openCopy opens the copy of the database and closes it when the test ends.
func openCopy(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("cannot open the copy %q: %v", path, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("cannot close the copy: %v", err)
		}
	})
	return db
}

// pragmaOfCopy reads a pragma that returns a text from the copy.
func pragmaOfCopy(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	var value string
	if err := db.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&value); err != nil {
		t.Fatalf("cannot read PRAGMA %s: %v", name, err)
	}
	return value
}

// rowsOfStore reads the rows of a query of the database as texts.
func rowsOfStore(t *testing.T, s *Store, query string) []string {
	t.Helper()
	return rowsOfQuery(t, s.db, query)
}

// rowsOfCopy reads the rows of a query of the copy as texts.
func rowsOfCopy(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	return rowsOfQuery(t, db, query)
}

// rowsOfQuery reads the rows of a query as texts. Every row becomes one text
// with the values of its columns, separated by a bar.
func rowsOfQuery(t *testing.T, q queryer, query string) []string {
	t.Helper()
	rows, err := q.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatalf("cannot run %q: %v", query, err)
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		columns, err := rows.Columns()
		if err != nil {
			t.Fatalf("cannot read the columns of %q: %v", query, err)
		}
		cells := make([]any, len(columns))
		for i := range cells {
			cells[i] = new(sql.NullString)
		}
		if err := rows.Scan(cells...); err != nil {
			t.Fatalf("cannot read a row of %q: %v", query, err)
		}
		parts := make([]string, 0, len(cells))
		for _, cell := range cells {
			parts = append(parts, cell.(*sql.NullString).String)
		}
		values = append(values, strings.Join(parts, "|"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot read the rows of %q: %v", query, err)
	}
	return values
}
