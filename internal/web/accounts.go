package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The states of an account, as the list shows them.
//
// The state is written as a word, so the list works without colour and
// stays readable for everyone.
const (
	accountStatusActive   = "aktiv"
	accountStatusInactive = "deaktiviert"
)

// The messages of the validation of the account name.
const (
	// accountNameEmpty is the message for a name without any character.
	accountNameEmpty = "Der Name darf nicht leer sein."
	// accountNameTaken is the message for a name that another account
	// already has.
	accountNameTaken = "Dieser Name ist bereits vergeben."
)

// AccountRow holds one account of the list with the texts of the display.
type AccountRow struct {
	// ID is the identifier of the account.
	ID int64
	// Name is the name of the account.
	Name string
	// IsActive reports whether the account counts in the summaries.
	IsActive bool
	// Status is the state of the account in German.
	Status string
	// EntryCount is the number of entries of the account.
	EntryCount int
	// IsFirst reports whether the account is the first one of the list.
	IsFirst bool
	// IsLast reports whether the account is the last one of the list.
	IsLast bool
}

// accountsData holds the data of the list of the accounts.
type accountsData struct {
	// Rows are the accounts with the texts of the display, in the order of
	// the accounts.
	Rows []AccountRow
	// NewName is the name the visitor typed into the form of a new account.
	// It is filled again when the form was rejected.
	NewName string
}

// accountEditData holds the data of the form that renames an account.
type accountEditData struct {
	// ID is the identifier of the account.
	ID int64
	// Name is the name of the account, the value the visitor typed in.
	Name string
	// Action is the address the form sends to.
	Action string
}

// AccountRows builds the rows of the list of the accounts.
//
// The function is pure: it counts the entries of the accounts and marks the
// first and the last row, so the template can switch off the buttons that
// would not change anything. The accounts keep the order they come in, that
// is the order of the store.
func AccountRows(accounts []store.Account, entries []store.Entry) []AccountRow {
	counted := make(map[int64]int, len(accounts))
	for _, entry := range entries {
		counted[entry.AccountID]++
	}

	rows := make([]AccountRow, 0, len(accounts))
	for i, account := range accounts {
		rows = append(rows, AccountRow{
			ID:         account.ID,
			Name:       account.Name,
			IsActive:   account.IsActive,
			Status:     accountStatus(account.IsActive),
			EntryCount: counted[account.ID],
			IsFirst:    i == 0,
			IsLast:     i == len(accounts)-1,
		})
	}
	return rows
}

// accountStatus returns the state of an account in German.
func accountStatus(active bool) string {
	if active {
		return accountStatusActive
	}
	return accountStatusInactive
}

// ValidateAccountName trims the raw name of an account and checks it.
//
// The function is pure: it reads nothing but its argument, so a test can
// check it without a server. It returns the cleaned name and the message of
// the error. An empty message means that the name is valid.
//
// The name of an account may be as long as the name of an entry. The
// uniqueness of the name is not checked here, the store reports a name that
// is already taken with store.ErrDuplicateName.
func ValidateAccountName(raw string) (string, string) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", accountNameEmpty
	case len([]rune(name)) > maxNameLength:
		return "", fmt.Sprintf("Der Name darf höchstens %d Zeichen lang sein.", maxNameLength)
	}
	return name, ""
}

// accountsPage serves the list of the accounts.
//
// The list shows all accounts, the active ones and the deactivated ones, in
// their order. Every row counts the entries of the account and offers the
// actions: move up, move down, rename and activate or deactivate.
//
// The accounts cannot be deleted. Konzept.md §6 offers only anlegen,
// umbenennen, sortieren and deaktivieren, so the interface does not use the
// DeleteAccount of the store, which deletes the accounts without entries.
func (s *server) accountsPage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.accountRows(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	s.renderAccounts(w, http.StatusOK, accountsData{Rows: rows}, nil)
}

// accountCreate stores a new account.
//
// The name is checked before the account is written. A form with an error is
// shown again with the value of the visitor and the status 422, so the
// browser knows that the answer is a new one.
func (s *server) accountCreate(w http.ResponseWriter, r *http.Request) {
	raw, err := readAccountName(w, r)
	if err != nil {
		s.renderFormFailure(w, r, err)
		return
	}

	name, message := ValidateAccountName(raw)
	if message != "" {
		s.renderAccountCreateFailure(w, r, raw, message)
		return
	}

	if _, err := s.store.CreateAccount(r.Context(), name); err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidName):
			s.renderAccountCreateFailure(w, r, raw, accountNameEmpty)
		case errors.Is(err, store.ErrDuplicateName):
			s.renderAccountCreateFailure(w, r, raw, accountNameTaken)
		default:
			s.internalError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

// accountEditPage serves the form that renames an account.
//
// The name of the account fills the form. An unknown account gets the page
// for 404.
func (s *server) accountEditPage(w http.ResponseWriter, r *http.Request) {
	id, ok := accountID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	account, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	s.render(w, http.StatusOK, "accountedit", PageData{
		Title:  "Konto umbenennen",
		Active: "accounts",
		Data: accountEditData{
			ID:     account.ID,
			Name:   account.Name,
			Action: accountPath(account.ID),
		},
	})
}

