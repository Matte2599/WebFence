package scope_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/scope"
)

// This laboratory adapter is deliberately test-only. It is not the production
// network broker: no public DNS, proxy, credentials, browser or scanning rules.
type labTransport struct {
	policy scope.Policy
	base   *http.Transport
}

func (l labTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := l.policy.Check(req.URL.String())
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.URL = u
	return l.base.RoundTrip(clone)
}

func TestLoopbackLabChecksInitialRequestsAndRedirects(t *testing.T) {
	var outsideHits, fixtureHits, forbiddenDials atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outsideHits.Add(1)
		_, _ = io.WriteString(w, "OUTSIDE SCOPE")
	}))
	defer outside.Close()
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixtureHits.Add(1)
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, "/final?x=1&x=2", http.StatusFound)
		case "/other-port":
			http.Redirect(w, r, outside.URL+"/canary", http.StatusFound)
		case "/external":
			http.Redirect(w, r, "https://outside.invalid/canary", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusTemporaryRedirect)
		default:
			_, _ = io.WriteString(w, "SYNTHETIC "+r.RequestURI)
		}
	}))
	defer fixture.Close()
	p, err := scope.New([]string{fixture.URL})
	if err != nil {
		t.Fatal(err)
	}
	// An independent dial fence guarantees the test cannot contact anything
	// except its two owned loopback fixtures, even if scope regresses.
	addresses := make(map[string]bool)
	for _, raw := range []string{fixture.URL, outside.URL} {
		u, _ := url.Parse(raw)
		if ip := net.ParseIP(u.Hostname()); ip == nil || !ip.IsLoopback() {
			t.Fatal("fixture must use literal loopback")
		}
		addresses[u.Host] = true
	}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			if !addresses[address] {
				forbiddenDials.Add(1)
				return nil, errors.New("lab_destination_denied")
			}
			return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, address)
		},
	}
	defer transport.CloseIdleConnections()
	errRedirectLimit := errors.New("lab_redirect_limit")
	client := &http.Client{
		Transport: labTransport{policy: p, base: transport}, Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errRedirectLimit
			}
			_, err := p.Check(req.URL.String())
			return err
		},
	}
	for _, test := range []struct {
		name, target, body string
		want               error
	}{
		{"in scope", fixture.URL + "/final", "SYNTHETIC /final", nil},
		{"relative redirect", fixture.URL + "/same", "SYNTHETIC /final?x=1&x=2", nil},
		{"direct other port", outside.URL + "/canary", "", scope.ErrOutOfScope},
		{"redirect other port", fixture.URL + "/other-port", "", scope.ErrOutOfScope},
		{"external redirect", fixture.URL + "/external", "", scope.ErrOutOfScope},
		{"redirect loop budget", fixture.URL + "/loop", "", errRedirectLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			resp, err := client.Get(test.target)
			if resp != nil {
				defer resp.Body.Close()
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
			if err != nil {
				return
			}
			body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
			if err != nil || string(body) != test.body {
				t.Fatalf("body %q, error %v", body, err)
			}
		})
	}
	if outsideHits.Load() != 0 || forbiddenDials.Load() != 0 {
		t.Fatalf("out of scope reached HTTP=%d dial=%d", outsideHits.Load(), forbiddenDials.Load())
	}
	if fixtureHits.Load() != 8 {
		t.Fatalf("unexpected fixture requests: %d", fixtureHits.Load())
	}
}
