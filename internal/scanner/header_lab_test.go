package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
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

type fixedResolver struct{ ip netip.Addr }

func (r fixedResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{r.ip}, nil
}

func fixtureGrant(t *testing.T, server *httptest.Server) transport.Grant {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if err != nil {
		t.Fatal(err)
	}
	return transport.Grant{Origin: server.URL, Addresses: []netip.Addr{ip}}
}

func fixtureStore(t *testing.T, origins ...string) (*storage.Store, string) {
	t.Helper()
	s, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "scanner.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	p, err := project.New(project.Draft{ID: "header-lab", Name: "Synthetic header lab",
		TargetOwner: "Fixture owner", AuthorizationReference: "local fixture permission",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: origins})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	return s, p.ID()
}

func fixturePlan(id string, urls []string, grant transport.Grant) HeaderLabPlan {
	return HeaderLabPlan{ProjectID: id, URLs: urls, Grants: []transport.Grant{grant},
		Limits: transport.Limits{MaxRequests: 8, MaxConcurrent: 1, MaxRedirects: 2,
			MaxBodyBytes: 1024, RequestTimeout: 3 * time.Second, RunTimeout: 5 * time.Second},
		Resolver: fixedResolver{grant.Addresses[0]}}
}

func TestHeaderLabClassifiesResponsesWithoutRetainingRawEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=private-cookie")
		switch r.URL.Path {
		case "/present":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		case "/absent":
			w.Header().Set("Content-Type", "text/html")
		case "/ambiguous":
			w.Header().Set("Content-Type", "text/html")
			w.Header().Add("X-Content-Type-Options", "nosniff")
			w.Header().Add("X-Content-Type-Options", "invalid-secret-value")
		case "/json":
			w.Header().Set("Content-Type", "application/json")
		case "/unknown":
			w.Header().Set("Content-Type", "not a media type")
		case "/conflicting-type":
			w.Header().Add("Content-Type", "text/html")
			w.Header().Add("Content-Type", "application/json")
		case "/failure":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, _ = io.WriteString(w, "private-body-token")
	}))
	defer server.Close()
	s, id := fixtureStore(t, server.URL)
	grant := fixtureGrant(t, server)
	paths := []string{"/present", "/absent", "/ambiguous", "/json", "/unknown", "/conflicting-type", "/failure"}
	urls := make([]string, len(paths))
	for i, path := range paths {
		urls[i] = server.URL + path + "?token=private-query"
	}
	report, err := RunHeaderLab(t.Context(), s, fixturePlan(id, urls, grant))
	if err != nil || !report.SeedsComplete || report.PlannedSeeds != 7 || report.CompletedSeeds != 7 || report.RequestsUsed != 7 {
		t.Fatalf("report: %+v, %v", report, err)
	}
	want := []struct {
		outcome  Outcome
		evidence string
	}{
		{Observed, "nosniff_present"}, {NotObserved, "nosniff_absent"},
		{Inconclusive, "nosniff_unrecognized"}, {Skipped, "non_html_response"},
		{Inconclusive, "content_type_unknown"}, {Inconclusive, "content_type_unknown"},
		{Skipped, "http_status_not_applicable"},
	}
	for i, check := range report.Checks {
		if check.SeedIndex != i || check.RuleID != HeaderRuleID || check.RuleRevision != HeaderRuleRevision ||
			check.Outcome != want[i].outcome || check.EvidenceCode != want[i].evidence {
			t.Fatalf("check %d: %+v", i, check)
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-cookie", "private-body-token", "private-query", "invalid-secret-value", server.URL} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("report retained unredacted response/URL: %s", secret)
		}
	}
}

