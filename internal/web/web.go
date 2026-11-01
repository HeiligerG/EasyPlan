package web

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
	"github.com/HeiligerG/EasyPlan/internal/store"
)

// Store is the part of the persistence that the web layer uses.
//
// *store.Store implements it. The interface keeps the handlers testable with
// the SQLite database in memory.
//
// The interface has no DeleteAccount: the interface of EasyPlan does not
// delete an account, it deactivates it.
type Store interface {
	ListAccounts(ctx context.Context, includeInactive bool) ([]store.Account, error)
	ListEntries(ctx context.Context) ([]store.Entry, error)
	ListEntriesByAccount(ctx context.Context, accountID int64) ([]store.Entry, error)
	GetAccount(ctx context.Context, id int64) (store.Account, error)
	GetEntry(ctx context.Context, id int64) (store.Entry, error)
	CreateAccount(ctx context.Context, name string) (int64, error)
	RenameAccount(ctx context.Context, id int64, name string) error
	SetAccountActive(ctx context.Context, id int64, active bool) error
	SetAccountOrder(ctx context.Context, id int64, order int) error
	MoveAccountUp(ctx context.Context, id int64) error
	MoveAccountDown(ctx context.Context, id int64) error
	CreateEntry(ctx context.Context, entry plan.Entry) (int64, error)
	UpdateEntry(ctx context.Context, entry plan.Entry) error
	DeleteEntry(ctx context.Context, id int64) error
	// Ping checks that the database can be reached. The health check of the
	// application uses it.
	Ping(ctx context.Context) error
	// VacuumInto writes a copy of the database into the file path. The
	// backup of the interface uses it.
	VacuumInto(ctx context.Context, path string) error
}

// The store of the application implements the interface of the web layer.
var _ Store = (*store.Store)(nil)

// Options holds the dependencies of the web layer.
type Options struct {
	// Store is the persistence the handlers read from and write to. It is
	// required.
	Store Store
	// Now returns the current time. It defaults to time.Now. The handlers
	// use it to determine the current day, so a test can inject a fixed
	// date.
	Now func() time.Time
	// Location is the time zone of the application. It defaults to
	// time.Local.
	Location *time.Location
	// Logger receives the messages of the server. It defaults to
	// slog.Default().
	Logger *slog.Logger
	// BackupDir is the directory in which the temporary copy of the database
	// is written before it is sent as a download. It defaults to the empty
	// string, then the backup is not available.
	//
	// The directory has to be writable. In the container it is the data
	// directory, so the backup works even when the file system of the
	// container is read-only.
	BackupDir string
}

// server holds the state that the handlers share.
type server struct {
	store     Store
	now       func() time.Time
	location  *time.Location
	logger    *slog.Logger
	templates map[string]*template.Template
	static    http.Handler
	backupDir string
}

