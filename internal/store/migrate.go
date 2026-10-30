package store

import (
	"cmp"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migration is one embedded schema migration.
type migration struct {
	// version is the schema version the migration leads to.
	version int
	// name is the file name of the migration, it is part of the error
	// messages.
	name string
	// script is the SQL of the migration.
	script string
}

// loadMigrations reads the embedded migrations. The result is cached, the
// embedded file system never changes while the program runs.
var loadMigrations = sync.OnceValues(func() ([]migration, error) {
	dir, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("cannot open embedded migrations: %w", err)
	}
	return readMigrations(dir)
})

// readMigrations reads the migration scripts of the file system and returns
// them sorted by version.
//
// Two migrations with the same version are rejected: the order in which they
// would be applied is not defined.
func readMigrations(fsys fs.FS) ([]migration, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("cannot list migrations: %w", err)
	}
	migrations := make([]migration, 0, len(names))
	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return nil, err
		}
		script, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("cannot read migration %s: %w", name, err)
		}
		migrations = append(migrations, migration{version: version, name: name, script: string(script)})
	}
	slices.SortStableFunc(migrations, func(a, b migration) int {
		return cmp.Compare(a.version, b.version)
	})
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].version == migrations[i].version {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", migrations[i].version, migrations[i-1].name, migrations[i].name)
		}
	}
	return migrations, nil
}

// migrationVersion reads the version from the file name of a migration. The
// name of a migration is its version, a separator and a description, for
// example 0001_init.sql.
func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(path.Base(name), "_")
	if !ok {
		return 0, fmt.Errorf("migration %s has no version", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %s has no valid version: %w", name, err)
	}
	return version, nil
}

// migrate applies the migrations that are newer than the schema of the
// database.
func (s *Store) migrate(ctx context.Context) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	return applyMigrations(ctx, s.db, migrations)
}

// applyMigrations applies the migrations that are newer than the schema of
// the database.
//
// The version of the schema is kept in the header of the database file
// (PRAGMA user_version), so no extra table is needed. It is read inside the
// transaction that applies the migrations: the transaction takes the write
// lock of the database right away, so a second process that starts at the
// same time waits for the first one and then sees the new version. The
// scripts and the new version either both happen or neither does.
//
// A database whose schema is newer than the highest known migration is
// refused, because the binary does not know what the newer schema looks like.
func applyMigrations(ctx context.Context, db *sql.DB, migrations []migration) error {
	ordered := slices.Clone(migrations)
	slices.SortStableFunc(ordered, func(a, b migration) int {
		return cmp.Compare(a.version, b.version)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := schemaVersion(ctx, tx)
	if err != nil {
		return err
	}
	latest := 0
	if len(ordered) > 0 {
		latest = ordered[len(ordered)-1].version
	}
	if current > latest {
		return fmt.Errorf("database schema version %d is newer than this binary supports (%d)", current, latest)
	}
	for _, m := range ordered {
		if m.version <= current {
			continue
		}
		if err := applyMigration(ctx, tx, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit migrations: %w", err)
	}
	return nil
}

// schemaVersion returns the version of the schema of the database.
func schemaVersion(ctx context.Context, q queryer) (int, error) {
	var version int
	if err := q.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("cannot read schema version: %w", err)
	}
	return version, nil
}

// applyMigration runs the script of a migration and sets the new schema
// version. The caller owns the transaction.
func applyMigration(ctx context.Context, tx *sql.Tx, m migration) error {
	if _, err := tx.ExecContext(ctx, m.script); err != nil {
		return fmt.Errorf("cannot apply script: %w", err)
	}
	// A pragma does not accept parameters, the version is a number of our
	// own file names.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
		return fmt.Errorf("cannot set schema version: %w", err)
	}
	return nil
}
