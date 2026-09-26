// Package intelligence keeps a bounded, local CVE/NVD cache. Source records
// are data, never instructions or evidence that a target is vulnerable.
package intelligence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MaxRecordBytes = 1 << 20
	MaxCacheBytes  = 512 << 20
	MaxPageCount   = 128
)

var (
	ErrInvalid     = errors.New("intelligence_invalid_input")
	ErrUnavailable = errors.New("intelligence_unavailable")
	ErrSource      = errors.New("intelligence_source_invalid")
	ErrLimit       = errors.New("intelligence_limit_exceeded")
	ErrNotFound    = errors.New("intelligence_not_found")
	ErrCorrupt     = errors.New("intelligence_cache_corrupt")
)

var cveIDPattern = regexp.MustCompile(`^CVE-[0-9]{4}-[0-9]{4,}$`)

type Record struct {
	Source, ID, State, SHA256     string
	Published, Modified, Acquired time.Time
	JSON                          []byte
}

type Snapshot struct {
	Source                              string
	WindowStart, Watermark, LastSuccess time.Time
	Records                             int
}

type Cache struct{ db *sql.DB }

// Open requires a trusted local directory. This separate cache contains public
// feed records, never project data, and is not part of project deletion.
func Open(ctx context.Context, path string) (*Cache, error) {
	if ctx == nil || !filepath.IsAbs(path) || filepath.Base(path) == "." {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err == nil && (!info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0) {
		return nil, ErrInvalid
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, ErrUnavailable
	}
	if errors.Is(err, os.ErrNotExist) {
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if e != nil {
			return nil, ErrUnavailable
		}
		if f.Close() != nil {
			return nil, ErrUnavailable
		}
	}
	filePath := filepath.ToSlash(path)
	if filePath[0] != '/' {
		filePath = "/" + filePath
	}
	u := url.URL{Scheme: "file", Path: filePath}
	q := url.Values{}
	q.Set("_foreign_keys", "on")
	q.Set("_busy_timeout", "500")
	q.Set("_synchronous", "FULL")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, ErrUnavailable
	}
	db.SetMaxOpenConns(1)
	c := &Cache{db: db}
	if err := c.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Cache) initialize(ctx context.Context) error {
	var version int
	if err := c.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return ErrUnavailable
	}
	if version != 0 && version != 1 {
		return ErrCorrupt
	}
	if version == 0 {
		var count int
		if err := c.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%'").Scan(&count); err != nil || count != 0 {
			return ErrCorrupt
		}
		tx, err := c.db.BeginTx(ctx, nil)
		if err != nil {
			return ErrUnavailable
		}
		defer tx.Rollback()
		_, err = tx.ExecContext(ctx, `CREATE TABLE records (
			source TEXT NOT NULL, id TEXT NOT NULL, state TEXT NOT NULL,
			published TEXT, modified TEXT NOT NULL, acquired TEXT NOT NULL,
			sha256 TEXT NOT NULL, body BLOB NOT NULL,
			PRIMARY KEY(source,id));
			CREATE TABLE snapshots (source TEXT PRIMARY KEY, window_start TEXT NOT NULL,
				watermark TEXT NOT NULL, last_success TEXT NOT NULL, records INTEGER NOT NULL);
			PRAGMA user_version=1`)
		if err != nil {
			return ErrUnavailable
		}
		if err := tx.Commit(); err != nil {
			return ErrUnavailable
		}
	}
	var integrity string
	if err := c.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil || integrity != "ok" {
		return ErrCorrupt
	}
	var pageSize int
	if err := c.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil || pageSize < 1 {
		return ErrUnavailable
	}
	var applied int
	if err := c.db.QueryRowContext(ctx, "PRAGMA max_page_count="+itoa(MaxCacheBytes/pageSize)).Scan(&applied); err != nil || applied != MaxCacheBytes/pageSize {
		return ErrLimit
	}
	_, _ = c.db.ExecContext(ctx, "PRAGMA journal_size_limit=8388608")
	return nil
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func validRecord(r Record) bool {
	if (r.Source != "nvd" && r.Source != "cve") || !cveIDPattern.MatchString(r.ID) ||
		len(r.State) < 1 || len(r.State) > 64 || len(r.JSON) < 2 || len(r.JSON) > MaxRecordBytes ||
		r.Modified.IsZero() || r.Acquired.IsZero() || !json.Valid(r.JSON) {
		return false
	}
	for _, c := range r.State {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || c == '-' || c == ' ') {
			return false
		}
	}
	return true
}

