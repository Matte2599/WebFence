// Package storage persists the M1 project metadata in a local SQLite database.
// It does not store credentials, scan results, evidence or report artifacts.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/mattn/go-sqlite3"
)

const driverName = "webfence-project-store"
const schemaVersion = 2

// Errors are stable codes and never include a database path or project data.
var (
	ErrInvalidPath       = errors.New("storage_invalid_path")
	ErrUnavailable       = errors.New("storage_unavailable")
	ErrUnsupportedSchema = errors.New("storage_unsupported_schema")
	ErrCorrupt           = errors.New("storage_corrupt_data")
	ErrAlreadyExists     = errors.New("storage_project_exists")
	ErrNotFound          = errors.New("storage_project_not_found")
	ErrRevisionConflict  = errors.New("storage_revision_conflict")
)

func init() {
	sql.Register(driverName, &sqlite3.SQLiteDriver{ConnectHook: func(c *sqlite3.SQLiteConn) error {
		_, err := c.Exec("PRAGMA trusted_schema=OFF", nil)
		return err
	}})
}

// Store owns one connection to a caller-selected local database. The caller
// must keep the file and its parent directory on a trusted local filesystem.
type Store struct{ db *sql.DB }

// Open creates a private database file if absent, initializes schema v2, and
// refuses an unknown future schema or an existing non-WebFence database.
// The parent directory must already exist; this function never chooses it.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Base(path) == "." {
		return nil, ErrInvalidPath
	}
	if err := prepareFile(path); err != nil {
		return nil, err
	}
	filePath := filepath.ToSlash(path)
	if !strings.HasPrefix(filePath, "/") { // file:///C:/... on Windows
		filePath = "/" + filePath
	}
	u := url.URL{Scheme: "file", Path: filePath}
	q := url.Values{}
	q.Set("_foreign_keys", "on")
	q.Set("_synchronous", "FULL")
	q.Set("_busy_timeout", "500")
	u.RawQuery = q.Encode()
	db, err := sql.Open(driverName, u.String())
	if err != nil {
		return nil, ErrUnavailable
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, storageError(ctx, err)
	}
	s := &Store{db: db}
	if err := s.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func prepareFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		f, createErr := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if createErr == nil {
			if closeErr := f.Close(); closeErr != nil {
				return ErrUnavailable
			}
			return nil
		}
		if !errors.Is(createErr, os.ErrExist) {
			return ErrUnavailable
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return ErrUnavailable
	}
	if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return ErrInvalidPath
	}
	return nil
}

