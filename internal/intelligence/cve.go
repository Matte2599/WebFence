package intelligence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

const cveEndpoint = "https://cveawg.mitre.org/api/cve/"

// CVEClient retrieves explicitly selected CVE Program records. NVD is a
// separate source; neither record is silently substituted for the other.
type CVEClient struct {
	HTTP        *http.Client
	Endpoint    string
	MinInterval time.Duration
	lastRequest time.Time
}

func (a *CVEClient) endpoint() (string, error) {
	if a.Endpoint == "" {
		return cveEndpoint, nil
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

type cveMetadata struct {
	ID        string `json:"cveId"`
	State     string `json:"state"`
	Published string `json:"datePublished"`
	Updated   string `json:"dateUpdated"`
}

func (a *CVEClient) fetch(ctx context.Context, id string) (Record, error) {
	if !cveIDPattern.MatchString(id) {
		return Record{}, ErrInvalid
	}
	endpoint, err := a.endpoint()
	if err != nil {
		return Record{}, err
	}
	interval := a.MinInterval
	if interval == 0 && a.Endpoint == "" {
		interval = time.Second
	}
	if interval < 0 {
		return Record{}, ErrInvalid
	}
	if wait := interval - time.Since(a.lastRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Record{}, ctx.Err()
		case <-timer.C:
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+id, nil)
	if err != nil {
		return Record{}, ErrInvalid
	}
	req.Header.Set("Accept", "application/json")
	client := a.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	a.lastRequest = time.Now()
	data, err := sourceGET(ctx, client, req, MaxRecordBytes)
	if err != nil {
		return Record{}, err
	}
	var parsed struct {
		Metadata cveMetadata `json:"cveMetadata"`
	}
	if json.Unmarshal(data, &parsed) != nil || parsed.Metadata.ID != id {
		return Record{}, ErrSource
	}
	modified, err := parseSourceTime(parsed.Metadata.Updated)
	if err != nil {
		return Record{}, err
	}
	var published time.Time
	if parsed.Metadata.Published != "" {
		published, err = parseSourceTime(parsed.Metadata.Published)
		if err != nil {
			return Record{}, err
		}
	}
	r := Record{Source: "cve", ID: id, State: parsed.Metadata.State, Published: published, Modified: modified, Acquired: time.Now().UTC(), JSON: data}
	if !validRecord(r) {
		return Record{}, ErrSource
	}
	return r, nil
}

// RefreshCVE imports at most 32 explicitly selected IDs in one transaction.
// Failure retains all previously cached records and the last success marker.
func (c *Cache) RefreshCVE(ctx context.Context, a *CVEClient, ids []string) (Snapshot, error) {
	if c == nil || c.db == nil || ctx == nil || a == nil || len(ids) < 1 || len(ids) > 32 {
		return Snapshot{}, ErrInvalid
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !cveIDPattern.MatchString(id) || seen[id] {
			return Snapshot{}, ErrInvalid
		}
		seen[id] = true
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	defer tx.Rollback()
	for _, id := range ids {
		r, e := a.fetch(ctx, id)
		if e != nil {
			return Snapshot{}, e
		}
		if e = c.upsert(ctx, tx, r); e != nil {
			return Snapshot{}, e
		}
	}
	now := time.Now().UTC()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM records WHERE source='cve'`).Scan(&count); err != nil {
		return Snapshot{}, ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO snapshots(source,window_start,watermark,last_success,records) VALUES('cve',?,?,?,?)
		ON CONFLICT(source) DO UPDATE SET watermark=excluded.watermark,last_success=excluded.last_success,records=excluded.records`,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), count)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if err := tx.Commit(); err != nil {
		return Snapshot{}, ErrUnavailable
	}
	return c.Snapshot(ctx, "cve")
}
