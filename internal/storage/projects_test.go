package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

func syntheticProject(t *testing.T, id string, origins []string) project.Project {
	t.Helper()
	p, err := project.New(project.Draft{
		ID: id, Name: "Synthetic caffè ☕", TargetOwner: "Fixture owner",
		AuthorizationReference: "local fixture approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: origins,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func openFixture(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestProjectStoreReopenListAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progetti caffè #1.sqlite")
	s := openFixture(t, path)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("new database is readable beyond owner: mode=%v", info.Mode())
		}
	}
	first := syntheticProject(t, "b-project", []string{"https://lab.invalid", "http://127.0.0.1:8080"})
	second := syntheticProject(t, "a-project", []string{"https://other.invalid"})
	for _, p := range []project.Project{first, second} {
		if err := s.CreateProject(t.Context(), p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateProject(t.Context(), first); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate project should be rejected: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	projects, err := s.ListProjects(t.Context())
	if err != nil || len(projects) != 2 || projects[0].ID() != "a-project" || projects[1].ID() != "b-project" {
		t.Fatalf("reopened project list: count=%d err=%v", len(projects), err)
	}
	loaded, err := s.LoadProject(t.Context(), first.ID())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Record().Name != first.Record().Name || loaded.Record().AuthorizationReference != first.Record().AuthorizationReference ||
		len(loaded.Origins()) != 2 || loaded.Origins()[0] != "https://lab.invalid:443" {
		t.Fatal("project metadata or canonical scope changed after reopen")
	}
	run, err := loaded.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.CheckOrigin("https://lab.invalid/synthetic"); err != nil {
		t.Fatalf("restored scope rejected its own origin: %v", err)
	}
	if err := s.DeleteProject(t.Context(), first.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadProject(t.Context(), first.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted project was loaded: %v", err)
	}
	var childCount int
	if err := s.db.QueryRowContext(t.Context(), "SELECT count(*) FROM authorization_revisions WHERE project_id = ?", first.ID()).Scan(&childCount); err != nil || childCount != 0 {
		t.Fatalf("origins remained after delete: count=%d err=%v", childCount, err)
	}
	if err := s.DeleteProject(t.Context(), first.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete should report absence: %v", err)
	}
	if _, err := s.LoadProject(t.Context(), second.ID()); err != nil {
		t.Fatalf("unrelated project was deleted: %v", err)
	}
}

func TestExpiredProjectRemainsReadableButCannotRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expired.sqlite")
	s := openFixture(t, path)
	p, err := project.Restore(project.Draft{
		ID: "expired", Name: "Synthetic expired", TargetOwner: "Fixture owner",
		AuthorizationReference: "fixture approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(-time.Hour), Origins: []string{"https://lab.invalid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	loaded, err := s.LoadProject(t.Context(), p.ID())
	if err != nil || loaded.Record().AuthorizationReference != "fixture approval" {
		t.Fatalf("expired project disappeared: %v", err)
	}
	if _, err := loaded.BeginRun(); !errors.Is(err, project.ErrAuthorizationExpired) {
		t.Fatalf("restored expired project started a run: %v", err)
	}
}

func TestCreateRollbackAfterOriginFailure(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "rollback.sqlite"))
	_, err := s.db.ExecContext(t.Context(), `CREATE TRIGGER fail_second_origin BEFORE INSERT ON authorization_origins
		WHEN NEW.position = 1 BEGIN SELECT RAISE(ABORT, 'synthetic failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	p := syntheticProject(t, "rollback", []string{"https://first.invalid", "https://second.invalid"})
	if err := s.CreateProject(t.Context(), p); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("second origin failure was not reported: %v", err)
	}
	var count int
	if err := s.db.QueryRowContext(t.Context(), "SELECT count(*) FROM projects WHERE id = ?", p.ID()).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial project survived rollback: count=%d err=%v", count, err)
	}
	if _, err := s.db.ExecContext(t.Context(), "DROP TRIGGER fail_second_origin"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatalf("store did not recover after rollback: %v", err)
	}
}

func TestOpenRejectsUnknownOrForeignSchema(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup string
	}{
		{"future version", "PRAGMA user_version = 99"},
		{"unversioned foreign table", "CREATE TABLE foreign_data (value TEXT)"},
		{"incomplete version one", "PRAGMA user_version = 1"},
		{"missing cascade", `CREATE TABLE projects (
			id TEXT PRIMARY KEY NOT NULL, name TEXT, target_owner TEXT,
			authorization_reference TEXT, authorization_confirmed INTEGER, expires_at TEXT);
			CREATE TABLE project_origins (project_id TEXT, position INTEGER, origin TEXT);
			PRAGMA user_version = 1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "other.sqlite")
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(t.Context(), tc.setup); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" {
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if s, err := Open(t.Context(), path); s != nil || !errors.Is(err, ErrUnsupportedSchema) {
				t.Fatalf("foreign schema accepted: store=%v err=%v", s, err)
			}
		})
	}
}