func (s *Store) initialize(ctx context.Context) error {
	for name, want := range map[string]int{"foreign_keys": 1, "trusted_schema": 0, "synchronous": 2} {
		var got int
		if err := s.db.QueryRowContext(ctx, "PRAGMA "+name).Scan(&got); err != nil {
			return storageError(ctx, err)
		}
		if got != want {
			return ErrUnavailable
		}
	}
	// Inspect before changing the journal mode so a newer or unrelated database
	// is not altered merely by attempting to open it.
	var initialVersion int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&initialVersion); err != nil {
		return storageError(ctx, err)
	}
	switch initialVersion {
	case 0:
		var objects int
		if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%'").Scan(&objects); err != nil {
			return storageError(ctx, err)
		}
		if objects != 0 {
			return ErrUnsupportedSchema
		}
	case 1:
		if err := s.checkTablesV1(ctx); err != nil {
			return err
		}
	case schemaVersion:
		if err := s.checkTablesV2(ctx); err != nil {
			return err
		}
	default:
		return ErrUnsupportedSchema
	}
	var journal string
	if err := s.db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journal); err != nil {
		return storageError(ctx, err)
	}
	if journal != "wal" {
		return ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storageError(ctx, err)
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return storageError(ctx, err)
	}
	switch version {
	case 0:
		var objects int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%'").Scan(&objects); err != nil {
			return storageError(ctx, err)
		}
		if objects != 0 {
			return ErrUnsupportedSchema
		}
		if _, err := tx.ExecContext(ctx, `CREATE TABLE projects (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			current_revision INTEGER NOT NULL CHECK (current_revision >= 1)
		);
		CREATE TABLE authorization_revisions (
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			revision INTEGER NOT NULL CHECK (revision >= 1),
			target_owner TEXT NOT NULL,
			authorization_reference TEXT NOT NULL,
			authorization_confirmed INTEGER NOT NULL CHECK (authorization_confirmed = 1),
			expires_at TEXT NOT NULL,
			PRIMARY KEY (project_id, revision)
		);
		CREATE TABLE authorization_origins (
			project_id TEXT NOT NULL,
			revision INTEGER NOT NULL,
			position INTEGER NOT NULL CHECK (position >= 0 AND position < 32),
			origin TEXT NOT NULL,
			PRIMARY KEY (project_id, revision, position),
			UNIQUE (project_id, revision, origin),
			FOREIGN KEY (project_id, revision) REFERENCES authorization_revisions(project_id, revision) ON DELETE CASCADE
		);
		PRAGMA user_version = 2`); err != nil {
			return storageError(ctx, err)
		}
	case 1:
		// DDL and data copy are one transaction: failure preserves readable v1.
		if _, err := tx.ExecContext(ctx, `ALTER TABLE projects RENAME TO projects_legacy;
		CREATE TABLE projects (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			current_revision INTEGER NOT NULL CHECK (current_revision >= 1)
		);
		CREATE TABLE authorization_revisions (
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			revision INTEGER NOT NULL CHECK (revision >= 1),
			target_owner TEXT NOT NULL,
			authorization_reference TEXT NOT NULL,
			authorization_confirmed INTEGER NOT NULL CHECK (authorization_confirmed = 1),
			expires_at TEXT NOT NULL,
			PRIMARY KEY (project_id, revision)
		);
		CREATE TABLE authorization_origins (
			project_id TEXT NOT NULL,
			revision INTEGER NOT NULL,
			position INTEGER NOT NULL CHECK (position >= 0 AND position < 32),
			origin TEXT NOT NULL,
			PRIMARY KEY (project_id, revision, position),
			UNIQUE (project_id, revision, origin),
			FOREIGN KEY (project_id, revision) REFERENCES authorization_revisions(project_id, revision) ON DELETE CASCADE
		);
		INSERT INTO projects (id, name, current_revision)
			SELECT id, name, 1 FROM projects_legacy;
		INSERT INTO authorization_revisions
			(project_id, revision, target_owner, authorization_reference, authorization_confirmed, expires_at)
			SELECT id, 1, target_owner, authorization_reference, authorization_confirmed, expires_at FROM projects_legacy;
		INSERT INTO authorization_origins (project_id, revision, position, origin)
			SELECT project_id, 1, position, origin FROM project_origins;
		DROP TABLE project_origins;
		DROP TABLE projects_legacy;
		PRAGMA user_version = 2`); err != nil {
			return storageError(ctx, err)
		}
	case schemaVersion:
	default:
		return ErrUnsupportedSchema
	}
	if err := tx.Commit(); err != nil {
		return storageError(ctx, err)
	}
	if err := s.checkTablesV2(ctx); err != nil {
		return err
	}
	var integrity string
	if err := s.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return storageError(ctx, err)
	}
	if integrity != "ok" {
		return ErrCorrupt
	}
	rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return storageError(ctx, err)
	}
	broken := rows.Next()
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return storageError(ctx, err)
	}
	if broken {
		return ErrCorrupt
	}
	return nil
}

func (s *Store) checkTablesV1(ctx context.Context) error {
	for _, query := range []string{
		"SELECT id, name, target_owner, authorization_reference, authorization_confirmed, expires_at FROM projects LIMIT 0",
		"SELECT project_id, position, origin FROM project_origins LIMIT 0",
	} {
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return ErrUnsupportedSchema
		}
		if err := rows.Close(); err != nil {
			return storageError(ctx, err)
		}
	}
	rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_list(project_origins)")
	if err != nil {
		return storageError(ctx, err)
	}
	var count int
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			_ = rows.Close()
			return storageError(ctx, err)
		}
		if table != "projects" || from != "project_id" || to != "id" || onDelete != "CASCADE" {
			_ = rows.Close()
			return ErrUnsupportedSchema
		}
		count++
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return storageError(ctx, err)
	}
	if count != 1 {
		return ErrUnsupportedSchema
	}
	return nil
}

func (s *Store) checkTablesV2(ctx context.Context) error {
	for _, query := range []string{
		"SELECT id, name, current_revision FROM projects LIMIT 0",
		"SELECT project_id, revision, target_owner, authorization_reference, authorization_confirmed, expires_at FROM authorization_revisions LIMIT 0",
		"SELECT project_id, revision, position, origin FROM authorization_origins LIMIT 0",
	} {
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return ErrUnsupportedSchema
		}
		if err := rows.Close(); err != nil {
			return storageError(ctx, err)
		}
	}
	for _, table := range []string{"authorization_revisions", "authorization_origins"} {
		rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_list("+table+")")
		if err != nil {
			return storageError(ctx, err)
		}
		var count int
		for rows.Next() {
			var id, seq int
			var parent, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &parent, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				_ = rows.Close()
				return storageError(ctx, err)
			}
			if onDelete != "CASCADE" || table == "authorization_revisions" && (parent != "projects" || from != "project_id" || to != "id") ||
				table == "authorization_origins" && (parent != "authorization_revisions" || from != "project_id" && from != "revision") {
				_ = rows.Close()
				return ErrUnsupportedSchema
			}
			count++
		}
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil || closeErr != nil {
			return storageError(ctx, err)
		}
		want := 1
		if table == "authorization_origins" {
			want = 2
		}
		if count != want {
			return ErrUnsupportedSchema
		}
	}
	return nil
}

