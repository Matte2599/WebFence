package intelligence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"
)

const nvdEndpoint = "https://services.nvd.nist.gov/rest/json/cves/2.0"
const maxResponseBytes = 16 << 20

// NVDClient is deliberately bound to the official HTTPS endpoint in normal
// use. An injected loopback URL is only accepted for synthetic tests.
type NVDClient struct {
	HTTP        *http.Client
	APIKey      string
	Endpoint    string
	MinInterval time.Duration
	lastRequest time.Time
}

func (a *NVDClient) endpoint() (string, error) {
	if a.Endpoint == "" {
		return nvdEndpoint, nil
	}
	u, e := url.Parse(a.Endpoint)
	if e != nil || u == nil {
		return "", ErrInvalid
	}
	ip, ipErr := netip.ParseAddr(u.Hostname())
	if u.Scheme != "http" || ipErr != nil || !ip.IsLoopback() || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", ErrInvalid
	}
	return a.Endpoint, nil
}

func (a *NVDClient) client() *http.Client {
	if a.HTTP != nil {
		return a.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

type nvdPage struct {
	Format          string `json:"format"`
	Version         string `json:"version"`
	ResultsPerPage  int    `json:"resultsPerPage"`
	StartIndex      int    `json:"startIndex"`
	TotalResults    int    `json:"totalResults"`
	Vulnerabilities []struct {
		CVE json.RawMessage `json:"cve"`
	} `json:"vulnerabilities"`
}
type nvdRecord struct {
	ID        string `json:"id"`
	Status    string `json:"vulnStatus"`
	Published string `json:"published"`
	Modified  string `json:"lastModified"`
}

func parseSourceTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000", "2006-01-02T15:04:05"} {
		if t, e := time.Parse(layout, value); e == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, ErrSource
}

func (a *NVDClient) page(ctx context.Context, start, end time.Time, index int) ([]Record, int, error) {
	endpoint, err := a.endpoint()
	if err != nil {
		return nil, 0, err
	}
	interval := a.MinInterval
	if interval == 0 && a.Endpoint == "" {
		if a.APIKey == "" {
			interval = 6 * time.Second
		} else {
			interval = 700 * time.Millisecond
		}
	}
	if interval < 0 {
		return nil, 0, ErrInvalid
	}
	if wait := interval - time.Since(a.lastRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-timer.C:
		}
	}
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("lastModStartDate", start.UTC().Format("2006-01-02T15:04:05.000"))
	q.Set("lastModEndDate", end.UTC().Format("2006-01-02T15:04:05.000"))
	q.Set("resultsPerPage", "500")
	q.Set("startIndex", strconv.Itoa(index))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, ErrInvalid
	}
	req.Header.Set("Accept", "application/json")
	if a.APIKey != "" {
		req.Header.Set("apiKey", a.APIKey)
	}
	a.lastRequest = time.Now()
	data, err := sourceGET(ctx, a.client(), req, maxResponseBytes)
	if err != nil {
		return nil, 0, err
	}
	var p nvdPage
	if json.Unmarshal(data, &p) != nil || p.Format != "NVD_CVE" || p.Version != "2.0" || p.StartIndex != index || p.TotalResults < 0 || p.TotalResults > MaxPageCount*500 || p.ResultsPerPage < 1 || p.ResultsPerPage > 500 || len(p.Vulnerabilities) > p.ResultsPerPage || (index < p.TotalResults && len(p.Vulnerabilities) == 0) {
		return nil, 0, ErrSource
	}
	now := time.Now().UTC()
	records := make([]Record, 0, len(p.Vulnerabilities))
	for _, v := range p.Vulnerabilities {
		var parsed nvdRecord
		if json.Unmarshal(v.CVE, &parsed) != nil {
			return nil, 0, ErrSource
		}
		published, e := parseSourceTime(parsed.Published)
		if e != nil {
			return nil, 0, e
		}
		modified, e := parseSourceTime(parsed.Modified)
		if e != nil {
			return nil, 0, e
		}
		r := Record{Source: "nvd", ID: parsed.ID, State: parsed.Status, Published: published, Modified: modified, Acquired: now, JSON: v.CVE}
		if !validRecord(r) {
			return nil, 0, ErrSource
		}
		records = append(records, r)
	}
	return records, p.TotalResults, nil
}

