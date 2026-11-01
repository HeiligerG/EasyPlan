package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// busyTimeout is the number of milliseconds SQLite waits for a lock that is
// held by another connection before it reports the database as locked.
const busyTimeout = 5000

// filePoolSize is the number of connections of the pool of a database in a
// file. WAL mode lets the readers work in parallel, the writers are
// serialised by SQLite itself.
const filePoolSize = 4

// Store is the SQLite persistence of EasyPlan.
//
// A Store is safe for concurrent use, database/sql takes care of the
// connections.
type Store struct {
	db *sql.DB
}

// Open opens the SQLite database in the file path.
//
// The file is created if it does not exist yet, the directory in which it
// lives has to exist already: Open never creates a directory, so a wrong
// EASYPLAN_DATA fails with a clear message instead of scattering database
// files over the file system.
//
// The schema is migrated to the current version. The default accounts are
// part of the first migration, so every new database holds them and a
// deleted default account does not come back.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("store: empty database path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("store: cannot resolve database path %q: %w", path, err)
	}
	dir := filepath.Dir(abs)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("store: directory %q of the database is not accessible: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("store: directory %q of the database is not a directory", dir)
	}
	return open(fileDSN(abs), filePoolSize)
}

// OpenMemory opens a database in memory.
//
// The database lives as long as the Store and is gone when it is closed. It
// exists for the smoke test with SQLite in memory that Konzept.md §7 asks
// for.
func OpenMemory() (*Store, error) {
	return open(memoryDSN(), 1)
}

// open opens the database and migrates the schema.
//
// maxOpenConns limits the size of the connection pool. The file database
// uses a small pool: in WAL mode the readers work in parallel, the writers
// are serialised by SQLite itself and wait for the busy timeout if another
// connection holds the write lock.
//
// The in-memory database uses a single connection. Every connection of a
// "file::memory:" database gets an own empty database, so the data would be
// gone as soon as the pool opens a second connection.
func open(dsn string, maxOpenConns int) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: cannot open database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)

	s := &Store{db: db}
	if err := s.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// init prepares a freshly opened database.
func (s *Store) init(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("store: cannot open database: %w", err)
	}
	if err := s.migrate(ctx); err != nil {
		return fmt.Errorf("store: cannot migrate database: %w", err)
	}
	return nil
}

// Ping checks that the database can be reached.
//
// The check takes a connection from the pool and lets the driver test it. The
// driver answers with a select of a constant (select 1), it reads no data of
// the database. The check does not look at the file of the database and it
// does not try to write, so a check of the health of the application does not
// change the state of the application.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("store: cannot reach the database: %w", err)
	}
	return nil
}

// Close closes the database.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store: cannot close database: %w", err)
	}
	return nil
}

// fileDSN builds the data source name of a database in a file.
//
// The pragmas are part of the data source name because the driver applies
// them to every connection it opens. foreign_keys has to be set on every
// connection, otherwise the foreign keys of the schema are not enforced.
//
// The journal mode WAL lets the readers work while a writer is active. It
// needs a file system with shared memory, which the data directory of the
// container provides.
func fileDSN(path string) string {
	u := url.URL{Scheme: "file", Path: path}
	return dsn(u.String(),
		"foreign_keys(1)",
		"journal_mode(WAL)",
		fmt.Sprintf("busy_timeout(%d)", busyTimeout),
	)
}

// memoryDSN builds the data source name of a database in memory.
//
// An in-memory database cannot use the journal mode WAL, so it is not
// requested.
func memoryDSN() string {
	return dsn("file::memory:",
		"foreign_keys(1)",
		fmt.Sprintf("busy_timeout(%d)", busyTimeout),
	)
}

// dsn builds the data source name of the SQLite driver.
//
// _txlock=immediate lets every transaction take the write lock right away. A
// transaction that starts as a reader and wants to write later cannot be
// retried by the busy timeout, so the writers are serialised at the
// beginning of the transaction instead.
func dsn(name string, pragmas ...string) string {
	q := url.Values{}
	for _, pragma := range pragmas {
		q.Add("_pragma", pragma)
	}
	q.Set("_txlock", "immediate")
	return name + "?" + q.Encode()
}

// queryer is the part of *sql.DB and *sql.Tx that the store needs to read
// rows.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
