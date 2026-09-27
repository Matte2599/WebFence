package browser

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/transport"
)

type proxyTestSecrets struct{}

func (proxyTestSecrets) Get(context.Context, string) ([]byte, error) {
	return []byte("synthetic-pass"), nil
}

func TestAuthenticatedProxyKeepsCookieInParent(t *testing.T) {
	var authorizedHits, wrongCookieHits, outsideHits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		outsideHits.Add(1)
	}))
	defer outside.Close()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Authorization") != "" ||
			r.Header.Get("Cookie") == "attacker=browser" {
			wrongCookieHits.Add(1)
		}
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != "POST" || r.Header.Get("Cookie") != "" {
				wrongCookieHits.Add(1)
			}
			_ = r.ParseForm()
			if r.Form.Get("username") != "alice" || r.Form.Get("password") != "synthetic-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "alice-token", Path: "/app", HttpOnly: true})
		case "/app/verify", "/app/data", "/app/redirect":
			if r.Header.Get("Cookie") != "" && r.Header.Get("Cookie") != "sid=alice-token" {
				wrongCookieHits.Add(1)
			}
			cookie, err := r.Cookie("sid")
			if err != nil || cookie.Value != "alice-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			authorizedHits.Add(1)
			if r.URL.Path == "/app/redirect" {
				http.Redirect(w, r, outside.URL+"/secret", http.StatusFound)
				return
			}
			if r.URL.Path == "/app/verify" {
				_, _ = io.WriteString(w, "alice")
			} else {
				w.Header().Set("Set-Cookie", "sid=rotated; Path=/app")
				_, _ = io.WriteString(w, "private fixture")
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer target.Close()
	origin := target.URL
	p, err := project.New(project.Draft{ID: "proxy-session", Name: "Synthetic proxy session",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin}})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, cancel := context.WithCancel(context.Background())
	defer cancel()
	permit, err = permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := NewGate(context.Background(), permit, policy,
		Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRuntime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Close()
	broker, err := transport.NewAuthorizedLabWithSession(context.Background(), permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 30, MaxConcurrent: 1, MaxRedirects: 1,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, net.DefaultResolver, policy,
		transport.SessionRoutes{LoginURL: origin + "/auth/login", VerifyURL: origin + "/app/verify", LoginConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	manager, err := session.NewManager(broker, proxyTestSecrets{})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := manager.Login(context.Background(), session.Account{ID: "alice", Username: "alice",
		SecretID: "alice-ref", UsernameField: "username", PasswordField: "password",
		CookieName: "sid", ExpectedBody: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	defer identity.Close()
	proxy, err := NewObservedProxyWithSession(context.Background(), gate, broker, identity, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	proxyURL, _ := url.Parse(proxy.Endpoint())
	username, password := proxy.Credentials()
	proxyURL.User = url.UserPassword(username, password)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	defer client.CloseIdleConnections()
	req, _ := http.NewRequest("GET", origin+"/app/data?token=synthetic", nil)
	req.Header.Set("Cookie", "attacker=browser")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || string(body) != "private fixture" ||
		response.Header.Get("Set-Cookie") != "" || wrongCookieHits.Load() != 0 || authorizedHits.Load() != 3 {
		t.Fatalf("authenticated proxy result: status=%d body=%q wrong=%d authorized=%d",
			response.StatusCode, body, wrongCookieHits.Load(), authorizedHits.Load())
	}
	response, err = client.Get(origin + "/app/redirect")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadGateway || outsideHits.Load() != 0 {
		t.Fatalf("credential redirect escaped: status=%d outside=%d", response.StatusCode, outsideHits.Load())
	}
	observed, dropped := proxy.Observations()
	if len(observed) != 1 || dropped != 0 || observed[0].Path != "/app/data" ||
		observed[0].FinalPath != "/app/data" {
		t.Fatalf("authenticated observations: %+v dropped=%d", observed, dropped)
	}
	if _, err := NewProxyWithSession(context.Background(), nil, broker, identity); !errors.Is(err, ErrConfig) {
		t.Fatalf("proxy accepted an unbound gate: %v", err)
	}
	otherBroker, err := transport.NewAuthorizedLabWithSession(context.Background(), permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 10, MaxConcurrent: 1, MaxRedirects: 1,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, net.DefaultResolver, policy,
		transport.SessionRoutes{LoginURL: origin + "/auth/login", VerifyURL: origin + "/app/verify", LoginConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	defer otherBroker.Close()
	if _, err := NewProxyWithSession(context.Background(), gate, otherBroker, identity); !errors.Is(err, ErrConfig) {
		t.Fatalf("proxy accepted a session from another run broker: %v", err)
	}
}