func TestStoreConnectionSettingsSurviveReplacement(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "settings.sqlite"))
	for i := 0; i < 2; i++ {
		for pragma, want := range map[string]int{"foreign_keys": 1, "trusted_schema": 0, "synchronous": 2, "busy_timeout": 500} {
			var got int
			if err := s.db.QueryRowContext(t.Context(), "PRAGMA "+pragma).Scan(&got); err != nil || got != want {
				t.Fatalf("%s=%d want %d: %v", pragma, got, want, err)
			}
		}
		s.db.SetMaxIdleConns(0)
	}
}

func TestStoreRejectsUnsafeExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs require native filesystem trials")
	}
	path := filepath.Join(t.TempDir(), "public.sqlite")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(t.Context(), path); s != nil || !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("public database accepted: store=%v err=%v", s, err)
	}
	link := filepath.Join(t.TempDir(), "link.sqlite")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(t.Context(), link); s != nil || !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("symlink database accepted: store=%v err=%v", s, err)
	}
}

func TestCorruptProjectRecordIsNeverAuthorized(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "corrupt.sqlite"))
	p := syntheticProject(t, "corrupt", []string{"https://lab.invalid"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "UPDATE authorization_origins SET origin = ? WHERE project_id = ?", "https://outside.invalid/path", p.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadProject(t.Context(), p.ID()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("invalid persisted scope accepted: %v", err)
	}
}

func TestOpenAndCreateHonorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	path := filepath.Join(t.TempDir(), "cancel.sqlite")
	if _, err := Open(ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled open: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled open created a file: %v", err)
	}
	s := openFixture(t, path)
	p := syntheticProject(t, "cancel", []string{"https://lab.invalid"})
	if err := s.CreateProject(ctx, p); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled create: %v", err)
	}
	if _, err := s.LoadProject(t.Context(), p.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("canceled create persisted data: %v", err)
	}
}

