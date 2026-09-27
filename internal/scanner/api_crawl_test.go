package scanner

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Matte2599/WebFence/internal/transport"
)

const apiFixture = `{"openapi":"3.1.0","servers":[{"url":"https://outside.invalid"}],"paths":{"/allowed/a":{"get":{}},"/allowed/b":{"get":{}},"/allowed/excluded/c":{"get":{}},"/allowed/{id}":{"get":{}},"/allowed/write":{"post":{}},"/allowed/private":{"get":{"security":[{"key":[]}]}}}}`

func TestAPICrawlExplicitStaticBatchAndRedactedLedger(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodGet || (r.URL.Path != "/allowed/a" && r.URL.Path != "/allowed/b") {
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"private":"body-marker"}`))
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	plan := APICrawlPlan{Document: []byte(apiFixture), Origin: server.URL,
		SelectedPaths: []string{"/allowed/b", "/allowed/a"}, Crawl: crawl}
	report, err := RunAPICrawl(t.Context(), store, plan)
	if err != nil || hits.Load() != 2 || report.PlannedSeeds != 2 || report.CompletedVisits != 2 ||
		report.RequestsUsed != 2 || !report.QueueDrained || len(report.Visits) != 2 {
		t.Fatalf("API batch: %+v hits=%d err=%v", report, hits.Load(), err)
	}
	for _, visit := range report.Visits {
		if visit.Depth != 0 || visit.EvidenceCode != "non_html_response" {
			t.Fatalf("unexpected API visit: %+v", visit)
		}
	}
	stored, err := store.LoadScanRun(t.Context(), report.RunID)
	if err != nil || stored.CompletedVisits != 2 || stored.State != "complete" {
		t.Fatalf("API ledger: %+v err=%v", stored, err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{server.URL, "outside.invalid", "body-marker", "/allowed/a"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("API report leaked input: %s", encoded)
		}
	}
}

func TestAPICrawlRejectsUnselectedAndDeniedRoutesBeforeNetworking(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	base := APICrawlPlan{Document: []byte(apiFixture), Origin: server.URL, Crawl: crawl}
	for _, paths := range [][]string{
		nil, {"/allowed/a", "/allowed/a"}, {"/allowed/a", "/allowed/excluded/c"},
		{"/allowed/write"}, {"/allowed/private"}, {"/allowed/{id}"},
		{"/allowed/a", "/allowed/not-in-document"},
	} {
		plan := base
		plan.SelectedPaths = paths
		if _, err := RunAPICrawl(t.Context(), store, plan); !errors.Is(err, ErrAPISelection) || hits.Load() != 0 {
			t.Fatalf("selection %v: hits=%d err=%v", paths, hits.Load(), err)
		}
	}
	plan := base
	plan.SelectedPaths = []string{"/allowed/a"}
	plan.Crawl.SeedURLs = []string{server.URL + "/allowed/extra"}
	if _, err := RunAPICrawl(t.Context(), store, plan); !errors.Is(err, ErrAPISelection) || hits.Load() != 0 {
		t.Fatalf("implicit seed: hits=%d err=%v", hits.Load(), err)
	}
	plan.Crawl.SeedURLs = nil
	plan.Crawl.Grants = []transport.Grant{{Origin: "https://outside.invalid"}}
	if _, err := RunAPICrawl(t.Context(), store, plan); err == nil || hits.Load() != 0 {
		t.Fatalf("invalid grant: hits=%d err=%v", hits.Load(), err)
	}
}

func TestAPICrawlDoesNotFollowRedirectToUnselectedRoute(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/allowed/a" {
			t.Errorf("unselected redirect reached %q", r.URL.Path)
		}
		http.Redirect(w, r, "/allowed/b", http.StatusFound)
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	plan := APICrawlPlan{Document: []byte(apiFixture), Origin: server.URL,
		SelectedPaths: []string{"/allowed/a"}, Crawl: crawl}
	report, err := RunAPICrawl(t.Context(), store, plan)
	if !errors.Is(err, transport.ErrRedirectLimit) || hits.Load() != 1 ||
		report.RequestsUsed != 1 || report.CompletedVisits != 0 || !report.CoverageLimited {
		t.Fatalf("redirect handling: %+v hits=%d err=%v", report, hits.Load(), err)
	}
}
