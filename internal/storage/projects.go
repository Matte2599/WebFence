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
const schemaVersion = 1

// Errors are stable codes and never include a database path or project data.
var (
	ErrInvalidPath       = errors.New("storage_invalid_path")
	ErrUnavailable       = errors.New("storage_unavailable")
	ErrUnsupportedSchema = errors.New("storage_unsupported_schema")
	ErrCorrupt           = errors.New("storage_corrupt_data")
	ErrAlreadyExists     = errors.New("storage_project_exists")
	ErrNotFound          = errors.New("storage_project_not_found")
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

// Open creates a private database file if absent, initializes schema v1, and
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
	case schemaVersion:
		if err := s.checkTables(ctx); err != nil {
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
			target_owner TEXT NOT NULL,
			authorization_reference TEXT NOT NULL,
			authorization_confirmed INTEGER NOT NULL CHECK (authorization_confirmed = 1),
			expires_at TEXT NOT NULL
		);
		CREATE TABLE project_origins (
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			position INTEGER NOT NULL CHECK (position >= 0 AND position < 32),
			origin TEXT NOT NULL,
			PRIMARY KEY (project_id, position),
			UNIQUE (project_id, origin)
		);
		PRAGMA user_version = 1`); err != nil {
			return storageError(ctx, err)
		}
	case schemaVersion:
	default:
		return ErrUnsupportedSchema
	}
	if err := tx.Commit(); err != nil {
		return storageError(ctx, err)
	}
	if err := s.checkTables(ctx); err != nil {
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

func (s *Store) checkTables(ctx context.Context) error {
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

// CreateProject atomically records one immutable project and its origins.
// A later authorization renewal will require a separate versioned contract.
func (s *Store) CreateProject(ctx context.Context, p project.Project) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	d := p.Record()
	if d.ID == "" {
		return project.ErrInvalidProject
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storageError(ctx, err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO projects
		(id, name, target_owner, authorization_reference, authorization_confirmed, expires_at)
		VALUES (?, ?, ?, ?, 1, ?)`, d.ID, d.Name, d.TargetOwner, d.AuthorizationReference,
		d.AuthorizationExpiresAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrAlreadyExists
		}
		return storageError(ctx, err)
	}
	for i, origin := range d.Origins {
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_origins (project_id, position, origin)
			VALUES (?, ?, ?)`, d.ID, i, origin); err != nil {
			return storageError(ctx, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return storageError(ctx, err)
	}
	return nil
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
	d := project.Draft{ID: id}
	var expiry string
	var confirmed int
	err := tx.QueryRowContext(ctx, `SELECT name, target_owner, authorization_reference,
		authorization_confirmed, expires_at FROM projects WHERE id = ?`, id).
		Scan(&d.Name, &d.TargetOwner, &d.AuthorizationReference, &confirmed, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Project{}, ErrNotFound
	}
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	if confirmed != 1 {
		return project.Project{}, ErrCorrupt
	}
	d.AuthorizationConfirmed = true
	d.AuthorizationExpiresAt, err = time.Parse(time.RFC3339Nano, expiry)
	if err != nil {
		return project.Project{}, ErrCorrupt
	}
	rows, err := tx.QueryContext(ctx, `SELECT origin FROM project_origins WHERE project_id = ? ORDER BY position`, id)
	if err != nil {
		return project.Project{}, storageError(ctx, err)
	}
	for rows.Next() {
		var origin string
		if err := rows.Scan(&origin); err != nil {
			_ = rows.Close()
			return project.Project{}, storageError(ctx, err)
		}
		d.Origins = append(d.Origins, origin)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return project.Project{}, storageError(ctx, err)
	}
	p, err := project.Restore(d)
	if err != nil {
		return project.Project{}, ErrCorrupt
	}
	return p, nil
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

// DeleteProject removes only this schema's project metadata and origins.
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
