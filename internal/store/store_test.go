package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// newMemoryStore opens a fresh database in memory for a test. The database is
// closed when the test ends.
func newMemoryStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	})
	return s
}

// newFileStore opens a database in the file path.
func newFileStore(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", path, err)
	}
	return s
}

// mustDate parses a calendar day of the tests and fails the test when the
// literal is wrong.
func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	day, err := plan.ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) failed: %v", value, err)
	}
	return day
}

// createAccount creates an account and fails the test when it does not work.
func createAccount(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	id, err := s.CreateAccount(context.Background(), name)
	if err != nil {
		t.Fatalf("CreateAccount(%q) failed: %v", name, err)
	}
	return id
}

// createEntry creates an entry and fails the test when it does not work.
func createEntry(t *testing.T, s *Store, entry plan.Entry) int64 {
	t.Helper()
	id, err := s.CreateEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("CreateEntry(%q) failed: %v", entry.Name, err)
	}
	return id
}

// planEntryForAccount returns a monthly entry of an account.
func planEntryForAccount(t *testing.T, accountID int64, name string, amount int64, startDate string) plan.Entry {
	t.Helper()
	return plan.Entry{
		AccountID:   accountID,
		Name:        name,
		AmountMinor: amount,
		StartDate:   mustDate(t, startDate),
		Unit:        plan.Month,
		Interval:    1,
	}
}

// accountByName returns the account with the given name.
func accountByName(t *testing.T, s *Store, name string) Account {
	t.Helper()
	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	for _, account := range accounts {
		if account.Name == name {
			return account
		}
	}
	t.Fatalf("the account %q does not exist", name)
	return Account{}
}

// accountNames returns the names of the accounts in the order of the list.
func accountNames(accounts []Account) []string {
	names := make([]string, 0, len(accounts))
	for _, account := range accounts {
		names = append(names, account.Name)
	}
	return names
}

// entryNames returns the names of the entries in the order of the list.
func entryNames(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	return names
}

// pragmaInt reads a pragma that returns a number.
func pragmaInt(t *testing.T, s *Store, name string) int {
	t.Helper()
	var value int
	if err := s.db.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&value); err != nil {
		t.Fatalf("cannot read PRAGMA %s: %v", name, err)
	}
	return value
}

// pragmaString reads a pragma that returns a text.
func pragmaString(t *testing.T, s *Store, name string) string {
	t.Helper()
	var value string
	if err := s.db.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&value); err != nil {
		t.Fatalf("cannot read PRAGMA %s: %v", name, err)
	}
	return value
}

// tableExists reports whether a table is part of the schema.
func tableExists(t *testing.T, s *Store, name string) bool {
	t.Helper()
	const query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	var count int
	if err := s.db.QueryRowContext(context.Background(), query, name).Scan(&count); err != nil {
		t.Fatalf("cannot read the schema: %v", err)
	}
	return count > 0
}

// countEntries returns the number of entries in the database.
func countEntries(t *testing.T, s *Store) int {
	t.Helper()
	var count int
	if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM entries").Scan(&count); err != nil {
		t.Fatalf("cannot count the entries: %v", err)
	}
	return count
}

// countAccounts returns the number of accounts in the database.
func countAccounts(t *testing.T, s *Store) int {
	t.Helper()
	var count int
	if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		t.Fatalf("cannot count the accounts: %v", err)
	}
	return count
}

// rawEndsOn reads the stored end date of an entry without the conversion of
// the store, so the test can see the NULL of a missing end date.
func rawEndsOn(t *testing.T, s *Store, id int64) sql.NullString {
	t.Helper()
	const query = "SELECT ends_on FROM entries WHERE id = ?"
	var endsOn sql.NullString
	if err := s.db.QueryRowContext(context.Background(), query, id).Scan(&endsOn); err != nil {
		t.Fatalf("cannot read ends_on of the entry %d: %v", id, err)
	}
	return endsOn
}

// rawInsertEntry writes an entry with plain SQL, bypassing the store, so the
// constraints of the database can be tested on their own.
func rawInsertEntry(t *testing.T, s *Store, accountID int64, name string, amount int64, startDate, unit string, interval int) error {
	t.Helper()
	const insert = `INSERT INTO entries (account_id, name, amount_minor, start_date, unit, interval, ends_on, notes)
VALUES (?, ?, ?, ?, ?, ?, NULL, '')`
	_, err := s.db.ExecContext(context.Background(), insert, accountID, name, amount, startDate, unit, interval)
	return err
}

