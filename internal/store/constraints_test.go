package store

import (
	"context"
	"testing"
)

// The tests of this file write to the database with plain SQL, so the
// constraints of the schema are checked without the store in between.

func TestDatabase_accepts_a_valid_entry(t *testing.T) {
	// The control of the following tests: the plain SQL insert of a valid
	// entry works, so the rejected ones fail because of their constraint and
	// not because of the statement.
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")

	err := rawInsertEntry(t, s, lifestyle.ID, "Miete", 200000, "2026-01-01", "month", 1)
	if err != nil {
		t.Fatalf("the insert of a valid entry failed: %v", err)
	}
	if got := countEntries(t, s); got != 1 {
		t.Errorf("the database holds %d entries, want 1", got)
	}
}

func TestDatabase_rejects_a_non_positive_amount(t *testing.T) {
	for _, amount := range []int64{0, -1, -200000} {
		s := newMemoryStore(t)
		lifestyle := accountByName(t, s, "Lifestyle")

		err := rawInsertEntry(t, s, lifestyle.ID, "Miete", amount, "2026-01-01", "month", 1)
		if err == nil {
			t.Fatalf("the insert of the amount %d succeeded, want an error", amount)
		}
		if !isCheckViolation(err) {
			t.Errorf("the error %v is not a check violation", err)
		}
		if got := countEntries(t, s); got != 0 {
			t.Errorf("the database holds %d entries, want none", got)
		}
	}
}

func TestDatabase_rejects_an_unknown_unit(t *testing.T) {
	for _, unit := range []string{"day", "MONTH", "months", ""} {
		s := newMemoryStore(t)
		lifestyle := accountByName(t, s, "Lifestyle")

		err := rawInsertEntry(t, s, lifestyle.ID, "Miete", 200000, "2026-01-01", unit, 1)
		if err == nil {
			t.Fatalf("the insert of the unit %q succeeded, want an error", unit)
		}
		if !isCheckViolation(err) {
			t.Errorf("the error %v is not a check violation", err)
		}
		if got := countEntries(t, s); got != 0 {
			t.Errorf("the database holds %d entries, want none", got)
		}
	}
}

func TestDatabase_rejects_a_non_positive_interval(t *testing.T) {
	for _, interval := range []int{0, -1} {
		s := newMemoryStore(t)
		lifestyle := accountByName(t, s, "Lifestyle")

		err := rawInsertEntry(t, s, lifestyle.ID, "Miete", 200000, "2026-01-01", "month", interval)
		if err == nil {
			t.Fatalf("the insert of the interval %d succeeded, want an error", interval)
		}
		if !isCheckViolation(err) {
			t.Errorf("the error %v is not a check violation", err)
		}
		if got := countEntries(t, s); got != 0 {
			t.Errorf("the database holds %d entries, want none", got)
		}
	}
}

func TestDatabase_rejects_a_foreign_key_violation(t *testing.T) {
	s := newMemoryStore(t)

	err := rawInsertEntry(t, s, 4242, "Miete", 200000, "2026-01-01", "month", 1)
	if err == nil {
		t.Fatal("the insert with an unknown account succeeded, want an error")
	}
	if !isForeignKeyViolation(err) {
		t.Errorf("the error %v is not a foreign key violation", err)
	}
	if got := countEntries(t, s); got != 0 {
		t.Errorf("the database holds %d entries, want none", got)
	}
}

func TestSQLInjection_keeps_the_name_of_an_account_as_plain_text(t *testing.T) {
	const name = "'); DROP TABLE accounts;--"
	s := newMemoryStore(t)

	id := createAccount(t, s, name)

	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != name {
		t.Errorf("the name is %q, want %q", account.Name, name)
	}
	if !tableExists(t, s, "accounts") {
		t.Error("the table accounts is missing, the name was not stored as plain text")
	}
	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if len(accounts) != 4 {
		t.Errorf("the database holds %d accounts, want 4", len(accounts))
	}
}

func TestSQLInjection_keeps_the_name_of_an_entry_as_plain_text(t *testing.T) {
	const name = "'); DROP TABLE entries;--"
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")
	entry := planEntryForAccount(t, lifestyle.ID, name, 200000, "2026-01-01")

	id := createEntry(t, s, entry)

	stored, err := s.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEntry() failed: %v", err)
	}
	if stored.Name != name {
		t.Errorf("the name is %q, want %q", stored.Name, name)
	}
	if !tableExists(t, s, "entries") {
		t.Error("the table entries is missing, the name was not stored as plain text")
	}
	entries, err := s.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("the database holds %d entries, want 1", len(entries))
	}
}
