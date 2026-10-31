package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// The limits of the entry form.
//
// The limits keep the entries short enough to be read in the list. The server
// checks them, the attributes of the form only tell the browser the same
// limits.
const (
	// maxNameLength is the number of characters the name of an entry may
	// have.
	maxNameLength = 100
	// maxNotesLength is the number of characters the note of an entry may
	// have.
	maxNotesLength = 1000
	// maxFormBytes is the largest body the server reads from a form.
	maxFormBytes = 64 * 1024
)

// The names of the fields of the entry form.
//
// The names are the keys of the errors of the validation, so the template can
// show the message of a field next to it.
const (
	fieldAccount  = "account_id"
	fieldName     = "name"
	fieldAmount   = "amount"
	fieldStart    = "start_date"
	fieldUnit     = "unit"
	fieldInterval = "interval"
	fieldEndsOn   = "ends_on"
	fieldNotes    = "notes"
)

// EntryForm holds the values of the entry form.
//
// The values are the raw strings of the form fields. A form that was rejected
// is shown again with the values the visitor typed in.
type EntryForm struct {
	// AccountID is the identifier of the account of the entry.
	AccountID string
	// Name is the label of the entry.
	Name string
	// Amount is the amount in francs, as the visitor typed it in.
	Amount string
	// StartDate is the first day of the entry, as YYYY-MM-DD.
	StartDate string
	// Unit is the recurrence unit, as the visitor typed it in.
	Unit string
	// Interval is the number of units between two occurrences.
	Interval string
	// EndsOn is the last day of the entry, as YYYY-MM-DD. It may be empty.
	EndsOn string
	// Notes is the note of the entry.
	Notes string
}

// EntryFormResult is the result of the validation of the entry form.
type EntryFormResult struct {
	// Entry is the entry the values of the form describe.
	Entry plan.Entry
	// Errors holds the errors of the fields, keyed by the name of the field.
	Errors map[string]string
	// Error is the message for the whole page. It is only set when the entry
	// could not be checked against the plan package.
	Error string
}

// accountOption is an account in the selection of the form.
type accountOption struct {
	// ID is the identifier of the account.
	ID int64
	// Name is the name of the account.
	Name string
	// IsActive reports whether the account counts in the summaries.
	IsActive bool
	// Selected reports whether the account is the one of the entry.
	Selected bool
}

// unitOption is a unit in the selection of the form.
type unitOption struct {
	// Value is the value of the field, the unit of the plan package.
	Value string
	// Label is the German name of the unit.
	Label string
}

// unitOptions are the units of an entry with their German names.
var unitOptions = []unitOption{
	{Value: string(plan.Once), Label: "einmalig"},
	{Value: string(plan.Week), Label: "wöchentlich"},
	{Value: string(plan.Month), Label: "monatlich"},
	{Value: string(plan.Year), Label: "jährlich"},
}

// entryFormData holds the data of the entry form.
//
// The page is the same for a new and for an edited entry, only the action of
// the form and the values differ.
type entryFormData struct {
	// IsNew reports whether the form creates a new entry.
	IsNew bool
	// EntryID is the identifier of the entry that is edited.
	EntryID int64
	// Action is the address the form sends to.
	Action string
	// Values holds the values of the form fields.
	Values EntryForm
	// Accounts are the accounts that can be chosen.
	Accounts []accountOption
	// Units are the units that can be chosen.
	Units []unitOption
}

// title returns the title of the page of the form.
func (d entryFormData) title() string {
	if d.IsNew {
		return "Eintrag anlegen"
	}
	return "Eintrag bearbeiten"
}