func TestRenewalIsAtomicVersionedAndDoesNotWidenRunningScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "renewal.sqlite")
	s := openFixture(t, path)
	old := syntheticProject(t, "renewal", []string{"https://old.invalid"})
	if err := s.CreateProject(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	running, err := old.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	change := project.AuthorizationDraft{
		TargetOwner: "New fixture owner", AuthorizationReference: "renewal-2",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(2 * time.Hour),
		Origins: []string{"https://new.invalid"},
	}
	next, err := s.ReviseAuthorization(t.Context(), old.ID(), 1, change)
	if err != nil || next.Revision() != 2 {
		t.Fatalf("renewal: revision=%d err=%v", next.Revision(), err)
	}
	if _, err := s.ReviseAuthorization(t.Context(), old.ID(), 1, change); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	if _, err := running.CheckOrigin("https://new.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("running revision acquired new origin: %v", err)
	}
	if _, err := running.CheckOrigin("https://old.invalid"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	current, err := s.LoadProject(t.Context(), old.ID())
	if err != nil || current.Revision() != 2 || current.Record().AuthorizationReference != "renewal-2" {
		t.Fatalf("current revision after reopen: %+v %v", current.Record(), err)
	}
	currentRun, err := current.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := currentRun.CheckOrigin("https://new.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := currentRun.CheckOrigin("https://old.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("removed origin still in current scope: %v", err)
	}
	history, err := s.ListAuthorizationRevisions(t.Context(), old.ID())
	if err != nil || len(history) != 2 || history[0].Revision != 1 || history[1].Revision != 2 ||
		history[0].Origins[0] != "https://old.invalid:443" || history[1].Origins[0] != "https://new.invalid:443" {
		t.Fatalf("revision history: %+v %v", history, err)
	}
	if err := s.DeleteProject(t.Context(), old.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListAuthorizationRevisions(t.Context(), old.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted history remained: %v", err)
	}
}

func TestExpiredAuthorizationCanBeRenewedButNotExtendedWithoutConfirmation(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "expired-renewal.sqlite"))
	p, err := project.Restore(project.Draft{
		ID: "expired-renewal", Name: "Expired fixture", TargetOwner: "Fixture owner",
		AuthorizationReference: "old approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(-time.Hour), Origins: []string{"https://old.invalid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	change := project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "new approval",
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{"https://new.invalid"},
	}
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, change); !errors.Is(err, project.ErrAuthorizationMissing) {
		t.Fatalf("unconfirmed renewal accepted: %v", err)
	}
	change.AuthorizationConfirmed = true
	change.AuthorizationExpiresAt = time.Now().Add(-time.Hour)
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, change); !errors.Is(err, project.ErrAuthorizationExpired) {
		t.Fatalf("already expired renewal accepted: %v", err)
	}
	change.AuthorizationExpiresAt = time.Now().Add(time.Hour)
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, change); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadProject(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loaded.BeginRun(); err != nil {
		t.Fatalf("renewed project cannot run: %v", err)
	}
}

func TestRevisionRollbackWhenOriginWriteFails(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "revision-rollback.sqlite"))
	p := syntheticProject(t, "revision-rollback", []string{"https://old.invalid"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), `CREATE TRIGGER fail_revised_origin BEFORE INSERT ON authorization_origins
		WHEN NEW.revision = 2 AND NEW.position = 1 BEGIN SELECT RAISE(ABORT, 'synthetic failure'); END`); err != nil {
		t.Fatal(err)
	}
	change := project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "new approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{"https://first.invalid", "https://second.invalid"},
	}
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, change); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("origin write failure: %v", err)
	}
	current, err := s.LoadProject(t.Context(), p.ID())
	if err != nil || current.Revision() != 1 {
		t.Fatalf("partial revision survived: %d %v", current.Revision(), err)
	}
	var count int
	if err := s.db.QueryRowContext(t.Context(), `SELECT count(*) FROM authorization_revisions WHERE project_id = ?`, p.ID()).Scan(&count); err != nil || count != 1 {
		t.Fatalf("partial history survived: count=%d err=%v", count, err)
	}
}