// SyncNVD commits records and watermark in one transaction. A failed,
// canceled or interrupted page leaves the last complete snapshot intact.
// The caller chooses a window (at most 120 days); overlapping windows are
// idempotent. It does not claim coverage outside the recorded window.
func (c *Cache) SyncNVD(ctx context.Context, a *NVDClient, start, end time.Time) (Snapshot, error) {
	if c == nil || c.db == nil || ctx == nil || a == nil || start.IsZero() || !end.After(start) || end.Sub(start) > 120*24*time.Hour || end.After(time.Now().Add(time.Minute)) {
		return Snapshot{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	defer tx.Rollback()
	seen := make(map[string]bool)
	expectedTotal := -1
	for index, pages := 0, 0; ; pages++ {
		if pages >= MaxPageCount {
			return Snapshot{}, ErrLimit
		}
		records, total, e := a.page(ctx, start, end, index)
		if e != nil {
			return Snapshot{}, e
		}
		if expectedTotal >= 0 && total != expectedTotal {
			return Snapshot{}, ErrSource
		}
		expectedTotal = total
		for _, r := range records {
			if seen[r.ID] {
				return Snapshot{}, ErrSource
			}
			seen[r.ID] = true
			if e := c.upsert(ctx, tx, r); e != nil {
				return Snapshot{}, e
			}
		}
		index += len(records)
		if index >= total {
			break
		}
		if len(records) == 0 {
			return Snapshot{}, ErrSource
		}
	}
	var oldStart, oldWatermark sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT window_start,watermark FROM snapshots WHERE source='nvd'`).Scan(&oldStart, &oldWatermark)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, ErrUnavailable
	}
	windowStart := start.UTC()
	if oldStart.Valid {
		oldFirst, e1 := time.Parse(time.RFC3339Nano, oldStart.String)
		oldLast, e2 := time.Parse(time.RFC3339Nano, oldWatermark.String)
		if e1 != nil || e2 != nil {
			return Snapshot{}, ErrCorrupt
		}
		if start.After(oldLast) || end.Before(oldFirst) {
			return Snapshot{}, ErrInvalid
		}
		if oldFirst.Before(windowStart) {
			windowStart = oldFirst
		}
		if oldLast.After(end) {
			end = oldLast
		}
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM records WHERE source='nvd'`).Scan(&count); err != nil {
		return Snapshot{}, ErrUnavailable
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO snapshots(source,window_start,watermark,last_success,records) VALUES('nvd',?,?,?,?)
		ON CONFLICT(source) DO UPDATE SET window_start=excluded.window_start,watermark=excluded.watermark,last_success=excluded.last_success,records=excluded.records`,
		windowStart.Format(time.RFC3339Nano), end.UTC().Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), count)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if err := tx.Commit(); err != nil {
		return Snapshot{}, ErrUnavailable
	}
	return Snapshot{Source: "nvd", WindowStart: windowStart, Watermark: end.UTC(), LastSuccess: now, Records: count}, nil
}

// NextWindow overlaps the last successful NVD watermark by one hour and caps
// a request at 120 days. The first sync must be given an explicit start date.
func (s Snapshot) NextWindow(now time.Time) (time.Time, time.Time, error) {
	if s.Source != "nvd" || s.Watermark.IsZero() || !now.After(s.Watermark) {
		return time.Time{}, time.Time{}, ErrInvalid
	}
	start := s.Watermark.Add(-time.Hour)
	end := now
	if end.Sub(start) > 120*24*time.Hour {
		end = start.Add(120 * 24 * time.Hour)
	}
	return start, end, nil
}
