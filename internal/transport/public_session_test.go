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
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/scope"
)

func publicSessionLimits() Limits {
	l := publicLimits()
	l.MinRequestInterval = 500 * time.Millisecond
	return l
}

func TestPublicSessionRequiresSeparateHTTPSConfirmation(t *testing.T) {
	ip := netip.MustParseAddr("8.8.8.8")
	for _, tc := range []struct {
		name, origin string
		confirmed    bool
		limit        func(*Limits)
	}{
		{"missing public confirmation", "https://public.fixture.test:443", false, nil},
		{"cleartext", "http://public.fixture.test:80", true, nil},
		{"too many attempts", "https://public.fixture.test:443", true, func(l *Limits) { l.MaxRequests = 129 }},
		{"too long", "https://public.fixture.test:443", true, func(l *Limits) { l.RunTimeout = 16 * time.Minute }},
		{"too fast", "https://public.fixture.test:443", true, func(l *Limits) { l.MinRequestInterval = 100 * time.Millisecond }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits := publicSessionLimits()
			if tc.limit != nil {
				tc.limit(&limits)
			}
			origin := tc.origin
			b, err := NewAuthorizedPublicWithSession(t.Context(), boundFixturePermit(t, origin),
				[]Grant{{Origin: origin, Addresses: []netip.Addr{ip}}}, limits, fixed(ip),
				publicRoute(t, []string{"GET"}), SessionRoutes{LoginURL: origin + "/login",
					VerifyURL: origin + "/allowed/verify", LoginConfirmed: true, PublicConfirmed: tc.confirmed})
			if b != nil || !errors.Is(err, ErrConfig) {
				t.Fatalf("unsafe public session accepted: broker=%v err=%v", b, err)
			}
		})
	}
}

func TestPublicSessionPinnedTLSAndNoCredentialRedirect(t *testing.T) {
	var loginHits, verifyHits, outsideHits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { outsideHits.Add(1) }))
	defer outside.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			loginHits.Add(1)
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || string(body) != "username=alice&password=synthetic" || r.Header.Get("Cookie") != "" {
				t.Error("unexpected login request")
			}
			w.Header().Set("Set-Cookie", "sid=synthetic; Path=/; Secure; HttpOnly")
			http.Redirect(w, r, outside.URL+"/stolen", http.StatusFound)
		case "/allowed/verify":
			verifyHits.Add(1)
			if r.Method != http.MethodGet || r.Header.Get("Cookie") != "sid=synthetic" {
				t.Error("unexpected verification request")
			}
			_, _ = io.WriteString(w, "alice")
		default:
			http.NotFound(w, r)
		}
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil || len(server.Certificate().DNSNames) == 0 {
		t.Fatal("TLS fixture lacks DNS name")
	}
	origin := "https://" + net.JoinHostPort(server.Certificate().DNSNames[0], u.Port())
	ip := netip.MustParseAddr("8.8.8.8")
	grant := Grant{Origin: origin, Addresses: []netip.Addr{ip}}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/allowed"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewAuthorizedPublicWithSession(t.Context(), boundFixturePermit(t, origin), []Grant{grant},
		publicSessionLimits(), fixed(ip), policy, SessionRoutes{
			LoginURL: origin + "/login", VerifyURL: origin + "/allowed/verify",
			LoginConfirmed: true, PublicConfirmed: true,
		})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	b.roots = roots
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
	if _, err := b.LoginForm(t.Context(), origin+"/other", []byte("password=synthetic")); !errors.Is(err, ErrSessionRoute) || dials.Load() != 0 {
		t.Fatalf("unconfirmed route reached socket: %v", err)
	}
	login, err := b.LoginForm(t.Context(), origin+"/login", []byte("username=alice&password=synthetic"))
	if err != nil || login.StatusCode != http.StatusFound || loginHits.Load() != 1 || outsideHits.Load() != 0 {
		t.Fatalf("login result: status=%d err=%v inside=%d outside=%d", login.StatusCode, err, loginHits.Load(), outsideHits.Load())
	}
	if _, err := b.FetchSession(t.Context(), outside.URL+"/stolen", "sid=synthetic"); !errors.Is(err, scope.ErrOutOfScope) || outsideHits.Load() != 0 {
		t.Fatalf("cookie escaped origin: %v", err)
	}
	verified, err := b.FetchSession(t.Context(), origin+"/allowed/verify", "sid=synthetic")
	if err != nil || verified.StatusCode != http.StatusOK || string(verified.Body) != "alice" ||
		verifyHits.Load() != 1 || dials.Load() != 2 || b.RequestsUsed() != 2 {
		t.Fatalf("verification: status=%d err=%v hits=%d dials=%d attempts=%d", verified.StatusCode, err, verifyHits.Load(), dials.Load(), b.RequestsUsed())
	}
	b.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		if err != nil {
			return nil, err
		}
		return publicPeerConn{Conn: conn, peer: &net.TCPAddr{IP: net.ParseIP("1.1.1.1"), Port: mustPort(t, u.Port())}}, nil
	}
	if _, err := b.FetchSession(t.Context(), origin+"/allowed/verify", "sid=synthetic"); !errors.Is(err, ErrAddress) || verifyHits.Load() != 1 {
		t.Fatalf("ungranted peer reached server: %v", err)
	}
}
