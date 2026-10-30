package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// The tests of this file work on the migrations themselves, they do not use
// the embedded migrations. The database is opened with the same data source
// name as the store, so the pragmas and the lock behaviour are the same.

// openMigrationDB opens a fresh database in a file for the tests of the
// migrations. The database has the schema version 0, no migration was
// applied yet.
func openMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "migrations.db")
	db, err := sql.Open("sqlite", fileDSN(path))
	if err != nil {
		t.Fatalf("cannot open the database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	})
	return db
}

// schemaVersionOf reads the schema version of a database.
func schemaVersionOf(t *testing.T, db *sql.DB) int {
	t.Helper()
	var version int
	if err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("cannot read the schema version: %v", err)
	}
	return version
}

// tableExistsIn reports whether a table is part of the schema of a database.
func tableExistsIn(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	const query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	var count int
	if err := db.QueryRowContext(context.Background(), query, name).Scan(&count); err != nil {
		t.Fatalf("cannot read the schema: %v", err)
	}
	return count > 0
}

// TestApplyMigrations_rolls_back_a_broken_script checks that a migration
// with a broken script changes nothing: the schema version stays the same
// and no table of the broken migration is left behind.
func TestApplyMigrations_rolls_back_a_broken_script(t *testing.T) {
	db := openMigrationDB(t)
	ctx := context.Background()

	migrations := []migration{
		{version: 1, name: "0001_first.sql", script: "CREATE TABLE first (id INTEGER PRIMARY KEY);"},
		{
			version: 2,
			name:    "0002_broken.sql",
			script:  "CREATE TABLE second (id INTEGER PRIMARY KEY); INSERT INTO missing_table (id) VALUES (1);",
		},
	}

	err := applyMigrations(ctx, db, migrations)
	if err == nil {
		t.Fatal("applyMigrations() = nil, want an error for the broken script")
	}
	if !strings.Contains(err.Error(), "0002_broken.sql") {
		t.Errorf("the error %q does not name the broken migration", err.Error())
	}
	if got := schemaVersionOf(t, db); got != 0 {
		t.Errorf("the schema version is %d, want 0", got)
	}
	for _, name := range []string{"first", "second"} {
		if tableExistsIn(t, db, name) {
			t.Errorf("the table %q exists, the transaction was not rolled back", name)
		}
	}
}

// TestApplyMigrations_refuses_a_database_that_is_newer checks that a
// database with a newer schema is not touched. The binary does not know what
// the newer schema looks like.
func TestApplyMigrations_refuses_a_database_that_is_newer(t *testing.T) {
	db := openMigrationDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "PRAGMA user_version = 5"); err != nil {
		t.Fatalf("cannot set the schema version: %v", err)
	}

	err := applyMigrations(ctx, db, []migration{
		{version: 1, name: "0001_first.sql", script: "CREATE TABLE first (id INTEGER PRIMARY KEY);"},
	})
	if err == nil {
		t.Fatal("applyMigrations() = nil, want an error for the newer database")
	}
	if !strings.Contains(err.Error(), "newer than this binary supports") {
		t.Errorf("the error %q does not mention the newer schema version", err.Error())
	}
	if got := schemaVersionOf(t, db); got != 5 {
		t.Errorf("the schema version is %d, want 5", got)
	}
	if tableExistsIn(t, db, "first") {
		t.Error("the table first exists, the migrations were applied to a newer database")
	}
}

// TestApplyMigrations_applies_the_migrations_in_order checks that the
// migrations are applied in the order of their version, no matter in which
// order they are handed over.
func TestApplyMigrations_applies_the_migrations_in_order(t *testing.T) {
	db := openMigrationDB(t)
	ctx := context.Background()

	migrations := []migration{
		{version: 3, name: "0003_third.sql", script: "CREATE TABLE third (id INTEGER PRIMARY KEY);"},
		{version: 1, name: "0001_first.sql", script: "CREATE TABLE first (id INTEGER PRIMARY KEY);"},
		{
			version: 2,
			name:    "0002_second.sql",
			script:  "INSERT INTO first (id) VALUES (1); CREATE TABLE second (id INTEGER PRIMARY KEY);",
		},
	}

	if err := applyMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("applyMigrations() failed: %v", err)
	}
	if got := schemaVersionOf(t, db); got != 3 {
		t.Errorf("the schema version is %d, want 3", got)
	}
	for _, name := range []string{"first", "second", "third"} {
		if !tableExistsIn(t, db, name) {
			t.Errorf("the table %q is missing", name)
		}
	}

	// The second migration writes into the table of the first migration, so
	// the row only exists when the migrations ran in the right order.
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM first").Scan(&count); err != nil {
		t.Fatalf("cannot read the table first: %v", err)
	}
	if count != 1 {
		t.Errorf("the table first holds %d rows, want 1", count)
	}
}

