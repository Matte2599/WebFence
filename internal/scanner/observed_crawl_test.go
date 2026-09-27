package scanner

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/transport"
)

func observedFixture(path string) browser.Observation {
	return browser.Observation{Method: http.MethodGet, Kind: browser.Fetch, Path: path,
		FinalPath: path, StatusCode: http.StatusOK}
}

func TestObservedCrawlExplicitFetchReplayAndRedactedLedger(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" ||
			(r.URL.Path != "/allowed/one" && r.URL.Path != "/allowed/two") {
			t.Errorf("unexpected replay %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secret":"synthetic-private-body"}`))
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	plan := ObservedCrawlPlan{Origin: server.URL, ReplayConfirmed: true, Crawl: crawl,
		Observations: []browser.Observation{observedFixture("/allowed/one"),
			observedFixture("/allowed/two"), observedFixture("/allowed/not-selected")},
		SelectedIndices: []int{1, 0}}
	report, err := RunObservedCrawl(t.Context(), store, plan)
	if err != nil || hits.Load() != 2 || report.CompletedVisits != 2 || report.RequestsUsed != 2 ||
		!report.QueueDrained || len(report.Visits) != 2 {
		t.Fatalf("observed replay: %+v hits=%d err=%v", report, hits.Load(), err)
	}
	stored, err := store.LoadScanRun(t.Context(), report.RunID)
	if err != nil || stored.State != "complete" || stored.CompletedVisits != 2 {
		t.Fatalf("redacted ledger: %+v err=%v", stored, err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{server.URL, "/allowed/one", "synthetic-private-body"} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("report leaked fixture data: %s", encoded)
		}
	}
}

func TestObservedCrawlRejectsUnconfirmedOrUnsafeHintsBeforeNetwork(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	base := ObservedCrawlPlan{Origin: server.URL, ReplayConfirmed: true, Crawl: crawl,
		Observations: []browser.Observation{observedFixture("/allowed/one")}, SelectedIndices: []int{0}}
	changes := []func(*ObservedCrawlPlan){
		func(p *ObservedCrawlPlan) { p.ReplayConfirmed = false },
		func(p *ObservedCrawlPlan) { p.Origin += "/path" },
		func(p *ObservedCrawlPlan) { p.SelectedIndices = []int{0, 0} },
		func(p *ObservedCrawlPlan) { p.SelectedIndices = []int{1} },
		func(p *ObservedCrawlPlan) { p.Observations[0].Method = http.MethodPost },
		func(p *ObservedCrawlPlan) { p.Observations[0].Kind = browser.Subresource },
		func(p *ObservedCrawlPlan) { p.Observations[0].StatusCode = http.StatusForbidden },
		func(p *ObservedCrawlPlan) { p.Observations[0].FinalPath = "/allowed/two" },
		func(p *ObservedCrawlPlan) { p.Observations[0].Path = "//outside.invalid/x" },
		func(p *ObservedCrawlPlan) { p.Observations[0].Path = "/allowed/one?write=1" },
		func(p *ObservedCrawlPlan) { p.Observations[0].Path = "/allowed/../other" },
	}
	for i, change := range changes {
		plan := base
		plan.Observations = append([]browser.Observation(nil), base.Observations...)
		change(&plan)
		_, err := RunObservedCrawl(t.Context(), store, plan)
		if err == nil || hits.Load() != 0 {
			t.Fatalf("unsafe hint %d: hits=%d err=%v", i, hits.Load(), err)
		}
	}
	plan := base
	plan.Crawl.Grants = []transport.Grant{{Origin: "https://outside.invalid"}}
	if _, err := RunObservedCrawl(t.Context(), store, plan); err == nil || hits.Load() != 0 {
		t.Fatalf("invalid grant: hits=%d err=%v", hits.Load(), err)
	}
}

func TestObservedCrawlStopsRedirectBeforeUnselectedRequest(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/allowed/one" {
			t.Errorf("redirect reached unselected path %s", r.URL.Path)
		}
		http.Redirect(w, r, "/allowed/two", http.StatusFound)
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	crawl := crawlFixturePlan(t, id, nil, fixtureGrant(t, server))
	crawl.FollowLinks, crawl.MaxDepth = false, 0
	plan := ObservedCrawlPlan{Origin: server.URL, ReplayConfirmed: true, Crawl: crawl,
		Observations: []browser.Observation{observedFixture("/allowed/one")}, SelectedIndices: []int{0}}
	report, err := RunObservedCrawl(t.Context(), store, plan)
	if !errors.Is(err, transport.ErrRedirectLimit) || hits.Load() != 1 || report.RequestsUsed != 1 ||
		report.CompletedVisits != 0 || !report.CoverageLimited {
		t.Fatalf("redirect replay: %+v hits=%d err=%v", report, hits.Load(), err)
	}
}
