package transport

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}
func limits() Limits {
	return Limits{MaxRequests: 20, MaxConcurrent: 2, MaxRedirects: 3, MaxBodyBytes: 1024, RequestTimeout: 3 * time.Second, RunTimeout: 10 * time.Second}
}
func fixtureGrant(t *testing.T, server *httptest.Server, host string) Grant {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ip := netip.MustParseAddr(u.Hostname())
	if !ip.IsLoopback() {
		t.Fatal("fixture must be loopback")
	}
	if host != "" {
		u.Host = net.JoinHostPort(host, u.Port())
	}
	return Grant{Origin: u.String(), Addresses: []netip.Addr{ip}}
}
func newFixtureBroker(t *testing.T, g Grant, l Limits, resolver Resolver) (*LabBroker, *atomic.Int32) {
	t.Helper()
	b, err := NewLab(context.Background(), []Grant{g}, l, resolver)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Close)
	u, _ := url.Parse(g.Origin)
	endpoints := map[string]bool{}
	for _, ip := range g.Addresses {
		endpoints[net.JoinHostPort(ip.String(), u.Port())] = true
	}
	count := new(atomic.Int32)
	// Independent test-only fence: never contact any address outside the fixture.
	b.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		count.Add(1)
		if !endpoints[address] {
			return nil, errors.New("fixture_dial_denied")
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	return b, count
}
func fixed(ip netip.Addr) Resolver {
	return resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return []netip.Addr{ip}, nil })
}
func await(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("fixture synchronization timed out")
	}
}

func TestDNSPinningAndRebinding(t *testing.T) {
	var hits atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !strings.HasPrefix(r.Host, "fixture.invalid:") {
			t.Error("Host changed to pinned IP")
		}
		_, _ = io.WriteString(w, r.RequestURI)
	}))
	defer s.Close()
	g := fixtureGrant(t, s, "fixture.invalid")
	var lookups atomic.Int32
	dns := resolverFunc(func(ctx context.Context, network, host string) ([]netip.Addr, error) {
		if network != "ip" || host != "fixture.invalid" {
			t.Error("unexpected lookup")
		}
		if lookups.Add(1) == 1 {
			return g.Addresses, nil
		}
		return []netip.Addr{netip.MustParseAddr("127.0.0.2")}, nil
	})
	b, dials := newFixtureBroker(t, g, limits(), dns)
	result, err := b.Fetch(context.Background(), g.Origin+"/a%2Fb?x=1&x=2")
	if err != nil || string(result.Body) != "/a%2Fb?x=1&x=2" {
		t.Fatalf("result=%q error=%v", result.Body, err)
	}
	if lookups.Load() != 1 || dials.Load() != 1 {
		t.Fatal("connection performed a second lookup")
	}
	_, err = b.Fetch(context.Background(), g.Origin)
	if !errors.Is(err, ErrAddress) || dials.Load() != 1 || hits.Load() != 1 || b.RequestsUsed() != 2 {
		t.Fatalf("rebinding allowed or budget refunded: %v", err)
	}
}

func TestDNSFailuresDenyBeforeDial(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("unexpected HTTP") }))
	defer s.Close()
	g := fixtureGrant(t, s, "fixture.invalid")
	for _, tc := range []struct {
		name              string
		ips               []netip.Addr
		dnsErr, errorWant error
	}{
		{"empty", nil, nil, ErrResolve},
		{"failure", nil, errors.New("PRIVATE DNS detail"), ErrResolve},
		{"mixed public", []netip.Addr{g.Addresses[0], netip.MustParseAddr("192.0.2.1")}, nil, ErrAddress},
		{"private", []netip.Addr{netip.MustParseAddr("10.0.0.1")}, nil, ErrAddress},
		{"link local metadata", []netip.Addr{netip.MustParseAddr("169.254.169.254")}, nil, ErrAddress},
		{"other loopback", []netip.Addr{netip.MustParseAddr("127.0.0.2")}, nil, ErrAddress},
		{"mapped", []netip.Addr{netip.MustParseAddr("::ffff:127.0.0.1")}, nil, ErrAddress},
		{"invalid", []netip.Addr{{}}, nil, ErrAddress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, dials := newFixtureBroker(t, g, limits(), resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return tc.ips, tc.dnsErr }))
			_, err := b.Fetch(context.Background(), g.Origin+"/?secret=SYNTHETIC")
			if !errors.Is(err, tc.errorWant) || dials.Load() != 0 {
				t.Fatalf("error=%v dials=%d", err, dials.Load())
			}
			if strings.Contains(err.Error(), "SYNTHETIC") || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("error leaked input")
			}
		})
	}
}