// ValidateEntryForm checks the values of the entry form.
//
// The function is pure: it reads nothing but its arguments, so a test can
// check it without a server. It returns the entry the values describe and the
// errors of the fields, keyed by the name of the field.
//
// The offered accounts are the accounts the visitor may choose: the active
// accounts and, when an entry is edited, the account of that entry. An
// account that is not in the list of all accounts does not exist, so it is an
// error of the field and not a failure of the server. An account that exists
// but is not offered is deactivated.
//
// The unit "once" ignores the interval and the end date, as the plan package
// does. The interval is then 1 and the entry has no end date.
//
// The check of plan.Entry.Validate is the last instance: it sees the entry
// that the checks of the fields have built.
func ValidateEntryForm(values EntryForm, offered []store.Account, accounts []store.Account) EntryFormResult {
	result := EntryFormResult{Errors: map[string]string{}}
	entry := plan.Entry{}

	// The account of the entry.
	accountID, err := strconv.ParseInt(strings.TrimSpace(values.AccountID), 10, 64)
	switch {
	case err != nil || accountID <= 0:
		result.Errors[fieldAccount] = "Bitte ein Konto auswählen."
	case !hasAccount(accounts, accountID):
		result.Errors[fieldAccount] = "Dieses Konto gibt es nicht."
	case !hasAccount(offered, accountID):
		result.Errors[fieldAccount] = "Dieses Konto ist deaktiviert."
	default:
		entry.AccountID = accountID
	}

	// The name of the entry.
	name := strings.TrimSpace(values.Name)
	switch {
	case name == "":
		result.Errors[fieldName] = "Der Name darf nicht leer sein."
	case len([]rune(name)) > maxNameLength:
		result.Errors[fieldName] = fmt.Sprintf("Der Name darf höchstens %d Zeichen lang sein.", maxNameLength)
	default:
		entry.Name = name
	}

	// The amount of one occurrence.
	amount, err := ParseAmount(values.Amount)
	if err != nil {
		result.Errors[fieldAmount] = amountMessage(err)
	} else {
		entry.AmountMinor = amount
	}

	// The unit of the recurrence.
	unit := plan.Unit(strings.TrimSpace(values.Unit))
	switch {
	case unit == "":
		result.Errors[fieldUnit] = "Bitte einen Rhythmus auswählen."
	case !unit.Valid():
		result.Errors[fieldUnit] = "Diese Wiederholung gibt es nicht."
	default:
		entry.Unit = unit
	}

	// The first day of the entry.
	startDate := strings.TrimSpace(values.StartDate)
	parsedStart, err := plan.ParseDate(startDate)
	startOK := false
	switch {
	case startDate == "":
		result.Errors[fieldStart] = "Das Startdatum darf nicht leer sein."
	case err != nil:
		result.Errors[fieldStart] = "Das Startdatum muss im Format JJJJ-MM-TT stehen."
	case !supportedYear(parsedStart):
		result.Errors[fieldStart] = "Das Startdatum muss zwischen dem Jahr 1 und 9999 liegen."
	default:
		entry.StartDate = parsedStart
		startOK = true
	}

	// The interval between two occurrences and the last day of the entry. The
	// unit "once" ignores both.
	if entry.Unit == plan.Once {
		entry.Interval = 1
	} else {
		interval, err := strconv.Atoi(strings.TrimSpace(values.Interval))
		switch {
		case err != nil:
			result.Errors[fieldInterval] = "Das Intervall muss eine ganze Zahl sein."
		case interval < 1:
			result.Errors[fieldInterval] = "Das Intervall muss mindestens 1 sein."
		case interval > plan.MaxInterval:
			result.Errors[fieldInterval] = fmt.Sprintf("Das Intervall darf höchstens %d sein.", plan.MaxInterval)
		default:
			entry.Interval = interval
		}

		if endsOn := strings.TrimSpace(values.EndsOn); endsOn != "" {
			parsedEnd, err := plan.ParseDate(endsOn)
			switch {
			case err != nil:
				result.Errors[fieldEndsOn] = "Das Enddatum muss im Format JJJJ-MM-TT stehen."
			case !supportedYear(parsedEnd):
				result.Errors[fieldEndsOn] = "Das Enddatum muss zwischen dem Jahr 1 und 9999 liegen."
			case startOK && parsedEnd.Before(entry.StartDate):
				result.Errors[fieldEndsOn] = "Das Enddatum darf nicht vor dem Startdatum liegen."
			default:
				entry.EndsOn = &parsedEnd
			}
		}
	}

	// The note of the entry.
	notes := strings.TrimSpace(values.Notes)
	if len([]rune(notes)) > maxNotesLength {
		result.Errors[fieldNotes] = fmt.Sprintf("Die Notiz darf höchstens %d Zeichen lang sein.", maxNotesLength)
	}
	entry.Notes = notes

	// The check of the plan package is the last instance.
	if len(result.Errors) == 0 {
		if err := entry.Validate(); err != nil {
			field, message := entryProblem(entry)
			if field == "" {
				result.Error = "Die Eingaben sind ungültig."
			} else {
				result.Errors[field] = message
			}
		}
	}

	result.Entry = entry
	return result
}