func (c *Cache) upsert(ctx context.Context, tx *sql.Tx, r Record) error {
	if !validRecord(r) {
		return ErrSource
	}
	sha := sha256.Sum256(r.JSON)
	_, err := tx.ExecContext(ctx, `INSERT INTO records(source,id,state,published,modified,acquired,sha256,body)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(source,id) DO UPDATE SET
		state=excluded.state,published=excluded.published,modified=excluded.modified,
		acquired=excluded.acquired,sha256=excluded.sha256,body=excluded.body`,
		r.Source, r.ID, r.State, nullableTime(r.Published), r.Modified.UTC().Format(time.RFC3339Nano),
		r.Acquired.UTC().Format(time.RFC3339Nano), hex.EncodeToString(sha[:]), r.JSON)
	if err != nil {
		return ErrLimit
	}
	return nil
}

func (c *Cache) Get(ctx context.Context, source, id string) (Record, error) {
	if c == nil || c.db == nil || ctx == nil || (source != "nvd" && source != "cve") || !cveIDPattern.MatchString(id) {
		return Record{}, ErrInvalid
	}
	r := Record{Source: source, ID: id}
	var published sql.NullString
	var modified, acquired string
	err := c.db.QueryRowContext(ctx, `SELECT state,published,modified,acquired,sha256,body FROM records WHERE source=? AND id=?`, source, id).
		Scan(&r.State, &published, &modified, &acquired, &r.SHA256, &r.JSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, ErrUnavailable
	}
	r.Modified, err = time.Parse(time.RFC3339Nano, modified)
	if err != nil {
		return Record{}, ErrCorrupt
	}
	r.Acquired, err = time.Parse(time.RFC3339Nano, acquired)
	if err != nil {
		return Record{}, ErrCorrupt
	}
	if published.Valid {
		r.Published, err = time.Parse(time.RFC3339Nano, published.String)
		if err != nil {
			return Record{}, ErrCorrupt
		}
	}
	h := sha256.Sum256(r.JSON)
	if !validRecord(r) || hex.EncodeToString(h[:]) != r.SHA256 {
		return Record{}, ErrCorrupt
	}
	return r, nil
}

func (c *Cache) Snapshot(ctx context.Context, source string) (Snapshot, error) {
	if c == nil || c.db == nil || ctx == nil || (source != "nvd" && source != "cve") {
		return Snapshot{}, ErrInvalid
	}
	s := Snapshot{Source: source}
	var start, watermark, success string
	err := c.db.QueryRowContext(ctx, `SELECT window_start,watermark,last_success,records FROM snapshots WHERE source=?`, source).Scan(&start, &watermark, &success, &s.Records)
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	s.WindowStart, err = time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return Snapshot{}, ErrCorrupt
	}
	s.Watermark, err = time.Parse(time.RFC3339Nano, watermark)
	if err != nil {
		return Snapshot{}, ErrCorrupt
	}
	s.LastSuccess, err = time.Parse(time.RFC3339Nano, success)
	if err != nil || s.Records < 0 {
		return Snapshot{}, ErrCorrupt
	}
	return s, nil
}

// Status is derived from a successful snapshot, not from the mere existence of
// a database. Offline describes the current mode; stale uses a caller-set TTL.
func (s Snapshot) Status(now time.Time, ttl time.Duration, offline bool) string {
	if s.LastSuccess.IsZero() {
		return "unavailable"
	}
	if offline {
		return "offline"
	}
	if ttl <= 0 || now.Before(s.LastSuccess) || now.Sub(s.LastSuccess) > ttl {
		return "stale"
	}
	return "fresh"
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func itoa(n int) string { return strconv.Itoa(n) }
