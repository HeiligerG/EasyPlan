package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestPragmas_are_set_on_every_connection_of_the_pool checks that the
// pragmas of the data source name are applied to every connection of the pool
// and not only to the first one.
//
// The foreign keys are only enforced on a connection that has the pragma
// foreign_keys set. A pool that hands out a connection without the pragma
// would therefore accept a row with an unknown account.
func TestPragmas_are_set_on_every_connection_of_the_pool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	s := newFileStore(t, path)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	ctx := context.Background()
	// The pool of a database in a file holds four connections. Three of them
	// are taken and kept open, so the pool has to open three different
	// connections.
	const connections = 3
	conns := make([]*sql.Conn, 0, connections)
	for i := 0; i < connections; i++ {
		conn, err := s.db.Conn(ctx)
		if err != nil {
			t.Fatalf("cannot take the connection %d: %v", i+1, err)
		}
		defer conn.Close()
		conns = append(conns, conn)
	}

	// The connections have to be different ones, otherwise the test would
	// check the same connection three times. A temporary table belongs to the
	// connection that created it, so it does not exist on the other ones.
	// Every connection uses its own table name, so the table of one
	// connection cannot be mistaken for the table of another one.
	for i := range conns {
		table := fmt.Sprintf("probe_%d", i+1)
		const create = "CREATE TEMP TABLE %s (id INTEGER)"
		if _, err := conns[i].ExecContext(ctx, fmt.Sprintf(create, table)); err != nil {
			t.Fatalf("cannot create the temporary table %q on the connection %d: %v", table, i+1, err)
		}
		for j := range conns {
			if i == j {
				continue
			}
			const count = "SELECT COUNT(*) FROM temp.%s"
			var rows int
			err := conns[j].QueryRowContext(ctx, fmt.Sprintf(count, table)).Scan(&rows)
			if err == nil {
				t.Errorf("the connection %d sees the temporary table %q of the connection %d, the pool handed out the same connection twice", j+1, table, i+1)
				continue
			}
			if !strings.Contains(err.Error(), "no such table") {
				t.Errorf("the connection %d returned the error %v, want a missing temporary table", j+1, err)
			}
		}
	}

	for i, conn := range conns {
		var foreignKeys int
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatalf("cannot read PRAGMA foreign_keys on the connection %d: %v", i+1, err)
		}
		if foreignKeys != 1 {
			t.Errorf("foreign_keys = %d on the connection %d, want 1", foreignKeys, i+1)
		}

		var journalMode string
		if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatalf("cannot read PRAGMA journal_mode on the connection %d: %v", i+1, err)
		}
		if journalMode != "wal" {
			t.Errorf("journal_mode = %q on the connection %d, want %q", journalMode, i+1, "wal")
		}

		// The foreign keys have to be enforced on every connection, a row
		// with an unknown account is rejected.
		const insert = `INSERT INTO entries (account_id, name, amount_minor, start_date, unit, interval, ends_on, notes)
VALUES (?, 'Miete', 200000, '2026-01-01', 'month', 1, NULL, '')`
		_, err := conn.ExecContext(ctx, insert, 4242)
		if err == nil {
			t.Errorf("the insert with an unknown account succeeded on the connection %d, want an error", i+1)
			continue
		}
		if !isForeignKeyViolation(err) {
			t.Errorf("the error %v on the connection %d is not a foreign key violation", err, i+1)
		}
	}

	if got := countEntries(t, s); got != 0 {
		t.Errorf("the database holds %d entries, want none", got)
	}
}