// accountUpdate stores the new name of an account.
//
// Like the creation, the update answers with the status 303 and the address
// of the list, so the browser does not send the form again.
func (s *server) accountUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := accountID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	if _, err := s.store.GetAccount(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	raw, err := readAccountName(w, r)
	if err != nil {
		s.renderFormFailure(w, r, err)
		return
	}

	name, message := ValidateAccountName(raw)
	if message != "" {
		s.renderAccountEdit(w, http.StatusUnprocessableEntity, id, raw, message)
		return
	}

	if err := s.store.RenameAccount(r.Context(), id, name); err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidName):
			s.renderAccountEdit(w, http.StatusUnprocessableEntity, id, raw, accountNameEmpty)
		case errors.Is(err, store.ErrDuplicateName):
			s.renderAccountEdit(w, http.StatusUnprocessableEntity, id, raw, accountNameTaken)
		case errors.Is(err, store.ErrNotFound):
			s.notFoundPage(w, r)
		default:
			s.internalError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

// accountMoveUp moves an account one position towards the front.
//
// The answer is the status 303 with the address of the list. Moving the
// first account changes nothing.
func (s *server) accountMoveUp(w http.ResponseWriter, r *http.Request) {
	s.changeAccount(w, r, s.store.MoveAccountUp)
}

// accountMoveDown moves an account one position towards the back.
//
// The answer is the status 303 with the address of the list. Moving the last
// account changes nothing.
func (s *server) accountMoveDown(w http.ResponseWriter, r *http.Request) {
	s.changeAccount(w, r, s.store.MoveAccountDown)
}

// accountActivate counts an account again in the summaries.
//
// The answer is the status 303 with the address of the list.
func (s *server) accountActivate(w http.ResponseWriter, r *http.Request) {
	s.changeAccount(w, r, func(ctx context.Context, id int64) error {
		return s.store.SetAccountActive(ctx, id, true)
	})
}

// accountDeactivate leaves an account out of the summaries.
//
// The entries of the account stay readable. The action can be taken back, so
// it needs no confirmation.
//
// The answer is the status 303 with the address of the list.
func (s *server) accountDeactivate(w http.ResponseWriter, r *http.Request) {
	s.changeAccount(w, r, func(ctx context.Context, id int64) error {
		return s.store.SetAccountActive(ctx, id, false)
	})
}

// changeAccount runs an action on one account.
//
// The action reports an unknown account with store.ErrNotFound, that gets the
// page for 404. Every other failure is a failure of the server.
func (s *server) changeAccount(w http.ResponseWriter, r *http.Request, action func(context.Context, int64) error) {
	id, ok := accountID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	if err := action(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

// accountRows reads the accounts with their entries and builds the rows of
// the list.
func (s *server) accountRows(r *http.Request) ([]AccountRow, error) {
	accounts, err := s.store.ListAccounts(r.Context(), true)
	if err != nil {
		return nil, err
	}
	entries, err := s.store.ListEntries(r.Context())
	if err != nil {
		return nil, err
	}
	return AccountRows(accounts, entries), nil
}

// renderAccounts shows the list of the accounts.
//
// The status code carries the reason of the answer, so the browser shows the
// page as a new answer.
func (s *server) renderAccounts(w http.ResponseWriter, status int, data accountsData, fieldErrors map[string]string) {
	s.render(w, status, "accounts", PageData{
		Title:  "Konten",
		Active: "accounts",
		Errors: fieldErrors,
		Data:   data,
	})
}

// renderAccountCreateFailure shows the list of the accounts after a rejected
// form of a new account.
//
// The list stays visible, so the visitor sees the accounts that are already
// there and can correct the name.
func (s *server) renderAccountCreateFailure(w http.ResponseWriter, r *http.Request, raw string, message string) {
	rows, err := s.accountRows(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	s.renderAccounts(w, http.StatusUnprocessableEntity, accountsData{
		Rows:    rows,
		NewName: raw,
	}, map[string]string{fieldName: message})
}

// renderAccountEdit shows the form of the rename with the value of the
// visitor.
func (s *server) renderAccountEdit(w http.ResponseWriter, status int, id int64, name string, message string) {
	s.render(w, status, "accountedit", PageData{
		Title:  "Konto umbenennen",
		Active: "accounts",
		Errors: map[string]string{fieldName: message},
		Data: accountEditData{
			ID:     id,
			Name:   name,
			Action: accountPath(id),
		},
	})
}

// readAccountName reads the name of the account form from the request.
//
// The body is limited, so a request with a huge body cannot take the memory
// of the server. Only the body is read, the values of the query string do
// not change the form.
func readAccountName(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.PostForm.Get(fieldName), nil
}

// accountID reads the identifier of the account from the path.
//
// A malformed identifier is reported with false, the caller answers with the
// page for 404.
func accountID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// accountPath returns the address of the form of an account.
func accountPath(id int64) string {
	return "/accounts/" + strconv.FormatInt(id, 10)
}
