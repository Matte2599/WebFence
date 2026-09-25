package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
)

func crawlFixturePlan(t *testing.T, id string, seeds []string, grant transport.Grant) CrawlPlan {
	t.Helper()
	route, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/allowed"}, []string{"/allowed/excluded"})
	if err != nil {
		t.Fatal(err)
	}
	base := fixturePlan(id, seeds, grant)
	base.Limits.MinRequestInterval = 100 * time.Millisecond
	return CrawlPlan{ProjectID: id, SeedURLs: seeds, Mode: CrawlLoopback,
		Grants: base.Grants, Policy: route, Limits: base.Limits, Resolver: base.Resolver,
		MaxPages: 3, MaxDepth: 2, FollowLinks: true}
}

func TestCrawlVisitsOnlyAllowedGETLinksAndRedactsReport(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		switch r.URL.Path {
		case "/allowed/start":
			_, _ = io.WriteString(w, `<a href="/allowed/page?token=private-query">one</a><a href="/allowed/page?token=private-query">duplicate</a><a href="/allowed/excluded/delete">excluded</a><a href="/outside">outside path</a><a href="https://outside.invalid/">outside origin</a><form action="/allowed/submit?token=private-form" method="POST"><input value="private-field"></form>`)
		case "/allowed/page":
			_, _ = io.WriteString(w, `<a href="/allowed/deep">deep</a>`)
		case "/allowed/deep":
			_, _ = io.WriteString(w, "done")
		default:
			t.Errorf("unexpected visit to %q", r.URL.Path)
		}
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	plan := crawlFixturePlan(t, id, []string{server.URL + "/allowed/start?token=private-seed"}, fixtureGrant(t, server))
	report, err := RunCrawl(t.Context(), store, plan)
	if err != nil || hits.Load() != 3 || report.CompletedVisits != 3 || report.RequestsUsed != 3 ||
		!report.QueueDrained || !report.CoverageLimited || report.PolicyOmitted != 2 ||
		len(report.Visits) != 3 || report.Visits[0].Discovery.Forms != 1 ||
		report.Visits[0].Discovery.OutOfScope != 1 {
		t.Fatalf("crawl report: %+v hits=%d err=%v", report, hits.Load(), err)
	}
	for i, visit := range report.Visits {
		if visit.Depth != i || visit.Outcome != Observed || visit.EvidenceCode != "nosniff_present" {
			t.Fatalf("visit %d: %+v", i, visit)
		}
	}
	stored, err := store.LoadScanRun(t.Context(), report.RunID)
	if err != nil || stored.State != "complete" || stored.CompletedVisits != 3 || len(stored.Visits) != 3 ||
		!stored.CoverageLimited || stored.Visits[0].EvidenceCode != "nosniff_present" {
		t.Fatalf("persistent crawl result: %+v err=%v", stored, err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{server.URL, "private-query", "private-seed", "private-form", "private-field", "outside.invalid"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("crawl report leaked input: %s", encoded)
		}
	}
}

func TestCrawlHeaderBeforeAndAfterFixSurvivesRestart(t *testing.T) {
	var fixed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if fixed.Load() {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		_, _ = io.WriteString(w, "<html>synthetic fixture</html>")
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "end-to-end.sqlite")
	store, err := storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	p, err := project.New(project.Draft{ID: "fix-fixture", Name: "Synthetic fix fixture", TargetOwner: "Test fixture",
		AuthorizationReference: "owned local server", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	plan := crawlFixturePlan(t, p.ID(), []string{server.URL + "/allowed"}, fixtureGrant(t, server))
	plan.MaxPages, plan.MaxDepth, plan.FollowLinks = 1, 0, false
	before, err := RunCrawl(t.Context(), store, plan)
	if err != nil || len(before.Visits) != 1 || before.Visits[0].EvidenceCode != "nosniff_absent" {
		t.Fatalf("before fix: %+v %v", before, err)
	}
	fixed.Store(true)
	after, err := RunCrawl(t.Context(), store, plan)
	if err != nil || len(after.Visits) != 1 || after.Visits[0].EvidenceCode != "nosniff_present" {
		t.Fatalf("after fix: %+v %v", after, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ id, want string }{{before.RunID, "nosniff_absent"}, {after.RunID, "nosniff_present"}} {
		run, err := store.LoadScanRun(t.Context(), pair.id)
		if err != nil || run.State != "complete" || len(run.Visits) != 1 || run.Visits[0].EvidenceCode != pair.want ||
			run.Visits[0].RuleID != HeaderRuleID || run.Visits[0].RuleRevision != HeaderRuleRevision {
			t.Fatalf("reopened %s: %+v %v", pair.want, run, err)
		}
	}
}

func TestCrawlPanickingObserverCannotPersistFalseCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>synthetic fixture</html>")
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	plan := crawlFixturePlan(t, id, []string{server.URL + "/allowed"}, fixtureGrant(t, server))
	plan.MaxPages, plan.MaxDepth, plan.FollowLinks = 1, 0, false
	plan.OnProgress = func(int) { panic("synthetic observer crash") }
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("observer did not panic")
			}
		}()
		_, _ = RunCrawl(t.Context(), store, plan)
	}()
	runs, err := store.ListScanRuns(t.Context(), id)
	if err != nil || len(runs) != 1 || runs[0].State != "interrupted" || !runs[0].CoverageLimited || runs[0].StopCode != "process_interrupted" {
		t.Fatalf("false completion after panic: %+v %v", runs, err)
	}
}

func TestCrawlReportsPageDepthAndOptInLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/allowed/start":
			_, _ = io.WriteString(w, `<a href="/allowed/one">one</a><a href="/allowed/two">two</a>`)
		case "/allowed/one":
			_, _ = io.WriteString(w, `<a href="/allowed/deep">deep</a>`)
		}
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	plan := crawlFixturePlan(t, id, []string{server.URL + "/allowed/start"}, fixtureGrant(t, server))
	plan.MaxPages = 2
	plan.MaxDepth = 1
	report, err := RunCrawl(t.Context(), store, plan)
	if err != nil || report.CompletedVisits != 2 || report.PageLimitOmitted != 1 || report.DepthOmitted != 1 ||
		!report.QueueDrained || !report.CoverageLimited {
		t.Fatalf("bounded crawl: %+v err=%v", report, err)
	}
	plan.FollowLinks, plan.MaxDepth, plan.MaxPages = false, 0, 1
	report, err = RunCrawl(t.Context(), store, plan)
	if err != nil || report.CompletedVisits != 1 || report.NotScheduled != 2 || !report.CoverageLimited || !report.QueueDrained {
		t.Fatalf("opt-out crawl: %+v err=%v", report, err)
	}
}

func TestCrawlPreflightsAllSeedsAndRechecksRedirect(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/allowed/redirect" {
			http.Redirect(w, r, "/allowed/excluded/target", http.StatusFound)
		}
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	plan := crawlFixturePlan(t, id, []string{server.URL + "/allowed/start", server.URL + "/allowed/excluded/seed"}, fixtureGrant(t, server))
	if _, err := RunCrawl(t.Context(), store, plan); !errors.Is(err, scope.ErrPathDenied) || hits.Load() != 0 {
		t.Fatalf("excluded seed reached network: hits=%d err=%v", hits.Load(), err)
	}
	plan.SeedURLs = []string{server.URL + "/allowed/redirect"}
	report, err := RunCrawl(t.Context(), store, plan)
	if !errors.Is(err, scope.ErrPathDenied) || hits.Load() != 1 || report.RequestsUsed != 1 ||
		report.CompletedVisits != 0 || report.StopCode != scope.ErrPathDenied.Error() || report.QueueDrained || !report.CoverageLimited {
		t.Fatalf("excluded redirect: %+v hits=%d err=%v", report, hits.Load(), err)
	}
	plan.Mode = CrawlPinnedPublic
	if _, err := RunCrawl(t.Context(), store, plan); !errors.Is(err, transport.ErrConfig) || hits.Load() != 1 {
		t.Fatalf("loopback granted to public mode: hits=%d err=%v", hits.Load(), err)
	}
}

func TestCrawlManagedRevocationStopsInFlight(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	plan := crawlFixturePlan(t, id, []string{server.URL + "/allowed/wait"}, fixtureGrant(t, server))
	done := make(chan struct {
		report CrawlReport
		err    error
	}, 1)
	go func() {
		report, err := RunCrawl(t.Context(), store, plan)
		done <- struct {
			report CrawlReport
			err    error
		}{report, err}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("crawl did not start")
	}
	if _, err := store.RevokeAuthorization(context.Background(), id, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if !errors.Is(got.err, project.ErrAuthorizationRevoked) || got.report.RequestsUsed != 1 ||
			got.report.CompletedVisits != 0 || got.report.StopCode != project.ErrAuthorizationRevoked.Error() {
			t.Fatalf("revoked crawl: %+v err=%v", got.report, got.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("revoked crawl did not stop")
	}
}
