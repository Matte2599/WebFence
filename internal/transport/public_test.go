package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

func publicLimits() Limits {
	return Limits{MaxRequests: 8, MaxConcurrent: 1, MaxRedirects: 2,
		MaxBodyBytes: 1024, RequestTimeout: 3 * time.Second, RunTimeout: 8 * time.Second,
		MinRequestInterval: 100 * time.Millisecond}
}

func publicRoute(t *testing.T, methods []string) scope.RequestPolicy {
	t.Helper()
	p, err := scope.NewRequestPolicy(methods, []string{"/allowed"}, []string{"/allowed/excluded"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func boundFixturePermit(t *testing.T, origin string) project.RunScope {
	t.Helper()
	permit := fixturePermit(t, origin, time.Now().Add(time.Hour))
	bound, err := permit.BindLifecycle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func TestPublicAddressClassifierAndConstructor(t *testing.T) {
	for _, raw := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if !publicDestination(netip.MustParseAddr(raw)) {
			t.Fatalf("rejected public fixture pin %s", raw)
		}
	}
	for _, raw := range []string{"0.1.2.3", "10.1.2.3", "100.64.0.1", "127.0.0.1", "169.254.169.254", "172.16.0.1", "192.0.0.9", "192.0.2.1", "192.168.1.1", "198.18.0.1", "198.51.100.1", "203.0.113.1", "240.0.0.1", "::1", "::ffff:8.8.8.8", "fc00::1", "fe80::1", "2001:db8::1", "2002::1", "3fff::1"} {
		if publicDestination(netip.MustParseAddr(raw)) {
			t.Fatalf("accepted special/private pin %s", raw)
		}
	}
	origin := "http://public.fixture.test:8080"
	permit := boundFixturePermit(t, origin)
	route := publicRoute(t, []string{"GET"})
	for _, raw := range []string{"127.0.0.1", "169.254.169.254", "192.0.2.1", "fc00::1"} {
		ip := netip.MustParseAddr(raw)
		b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{{Origin: origin, Addresses: []netip.Addr{ip}}}, publicLimits(), fixed(ip), route)
		if b != nil || !errors.Is(err, ErrConfig) {
			t.Fatalf("accepted grant %s: %v", raw, err)
		}
	}
	good := Grant{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}
	unmanaged := fixturePermit(t, origin, time.Now().Add(time.Hour))
	if b, err := NewAuthorizedPublic(t.Context(), unmanaged, []Grant{good}, publicLimits(), fixed(good.Addresses[0]), route); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("unmanaged public run accepted: %v", err)
	}
	short := publicLimits()
	short.MinRequestInterval = time.Millisecond
	if b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{good}, short, fixed(good.Addresses[0]), route); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("unbounded rate accepted: %v", err)
	}
	parallel := publicLimits()
	parallel.MaxConcurrent = 2
	if b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{good}, parallel, fixed(good.Addresses[0]), route); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("parallel public requests accepted: %v", err)
	}
	if b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{good}, publicLimits(), fixed(good.Addresses[0]), scope.RequestPolicy{}); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("missing route policy accepted: %v", err)
	}
	if b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{{Origin: "http://other.fixture.test:8080", Addresses: good.Addresses}}, publicLimits(), fixed(good.Addresses[0]), route); b != nil || !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("grant outside authorization accepted: %v", err)
	}
}

type publicPeerConn struct {
	net.Conn
	peer net.Addr
}

func (c publicPeerConn) RemoteAddr() net.Addr { return c.peer }

func TestPublicBrokerUsesPinnedPeerAndChecksRedirects(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/allowed/redirect" {
			http.Redirect(w, r, "/allowed/excluded/action", http.StatusFound)
			return
		}
		if r.URL.Path != "/allowed/page" || r.Host == "" {
			t.Error("unexpected fixture request")
		}
		_, _ = io.WriteString(w, "synthetic-public-response")
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	origin := "http://public.fixture.test:" + u.Port()
	ip := netip.MustParseAddr("8.8.8.8")
	permit := boundFixturePermit(t, origin)
	b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{{Origin: origin, Addresses: []netip.Addr{ip}}}, publicLimits(), fixed(ip), publicRoute(t, []string{"GET"}))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var dials atomic.Int32
	b.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		dials.Add(1)
		if address != net.JoinHostPort(ip.String(), u.Port()) {
			return nil, errors.New("fixture_dial_denied")
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		if err != nil {
			return nil, err
		}
		return publicPeerConn{Conn: conn, peer: &net.TCPAddr{IP: net.ParseIP(ip.String()), Port: mustPort(t, u.Port())}}, nil
	}
	result, err := b.Fetch(t.Context(), origin+"/allowed/page?secret=synthetic")
	if err != nil || string(result.Body) != "synthetic-public-response" || result.FinalURL != origin+"/allowed/page?secret=synthetic" {
		t.Fatalf("pinned public result: status=%d err=%v", result.StatusCode, err)
	}
	_, err = b.Fetch(t.Context(), origin+"/allowed/redirect")
	if !errors.Is(err, scope.ErrPathDenied) || hits.Load() != 2 || dials.Load() != 2 || b.RequestsUsed() != 2 {
		t.Fatalf("excluded redirect reached socket: hits=%d dials=%d requests=%d err=%v", hits.Load(), dials.Load(), b.RequestsUsed(), err)
	}
	if _, err := b.FetchMethod(t.Context(), "HEAD", origin+"/allowed/page"); !errors.Is(err, scope.ErrMethodDenied) || dials.Load() != 2 {
		t.Fatalf("denied HEAD reached socket: %v", err)
	}
	b.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		if err != nil {
			return nil, err
		}
		return publicPeerConn{Conn: conn, peer: &net.TCPAddr{IP: net.ParseIP("1.1.1.1"), Port: mustPort(t, u.Port())}}, nil
	}
	if _, err := b.Fetch(t.Context(), origin+"/allowed/page"); !errors.Is(err, ErrAddress) || hits.Load() != 2 {
		t.Fatalf("ungranted peer reached HTTP: hits=%d err=%v", hits.Load(), err)
	}
}

