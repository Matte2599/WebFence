package scanner

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/transport"
)

func fixturePermit(t *testing.T, origin string) project.RunScope {
	t.Helper()
	store, id := fixtureStore(t, origin)
	run, err := store.BeginRun(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(run.Close)
	return run.Scope()
}

func TestObserveHTMLKeepsOnlyAuthorizedCandidates(t *testing.T) {
	permit := fixturePermit(t, "http://example.test:80")
	body := []byte(`<base href="http://example.test/dir/">
	<a href="one?x=private-query#part">one</a><a href="one?x=private-query">duplicate</a>
	<a href="../a%2Fb?b=2&a=1">encoded</a><a href="https://other.test/">outside</a>
	<a href="javascript:alert(1)">bad</a><a href="/slash\\other">bad</a>
	<form action="submit?token=private-form" method="post"><input value="private-value"></form>
	<form method="get"></form><form action="//other.test/form"></form>`)
	surface, err := ObserveHTML(permit, "http://example.test:80/start", body)
	if err != nil || surface.Incomplete || surface.OutOfScope != 2 || surface.Invalid != 2 ||
		len(surface.Links) != 2 || len(surface.Forms) != 2 {
		t.Fatalf("surface: %+v err=%v", surface, err)
	}
	if surface.Links[0] != "http://example.test:80/dir/one?x=private-query" ||
		surface.Links[1] != "http://example.test:80/a%2Fb?b=2&a=1" ||
		surface.Forms[0].Method != "POST" || surface.Forms[1].Method != "GET" {
		t.Fatalf("resolution/method mismatch: %+v", surface)
	}
}

func TestObserveHTMLExternalBaseAndParserLimits(t *testing.T) {
	permit := fixturePermit(t, "http://example.test:80")
	beforeBase, err := ObserveHTML(permit, "http://example.test:80/start", []byte(`<a href="relative">x</a><base href="/later/">`))
	if err != nil || len(beforeBase.Links) != 1 || beforeBase.Links[0] != "http://example.test:80/later/relative" {
		t.Fatalf("base applies to earlier link: %+v err=%v", beforeBase, err)
	}
	surface, err := ObserveHTML(permit, "http://example.test:80/start", []byte(`<base href="https://third.test/path/"><a href="relative">x</a><a href="http://example.test/absolute">y</a><form></form>`))
	if err != nil || surface.OutOfScope != 1 || len(surface.Links) != 1 || surface.Links[0] != "http://example.test:80/absolute" || len(surface.Forms) != 1 || surface.Forms[0].Action != "http://example.test:80/start" {
		t.Fatalf("external base: %+v err=%v", surface, err)
	}
	var many strings.Builder
	for range MaxDiscoveryReferences + 1 {
		many.WriteString(`<a href="/next">x</a>`)
	}
	surface, err = ObserveHTML(permit, "http://example.test:80/start", []byte(many.String()))
	if err != nil || !surface.Incomplete || len(surface.Links) != 1 {
		t.Fatalf("reference limit: %+v err=%v", surface, err)
	}
	surface, err = ObserveHTML(permit, "http://example.test:80/start", []byte{0xff})
	if err != nil || !surface.Incomplete || len(surface.Links) != 0 {
		t.Fatalf("invalid UTF-8: %+v err=%v", surface, err)
	}
	surface, err = ObserveHTML(permit, "http://example.test:80/start", []byte("<a "+strings.Repeat("x", MaxDiscoveryTokenBytes)+`="/next">x</a>`))
	if err != nil || !surface.Incomplete {
		t.Fatalf("oversize token: %+v err=%v", surface, err)
	}
}

func TestDiscoveryMarksUnsupportedCharsetIncomplete(t *testing.T) {
	permit := fixturePermit(t, "http://example.test:80")
	summary, err := observeResponse(permit, 0, transport.Result{StatusCode: 200,
		Header: http.Header{"Content-Type": []string{"text/html; charset=iso-8859-1"}},
		Body:   []byte(`<a href="/hidden">x</a>`), FinalURL: "http://example.test:80/"})
	if err != nil || summary.Status != "incomplete" || summary.ReasonCode != "unsupported_charset" || summary.Links != 0 {
		t.Fatalf("charset summary: %+v err=%v", summary, err)
	}
	summary, err = observeResponse(permit, 0, transport.Result{StatusCode: 200,
		Header: http.Header{"Content-Type": []string{"invalid media type"}},
		Body:   []byte(`<a href="/hidden">x</a>`), FinalURL: "http://example.test:80/"})
	if err != nil || summary.Status != "incomplete" || summary.ReasonCode != "content_type_unknown" {
		t.Fatalf("unknown content type: %+v err=%v", summary, err)
	}
}

func TestHeaderLabObservesRedirectedHTMLWithoutFollowingCandidates(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/from" {
			http.Redirect(w, r, "/nested/final?secret=seed-query", http.StatusFound)
			return
		}
		if r.URL.Path != "/nested/final" {
			t.Error("discovered candidate was fetched")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<a href="next?secret=private-link">next</a><a href="https://other.test/">other</a><form action="submit?secret=private-form"><input value="private-value"></form>`)
	}))
	defer server.Close()
	store, id := fixtureStore(t, server.URL)
	report, err := RunHeaderLab(t.Context(), store, fixturePlan(id, []string{server.URL + "/from"}, fixtureGrant(t, server)))
	if err != nil || hits.Load() != 2 || report.RequestsUsed != 2 || !report.SeedsComplete ||
		len(report.Discovery) != 1 || report.Discovery[0].Status != "observed" ||
		report.Discovery[0].Links != 1 || report.Discovery[0].Forms != 1 || report.Discovery[0].OutOfScope != 1 {
		t.Fatalf("report: %+v hits=%d err=%v", report, hits.Load(), err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{server.URL, "seed-query", "private-link", "private-form", "private-value", "other.test"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("raw discovery leaked in report: %s", encoded)
		}
	}
}