func storageError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return ErrUnavailable
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	if err := s.db.Close(); err != nil {
		return ErrUnavailable
	}
	return nil
}

// CreateProject atomically records revision 1 and its origins.
func (s *Store) CreateProject(ctx context.Context, p project.Project) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	d := p.Record()
	if d.ID == "" || p.Revision() != 1 {
		return project.ErrInvalidProject
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storageError(ctx, err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO projects (id, name, current_revision)
		VALUES (?, ?, 1)`, d.ID, d.Name)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrAlreadyExists
		}
		return storageError(ctx, err)
	}
	if err := insertRevision(ctx, tx, p); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return storageError(ctx, err)
	}
	return nil
}

func insertRevision(ctx context.Context, tx *sql.Tx, p project.Project) error {
	d := p.Record()
	if _, err := tx.ExecContext(ctx, `INSERT INTO authorization_revisions
		(project_id, revision, target_owner, authorization_reference, authorization_confirmed, expires_at)
		VALUES (?, ?, ?, ?, 1, ?)`, d.ID, p.Revision(), d.TargetOwner,
		d.AuthorizationReference, d.AuthorizationExpiresAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return storageError(ctx, err)
	}
	for i, origin := range d.Origins {
		if _, err := tx.ExecContext(ctx, `INSERT INTO authorization_origins (project_id, revision, position, origin)
			VALUES (?, ?, ?, ?)`, d.ID, p.Revision(), i, origin); err != nil {
			return storageError(ctx, err)
		}
	}
	return nil
}

// ReviseAuthorization atomically stores a fresh declaration and moves the
// current pointer only when the caller saw the expected revision. An expired
// project can be renewed, but the new declaration itself must be unexpired.
func (s *Store) ReviseAuthorization(ctx context.Context, id string, expectedRevision uint64, change project.AuthorizationDraft) (project.Project, error) {
	if s == nil || s.db == nil {
		return project.Project{}, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	defer tx.Rollback()
	current, err := loadProject(ctx, tx, id)
	if err != nil {
		return project.Project{}, err
	}
	if current.Revision() != expectedRevision {
		return project.Project{}, ErrRevisionConflict
	}
	next, err := current.ReviseAuthorization(change)
	if err != nil {
		return project.Project{}, err
	}
	if err := insertRevision(ctx, tx, next); err != nil {
		return project.Project{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE projects SET current_revision = ?
		WHERE id = ? AND current_revision = ?`, next.Revision(), id, expectedRevision)
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	if count != 1 {
		return project.Project{}, ErrRevisionConflict
	}
	if err := tx.Commit(); err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	return next, nil
}

