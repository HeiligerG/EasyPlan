package web

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/store"
)

// backupDirPrefixLiteral is the prefix of the name of the temporary directory
// of a backup as the specification names it.
//
// The tests that build such a directory use the literal and not the constant
// of the production code, so a change of the constant is visible in the tests.
const backupDirPrefixLiteral = "easyplan-backup-"

// backupApp is the application under test for the backup: the handler, the
// database behind it and the directory of the temporary copy.
type backupApp struct {
	*testApp
	dir string
}

// newBackupApp starts the application with a database in memory, a fixed
// clock and a directory for the temporary copy of the backup.
func newBackupApp(t *testing.T, now time.Time) *backupApp {
	t.Helper()

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})

	handler, dir := newBackupHandler(t, st, now)
	return &backupApp{testApp: &testApp{handler: handler, store: st}, dir: dir}
}

// newBackupHandler starts the application with the given store, a fixed clock
// and a directory for the temporary copy of the backup. It returns the
// handler and the directory.
func newBackupHandler(t *testing.T, st Store, now time.Time) (http.Handler, string) {
	t.Helper()

	dir := t.TempDir()
	return newBackupHandlerIn(t, st, now, dir), dir
}

// newBackupHandlerIn starts the application with the given store, a fixed
// clock and the given directory for the temporary copy of the backup.
func newBackupHandlerIn(t *testing.T, st Store, now time.Time, dir string) http.Handler {
	t.Helper()

	handler, err := New(Options{
		Store:     st,
		Now:       func() time.Time { return now },
		Location:  testLocation,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		BackupDir: dir,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return handler
}

// failingBackupStore is a store whose backup fails.
type failingBackupStore struct {
	*store.Store
	err error
}

// VacuumInto fails with the error of the store.
func (s failingBackupStore) VacuumInto(ctx context.Context, path string) error {
	return s.err
}

// blockingBackupStore is a store whose backup waits until the test lets it
// continue.
//
// The store behind it works, the copy is only made when the test releases it,
// so the test can see when the answer of the server starts.
type blockingBackupStore struct {
	*store.Store
	entered chan struct{}
	release chan struct{}
}

// VacuumInto waits for the release of the test and makes the copy then.
func (s *blockingBackupStore) VacuumInto(ctx context.Context, path string) error {
	select {
	case s.entered <- struct{}{}:
	default:
	}
	select {
	case <-s.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return s.Store.VacuumInto(ctx, path)
}

// recordingBackupStore is a store that remembers the path of every copy of
// the database it writes.
//
// The paths show which file every backup used, so a test can see whether two
// backups at the same time shared a file.
type recordingBackupStore struct {
	*store.Store
	mu    sync.Mutex
	paths []string
}

// VacuumInto writes the copy and remembers its path.
func (s *recordingBackupStore) VacuumInto(ctx context.Context, path string) error {
	s.mu.Lock()
	s.paths = append(s.paths, path)
	s.mu.Unlock()
	return s.Store.VacuumInto(ctx, path)
}

// recordedPaths returns the paths of the copies that were written.
func (s *recordingBackupStore) recordedPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.paths...)
}

// TestBackupDownload_sends_the_database_as_a_download checks the answer of
// the backup and the content of the file.
func TestBackupDownload_sends_the_database_as_a_download(t *testing.T) {
	app := newBackupApp(t, conceptToday)
	lifestyle := accountByName(t, app.testApp, "Lifestyle")
	createEntry(t, app.testApp, monthlyEntry(t, lifestyle.ID, "Miete", 200000, "2026-01-01"))

	rec := app.postForm(t, "/backup", url.Values{})

	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != backupContentType {
		t.Errorf("the content type is %q, want %q", contentType, backupContentType)
	}
	const disposition = `attachment; filename="easyplan-20261001-000000.db"`
	if got := rec.Header().Get("Content-Disposition"); got != disposition {
		t.Errorf("the disposition is %q, want %q", got, disposition)
	}
	if cache := rec.Header().Get("Cache-Control"); cache != "no-store" {
		t.Errorf("the cache header is %q, want %q", cache, "no-store")
	}
	length := rec.Header().Get("Content-Length")
	if length == "" {
		t.Fatal("the answer has no content length")
	}
	size, err := strconv.Atoi(length)
	if err != nil {
		t.Fatalf("the content length %q is not a number: %v", length, err)
	}
	if size != rec.Body.Len() {
		t.Errorf("the content length is %d, want the %d bytes of the body", size, rec.Body.Len())
	}

	// The body has to be the database with the data of the source.
	path := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(path, rec.Body.Bytes(), 0o600); err != nil {
		t.Fatalf("cannot write the download: %v", err)
	}
	copyStore, err := store.Open(path)
	if err != nil {
		t.Fatalf("the download is not a valid database: %v", err)
	}
	defer func() {
		if err := copyStore.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}()

	accounts, err := copyStore.ListAccounts(context.Background(), true)
	if err != nil {
		t.Fatalf("ListAccounts() of the download failed: %v", err)
	}
	if len(accounts) != 3 {
		t.Errorf("the download holds %d accounts, want 3: %v", len(accounts), accountNames(accounts))
	}
	entries, err := copyStore.ListEntries(context.Background())
	if err != nil {
		t.Fatalf("ListEntries() of the download failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("the download holds %d entries, want 1", len(entries))
	}
	if entries[0].Name != "Miete" || entries[0].AmountMinor != 200000 {
		t.Errorf("the entry of the download is %+v, want the entry Miete with 200000 Rappen", entries[0])
	}
}

// TestBackupDownload_removes_the_temporary_copy checks that nothing is left
// in the directory of the backup after the download.
func TestBackupDownload_removes_the_temporary_copy(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.postForm(t, "/backup", url.Values{})
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}

	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_answers_with_500_when_the_copy_fails checks the answer of
// the backup when the database cannot be copied.
func TestBackupDownload_answers_with_500_when_the_copy_fails(t *testing.T) {
	const secret = "SECRET-BACKUP-FAILURE-4711"
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	handler, dir := newBackupHandler(t, failingBackupStore{Store: st, err: errors.New(secret)}, conceptToday)

	req := newRequest(http.MethodPost, "/backup", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if contains(body, secret) {
		t.Errorf("the page shows the details of the failure:\n%s", body)
	}
	if !contains(body, "Die Seite konnte nicht geladen werden.") {
		t.Errorf("the page does not show the message of the failure:\n%s", body)
	}
	if leftovers := directoryEntries(t, dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_rejects_a_foreign_origin checks that the backup is
// protected against a request of another origin.
func TestBackupDownload_rejects_a_foreign_origin(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.postFormWithOrigin(t, "/backup", url.Values{}, "http://evil.example")

	if rec.Code != http.StatusForbidden {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_rejects_a_request_without_an_origin checks that the
// backup is protected against a request without an origin.
func TestBackupDownload_rejects_a_request_without_an_origin(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.postFormWithOrigin(t, "/backup", url.Values{}, "")

	if rec.Code != http.StatusForbidden {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusForbidden)
	}
	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_answers_with_404_for_a_reading_request checks that the
// backup is not reachable with a GET.
//
// The patterns of the routes carry the method, so the catch-all route at the
// end of the router answers with the page for 404.
func TestBackupDownload_answers_with_404_for_a_reading_request(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.get(t, "/backup")

	if rec.Code != http.StatusNotFound {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !contains(rec.Body.String(), "Seite nicht gefunden") {
		t.Errorf("the page does not show the page for 404:\n%s", rec.Body.String())
	}
}

// TestBackupDownload_answers_with_503_without_a_backup_directory checks the
// answer of the backup when the application has no directory for the
// temporary copy.
func TestBackupDownload_answers_with_503_without_a_backup_directory(t *testing.T) {
	app := newTestApp(t, conceptToday)

	rec := app.postForm(t, "/backup", url.Values{})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !contains(rec.Body.String(), "Das Backup ist nicht eingerichtet.") {
		t.Errorf("the page does not explain the problem:\n%s", rec.Body.String())
	}
}

// TestBackupDownload_answers_with_500_when_the_request_was_cancelled checks
// the answer of the backup when the context of the request is already
// cancelled when the request arrives.
//
// The copy of the database cannot be made with a cancelled context, so the
// answer is the page for 500 and nothing may stay in the directory of the
// backup.
func TestBackupDownload_answers_with_500_when_the_request_was_cancelled(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := newRequest(http.MethodPost, "/backup", nil).WithContext(ctx)
	req.Header.Set("Origin", testOrigin)

	rec := app.serve(t, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_cleans_up_when_the_visitor_goes_away checks that nothing
// is left in the directory of the backup when the visitor breaks the request
// off while the copy of the database is made.
//
// The store behind the handler waits, so the test can break the request off
// while the copy runs. The temporary directory of the copy has to be gone
// again, also when the visitor is the one who ends the request.
func TestBackupDownload_cleans_up_when_the_visitor_goes_away(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	blocking := &blockingBackupStore{
		Store:   st,
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	handler, dir := newBackupHandler(t, blocking, conceptToday)

	ctx, cancel := context.WithCancel(context.Background())
	req := newRequest(http.MethodPost, "/backup", nil).WithContext(ctx)
	req.Header.Set("Origin", testOrigin)

	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(rec, req)
	}()

	// The copy of the database has to be running, then the visitor goes
	// away.
	select {
	case <-blocking.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the copy of the database did not start")
	}
	cancel()
	<-done

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if leftovers := directoryEntries(t, dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_uses_its_own_file_for_every_backup checks that two
// backups at the same time do not use the same file.
//
// The store behind the handler remembers the path of every copy it writes, so
// the test can see that every backup got a path of its own and that every
// path lies in the directory of the backup.
func TestBackupDownload_uses_its_own_file_for_every_backup(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	recording := &recordingBackupStore{Store: st}
	handler, dir := newBackupHandler(t, recording, conceptToday)

	const count = 4
	var group sync.WaitGroup
	statuses := make([]int, count)
	for i := range statuses {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			req := newRequest(http.MethodPost, "/backup", nil)
			req.Header.Set("Origin", testOrigin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			statuses[index] = rec.Code
		}(i)
	}
	group.Wait()

	for i, status := range statuses {
		if status != http.StatusOK {
			t.Errorf("the backup %d has the status %d, want %d", i, status, http.StatusOK)
		}
	}

	paths := recording.recordedPaths()
	if len(paths) != count {
		t.Fatalf("the store wrote %d copies, want %d", len(paths), count)
	}
	seen := make(map[string]bool, count)
	for _, path := range paths {
		if seen[path] {
			t.Errorf("two backups used the file %q", path)
		}
		seen[path] = true

		// The copy has to lie in the directory of the backup.
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			t.Fatalf("cannot read the path %q relative to %q: %v", path, dir, err)
		}
		if relative == "." || strings.HasPrefix(relative, "..") {
			t.Errorf("the copy %q lies outside of the directory %q", path, dir)
		}
		// The temporary directory of the copy has to carry the prefix, so
		// the next start can find it.
		name := filepath.Base(filepath.Dir(path))
		if !strings.HasPrefix(name, BackupDirPrefix) {
			t.Errorf("the temporary directory %q does not begin with %q", name, BackupDirPrefix)
		}
	}
	if leftovers := directoryEntries(t, dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestThePages_show_the_button_of_the_backup checks that the button of the
// backup is in the footer of the pages.
func TestThePages_show_the_button_of_the_backup(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	body := app.get(t, "/").Body.String()

	if !contains(body, `<form class="site-footer__backup" method="post" action="/backup">`) {
		t.Errorf("the page has no form for the backup:\n%s", body)
	}
	if !contains(body, ">Backup herunterladen</button>") {
		t.Errorf("the page has no button for the backup:\n%s", body)
	}
}

// TestThePages_hold_no_button_of_the_backup_without_a_backup_directory checks
// that the pages do not offer a backup that cannot work.
func TestThePages_hold_no_button_of_the_backup_without_a_backup_directory(t *testing.T) {
	app := newTestApp(t, conceptToday)

	body := app.get(t, "/").Body.String()

	if contains(body, `action="/backup"`) {
		t.Errorf("the page offers the backup without a directory:\n%s", body)
	}
}

// TestWriteBackup_writes_the_copy_for_the_application_alone checks the rights
// of the temporary directory and of the copy of the database.
func TestWriteBackup_writes_the_copy_for_the_application_alone(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	s := &server{
		store:     app.store,
		now:       func() time.Time { return conceptToday },
		location:  testLocation,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		backupDir: app.dir,
	}

	path, cleanup, err := s.writeBackup(context.Background())
	if err != nil {
		t.Fatalf("writeBackup() failed: %v", err)
	}
	defer cleanup()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("cannot read the copy: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("the copy has the rights %o, want %o", got, 0o600)
	}

	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("cannot read the directory of the copy: %v", err)
	}
	if got := dir.Mode().Perm(); got != 0o700 {
		t.Errorf("the directory of the copy has the rights %o, want %o", got, 0o700)
	}
}

// TestBackupDownload_works_with_a_recorder_without_a_deadline checks that the
// backup works with a response writer that cannot set a deadline.
//
// The recorder of the tests does not support a deadline. The route has to
// accept that without an error, the answer is the same as with a server.
func TestBackupDownload_works_with_a_recorder_without_a_deadline(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.postForm(t, "/backup", url.Values{})

	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte(sqliteHeader)) {
		t.Error("the answer does not begin with the magic number of SQLite")
	}
	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_sends_the_database_with_a_short_write_timeout_of_the_server
// checks that the download arrives completely when the copy of the database
// takes longer than the write timeout of the server.
//
// The server sets the deadline of the connection before the handler runs, so
// the copy of the database eats the time of the write timeout. The route sets
// its own deadline for the answer, so the download is not cut short.
func TestBackupDownload_sends_the_database_with_a_short_write_timeout_of_the_server(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	blocking := &blockingBackupStore{
		Store:   st,
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	handler, dir := newBackupHandler(t, blocking, conceptToday)

	// The write timeout of the server is shorter than the copy of the
	// database, so the deadline of the server runs out before the answer
	// begins.
	const writeTimeout = 200 * time.Millisecond
	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = writeTimeout
	server.Start()
	t.Cleanup(server.Close)

	answers := make(chan backupAnswer, 1)
	go func() {
		answers <- readBackup(t, server.URL)
	}()

	// The copy of the database waits for the release of the test, so the
	// deadline of the server runs out while the copy is made.
	select {
	case <-blocking.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the copy of the database did not start")
	}
	timer := time.NewTimer(2 * writeTimeout)
	defer timer.Stop()
	<-timer.C

	close(blocking.release)

	answer := <-answers
	if answer.err != nil {
		t.Fatalf("the backup failed: %v", answer.err)
	}
	if answer.status != http.StatusOK {
		t.Fatalf("the status is %d, want %d", answer.status, http.StatusOK)
	}
	if !bytes.HasPrefix(answer.body, []byte(sqliteHeader)) {
		t.Error("the answer does not begin with the magic number of SQLite")
	}
	if answer.length != "" && answer.length != strconv.Itoa(len(answer.body)) {
		t.Errorf("the answer holds %d bytes, want the %s bytes of the content length", len(answer.body), answer.length)
	}

	// The database has to be complete, a visitor has to be able to restore
	// it.
	path := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(path, answer.body, 0o600); err != nil {
		t.Fatalf("cannot write the download: %v", err)
	}
	if got := integrityOfDatabase(t, path); got != "ok" {
		t.Errorf("the integrity check of the download is %q, want %q", got, "ok")
	}
	if leftovers := directoryEntries(t, dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// backupAnswer holds the answer of the server of a backup.
type backupAnswer struct {
	status int
	length string
	body   []byte
	err    error
}

// readBackup asks the server of the URL for a backup and reads the answer.
func readBackup(t *testing.T, baseURL string) backupAnswer {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/backup", nil)
	if err != nil {
		return backupAnswer{err: err}
	}
	req.Header.Set("Origin", baseURL)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return backupAnswer{err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return backupAnswer{
		status: resp.StatusCode,
		length: resp.Header.Get("Content-Length"),
		body:   body,
		err:    err,
	}
}

// TestCleanupBackups_removes_the_leftovers_of_a_backup checks that the
// temporary directories of the backups are removed and that everything else
// in the directory stays untouched.
//
// The names of the leftovers carry the prefix of the specification and not the
// constant of the production code, so a change of the constant is visible in
// this test.
func TestCleanupBackups_removes_the_leftovers_of_a_backup(t *testing.T) {
	dir := t.TempDir()

	// The leftover of a backup that was killed hard.
	leftover := filepath.Join(dir, backupDirPrefixLiteral+"123456789")
	if err := os.Mkdir(leftover, 0o700); err != nil {
		t.Fatalf("cannot create the leftover: %v", err)
	}
	if err := os.WriteFile(filepath.Join(leftover, backupFileName), []byte("copy"), 0o600); err != nil {
		t.Fatalf("cannot write the copy: %v", err)
	}

	// The database of the application and its journal stay.
	database := filepath.Join(dir, "easyplan.db")
	if err := os.WriteFile(database, []byte("database"), 0o600); err != nil {
		t.Fatalf("cannot write the database: %v", err)
	}
	journal := filepath.Join(dir, "easyplan.db-wal")
	if err := os.WriteFile(journal, []byte("journal"), 0o600); err != nil {
		t.Fatalf("cannot write the journal: %v", err)
	}

	// Another directory stays.
	other := filepath.Join(dir, "other")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatalf("cannot create the other directory: %v", err)
	}

	// A file with the name of a backup stays, the cleanup removes only
	// directories.
	file := filepath.Join(dir, backupDirPrefixLiteral+"file")
	if err := os.WriteFile(file, []byte("file"), 0o600); err != nil {
		t.Fatalf("cannot write the file: %v", err)
	}

	// A link with the name of a backup stays, the cleanup follows no link.
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("cannot create the target of the link: %v", err)
	}
	link := filepath.Join(dir, backupDirPrefixLiteral+"link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symbolic link: %v", err)
	}

	if err := CleanupBackups(dir); err != nil {
		t.Fatalf("CleanupBackups() failed: %v", err)
	}

	if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the leftover %q is still there", leftover)
	}
	for _, path := range []string{database, journal, other, file, link, target} {
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("%q is gone: %v", path, err)
		}
	}
}

// TestCleanupBackups_fails_with_a_directory_that_does_not_exist checks the
// answer of the cleanup for a directory that cannot be read.
func TestCleanupBackups_fails_with_a_directory_that_does_not_exist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gibt-es-nicht")

	if err := CleanupBackups(missing); err == nil {
		t.Error("CleanupBackups() worked with a missing directory, want an error")
	}
}

// directoryEntries returns the names of the entries of a directory.
func directoryEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read the directory %q: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// The magic number at the beginning of every SQLite file.
const sqliteHeader = "SQLite format 3\x00"

// TestBackupDownload_sends_a_valid_database checks that the download is a
// valid SQLite file.
//
// The file has to begin with the magic number of SQLite and the integrity
// check of the file has to succeed, so the copy can be restored.
func TestBackupDownload_sends_a_valid_database(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	rec := app.postForm(t, "/backup", url.Values{})
	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.Bytes()
	if !bytes.HasPrefix(body, []byte(sqliteHeader)) {
		t.Fatalf("the download does not begin with the magic number of SQLite: %q", body[:min(len(body), 16)])
	}

	path := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("cannot write the download: %v", err)
	}
	if got := integrityOfDatabase(t, path); got != "ok" {
		t.Errorf("the integrity check of the download is %q, want %q", got, "ok")
	}
}

// TestBackupDownload_answers_with_500_when_the_backup_directory_is_gone
// checks the answer of the backup when the directory of the temporary copy
// does not exist.
//
// The directory can be gone when the data directory was removed while the
// application runs.
func TestBackupDownload_answers_with_500_when_the_backup_directory_is_gone(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	missing := filepath.Join(t.TempDir(), "gibt-es-nicht")
	handler := newBackupHandlerIn(t, st, conceptToday, missing)

	req := newRequest(http.MethodPost, "/backup", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !contains(rec.Body.String(), "Die Seite konnte nicht geladen werden.") {
		t.Errorf("the page does not show the message of the failure:\n%s", rec.Body.String())
	}
}

// TestBackupDownload_starts_the_answer_after_the_copy_is_ready checks that
// the answer of the backup begins only when the copy is complete.
//
// A visitor that gets a half database cannot restore it, so the answer has to
// wait for the copy.
func TestBackupDownload_starts_the_answer_after_the_copy_is_ready(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	blocking := &blockingBackupStore{
		Store:   st,
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	handler, dir := newBackupHandler(t, blocking, conceptToday)

	writer := newHeaderWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := newRequest(http.MethodPost, "/backup", nil)
		req.Header.Set("Origin", testOrigin)
		handler.ServeHTTP(writer, req)
	}()

	// The copy has to start before the answer can begin.
	select {
	case <-blocking.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the copy of the database did not start")
	}
	select {
	case <-writer.started:
		t.Error("the answer began before the copy of the database was ready")
	default:
	}

	close(blocking.release)
	<-done

	if status := writer.statusOf(); status != http.StatusOK {
		t.Errorf("the status is %d, want %d", status, http.StatusOK)
	}
	if !bytes.HasPrefix(writer.body.Bytes(), []byte(sqliteHeader)) {
		t.Error("the answer does not begin with the magic number of SQLite")
	}
	if leftovers := directoryEntries(t, dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_cleans_up_after_a_broken_connection checks that nothing
// is left in the directory of the backup when the connection breaks while
// the answer is sent.
func TestBackupDownload_cleans_up_after_a_broken_connection(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	req := newRequest(http.MethodPost, "/backup", nil)
	req.Header.Set("Origin", testOrigin)
	writer := &brokenWriter{header: make(http.Header)}
	app.handler.ServeHTTP(writer, req)

	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_works_with_a_read_only_parent_directory checks that the
// backup works when the file system around the data directory cannot be
// changed.
//
// The container of EasyPlan runs with a read-only file system, only the data
// directory can be written. The copy of the database is written into the data
// directory, so the backup needs no right outside of it.
func TestBackupDownload_works_with_a_read_only_parent_directory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the test runs as root, the rights of a directory do not stop the write")
	}

	root := t.TempDir()
	data := filepath.Join(root, "data")
	if err := os.Mkdir(data, 0o700); err != nil {
		t.Fatalf("cannot create the data directory: %v", err)
	}
	// The directory around the data directory cannot be changed anymore.
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatalf("cannot change the rights of the directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o700); err != nil {
			t.Errorf("cannot change the rights of the directory: %v", err)
		}
	})

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("store.OpenMemory() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close() failed: %v", err)
		}
	})
	handler := newBackupHandlerIn(t, st, conceptToday, data)

	req := newRequest(http.MethodPost, "/backup", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte(sqliteHeader)) {
		t.Error("the answer does not begin with the magic number of SQLite")
	}
	if leftovers := directoryEntries(t, data); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestBackupDownload_answers_with_a_valid_database_for_every_parallel_backup
// checks that backups at the same time do not disturb each other.
//
// Every backup has its own file, every answer is a complete database and
// nothing is left in the directory of the backup.
func TestBackupDownload_answers_with_a_valid_database_for_every_parallel_backup(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	const count = 8
	var group sync.WaitGroup
	statuses := make([]int, count)
	bodies := make([][]byte, count)
	for i := range statuses {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			req := newRequest(http.MethodPost, "/backup", nil)
			req.Header.Set("Origin", testOrigin)
			rec := httptest.NewRecorder()
			app.handler.ServeHTTP(rec, req)
			statuses[index] = rec.Code
			bodies[index] = rec.Body.Bytes()
		}(i)
	}
	group.Wait()

	for i, status := range statuses {
		if status != http.StatusOK {
			t.Errorf("the backup %d has the status %d, want %d", i, status, http.StatusOK)
			continue
		}
		if !bytes.HasPrefix(bodies[i], []byte(sqliteHeader)) {
			t.Errorf("the backup %d does not begin with the magic number of SQLite", i)
		}
	}
	if leftovers := directoryEntries(t, app.dir); len(leftovers) != 0 {
		t.Errorf("the directory of the backup holds %v, want nothing", leftovers)
	}
}

// TestThePages_show_the_button_of_the_backup_on_every_page checks that every
// page of the interface offers the backup.
//
// The button belongs to the layout, so it is on every page.
func TestThePages_show_the_button_of_the_backup_on_every_page(t *testing.T) {
	app := newBackupApp(t, conceptToday)

	for _, target := range []string{"/", "/entries", "/accounts", "/timeline", "/unknown-address"} {
		t.Run(target, func(t *testing.T) {
			rec := app.get(t, target)
			if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
				t.Fatalf("the status is %d, want %d or %d", rec.Code, http.StatusOK, http.StatusNotFound)
			}
			body := rec.Body.String()
			if !contains(body, `action="/backup"`) {
				t.Errorf("the page has no form for the backup:\n%s", body)
			}
			if !contains(body, ">Backup herunterladen</button>") {
				t.Errorf("the page has no button for the backup:\n%s", body)
			}
		})
	}
}

// TestThePages_hold_no_button_of_the_backup_without_a_backup_directory_on_every_page
// checks that no page offers a backup that cannot work.
func TestThePages_hold_no_button_of_the_backup_without_a_backup_directory_on_every_page(t *testing.T) {
	app := newTestApp(t, conceptToday)

	for _, target := range []string{"/", "/entries", "/accounts", "/timeline", "/unknown-address"} {
		t.Run(target, func(t *testing.T) {
			body := app.get(t, target).Body.String()
			if contains(body, `action="/backup"`) {
				t.Errorf("the page offers the backup without a directory:\n%s", body)
			}
		})
	}
}

// headerWriter is a response writer that remembers the status of the answer
// and tells the test when the answer begins.
//
// The channel makes the beginning of the answer visible for the test without
// a sleep: the test can look at the channel while the handler still runs.
type headerWriter struct {
	mu      sync.Mutex
	header  http.Header
	status  int
	body    bytes.Buffer
	started chan struct{}
}

// newHeaderWriter returns a writer that remembers the beginning of the
// answer.
func newHeaderWriter() *headerWriter {
	return &headerWriter{header: make(http.Header), started: make(chan struct{}, 1)}
}

// Header returns the header of the answer.
func (w *headerWriter) Header() http.Header { return w.header }

// WriteHeader remembers the status of the answer and signals that the answer
// begins.
func (w *headerWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = status
	}
	w.signal()
}

// Write writes the body of the answer and signals that the answer begins.
func (w *headerWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.signal()
	return w.body.Write(p)
}

// signal tells the test that the answer has begun.
func (w *headerWriter) signal() {
	select {
	case w.started <- struct{}{}:
	default:
	}
}

// statusOf returns the status of the answer.
func (w *headerWriter) statusOf() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// brokenWriter is a response writer whose write always fails, like a
// connection that breaks while the answer is sent.
type brokenWriter struct {
	header http.Header
}

// Header returns the header of the answer.
func (w *brokenWriter) Header() http.Header { return w.header }

// WriteHeader does nothing, a broken connection has no use for the header.
func (w *brokenWriter) WriteHeader(int) {}

// Write fails like a broken connection.
func (w *brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("the connection is broken")
}

// integrityOfDatabase reads the result of the integrity check of the database
// in the file path.
func integrityOfDatabase(t *testing.T, path string) string {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("cannot open the database %q: %v", path, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("cannot close the database: %v", err)
		}
	})

	var value string
	if err := db.QueryRowContext(context.Background(), "PRAGMA integrity_check").Scan(&value); err != nil {
		t.Fatalf("cannot read the integrity check of %q: %v", path, err)
	}
	return value
}
