package transport

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestConfigurationRejectsNonLabDestinations(t *testing.T) {
	for _, raw := range []string{"0.0.0.0", "10.0.0.1", "169.254.169.254", "192.0.2.1", "224.0.0.1", "::", "fe80::1", "fc00::1", "2001:db8::1", "::ffff:127.0.0.1", "::1%zone"} {
		g := dummyGrant()
		g.Addresses = []netip.Addr{netip.MustParseAddr(raw)}
		b, err := NewLab(context.Background(), []Grant{g}, limits(), fixed(g.Addresses[0]))
		if b != nil || !errors.Is(err, ErrConfig) {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, change := range []func(*Limits){
		func(l *Limits) { l.MaxRequests = 0 }, func(l *Limits) { l.MaxConcurrent = 0 }, func(l *Limits) { l.MaxConcurrent = 17 },
		func(l *Limits) { l.MaxBodyBytes = 0 }, func(l *Limits) { l.MaxBodyBytes = 1 << 62 }, func(l *Limits) { l.MaxRedirects = -1 },
		func(l *Limits) { l.MaxRedirects = 11 }, func(l *Limits) { l.RequestTimeout = 0 }, func(l *Limits) { l.RunTimeout = 0 },
	} {
		l := limits()
		change(&l)
		if b, err := NewLab(context.Background(), []Grant{dummyGrant()}, l, fixed(netip.MustParseAddr("127.0.0.1"))); b != nil || !errors.Is(err, ErrConfig) {
			t.Fatal("invalid limits accepted")
		}
	}
	for _, grants := range [][]Grant{nil, {{Origin: "http://fixture.invalid:8080"}}, {{Origin: "invalid", Addresses: dummyGrant().Addresses}}, {dummyGrant(), dummyGrant()}} {
		if b, err := NewLab(context.Background(), grants, limits(), fixed(dummyGrant().Addresses[0])); b != nil || !errors.Is(err, ErrConfig) {
			t.Fatal("invalid grants accepted")
		}
	}
	if b, err := NewLab(context.Background(), []Grant{dummyGrant()}, limits(), nil); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatal("nil resolver accepted")
	}
}

func TestIPv6LiteralLoopback(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	s.Listener.Close()
	s.Listener = listener
	s.Start()
	defer s.Close()
	g := fixtureGrant(t, s, "")
	b, dials := newFixtureBroker(t, g, limits(), resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		t.Error("literal IP triggered DNS")
		return nil, ErrResolve
	}))
	r, err := b.Fetch(context.Background(), g.Origin)
	if err != nil || r.StatusCode != 204 || dials.Load() != 1 {
		t.Fatalf("status=%d error=%v dials=%d", r.StatusCode, err, dials.Load())
	}
}

// A trusted dialer should connect exactly where asked. Enforce that invariant
// even when a test double (or a future connector) returns the wrong peer.
type wrongPeerConn struct{ net.Conn }

func (wrongPeerConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.2"), Port: 8080}
}
func TestRejectsUnexpectedConnectedPeer(t *testing.T) {
	g := dummyGrant()
	b, _ := newFixtureBroker(t, g, limits(), fixed(g.Addresses[0]))
	a, z := net.Pipe()
	defer z.Close()
	b.dial = func(context.Context, string, string) (net.Conn, error) { return wrongPeerConn{a}, nil }
	if _, err := b.Fetch(context.Background(), g.Origin); !errors.Is(err, ErrAddress) {
		t.Fatal(err)
	}
	if _, err := z.Write([]byte("x")); err == nil {
		t.Fatal("unexpected connection remained open")
	}
}