// LoadProject returns a complete immutable project even after authorization
// expiry. The caller must call BeginRun to recheck authorization before use.
func (s *Store) LoadProject(ctx context.Context, id string) (project.Project, error) {
	if s == nil || s.db == nil {
		return project.Project{}, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	defer tx.Rollback()
	p, err := loadProject(ctx, tx, id)
	if err != nil {
		return project.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	return p, nil
}

func loadProject(ctx context.Context, tx *sql.Tx, id string) (project.Project, error) {
	var name string
	var revision int64
	err := tx.QueryRowContext(ctx, `SELECT name, current_revision FROM projects WHERE id = ?`, id).
		Scan(&name, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Project{}, ErrNotFound
	}
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	if revision < 1 {
		return project.Project{}, ErrCorrupt
	}
	var latest sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT max(revision) FROM authorization_revisions WHERE project_id = ?`, id).Scan(&latest); err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	if !latest.Valid || latest.Int64 != revision {
		return project.Project{}, ErrCorrupt
	}
	d, err := loadRevision(ctx, tx, id, uint64(revision))
	if errors.Is(err, ErrNotFound) {
		return project.Project{}, ErrCorrupt
	}
	if err != nil {
		return project.Project{}, err
	}
	d.Name = name
	p, err := project.RestoreRevision(d, uint64(revision))
	if err != nil {
		return project.Project{}, ErrCorrupt
	}
	return p, nil
}

func loadRevision(ctx context.Context, tx *sql.Tx, id string, revision uint64) (project.Draft, error) {
	d := project.Draft{ID: id}
	var expiry string
	var confirmed int
	err := tx.QueryRowContext(ctx, `SELECT target_owner, authorization_reference,
		authorization_confirmed, expires_at FROM authorization_revisions
		WHERE project_id = ? AND revision = ?`, id, revision).
		Scan(&d.TargetOwner, &d.AuthorizationReference, &confirmed, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Draft{}, ErrNotFound
	}
	if err != nil {
		return project.Draft{}, storageError(ctx, err)
	}
	if confirmed != 1 {
		return project.Draft{}, ErrCorrupt
	}
	d.AuthorizationConfirmed = true
	d.AuthorizationExpiresAt, err = time.Parse(time.RFC3339Nano, expiry)
	if err != nil {
		return project.Draft{}, ErrCorrupt
	}
	rows, err := tx.QueryContext(ctx, `SELECT origin FROM authorization_origins
		WHERE project_id = ? AND revision = ? ORDER BY position`, id, revision)
	if err != nil {
		return project.Draft{}, storageError(ctx, err)
	}
	for rows.Next() {
		var origin string
		if err := rows.Scan(&origin); err != nil {
			_ = rows.Close()
			return project.Draft{}, storageError(ctx, err)
		}
		d.Origins = append(d.Origins, origin)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return project.Draft{}, storageError(ctx, err)
	}
	return d, nil
}

// AuthorizationRevision is descriptive audit data, not a runnable scope.
type AuthorizationRevision struct {
	Revision               uint64
	TargetOwner            string
	AuthorizationReference string
	ExpiresAt              time.Time
	Origins                []string
}

// ListAuthorizationRevisions returns validated history in revision order.
// Only LoadProject returns the current revision for starting a new run.
func (s *Store) ListAuthorizationRevisions(ctx context.Context, id string) ([]AuthorizationRevision, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, storageError(ctx, err)
	}
	defer tx.Rollback()
	var name string
	var currentRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT name, current_revision FROM projects WHERE id = ?`, id).Scan(&name, &currentRevision); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, storageError(ctx, err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT revision FROM authorization_revisions WHERE project_id = ? ORDER BY revision`, id)
	if err != nil {
		return nil, storageError(ctx, err)
	}
	var revisions []uint64
	for rows.Next() {
		var revision int64
		if err := rows.Scan(&revision); err != nil {
			_ = rows.Close()
			return nil, storageError(ctx, err)
		}
		if revision < 1 {
			_ = rows.Close()
			return nil, ErrCorrupt
		}
		revisions = append(revisions, uint64(revision))
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return nil, storageError(ctx, err)
	}
	if len(revisions) == 0 {
		return nil, ErrCorrupt
	}
	if revisions[len(revisions)-1] != uint64(currentRevision) {
		return nil, ErrCorrupt
	}
	for i, revision := range revisions {
		if revision != uint64(i+1) {
			return nil, ErrCorrupt
		}
	}
	result := make([]AuthorizationRevision, 0, len(revisions))
	for _, revision := range revisions {
		d, err := loadRevision(ctx, tx, id, revision)
		if err != nil {
			return nil, err
		}
		d.Name = name
		p, err := project.RestoreRevision(d, revision)
		if err != nil {
			return nil, ErrCorrupt
		}
		result = append(result, AuthorizationRevision{
			Revision: revision, TargetOwner: d.TargetOwner,
			AuthorizationReference: d.AuthorizationReference,
			ExpiresAt:              d.AuthorizationExpiresAt, Origins: p.Origins(),
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, storageError(ctx, err)
	}
	return result, nil
}

// ListProjects reads one consistent snapshot in ID order.
func (s *Store) ListProjects(ctx context.Context) ([]project.Project, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, storageError(ctx, err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id FROM projects ORDER BY id")
	if err != nil {
		return nil, storageError(ctx, err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, storageError(ctx, err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return nil, storageError(ctx, err)
	}
	projects := make([]project.Project, 0, len(ids))
	for _, id := range ids {
		p, err := loadProject(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	if err := tx.Commit(); err != nil {
		return nil, storageError(ctx, err)
	}
	return projects, nil
}

// DeleteProject removes this schema's project metadata and revision history.
// It is not secure erasure of SQLite pages, WAL, backups or future artifacts.
func (s *Store) DeleteProject(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	result, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return storageError(ctx, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return storageError(ctx, err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