// TestApplyMigrations_does_not_apply_a_migration_twice checks that a second
// run of the same migrations changes nothing.
func TestApplyMigrations_does_not_apply_a_migration_twice(t *testing.T) {
	db := openMigrationDB(t)
	ctx := context.Background()

	migrations := []migration{
		{version: 1, name: "0001_first.sql", script: "CREATE TABLE first (id INTEGER PRIMARY KEY);"},
		{version: 2, name: "0002_second.sql", script: "INSERT INTO first (id) VALUES (1);"},
	}

	if err := applyMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("the first applyMigrations() failed: %v", err)
	}
	if err := applyMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("the second applyMigrations() failed: %v", err)
	}
	if got := schemaVersionOf(t, db); got != 2 {
		t.Errorf("the schema version is %d, want 2", got)
	}

	// The second migration inserts a row. A second run would insert a second
	// row, a run of the first migration would fail because the table exists
	// already.
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM first").Scan(&count); err != nil {
		t.Fatalf("cannot read the table first: %v", err)
	}
	if count != 1 {
		t.Errorf("the table first holds %d rows, want 1", count)
	}
}

// TestReadMigrations_sorts_the_migrations_by_version checks that the
// migrations are read in the order of their version.
func TestReadMigrations_sorts_the_migrations_by_version(t *testing.T) {
	fsys := fstest.MapFS{
		"0002_second.sql": &fstest.MapFile{Data: []byte("CREATE TABLE second (id INTEGER PRIMARY KEY);")},
		"0001_first.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE first (id INTEGER PRIMARY KEY);")},
		"0010_tenth.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE tenth (id INTEGER PRIMARY KEY);")},
	}

	migrations, err := readMigrations(fsys)
	if err != nil {
		t.Fatalf("readMigrations() failed: %v", err)
	}
	want := []int{1, 2, 10}
	if len(migrations) != len(want) {
		t.Fatalf("readMigrations() returned %d migrations, want %d", len(migrations), len(want))
	}
	for i, m := range migrations {
		if m.version != want[i] {
			t.Errorf("the migration %d has the version %d, want %d", i, m.version, want[i])
		}
	}
	if !strings.Contains(migrations[0].script, "first") {
		t.Errorf("the script of the first migration is %q, want the script of 0001_first.sql", migrations[0].script)
	}
}

// TestReadMigrations_rejects_two_migrations_with_the_same_version checks that
// two migrations with the same version are refused. The order in which they
// would be applied is not defined.
func TestReadMigrations_rejects_two_migrations_with_the_same_version(t *testing.T) {
	fsys := fstest.MapFS{
		"0001_first.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE first (id INTEGER PRIMARY KEY);")},
		"0001_other.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE other (id INTEGER PRIMARY KEY);")},
		"0002_second.sql": &fstest.MapFile{Data: []byte("CREATE TABLE second (id INTEGER PRIMARY KEY);")},
	}

	migrations, err := readMigrations(fsys)
	if err == nil {
		t.Fatalf("readMigrations() = %v, want an error for the duplicate version", migrations)
	}
	if !strings.Contains(err.Error(), "duplicate migration version 1") {
		t.Errorf("the error %q does not mention the duplicate version", err.Error())
	}
}

// TestReadMigrations_rejects_a_file_without_a_version checks that a file
// whose name does not start with a version is refused.
func TestReadMigrations_rejects_a_file_without_a_version(t *testing.T) {
	fsys := fstest.MapFS{
		"init.sql": &fstest.MapFile{Data: []byte("CREATE TABLE first (id INTEGER PRIMARY KEY);")},
	}

	migrations, err := readMigrations(fsys)
	if err == nil {
		t.Fatalf("readMigrations() = %v, want an error for the missing version", migrations)
	}
	if !strings.Contains(err.Error(), "has no version") {
		t.Errorf("the error %q does not mention the missing version", err.Error())
	}
}