func TestAddressGrantsDoNotCrossOrigins(t *testing.T) {
	grants := []Grant{{"http://one.invalid:8080", []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, {"http://two.invalid:8080", []netip.Addr{netip.MustParseAddr("127.0.0.2")}}}
	b, err := NewLab(context.Background(), grants, limits(), fixed(grants[1].Addresses[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.dial = func(context.Context, string, string) (net.Conn, error) {
		t.Error("cross-origin address was dialed")
		return nil, ErrNetwork
	}
	grants[0].Addresses[0] = grants[1].Addresses[0] // Caller mutation must not broaden scope.
	_, err = b.Fetch(context.Background(), grants[0].Origin)
	if !errors.Is(err, ErrAddress) {
		t.Fatal(err)
	}
}

func TestRedirectBudgetAndNoCredentialsOrProxy(t *testing.T) {
	var canaryHits, hits atomic.Int32
	canary := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { canaryHits.Add(1) }))
	defer canary.Close()
	t.Setenv("HTTP_PROXY", canary.URL)
	t.Setenv("HTTPS_PROXY", canary.URL)
	t.Setenv("NO_PROXY", "")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != "GET" || r.Header.Get("Cookie") != "" || r.Header.Get("Referer") != "" || r.Header.Get("Authorization") != "" {
			t.Error("unexpected request metadata")
		}
		switch r.URL.Path {
		case "/same":
			w.Header().Set("Set-Cookie", "secret=SYNTHETIC")
			http.Redirect(w, r, "/final", 302)
		case "/out":
			http.Redirect(w, r, canary.URL, 302)
		case "/loop":
			http.Redirect(w, r, "/loop", 307)
		default:
			_, _ = io.WriteString(w, "fixture")
		}
	}))
	defer s.Close()
	g := fixtureGrant(t, s, "fixture.invalid")
	dns := fixed(g.Addresses[0])
	b, _ := newFixtureBroker(t, g, limits(), dns)
	result, err := b.Fetch(context.Background(), g.Origin+"/same")
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalURL != g.Origin+"/final" {
		t.Fatalf("redirect final URL mismatch: %q", result.FinalURL)
	}
	if b.RequestsUsed() != 2 {
		t.Fatal("redirect did not count")
	}
	if _, err := b.Fetch(context.Background(), g.Origin+"/out"); err == nil {
		t.Fatal("out-of-scope redirect followed")
	}
	if canaryHits.Load() != 0 {
		t.Fatal("canary contacted")
	}
	l := limits()
	l.MaxRequests = 2
	b, _ = newFixtureBroker(t, g, l, dns)
	before := hits.Load()
	if _, err := b.Fetch(context.Background(), g.Origin+"/loop"); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	if hits.Load()-before != 2 || b.RequestsUsed() != 2 {
		t.Fatal("request budget exceeded")
	}
	l = limits()
	l.MaxRedirects = 0
	b, _ = newFixtureBroker(t, g, l, dns)
	if _, err := b.Fetch(context.Background(), g.Origin+"/same"); !errors.Is(err, ErrRedirectLimit) {
		t.Fatal(err)
	}
}

func TestConcurrentBudget(t *testing.T) {
	var hits atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); _, _ = io.WriteString(w, "OK") }))
	defer s.Close()
	g := fixtureGrant(t, s, "")
	l := limits()
	l.MaxRequests = 7
	l.MaxConcurrent = 3
	b, _ := newFixtureBroker(t, g, l, fixed(g.Addresses[0]))
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 30 {
		wg.Go(func() {
			_, err := b.Fetch(context.Background(), g.Origin)
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrBudget) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 7 || hits.Load() != 7 || b.RequestsUsed() != 7 {
		t.Fatalf("overspend: successes=%d hits=%d used=%d", successes.Load(), hits.Load(), b.RequestsUsed())
	}
}

func TestResponseLimits(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/large":
			_, _ = io.WriteString(w, strings.Repeat("x", 33))
		case "/encoded":
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = io.WriteString(w, "not decoded")
		case "/headers":
			w.Header().Set("X-Large", strings.Repeat("x", 40<<10))
		default:
			_, _ = io.WriteString(w, strings.Repeat("x", 32))
		}
	}))
	defer s.Close()
	g := fixtureGrant(t, s, "")
	l := limits()
	l.MaxBodyBytes = 32
	b, _ := newFixtureBroker(t, g, l, fixed(g.Addresses[0]))
	for _, tc := range []struct {
		path string
		want error
	}{{"/ok", nil}, {"/large", ErrBodyLimit}, {"/encoded", ErrEncoding}, {"/headers", ErrNetwork}} {
		r, err := b.Fetch(context.Background(), g.Origin+tc.path)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if err != nil && len(r.Body) != 0 {
			t.Fatal("partial body escaped")
		}
	}
}

func TestTLSHostnameAndTrust(t *testing.T) {
	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "TLS fixture") }))
	s.Config.ErrorLog = log.New(io.Discard, "", 0)
	s.StartTLS()
	defer s.Close()
	roots := x509.NewCertPool()
	roots.AddCert(s.Certificate())
	// httptest's certificate names are read from the actual certificate.
	if len(s.Certificate().DNSNames) == 0 {
		t.Fatal("fixture certificate needs DNS SAN")
	}
	g := fixtureGrant(t, s, s.Certificate().DNSNames[0])
	for _, tc := range []struct {
		name               string
		trusted, wrongHost bool
		want               error
	}{{"trusted", true, false, nil}, {"untrusted", false, false, ErrNetwork}, {"wrong host", true, true, ErrNetwork}} {
		t.Run(tc.name, func(t *testing.T) {
			grant := g
			if tc.wrongHost {
				grant = fixtureGrant(t, s, "wrong.invalid")
			}
			b, _ := newFixtureBroker(t, grant, limits(), fixed(grant.Addresses[0]))
			if tc.trusted {
				b.roots = roots
			}
			_, err := b.Fetch(context.Background(), grant.Origin)
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
		})
	}
}

func TestDNSRevalidatedAfterRedirect(t *testing.T) {
	var hits, lookups atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); http.Redirect(w, r, "/next", 302) }))
	defer s.Close()
	g := fixtureGrant(t, s, "fixture.invalid")
	dns := resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		if lookups.Add(1) == 1 {
			return g.Addresses, nil
		}
		return []netip.Addr{netip.MustParseAddr("169.254.169.254")}, nil
	})
	b, dials := newFixtureBroker(t, g, limits(), dns)
	if _, err := b.Fetch(context.Background(), g.Origin); !errors.Is(err, ErrAddress) {
		t.Fatal(err)
	}
	if lookups.Load() != 2 || hits.Load() != 1 || dials.Load() != 1 || b.RequestsUsed() != 2 {
		t.Fatal("redirect bypassed address validation")
	}
}
