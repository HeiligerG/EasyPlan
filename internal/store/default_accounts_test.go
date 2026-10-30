package store

import (
	"context"
	"path/filepath"
	"testing"
)

// TestOpen_does_not_bring_back_a_deleted_default_account checks that the
// default accounts are part of the first migration. A default account that
// was deleted stays deleted, even after the database was closed and opened
// again.
func TestOpen_does_not_bring_back_a_deleted_default_account(t *testing.T) {
	path := filepath.Join(t.TempDir(), "easyplan.db")

	first := newFileStore(t, path)
	sparen := accountByName(t, first, "Sparen")
	if err := first.DeleteAccount(context.Background(), sparen.ID); err != nil {
		t.Fatalf("DeleteAccount() failed: %v", err)
	}
	if got := countAccounts(t, first); got != 2 {
		t.Fatalf("the database holds %d accounts, want 2", got)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	second := newFileStore(t, path)
	defer func() {
		if err := second.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	accounts, err := second.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if got := countAccounts(t, second); got != 2 {
		t.Errorf("the second open holds %d accounts, want 2: %v", got, accountNames(accounts))
	}
	for _, account := range accounts {
		if account.Name == "Sparen" {
			t.Error("the account Sparen is back, the default accounts must only be created with the first migration")
		}
	}
	// The other default accounts are still there.
	for _, name := range []string{"Lifestyle", "Steuern"} {
		found := false
		for _, account := range accounts {
			if account.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the account %q is missing", name)
		}
	}
}