// entryNewPage serves the form for a new entry.
//
// The form has its own page, so the list stays short and the form works on
// every device without JavaScript.
func (s *server) entryNewPage(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.ListAccounts(r.Context(), true)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	values := EntryForm{
		Unit:      string(plan.Month),
		Interval:  "1",
		StartDate: plan.FormatDate(s.today()),
	}
	offered := offeredAccounts(accounts, nil)
	if len(offered) > 0 {
		values.AccountID = strconv.FormatInt(offered[0].ID, 10)
	}

	data := entryFormData{
		IsNew:    true,
		Action:   "/entries",
		Values:   values,
		Accounts: accountOptions(offered, values.AccountID),
		Units:    unitOptions,
	}
	s.render(w, http.StatusOK, "entryform", PageData{
		Title:  data.title(),
		Active: "entries",
		Data:   data,
	})
}

// entryCreate stores a new entry.
//
// The values are checked before the entry is written. A form with an error is
// shown again with the values of the visitor and the status 422, so the
// browser knows that the answer is a new one.
func (s *server) entryCreate(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.ListAccounts(r.Context(), true)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	values, err := readEntryForm(w, r)
	if err != nil {
		s.renderFormFailure(w, r, err)
		return
	}

	offered := offeredAccounts(accounts, nil)
	result := ValidateEntryForm(values, offered, accounts)
	data := entryFormData{
		IsNew:    true,
		Action:   "/entries",
		Values:   values,
		Accounts: accountOptions(offered, values.AccountID),
		Units:    unitOptions,
	}
	if len(result.Errors) > 0 || result.Error != "" {
		s.renderEntryForm(w, http.StatusUnprocessableEntity, data, result)
		return
	}

	if _, err := s.store.CreateEntry(r.Context(), result.Entry); err != nil {
		if errors.Is(err, store.ErrUnknownAccount) {
			result.Errors = map[string]string{fieldAccount: "Dieses Konto gibt es nicht."}
			s.renderEntryForm(w, http.StatusUnprocessableEntity, data, result)
			return
		}
		s.internalError(w, r, err)
		return
	}

	http.Redirect(w, r, "/entries", http.StatusSeeOther)
}

