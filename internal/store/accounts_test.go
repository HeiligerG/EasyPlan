package store

import (
	"context"
	"errors"
	"testing"
)

func TestCreateAccount_trims_the_name(t *testing.T) {
	s := newMemoryStore(t)

	id := createAccount(t, s, "  Wohnen  ")

	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != "Wohnen" {
		t.Errorf("the name is %q, want %q", account.Name, "Wohnen")
	}
	if !account.IsActive {
		t.Error("a new account is not active")
	}
}

func TestCreateAccount_rejects_an_empty_name(t *testing.T) {
	for _, name := range []string{"", "   ", "\t\n"} {
		t.Run(name, func(t *testing.T) {
			s := newMemoryStore(t)

			_, err := s.CreateAccount(context.Background(), name)
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("CreateAccount(%q) returned %v, want ErrInvalidName", name, err)
			}
			if got := countAccounts(t, s); got != 3 {
				t.Errorf("the database holds %d accounts, want the 3 default accounts", got)
			}
		})
	}
}

func TestCreateAccount_rejects_a_duplicate_name(t *testing.T) {
	s := newMemoryStore(t)

	createAccount(t, s, "Miete")

	_, err := s.CreateAccount(context.Background(), "Miete")
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("CreateAccount() returned %v, want ErrDuplicateName", err)
	}
	if got := countAccounts(t, s); got != 4 {
		t.Errorf("the database holds %d accounts, want 4", got)
	}
}

func TestCreateAccount_allows_a_name_with_another_spelling(t *testing.T) {
	// The schema declares the name as "text not null unique" without a
	// collation, so SQLite compares the names case-sensitive: "lifestyle" is
	// a different name than "Lifestyle".
	s := newMemoryStore(t)

	id := createAccount(t, s, "lifestyle")

	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != "lifestyle" {
		t.Errorf("the name is %q, want %q", account.Name, "lifestyle")
	}
	if account.ID == accountByName(t, s, "Lifestyle").ID {
		t.Error("the account with another spelling has the identifier of the default account")
	}
}

func TestRenameAccount_changes_the_name(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")

	if err := s.RenameAccount(context.Background(), id, "  Miete  "); err != nil {
		t.Fatalf("RenameAccount() failed: %v", err)
	}

	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != "Miete" {
		t.Errorf("the name is %q, want %q", account.Name, "Miete")
	}
}

func TestRenameAccount_rejects_a_duplicate_name(t *testing.T) {
	s := newMemoryStore(t)
	first := createAccount(t, s, "A")
	second := createAccount(t, s, "B")

	err := s.RenameAccount(context.Background(), second, "A")
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("RenameAccount() returned %v, want ErrDuplicateName", err)
	}
	account, err := s.GetAccount(context.Background(), second)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != "B" {
		t.Errorf("the name is %q, want %q", account.Name, "B")
	}
	if _, err := s.GetAccount(context.Background(), first); err != nil {
		t.Errorf("GetAccount() of the first account failed: %v", err)
	}
}

func TestRenameAccount_rejects_an_empty_name(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")

	err := s.RenameAccount(context.Background(), id, "   ")
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("RenameAccount() returned %v, want ErrInvalidName", err)
	}
	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.Name != "Wohnen" {
		t.Errorf("the name is %q, want %q", account.Name, "Wohnen")
	}
}

func TestRenameAccount_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	err := s.RenameAccount(context.Background(), 4242, "Miete")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("RenameAccount() returned %v, want ErrNotFound", err)
	}
}

func TestSetAccountActive_deactivates_and_activates_the_account(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")

	if err := s.SetAccountActive(context.Background(), id, false); err != nil {
		t.Fatalf("SetAccountActive(false) failed: %v", err)
	}
	account, err := s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if account.IsActive {
		t.Error("the account is still active")
	}

	if err := s.SetAccountActive(context.Background(), id, true); err != nil {
		t.Fatalf("SetAccountActive(true) failed: %v", err)
	}
	account, err = s.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAccount() failed: %v", err)
	}
	if !account.IsActive {
		t.Error("the account is not active any more")
	}
}

func TestSetAccountActive_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	err := s.SetAccountActive(context.Background(), 4242, false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetAccountActive() returned %v, want ErrNotFound", err)
	}
}

func TestGetAccount_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	_, err := s.GetAccount(context.Background(), 4242)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAccount() returned %v, want ErrNotFound", err)
	}
}

func TestSetAccountOrder_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	err := s.SetAccountOrder(context.Background(), 4242, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetAccountOrder() returned %v, want ErrNotFound", err)
	}
}

func TestListAccounts_leaves_out_the_inactive_accounts(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")
	if err := s.SetAccountActive(context.Background(), id, false); err != nil {
		t.Fatalf("SetAccountActive(false) failed: %v", err)
	}

	active, err := s.ListAccounts(context.Background(), false)
	if err != nil {
		t.Fatalf("ListAccounts(false) failed: %v", err)
	}
	if got := accountNames(active); !equalStrings(got, []string{"Lifestyle", "Sparen", "Steuern"}) {
		t.Errorf("the active accounts are %v, want [Lifestyle Sparen Steuern]", got)
	}

	all, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts(true) failed: %v", err)
	}
	if got := accountNames(all); !equalStrings(got, []string{"Lifestyle", "Sparen", "Steuern", "Wohnen"}) {
		t.Errorf("all accounts are %v, want [Lifestyle Sparen Steuern Wohnen]", got)
	}
}