func mustPort(t *testing.T, raw string) int {
	t.Helper()
	port, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestPublicBrokerRejectsMixedDNSBeforeSocket(t *testing.T) {
	origin := "http://public.fixture.test:8080"
	ip := netip.MustParseAddr("8.8.8.8")
	permit := boundFixturePermit(t, origin)
	b, err := NewAuthorizedPublic(t.Context(), permit, []Grant{{Origin: origin, Addresses: []netip.Addr{ip}}}, publicLimits(), resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{ip, netip.MustParseAddr("169.254.169.254")}, nil
	}), publicRoute(t, []string{"GET"}))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.dial = func(context.Context, string, string) (net.Conn, error) {
		t.Error("mixed DNS reached socket")
		return nil, errors.New("fixture_dial_denied")
	}
	if _, err := b.Fetch(t.Context(), origin+"/allowed/page"); !errors.Is(err, ErrAddress) || b.RequestsUsed() != 1 {
		t.Fatalf("mixed DNS result: %v requests=%d", err, b.RequestsUsed())
	}
}

func TestPacingCountsRedirectsAndCancelsWaitingRequest(t *testing.T) {
	var times []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		times = append(times, time.Now())
		if r.URL.Path == "/allowed/start" {
			http.Redirect(w, r, "/allowed/end", http.StatusFound)
		}
	}))
	defer server.Close()
	grant := fixtureGrant(t, server, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(time.Hour))
	l := limits()
	l.MinRequestInterval = 120 * time.Millisecond
	l.MaxConcurrent = 1
	b, err := NewAuthorizedLabWithPolicy(t.Context(), permit, []Grant{grant}, l, fixed(grant.Addresses[0]), publicRoute(t, []string{"GET"}))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := b.Fetch(t.Context(), grant.Origin+"/allowed/start"); err != nil {
		t.Fatal(err)
	}
	if len(times) != 2 || times[1].Sub(times[0]) < 100*time.Millisecond || b.RequestsUsed() != 2 {
		t.Fatalf("redirect not paced: times=%v requests=%d", times, b.RequestsUsed())
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Millisecond)
	defer cancel()
	if _, err := b.Fetch(ctx, grant.Origin+"/allowed/end"); !errors.Is(err, context.DeadlineExceeded) || len(times) != 2 || b.RequestsUsed() != 3 {
		t.Fatalf("canceled wait reached socket: times=%v requests=%d err=%v", times, b.RequestsUsed(), err)
	}
}

func TestHEADRequiresExplicitPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	grant := fixtureGrant(t, server, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(time.Hour))
	l := limits()
	legacy, err := NewAuthorizedLab(t.Context(), permit, []Grant{grant}, l, fixed(grant.Addresses[0]))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.FetchMethod(t.Context(), http.MethodHead, grant.Origin+"/allowed/page"); !errors.Is(err, scope.ErrMethodDenied) || legacy.RequestsUsed() != 0 {
		t.Fatalf("legacy HEAD: %v", err)
	}
	legacy.Close()
	l.MinRequestInterval = time.Millisecond
	l.MaxConcurrent = 1
	configured, err := NewAuthorizedLabWithPolicy(t.Context(), permit, []Grant{grant}, l, fixed(grant.Addresses[0]), publicRoute(t, []string{"HEAD"}))
	if err != nil {
		t.Fatal(err)
	}
	defer configured.Close()
	result, err := configured.FetchMethod(t.Context(), http.MethodHead, grant.Origin+"/allowed/page")
	if err != nil || result.StatusCode != http.StatusNoContent || configured.RequestsUsed() != 1 {
		t.Fatalf("configured HEAD: %+v err=%v", result, err)
	}
}
