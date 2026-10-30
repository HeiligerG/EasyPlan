package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// The tests of this file check that several starts at the same time work on
// the same database file.
//
// The first connection of a brand new file switches the journal mode to WAL.
// That switch needs an exclusive lock on the database and cannot wait for the
// busy timeout, so two starts at the same time on a brand new file can fail
// sporadically with SQLITE_BUSY. The case is documented and accepted: the
// operation runs a single container and therefore starts once.
//
// Every test of this file opens and closes the file once before the parallel
// phase starts. The switch of the journal mode is done at that point, the
// parallel phase measures the behaviour of the migration itself.

// parallelStarts is the number of starts that run at the same time in the
// tests of this file.
const parallelStarts = 8

// TestOpen_parallel_opens_on_the_same_file_keep_the_default_accounts_single
// checks that several starts that open the same database file at the same
// time all succeed and that the default accounts exist exactly once.
//
// The default accounts are part of the first migration. A start that runs the
// migration a second time must not insert them again, otherwise the database
// would hold one set of default accounts per start.
func TestOpen_parallel_opens_on_the_same_file_keep_the_default_accounts_single(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	// Settle the journal mode of the new file and migrate it once.
	openFileAndClose(t, path)

	stores := make([]*Store, parallelStarts)
	errs := make([]error, parallelStarts)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, err := Open(path)
			if err != nil {
				errs[i] = err
				return
			}
			stores[i] = s
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("the parallel open %d failed: %v", i+1, err)
		}
	}
	for _, s := range stores {
		if s == nil {
			continue
		}
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
	if t.Failed() {
		return
	}

	checkDefaultAccounts(t, path)
}

// TestApplyMigrations_parallel_runs_create_the_schema_once checks that
// several starts that apply the migrations to the same new database file at
// the same time all succeed and that the schema and the default accounts
// exist exactly once.
//
// The migration takes the write lock of the database at the beginning of its
// transaction. Only one start can hold it, the others wait for the busy
// timeout and see the new schema version afterwards, so they do not apply the
// migration a second time.
func TestApplyMigrations_parallel_runs_create_the_schema_once(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	// Settle the journal mode of the new file without migrating it, so the
	// parallel phase has to create the schema itself.
	createFileWithoutMigration(t, path)

	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations() failed: %v", err)
	}

	errs := make([]error, parallelStarts)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = applyMigrationsToFile(path, migrations)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("the parallel migration %d failed: %v", i+1, err)
		}
	}
	if t.Failed() {
		return
	}

	checkDefaultAccounts(t, path)
}

// openFileAndClose opens the database in the file path and closes it again.
// The file exists afterwards, the journal mode is WAL and the schema is
// migrated.
func openFileAndClose(t *testing.T, path string) {
	t.Helper()
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", path, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
}

// createFileWithoutMigration creates the database file and closes it again
// without applying a migration. The file exists afterwards and the journal
// mode is WAL, the schema version is still 0.
func createFileWithoutMigration(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", fileDSN(path))
	if err != nil {
		t.Fatalf("cannot open the database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		t.Fatalf("cannot open the database file %q: %v", path, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
}

// applyMigrationsToFile opens the database in the file path with its own
// connection and applies the migrations to it.
func applyMigrationsToFile(path string, migrations []migration) error {
	db, err := sql.Open("sqlite", fileDSN(path))
	if err != nil {
		return fmt.Errorf("cannot open the database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := applyMigrations(context.Background(), db, migrations); err != nil {
		return err
	}
	return nil
}

// checkDefaultAccounts checks that the database in the file path holds the
// schema version 1 and the three default accounts of Konzept.md §3, each of
// them exactly once.
func checkDefaultAccounts(t *testing.T, path string) {
	t.Helper()
	s := newFileStore(t, path)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	if got := pragmaInt(t, s, "user_version"); got != 1 {
		t.Errorf("user_version = %d, want 1", got)
	}
	if got := countAccounts(t, s); got != 3 {
		t.Errorf("the database holds %d accounts, want the 3 default accounts", got)
	}
	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	want := []string{"Lifestyle", "Sparen", "Steuern"}
	if got := accountNames(accounts); !equalStrings(got, want) {
		t.Errorf("the accounts are %v, want %v", got, want)
	}
}