// New returns the HTTP handler of the application.
//
// The handler serves the pages of the interface, the embedded static files
// and rejects the writing requests that come from a foreign origin.
func New(opts Options) (http.Handler, error) {
	if opts.Store == nil {
		return nil, errors.New("web: no store given")
	}

	s := &server{
		store:     opts.Store,
		now:       opts.Now,
		location:  opts.Location,
		logger:    opts.Logger,
		backupDir: opts.BackupDir,
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.location == nil {
		s.location = time.Local
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}

	if err := s.loadTemplates(); err != nil {
		return nil, err
	}
	static, err := s.newStaticHandler()
	if err != nil {
		return nil, err
	}
	s.static = static

	mux := http.NewServeMux()
	s.routes(mux)

	return s.securityHeaders(s.originCheck(mux)), nil
}

// routes registers the routes of the application.
//
// The patterns carry the method, so a page only answers the requests of its
// own method. The check of the health of the application has its own route,
// it is not a page of the interface. Every page of the navigation has its
// own route, the pages are filled in step by step. The entries have their
// own routes for the list, the form of an entry and the deletion. The
// accounts have their own routes for the list, the form of a rename and the
// actions on an account. The backup has its own route, it answers only the
// requests that write. The catch-all at the end answers every other address
// with the page for 404.
func (s *server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.healthPage)
	mux.HandleFunc("GET /{$}", s.dashboardPage)
	mux.HandleFunc("GET /entries", s.entriesPage)
	mux.HandleFunc("GET /entries/new", s.entryNewPage)
	mux.HandleFunc("POST /entries", s.entryCreate)
	mux.HandleFunc("GET /entries/{id}", s.entryEditPage)
	mux.HandleFunc("POST /entries/{id}", s.entryUpdate)
	mux.HandleFunc("GET /entries/{id}/delete", s.entryDeletePage)
	mux.HandleFunc("POST /entries/{id}/delete", s.entryDelete)
	mux.HandleFunc("GET /timeline", s.timelinePage)
	mux.HandleFunc("GET /accounts", s.accountsPage)
	mux.HandleFunc("POST /accounts", s.accountCreate)
	mux.HandleFunc("GET /accounts/{id}", s.accountEditPage)
	mux.HandleFunc("POST /accounts/{id}", s.accountUpdate)
	mux.HandleFunc("POST /accounts/{id}/up", s.accountMoveUp)
	mux.HandleFunc("POST /accounts/{id}/down", s.accountMoveDown)
	mux.HandleFunc("POST /accounts/{id}/activate", s.accountActivate)
	mux.HandleFunc("POST /accounts/{id}/deactivate", s.accountDeactivate)
	mux.HandleFunc("POST /backup", s.backupDownload)
	mux.Handle("GET /static/", s.static)
	// Every other path is unknown and gets the page for 404.
	mux.HandleFunc("/", s.notFoundPage)
}

// today returns the current calendar day in the time zone of the application.
func (s *server) today() time.Time {
	return s.now().In(s.location)
}

// PageData is the data model of the pages.
//
// The fields are the same for every page, so the layout can show the title,
// the navigation and the messages without knowing the page.
//
// Errors holds the validation errors of a form, keyed by the name of the
// field. Every form can use it the same way: the handler collects the errors
// of the input and the template shows them next to the field.
type PageData struct {
	// Title is the title of the page.
	Title string
	// Active names the entry of the navigation that belongs to the page.
	Active string
	// Errors holds the validation errors of the form, keyed by the name of
	// the field.
	Errors map[string]string
	// Error is a message that belongs to the whole page.
	Error string
	// Data holds the data of the page itself.
	Data any
	// BackupAvailable reports whether the button of the backup is shown. The
	// backup needs a directory for the temporary copy, without one it is not
	// possible. The rendering sets the flag, so every page shows the button
	// when the backup works.
	BackupAvailable bool
}

// render writes a page to the response.
//
// The page is written with the given status code. The template is rendered
// into a buffer first, so a template that fails halfway does not send a
// half-written page. The failure is written to the log instead and the
// visitor gets a plain error without any details of the failure.
func (s *server) render(w http.ResponseWriter, status int, page string, data PageData) {
	tmpl, ok := s.templates[page]
	if !ok {
		s.logger.Error("unknown page", "page", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// The button of the backup belongs to the layout, so the flag is set
	// here and not by every handler.
	data.BackupAvailable = s.backupDir != ""

	var buffer bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buffer, layoutTemplate, data); err != nil {
		s.logger.Error("cannot render page", "page", page, "error", err)
		// The page cannot be written, so the answer is a plain text. The
		// error page cannot be rendered either in this case, so it is not
		// tried and no recursion can happen.
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buffer.WriteTo(w); err != nil {
		s.logger.Error("cannot write page", "page", page, "error", err)
	}
}

// internalError reports a failure to the log and shows the error page.
//
// The browser never sees the details of the failure, they stay in the log of
// the server.
func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.Error("request failed",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
	s.render(w, http.StatusInternalServerError, errorPage, PageData{
		Title: "Fehler",
		Error: "Die Seite konnte nicht geladen werden.",
	})
}