func TestListAccounts_sorts_by_sort_order_and_identifier(t *testing.T) {
	s := newMemoryStore(t)
	first := createAccount(t, s, "A")
	second := createAccount(t, s, "B")
	third := createAccount(t, s, "C")

	// Two accounts with the same order are sorted by their identifier.
	if err := s.SetAccountOrder(context.Background(), first, 1); err != nil {
		t.Fatalf("SetAccountOrder() failed: %v", err)
	}
	if err := s.SetAccountOrder(context.Background(), second, 1); err != nil {
		t.Fatalf("SetAccountOrder() failed: %v", err)
	}
	if err := s.SetAccountOrder(context.Background(), third, 0); err != nil {
		t.Fatalf("SetAccountOrder() failed: %v", err)
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	// The accounts with the order 1 are sorted by their identifier: the
	// default account has the identifier 1, the new ones 4 and 5.
	if got := accountNames(accounts); !equalStrings(got, []string{"C", "Lifestyle", "A", "B", "Sparen", "Steuern"}) {
		t.Errorf("the accounts are %v, want [C Lifestyle A B Sparen Steuern]", got)
	}
}

func TestMoveAccountUp_moves_the_account_one_position_towards_the_front(t *testing.T) {
	s := newMemoryStore(t)
	extra := createAccount(t, s, "Extra")

	if err := s.MoveAccountUp(context.Background(), extra); err != nil {
		t.Fatalf("MoveAccountUp() failed: %v", err)
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if got := accountNames(accounts); !equalStrings(got, []string{"Lifestyle", "Sparen", "Extra", "Steuern"}) {
		t.Errorf("the accounts are %v, want [Lifestyle Sparen Extra Steuern]", got)
	}
	for i, account := range accounts {
		if account.SortOrder != i+1 {
			t.Errorf("the account %q has the sort order %d, want %d", account.Name, account.SortOrder, i+1)
		}
	}
}

func TestMoveAccountDown_moves_the_account_one_position_towards_the_back(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")

	if err := s.MoveAccountDown(context.Background(), lifestyle.ID); err != nil {
		t.Fatalf("MoveAccountDown() failed: %v", err)
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if got := accountNames(accounts); !equalStrings(got, []string{"Sparen", "Lifestyle", "Steuern"}) {
		t.Errorf("the accounts are %v, want [Sparen Lifestyle Steuern]", got)
	}
	for i, account := range accounts {
		if account.SortOrder != i+1 {
			t.Errorf("the account %q has the sort order %d, want %d", account.Name, account.SortOrder, i+1)
		}
	}
}

func TestMoveAccountUp_keeps_the_first_account_in_place(t *testing.T) {
	s := newMemoryStore(t)
	lifestyle := accountByName(t, s, "Lifestyle")

	if err := s.MoveAccountUp(context.Background(), lifestyle.ID); err != nil {
		t.Fatalf("MoveAccountUp() failed: %v", err)
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if got := accountNames(accounts); !equalStrings(got, []string{"Lifestyle", "Sparen", "Steuern"}) {
		t.Errorf("the accounts are %v, want [Lifestyle Sparen Steuern]", got)
	}
}

func TestMoveAccountDown_keeps_the_last_account_in_place(t *testing.T) {
	s := newMemoryStore(t)
	steuern := accountByName(t, s, "Steuern")

	if err := s.MoveAccountDown(context.Background(), steuern.ID); err != nil {
		t.Fatalf("MoveAccountDown() failed: %v", err)
	}

	accounts, err := s.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() failed: %v", err)
	}
	if got := accountNames(accounts); !equalStrings(got, []string{"Lifestyle", "Sparen", "Steuern"}) {
		t.Errorf("the accounts are %v, want [Lifestyle Sparen Steuern]", got)
	}
}

func TestMoveAccount_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	if err := s.MoveAccountUp(context.Background(), 4242); !errors.Is(err, ErrNotFound) {
		t.Errorf("MoveAccountUp() returned %v, want ErrNotFound", err)
	}
	if err := s.MoveAccountDown(context.Background(), 4242); !errors.Is(err, ErrNotFound) {
		t.Errorf("MoveAccountDown() returned %v, want ErrNotFound", err)
	}
}

func TestDeleteAccount_deletes_an_account_without_entries(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")

	if err := s.DeleteAccount(context.Background(), id); err != nil {
		t.Fatalf("DeleteAccount() failed: %v", err)
	}
	if _, err := s.GetAccount(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAccount() returned %v, want ErrNotFound", err)
	}
	if got := countAccounts(t, s); got != 3 {
		t.Errorf("the database holds %d accounts, want the 3 default accounts", got)
	}
}

func TestDeleteAccount_reports_an_account_with_entries(t *testing.T) {
	s := newMemoryStore(t)
	id := createAccount(t, s, "Wohnen")
	createEntry(t, s, planEntryForAccount(t, id, "Miete", 200000, "2026-01-01"))

	err := s.DeleteAccount(context.Background(), id)
	if !errors.Is(err, ErrAccountInUse) {
		t.Fatalf("DeleteAccount() returned %v, want ErrAccountInUse", err)
	}
	if _, err := s.GetAccount(context.Background(), id); err != nil {
		t.Errorf("the account was deleted although it has entries: %v", err)
	}
}

func TestDeleteAccount_reports_an_unknown_account(t *testing.T) {
	s := newMemoryStore(t)

	err := s.DeleteAccount(context.Background(), 4242)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteAccount() returned %v, want ErrNotFound", err)
	}
}