func TestHeaderLabRejectsAllSeedsBeforeNetworkAndKeepsBudgetPartial(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "fixture")
	}))
	defer server.Close()
	s, id := fixtureStore(t, server.URL)
	grant := fixtureGrant(t, server)
	plan := fixturePlan(id, []string{server.URL + "/first", "http://outside.invalid/second"}, grant)
	plan.Grants = make([]transport.Grant, MaxHeaderLabGrants+1)
	if _, err := RunHeaderLab(t.Context(), s, plan); !errors.Is(err, ErrInvalidPlan) || hits.Load() != 0 {
		t.Fatalf("unbounded grant list: hits=%d err=%v", hits.Load(), err)
	}
	plan.Grants = []transport.Grant{grant}
	if _, err := RunHeaderLab(t.Context(), s, plan); !errors.Is(err, scope.ErrOutOfScope) || hits.Load() != 0 {
		t.Fatalf("out-of-scope preflight: hits=%d err=%v", hits.Load(), err)
	}
	plan.URLs[1] = server.URL + "/second"
	plan.Limits.MaxRequests = 1
	report, err := RunHeaderLab(t.Context(), s, plan)
	if !errors.Is(err, transport.ErrBudget) || report.SeedsComplete || report.CompletedSeeds != 1 ||
		report.RequestsUsed != 1 || report.PlannedSeeds != 2 || hits.Load() != 1 ||
		len(report.Checks) != 2 || report.Checks[1].Outcome != CheckError ||
		report.StopCode != transport.ErrBudget.Error() {
		t.Fatalf("budget partial report: %+v, hits=%d, err=%v", report, hits.Load(), err)
	}
}

func TestHeaderLabOwnsCheckedSeedsDuringRun(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var secondHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/first" {
			close(started)
			<-release
		} else if r.URL.Path == "/second" {
			secondHits.Add(1)
		}
		_, _ = io.WriteString(w, "fixture")
	}))
	defer server.Close()
	s, id := fixtureStore(t, server.URL)
	plan := fixturePlan(id, []string{server.URL + "/first", server.URL + "/second"}, fixtureGrant(t, server))
	done := make(chan error, 1)
	go func() {
		report, err := RunHeaderLab(t.Context(), s, plan)
		if err == nil && (!report.SeedsComplete || report.CompletedSeeds != 2) {
			err = errors.New("incomplete owned-seed run")
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("first fixture request did not start")
	}
	plan.URLs[1] = "http://outside.invalid/changed-after-check"
	close(release)
	select {
	case err := <-done:
		if err != nil || secondHits.Load() != 1 {
			t.Fatalf("checked second seed changed: hits=%d err=%v", secondHits.Load(), err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("scan did not finish")
	}
}

func TestUnknownTransportErrorIsRedactedAtScannerBoundary(t *testing.T) {
	raw := errors.New("failed to fetch http://fixture.invalid/?token=private")
	if got := safeStopError(raw); !errors.Is(got, ErrFetchFailed) ||
		strings.Contains(got.Error(), "fixture.invalid") || strings.Contains(got.Error(), "private") {
		t.Fatalf("raw transport error escaped: %v", got)
	}
}

func TestHeaderLabRedirectCannotWidenProjectAndRevocationStopsWork(t *testing.T) {
	var outsideHits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outsideHits.Add(1)
	}))
	defer outside.Close()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, outside.URL+"/secret", http.StatusFound)
			return
		}
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	s, id := fixtureStore(t, server.URL)
	grant := fixtureGrant(t, server)
	plan := fixturePlan(id, []string{server.URL + "/redirect"}, grant)
	plan.Grants = append(plan.Grants, fixtureGrant(t, outside))
	report, err := RunHeaderLab(t.Context(), s, plan)
	if !errors.Is(err, scope.ErrOutOfScope) || report.SeedsComplete || report.RequestsUsed != 1 || outsideHits.Load() != 0 {
		t.Fatalf("redirect widened project: %+v, outside=%d, err=%v", report, outsideHits.Load(), err)
	}
	plan.URLs = []string{server.URL + "/wait"}
	result := make(chan struct {
		report Report
		err    error
	}, 1)
	go func() {
		report, err := RunHeaderLab(t.Context(), s, plan)
		result <- struct {
			report Report
			err    error
		}{report, err}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture request did not start")
	}
	if _, err := s.RevokeAuthorization(t.Context(), id, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-result:
		if !errors.Is(got.err, project.ErrAuthorizationRevoked) || got.report.SeedsComplete ||
			got.report.RequestsUsed != 1 || got.report.CompletedSeeds != 0 ||
			got.report.StopCode != project.ErrAuthorizationRevoked.Error() {
			t.Fatalf("revoked scan: %+v, err=%v", got.report, got.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("revoked scan did not stop")
	}
}
