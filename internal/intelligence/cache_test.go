package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func openTestCache(t *testing.T) (*Cache, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "intelligence.sqlite")
	c, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, path
}

func nvdFixture(id string) map[string]any {
	return map[string]any{"cve": map[string]any{"id": id, "vulnStatus": "Analyzed", "published": "2026-09-01T00:00:00.000", "lastModified": "2026-09-25T00:00:00.000", "configurations": []any{}}}
}

func TestNVDSyncAtomicPaginationAndOffline(t *testing.T) {
	c, path := openTestCache(t)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/rest/json/cves/2.0" || r.URL.Query().Get("lastModStartDate") == "" || r.URL.Query().Get("lastModEndDate") == "" || r.URL.Query().Get("resultsPerPage") != "500" {
			t.Errorf("unexpected request: %s", r.URL.String())
			w.WriteHeader(400)
			return
		}
		index, _ := strconv.Atoi(r.URL.Query().Get("startIndex"))
		if requestCount == 4 {
			_, _ = w.Write([]byte(`{"format":"NVD_CVE","version":"2.0","startIndex":1,"resultsPerPage":500,"totalResults":2,"vulnerabilities":[{"cve":{"id":"wrong"}}]}`))
			return
		}
		id := "CVE-2026-1000"
		if index == 1 {
			id = "CVE-2026-1001"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"format": "NVD_CVE", "version": "2.0", "startIndex": index, "resultsPerPage": 500, "totalResults": 2, "vulnerabilities": []any{nvdFixture(id)}})
	}))
	defer server.Close()
	client := &NVDClient{Endpoint: server.URL + "/rest/json/cves/2.0"}
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	snap, err := c.SyncNVD(context.Background(), client, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Records != 2 || snap.Status(time.Now(), 72*time.Hour, false) != "fresh" || snap.Status(time.Now(), 72*time.Hour, true) != "offline" {
		t.Fatalf("snapshot: %+v", snap)
	}
	r, err := c.Get(context.Background(), "nvd", "CVE-2026-1001")
	if err != nil || r.Source != "nvd" || len(r.SHA256) != 64 {
		t.Fatalf("record: %+v %v", r, err)
	}
	_, err = c.SyncNVD(context.Background(), client, start, end.Add(time.Hour))
	if !errors.Is(err, ErrSource) {
		t.Fatalf("invalid page: %v", err)
	}
	after, err := c.Snapshot(context.Background(), "nvd")
	if err != nil || !after.Watermark.Equal(end) {
		t.Fatalf("snapshot changed after failure: %+v %v", after, err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := reopened.Get(context.Background(), "nvd", "CVE-2026-1000"); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Get(context.Background(), "nvd", "CVE-2026-1001"); err != nil {
		t.Fatal(err)
	}
}

func TestCVEBatchRollbackAndIndependentSource(t *testing.T) {
	c, _ := openTestCache(t)
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/cve/"):]
		if fail && id == "CVE-2026-1002" {
			w.WriteHeader(503)
			return
		}
		_, _ = fmt.Fprintf(w, `{"cveMetadata":{"cveId":%q,"state":"PUBLISHED","datePublished":"2026-09-01T00:00:00Z","dateUpdated":"2026-09-25T00:00:00Z"},"containers":{"cna":{"affected":[]}}}`, id)
	}))
	defer server.Close()
	a := &CVEClient{Endpoint: server.URL + "/api/cve/"}
	_, err := c.RefreshCVE(context.Background(), a, []string{"CVE-2026-1000"})
	if err != nil {
		t.Fatal(err)
	}
	fail = true
	_, err = c.RefreshCVE(context.Background(), a, []string{"CVE-2026-1001", "CVE-2026-1002"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable: %v", err)
	}
	if _, err := c.Get(context.Background(), "cve", "CVE-2026-1001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial commit: %v", err)
	}
	if _, err := c.Get(context.Background(), "cve", "CVE-2026-1000"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), "nvd", "CVE-2026-1000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sources mixed: %v", err)
	}
	snap, err := c.Snapshot(context.Background(), "cve")
	if err != nil || snap.Records != 1 {
		t.Fatalf("snapshot: %+v %v", snap, err)
	}
}

func TestInvalidWindowsAndEndpoints(t *testing.T) {
	c, _ := openTestCache(t)
	start := time.Now().Add(-121 * 24 * time.Hour)
	if _, err := c.SyncNVD(context.Background(), &NVDClient{}, start, time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := (&NVDClient{Endpoint: "https://example.com/redirect"}).endpoint(); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := (&CVEClient{Endpoint: "http://example.com/api/cve/"}).endpoint(); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), "nvd", "CVE-2026-../../secret"); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	s := Snapshot{Source: "nvd", Watermark: time.Now().Add(-180 * 24 * time.Hour)}
	a, b, err := s.NextWindow(time.Now())
	if err != nil || b.Sub(a) > 120*24*time.Hour {
		t.Fatalf("window: %v %v %v", a, b, err)
	}
}

func TestInterruptedSyncRetainsLastSnapshot(t *testing.T) {
	c, _ := openTestCache(t)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index, _ := strconv.Atoi(r.URL.Query().Get("startIndex"))
		if index == 1 {
			cancel()
			<-r.Context().Done()
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"format": "NVD_CVE", "version": "2.0", "startIndex": 0, "resultsPerPage": 500, "totalResults": 2, "vulnerabilities": []any{nvdFixture("CVE-2026-1000")}})
	}))
	defer server.Close()
	_, err := c.SyncNVD(ctx, &NVDClient{Endpoint: server.URL}, start, end)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := c.Get(context.Background(), "nvd", "CVE-2026-1000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial record: %v", err)
	}
	if _, err := c.Snapshot(context.Background(), "nvd"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial snapshot: %v", err)
	}
}

func TestSourceHTTPRetryAfterAndRedirect(t *testing.T) {
	var attempts int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("followed redirect") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	_, err := sourceGET(context.Background(), server.Client(), req, 1024)
	if !errors.Is(err, ErrUnavailable) || attempts != 2 {
		t.Fatalf("retry/redirect: attempts=%d err=%v", attempts, err)
	}
}