func TestV1ProjectStoreMigratesToRevisionOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE projects (
		id TEXT PRIMARY KEY NOT NULL, name TEXT NOT NULL, target_owner TEXT NOT NULL,
		authorization_reference TEXT NOT NULL, authorization_confirmed INTEGER NOT NULL CHECK (authorization_confirmed = 1),
		expires_at TEXT NOT NULL);
		CREATE TABLE project_origins (
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		position INTEGER NOT NULL CHECK (position >= 0 AND position < 32), origin TEXT NOT NULL,
		PRIMARY KEY (project_id, position), UNIQUE (project_id, origin));
		INSERT INTO projects VALUES ('migrated', 'Migrated fixture', 'Fixture owner', 'v1 approval', 1, '2027-01-01T00:00:00Z');
		INSERT INTO project_origins VALUES ('migrated', 0, 'https://lab.invalid:443');
		PRAGMA user_version = 1`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := openFixture(t, path)
	loaded, err := s.LoadProject(t.Context(), "migrated")
	if err != nil || loaded.Revision() != 1 || loaded.Record().AuthorizationReference != "v1 approval" {
		t.Fatalf("migrated project: %+v %v", loaded.Record(), err)
	}
	var version int
	if err := s.db.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("schema version after migration: %d %v", version, err)
	}
	history, err := s.ListAuthorizationRevisions(t.Context(), "migrated")
	if err != nil || len(history) != 1 || history[0].Origins[0] != "https://lab.invalid:443" {
		t.Fatalf("migrated history: %+v %v", history, err)
	}
}

func TestV2ProjectStoreMigratesToRevocationState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE projects (
		id TEXT PRIMARY KEY NOT NULL, name TEXT NOT NULL,
		current_revision INTEGER NOT NULL CHECK (current_revision >= 1));
		CREATE TABLE authorization_revisions (
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		revision INTEGER NOT NULL CHECK (revision >= 1), target_owner TEXT NOT NULL,
		authorization_reference TEXT NOT NULL,
		authorization_confirmed INTEGER NOT NULL CHECK (authorization_confirmed = 1),
		expires_at TEXT NOT NULL, PRIMARY KEY (project_id, revision));
		CREATE TABLE authorization_origins (
		project_id TEXT NOT NULL, revision INTEGER NOT NULL,
		position INTEGER NOT NULL CHECK (position >= 0 AND position < 32), origin TEXT NOT NULL,
		PRIMARY KEY (project_id, revision, position), UNIQUE (project_id, revision, origin),
		FOREIGN KEY (project_id, revision) REFERENCES authorization_revisions(project_id, revision) ON DELETE CASCADE);
		INSERT INTO projects VALUES ('migrated-v2', 'Migrated fixture', 2);
		INSERT INTO authorization_revisions VALUES ('migrated-v2', 1, 'Fixture owner', 'v2 approval', 1, '2100-01-01T00:00:00Z');
		INSERT INTO authorization_revisions VALUES ('migrated-v2', 2, 'Fixture owner', 'v2 renewal', 1, '2101-01-01T00:00:00Z');
		INSERT INTO authorization_origins VALUES ('migrated-v2', 1, 0, 'https://lab.invalid:443');
		INSERT INTO authorization_origins VALUES ('migrated-v2', 2, 0, 'https://renewed.invalid:443');
		PRAGMA user_version = 2`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := openFixture(t, path)
	loaded, err := s.LoadProject(t.Context(), "migrated-v2")
	if err != nil || loaded.Revoked() || loaded.Revision() != 2 || loaded.Origins()[0] != "https://renewed.invalid:443" {
		t.Fatalf("v2 migration: revision=%d revoked=%v err=%v", loaded.Revision(), loaded.Revoked(), err)
	}
	history, err := s.ListAuthorizationRevisions(t.Context(), "migrated-v2")
	if err != nil || len(history) != 2 || history[0].Revoked || history[1].Revoked ||
		history[0].Origins[0] != "https://lab.invalid:443" || history[1].Origins[0] != "https://renewed.invalid:443" {
		t.Fatalf("v2 history after migration: %+v %v", history, err)
	}
	var version int
	if err := s.db.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("schema version after v2 migration: %d %v", version, err)
	}
	if _, err := s.RevokeAuthorization(t.Context(), "migrated-v2", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BeginRun(t.Context(), "migrated-v2"); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("migrated project revocation ineffective: %v", err)
	}
}

func TestRevisionPointerRollbackIsRejected(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "pointer.sqlite"))
	p := syntheticProject(t, "pointer", []string{"https://old.invalid"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	_, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "new approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{"https://new.invalid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), `UPDATE projects SET current_revision = 1 WHERE id = ?`, p.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadProject(t.Context(), p.ID()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("rolled-back pointer accepted: %v", err)
	}
}
