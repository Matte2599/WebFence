// Package foundation holds executable M0 dependency feasibility experiments.
// It does not implement the future project store or a persistent SQL schema.
package foundation

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
)

// Driver hooks apply connection-local invariants even after pool replacement.
func init() {
	sql.Register("webfence-sqlite-lab", &sqlite3.SQLiteDriver{ConnectHook: func(c *sqlite3.SQLiteConn) error {
		_, err := c.Exec("PRAGMA trusted_schema=OFF", nil)
		return err
	}})
}

func openDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path}
	u.RawQuery = "_foreign_keys=on&_journal_mode=WAL&_synchronous=FULL&_busy_timeout=50"
	db, err := sql.Open("webfence-sqlite-lab", u.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.PingContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	return db
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteTransactionsAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lab caffè #%&.sqlite")
	db := openDB(t, path)
	var version string
	if err := db.QueryRowContext(t.Context(), "SELECT sqlite_version()").Scan(&version); err != nil {
		t.Fatal(err)
	}
	t.Logf("SQLite runtime: %s", version)
	execSQL(t, db, "CREATE TABLE parent(id INTEGER PRIMARY KEY, value TEXT NOT NULL); CREATE TABLE child(parent_id INTEGER REFERENCES parent(id))")
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	value := "caffè ☕ '; DROP TABLE parent; --"
	if _, err = tx.ExecContext(t.Context(), "INSERT INTO parent VALUES(?,?)", 1, value); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	if _, err = tx.ExecContext(t.Context(), "INSERT INTO parent VALUES(?,?)", 2, "rollback"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db = openDB(t, path)
	var got string
	var count int
	if err = db.QueryRowContext(t.Context(), "SELECT value FROM parent WHERE id=?", 1).Scan(&got); err != nil || got != value {
		t.Fatalf("reopen: %v", err)
	}
	if err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM parent").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rollback count=%d: %v", count, err)
	}
	if _, err = db.ExecContext(t.Context(), "INSERT INTO child VALUES(?)", 999); err == nil {
		t.Fatal("foreign key accepted")
	}
	var integrity string
	if err = db.QueryRowContext(t.Context(), "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity: %v", err)
	}
}

func TestSQLiteConnectionSettingsAndLockBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lab.sqlite")
	db := openDB(t, path)
	for i := 0; i < 2; i++ {
		for pragma, want := range map[string]int{"foreign_keys": 1, "trusted_schema": 0, "synchronous": 2, "busy_timeout": 50} {
			var got int
			if err := db.QueryRowContext(t.Context(), "PRAGMA "+pragma).Scan(&got); err != nil || got != want {
				t.Fatalf("%s=%d want %d: %v", pragma, got, want, err)
			}
		}
		var mode string
		if err := db.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
			t.Fatalf("journal mode=%s: %v", mode, err)
		}
		db.SetMaxIdleConns(0) // next iteration must use newly initialized connections
	}
	execSQL(t, db, "CREATE TABLE fixture(id INTEGER PRIMARY KEY)")
	other := openDB(t, path)
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(t.Context(), "INSERT INTO fixture VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = other.QueryRowContext(t.Context(), "SELECT count(*) FROM fixture").Scan(&count); err != nil || count != 0 {
		t.Fatalf("reader sees uncommitted row: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_, err = other.ExecContext(ctx, "INSERT INTO fixture VALUES(2)")
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code != sqlite3.ErrBusy {
		t.Fatalf("expected bounded busy: %v", err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	execSQL(t, other, "INSERT INTO fixture VALUES(2)")
}

func TestSQLiteCancellationAndRecovery(t *testing.T) {
	db := openDB(t, filepath.Join(t.TempDir(), "lab.sqlite"))
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	var result int64
	err := db.QueryRowContext(ctx, `WITH RECURSIVE counter(x) AS (VALUES(0) UNION ALL SELECT x+1 FROM counter WHERE x<1000000000) SELECT sum(x) FROM counter`).Scan(&result)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("query cancellation: %v", err)
	}
	if err = db.QueryRowContext(t.Context(), "SELECT 42").Scan(&result); err != nil || result != 42 {
		t.Fatalf("connection recovery: %v", err)
	}
	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	if _, err = db.Conn(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("pool cancellation: %v", err)
	}
}
