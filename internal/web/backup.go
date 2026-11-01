package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The media type of the copy of the database.
//
// The type tells the browser that the file is a SQLite database and that it
// has to be saved instead of shown.
const backupContentType = "application/vnd.sqlite3"

// The name of the file of the copy inside the temporary directory.
//
// The name is fixed, the directory around it is unique for every backup, so
// two backups at the same time never use the same file.
const backupFileName = "easyplan.db"

// The format of the time in the name of the file of the download.
const backupTimeFormat = "20060102-150405"

// BackupDirPrefix is the prefix of the name of the temporary directory of a
// backup.
//
// The name of the directory is built with this prefix, so the next start can
// find the directories that a backup left behind when it was killed hard.
const BackupDirPrefix = "easyplan-backup-"

// backupWriteTimeout is the time the answer of the backup may take.
//
// The server sets a write timeout of 60 seconds and it begins before the copy
// of the database is made, so the copy of a big database would cut the answer
// short. The route sets its own deadline for the answer, it is long enough
// for the copy and for the download.
const backupWriteTimeout = 10 * time.Minute

// backupDownload sends a copy of the database as a download.
//
// The route answers only POST.
//
// A GET has to be safe, it must not change the state of the application
// (RFC 9110 §9.2.1). A backup copies the whole database, so a link, a
// prefetch of the browser, a crawler or a preview of a link would start the
// copy without the visitor asking for it and would put a load on the server.
//
// POST is a writing method, so the check of the origin protects the route as
// well: a page of another origin cannot start the backup. The check of the
// origin is an additional protection for the writing methods.
//
// The copy is written into a temporary directory of the backup directory and
// it is removed again in every normal path: after the download, after a
// failure and after a visitor who went away. A hard kill of the program
// leaves the directory behind, the next start removes it. The answer starts
// only after the copy is complete, so nobody gets half a database.
func (s *server) backupDownload(w http.ResponseWriter, r *http.Request) {
	if s.backupDir == "" {
		s.logger.Error("the backup is not configured", "path", r.URL.Path)
		s.render(w, http.StatusServiceUnavailable, errorPage, PageData{
			Title: "Backup nicht verfügbar",
			Error: "Das Backup ist nicht eingerichtet.",
		})
		return
	}

	s.setBackupDeadline(w)

	path, cleanup, err := s.writeBackup(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer cleanup()

	s.sendBackup(w, r, path)
}

// setBackupDeadline sets the deadline for the answer of the backup.
//
// The deadline replaces the write timeout of the server, which begins before
// the copy of the database is made. A response writer that does not support a
// deadline, like the recorder of the tests, is not an error: the answer works
// without the deadline as well.
func (s *server) setBackupDeadline(w http.ResponseWriter) {
	deadline := time.Now().Add(backupWriteTimeout)
	err := http.NewResponseController(w).SetWriteDeadline(deadline)
	if err == nil || errors.Is(err, http.ErrNotSupported) {
		return
	}
	s.logger.Warn("cannot set the deadline of the backup", "error", err)
}

// writeBackup writes a copy of the database into a temporary directory of the
// backup directory.
//
// It returns the path of the copy and the function that removes it again. The
// directory has the rights 0700 and the file the rights 0600, so nobody else
// can read the copy of the database.
func (s *server) writeBackup(ctx context.Context) (string, func(), error) {
	dir, err := os.MkdirTemp(s.backupDir, BackupDirPrefix)
	if err != nil {
		return "", nil, fmt.Errorf("web: cannot create the temporary directory in %q: %w", s.backupDir, err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			s.logger.Error("cannot remove the temporary backup", "dir", dir, "error", err)
		}
	}

	path := filepath.Join(dir, backupFileName)
	if err := s.store.VacuumInto(ctx, path); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("web: cannot write the backup: %w", err)
	}
	// SQLite creates the file with the rights of the umask of the process.
	// The copy of the database belongs to the application alone.
	if err := os.Chmod(path, 0o600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("web: cannot set the rights of the backup %q: %w", path, err)
	}
	return path, cleanup, nil
}

// sendBackup sends the copy of the database as a download.
//
// The name of the file carries the time of the backup in the time zone of the
// application, so the visitor can tell the copies apart.
func (s *server) sendBackup(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		s.internalError(w, r, fmt.Errorf("web: cannot open the backup %q: %w", path, err))
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		s.internalError(w, r, fmt.Errorf("web: cannot read the size of the backup: %w", err))
		return
	}

	header := w.Header()
	header.Set("Content-Type", backupContentType)
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.backupName()))
	header.Set("Cache-Control", "no-store")
	header.Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		// The visitor went away or the connection broke. The answer is
		// incomplete, there is nothing to do but to write it into the log.
		s.logger.Error("cannot send the backup", "error", err)
	}
}

// backupName returns the name of the file of the download.
//
// The time comes from the clock of the server and from the time zone of the
// application, so the name carries the local time of the backup.
func (s *server) backupName() string {
	return "easyplan-" + s.now().In(s.location).Format(backupTimeFormat) + ".db"
}

// CleanupBackups removes the temporary directories of the backups that are
// left in the directory dir.
//
// A backup that is killed hard cannot clean up after itself, its temporary
// directory stays in the data directory until the next start.
//
// Only the directories whose name begins with BackupDirPrefix are removed.
// Files and directories with other names stay untouched, and a symbolic link
// is never followed and never removed.
func CleanupBackups(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("web: cannot read the directory %q: %w", dir, err)
	}

	var failures []error
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, BackupDirPrefix) {
			continue
		}
		path := filepath.Join(dir, name)
		// The link is not followed, so a link into another directory cannot
		// make the cleanup remove something outside of the data directory.
		info, err := os.Lstat(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("web: cannot read %q: %w", path, err))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			failures = append(failures, fmt.Errorf("web: cannot remove the directory %q: %w", path, err))
		}
	}
	return errors.Join(failures...)
}