// entryEditPage serves the form for an existing entry.
//
// The values of the entry fill the form. An unknown entry gets the page for
// 404.
func (s *server) entryEditPage(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	entry, err := s.store.GetEntry(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	accounts, err := s.store.ListAccounts(r.Context(), true)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	current := store.Account{ID: entry.AccountID, Name: entry.AccountName, IsActive: entry.AccountActive}
	values := entryFormValues(entry)
	data := entryFormData{
		EntryID:  entry.ID,
		Action:   entryPath(entry.ID),
		Values:   values,
		Accounts: accountOptions(offeredAccounts(accounts, &current), values.AccountID),
		Units:    unitOptions,
	}
	s.render(w, http.StatusOK, "entryform", PageData{
		Title:  data.title(),
		Active: "entries",
		Data:   data,
	})
}

// entryUpdate stores the changes of an existing entry.
//
// Like the creation, the update answers with the status 303 and the address
// of the list, so the browser does not send the form again.
func (s *server) entryUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(r)
	if !ok {
		s.notFoundPage(w, r)
		return
	}

	entry, err := s.store.GetEntry(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.notFoundPage(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}

	accounts, err := s.store.ListAccounts(r.Context(), true)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	current := store.Account{ID: entry.AccountID, Name: entry.AccountName, IsActive: entry.AccountActive}
	offered := offeredAccounts(accounts, &current)

	values, err := readEntryForm(w, r)
	if err != nil {
		s.renderFormFailure(w, r, err)
		return
	}

	result := ValidateEntryForm(values, offered, accounts)
	data := entryFormData{
		EntryID:  id,
		Action:   entryPath(id),
		Values:   values,
		Accounts: accountOptions(offered, values.AccountID),
		Units:    unitOptions,
	}
	if len(result.Errors) > 0 || result.Error != "" {
		s.renderEntryForm(w, http.StatusUnprocessableEntity, data, result)
		return
	}

	result.Entry.ID = id
	if err := s.store.UpdateEntry(r.Context(), result.Entry); err != nil {
		switch {
		case errors.Is(err, store.ErrUnknownAccount):
			result.Errors = map[string]string{fieldAccount: "Dieses Konto gibt es nicht."}
			s.renderEntryForm(w, http.StatusUnprocessableEntity, data, result)
		case errors.Is(err, store.ErrNotFound):
			s.notFoundPage(w, r)
		default:
			s.internalError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/entries", http.StatusSeeOther)
}

// renderEntryForm shows the form with the values of the visitor.
//
// The status code carries the reason of the answer, so the browser shows the
// page as a new answer.
func (s *server) renderEntryForm(w http.ResponseWriter, status int, data entryFormData, result EntryFormResult) {
	s.render(w, status, "entryform", PageData{
		Title:  data.title(),
		Active: "entries",
		Errors: result.Errors,
		Error:  result.Error,
		Data:   data,
	})
}

// renderFormFailure shows the error page when the body of a form cannot be
// read.
//
// A body that is too large gets the status 413, a body that cannot be read
// at all the status 400.
func (s *server) renderFormFailure(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		s.render(w, http.StatusRequestEntityTooLarge, errorPage, PageData{
			Title: "Eingabe zu gross",
			Error: "Das Formular ist zu gross.",
		})
		return
	}
	s.logger.Warn("cannot read the form", "path", r.URL.Path, "error", err)
	s.render(w, http.StatusBadRequest, errorPage, PageData{
		Title: "Eingabe fehlerhaft",
		Error: "Das Formular konnte nicht gelesen werden.",
	})
}

// readEntryForm reads the values of the entry form from the request.
//
// The body is limited, so a request with a huge body cannot take the memory
// of the server. Only the body is read, the values of the query string do
// not change the form.
func readEntryForm(w http.ResponseWriter, r *http.Request) (EntryForm, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		return EntryForm{}, err
	}
	return EntryForm{
		AccountID: r.PostFormValue(fieldAccount),
		Name:      r.PostFormValue(fieldName),
		Amount:    r.PostFormValue(fieldAmount),
		StartDate: r.PostFormValue(fieldStart),
		Unit:      r.PostFormValue(fieldUnit),
		Interval:  r.PostFormValue(fieldInterval),
		EndsOn:    r.PostFormValue(fieldEndsOn),
		Notes:     r.PostFormValue(fieldNotes),
	}, nil
}

// entryFormValues returns the values of the form for an existing entry.
//
// The amount is written in francs with at most two decimals and without a
// thousands separator, so the parser of the web layer reads it back.
func entryFormValues(entry store.Entry) EntryForm {
	values := EntryForm{
		AccountID: strconv.FormatInt(entry.AccountID, 10),
		Name:      entry.Name,
		Amount:    FormatAmountInput(entry.AmountMinor),
		StartDate: plan.FormatDate(entry.StartDate),
		Unit:      string(entry.Unit),
		Interval:  strconv.Itoa(entry.Interval),
		Notes:     entry.Notes,
	}
	if entry.EndsOn != nil {
		values.EndsOn = plan.FormatDate(*entry.EndsOn)
	}
	return values
}

// offeredAccounts returns the accounts that the form offers.
//
// The active accounts are the choice. The account of the edited entry is
// added when it is deactivated, so the entry can stay on it.
func offeredAccounts(accounts []store.Account, current *store.Account) []store.Account {
	offered := make([]store.Account, 0, len(accounts))
	for _, account := range accounts {
		if account.IsActive || (current != nil && account.ID == current.ID) {
			offered = append(offered, account)
		}
	}
	return offered
}

// accountOptions turns the accounts into the options of the selection.
//
// The selected account is the one the visitor chose. An account that is not
// in the list leaves the selection empty, the browser then shows the first
// option.
func accountOptions(accounts []store.Account, selected string) []accountOption {
	options := make([]accountOption, 0, len(accounts))
	for _, account := range accounts {
		options = append(options, accountOption{
			ID:       account.ID,
			Name:     account.Name,
			IsActive: account.IsActive,
			Selected: strconv.FormatInt(account.ID, 10) == selected,
		})
	}
	return options
}

// entryProblem returns the field and the message of the problem that the
// check of the plan package reports.
//
// The checks repeat the ones of plan.Entry.Validate, because the message of
// the plan package is in English and the interface is in German. The function
// is the last instance: the checks of the fields come first.
func entryProblem(entry plan.Entry) (string, string) {
	switch {
	case entry.AmountMinor <= 0:
		return fieldAmount, "Der Betrag muss grösser als 0 sein."
	case entry.AmountMinor > plan.MaxAmountMinor:
		return fieldAmount, "Der Betrag ist zu gross."
	case !entry.Unit.Valid():
		return fieldUnit, "Diese Wiederholung gibt es nicht."
	case entry.Interval <= 0:
		return fieldInterval, "Das Intervall muss mindestens 1 sein."
	case entry.Unit != plan.Once && entry.Interval > plan.MaxInterval:
		return fieldInterval, fmt.Sprintf("Das Intervall darf höchstens %d sein.", plan.MaxInterval)
	case strings.TrimSpace(entry.Name) == "":
		return fieldName, "Der Name darf nicht leer sein."
	case !supportedYear(entry.StartDate):
		return fieldStart, "Das Startdatum muss zwischen dem Jahr 1 und 9999 liegen."
	case entry.EndsOn != nil && !supportedYear(*entry.EndsOn):
		return fieldEndsOn, "Das Enddatum muss zwischen dem Jahr 1 und 9999 liegen."
	case entry.EndsOn != nil && entry.EndsOn.Before(entry.StartDate):
		return fieldEndsOn, "Das Enddatum darf nicht vor dem Startdatum liegen."
	}
	return "", ""
}

// amountMessage returns the German message of the error of the amount parser.
func amountMessage(err error) string {
	switch {
	case errors.Is(err, ErrAmountEmpty):
		return "Der Betrag darf nicht leer sein."
	case errors.Is(err, ErrAmountNegative):
		return "Der Betrag darf nicht negativ sein."
	case errors.Is(err, ErrAmountZero):
		return "Der Betrag muss grösser als 0 sein."
	case errors.Is(err, ErrAmountPrecision):
		return "Höchstens 2 Nachkommastellen sind erlaubt."
	case errors.Is(err, ErrAmountTooLarge):
		return "Der Betrag ist zu gross."
	case errors.Is(err, ErrAmountFormat):
		return "Der Betrag muss eine Zahl sein."
	default:
		return "Der Betrag ist ungültig."
	}
}

// hasAccount reports whether the account is in the list.
func hasAccount(accounts []store.Account, id int64) bool {
	for _, account := range accounts {
		if account.ID == id {
			return true
		}
	}
	return false
}

// supportedYear reports whether the day lies inside the years the plan package
// supports.
func supportedYear(t time.Time) bool {
	year := t.Year()
	return year >= 1 && year <= 9999
}