func TestOpenMemory_migrates_the_schema_and_creates_the_default_accounts(t *testing.T) {
	s := newMemoryStore(t)

	if got := pragmaInt(t, s, "user_version"); got != 1 {
		t.Errorf("user_version = %d, want 1", got)
	}
	for _, table := range []string{"accounts", "entries"} {
		if !tableExists(t, s, table) {
			t.Errorf("the table %q is missing", table)
		}
	}
	if got := pragmaInt(t, s, "foreign_keys"); got != 1 {
		t.Errorf("foreign_keys = %d, want 1", got)
	}
	// An in-memory database cannot use the WAL mode, the file database can.
	if got := pragmaString(t, s, "journal_mode"); got != "memory" {
		t.Errorf("journal_mode = %q, want %q", got, "memory")
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	want := []struct {
		name  string
		order int
	}{
		{name: "Lifestyle", order: 1},
		{name: "Sparen", order: 2},
		{name: "Steuern", order: 3},
	}
	if len(accounts) != len(want) {
		t.Fatalf("the migration created %d accounts, want %d: %v", len(accounts), len(want), accountNames(accounts))
	}
	for i, expected := range want {
		if accounts[i].Name != expected.name {
			t.Errorf("account %d is %q, want %q", i, accounts[i].Name, expected.name)
		}
		if accounts[i].SortOrder != expected.order {
			t.Errorf("the account %q has the sort order %d, want %d", accounts[i].Name, accounts[i].SortOrder, expected.order)
		}
		if !accounts[i].IsActive {
			t.Errorf("the account %q is not active", accounts[i].Name)
		}
	}
}

func TestOpen_creates_the_database_file_and_migrates_the_schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	s := newFileStore(t, path)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the database file was not created: %v", err)
	}
	if got := pragmaInt(t, s, "user_version"); got != 1 {
		t.Errorf("user_version = %d, want 1", got)
	}
	for _, table := range []string{"accounts", "entries"} {
		if !tableExists(t, s, table) {
			t.Errorf("the table %q is missing", table)
		}
	}
	if got := pragmaString(t, s, "journal_mode"); got != "wal" {
		t.Errorf("journal_mode = %q, want %q", got, "wal")
	}
	if got := pragmaInt(t, s, "foreign_keys"); got != 1 {
		t.Errorf("foreign_keys = %d, want 1", got)
	}
	if got := countAccounts(t, s); got != 3 {
		t.Errorf("the migration created %d accounts, want 3", got)
	}
}

func TestOpen_second_open_keeps_the_data_and_does_not_migrate_again(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	first := newFileStore(t, path)
	lifestyle := accountByName(t, first, "Lifestyle")
	if err := first.RenameAccount(context.Background(), lifestyle.ID, "Wohnen"); err != nil {
		t.Fatalf("RenameAccount() failed: %v", err)
	}
	extra := createAccount(t, first, "Extra")
	entry := plan.Entry{
		AccountID:   extra,
		Name:        "Sparen",
		AmountMinor: 100000,
		StartDate:   mustDate(t, "2026-01-01"),
		Unit:        plan.Month,
		Interval:    1,
	}
	entryID := createEntry(t, first, entry)
	if err := first.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	second := newFileStore(t, path)
	defer func() {
		if err := second.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	if got := pragmaInt(t, second, "user_version"); got != 1 {
		t.Errorf("user_version = %d, want 1", got)
	}
	accounts, err := second.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	// The migration would create three more accounts, the count would be 7.
	if got := countAccounts(t, second); got != 4 {
		t.Errorf("the second open holds %d accounts, want 4: %v", got, accountNames(accounts))
	}
	if got := accountNames(accounts); !equalStrings(got, []string{"Wohnen", "Sparen", "Steuern", "Extra"}) {
		t.Errorf("the accounts are %v, want [Wohnen Sparen Steuern Extra]", got)
	}
	entries, err := second.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("the second open holds %d entries, want 1", len(entries))
	}
	if entries[0].ID != entryID || entries[0].AccountID != extra || entries[0].AmountMinor != 100000 {
		t.Errorf("the entry did not survive the restart: %+v", entries[0])
	}
}

func TestOpen_reports_a_missing_directory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := Open(filepath.Join(missing, "easyplan.db"))
	if err == nil {
		t.Fatal("Open() succeeded, want an error for the missing directory")
	}
	message := err.Error()
	if !strings.Contains(message, "store:") {
		t.Errorf("the error %q does not name the store", message)
	}
	if !strings.Contains(message, missing) {
		t.Errorf("the error %q does not mention the directory %q", message, missing)
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Errorf("Open() created the directory %q", missing)
	}
}

func TestOpen_reports_a_path_whose_parent_is_a_file(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("cannot write the file: %v", err)
	}

	_, err := Open(filepath.Join(file, "easyplan.db"))
	if err == nil {
		t.Fatal("Open() succeeded, want an error for the file as directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("the error %q does not explain the problem", err)
	}
}

func TestOpen_rejects_an_empty_path(t *testing.T) {
	_, err := Open("   ")
	if err == nil {
		t.Fatal("Open() succeeded, want an error for the empty path")
	}
	if !strings.Contains(err.Error(), "empty database path") {
		t.Errorf("the error %q does not explain the problem", err)
	}
}

func TestOpen_fails_when_the_path_is_a_directory(t *testing.T) {
	dir := t.TempDir()

	if _, err := Open(dir); err == nil {
		t.Fatal("Open() succeeded, want an error for the directory as database file")
	}
}

// TestPing_succeeds_with_an_open_database checks that the check of the
// connection to the database succeeds while the database is open.
func TestPing_succeeds_with_an_open_database(t *testing.T) {
	s := newMemoryStore(t)

	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}
}

// TestPing_fails_with_a_closed_database checks that the check of the
// connection to the database fails after the database was closed.
func TestPing_fails_with_a_closed_database(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() failed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	err = s.Ping(context.Background())
	if err == nil {
		t.Fatal("Ping() succeeded with a closed database, want an error")
	}
	if !strings.Contains(err.Error(), "store:") {
		t.Errorf("the error %q does not name the store", err)
	}
}

// equalStrings reports whether two string slices are equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
