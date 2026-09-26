package session

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

type testSecrets map[string][]byte

func (s testSecrets) Get(_ context.Context, id string) ([]byte, error) {
	value, ok := s[id]
	if !ok {
		return nil, errors.New("missing synthetic secret")
	}
	return append([]byte(nil), value...), nil
}

type loopbackResolver struct{}

func (loopbackResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func fixtureManager(t *testing.T, handler http.Handler) (*Manager, *transport.Broker, string) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.New(project.Draft{ID: "synthetic-project", Name: "Synthetic project",
		TargetOwner: "Fixture owner", AuthorizationReference: "fixture-only",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	permit, err = permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	route, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	broker, err := transport.NewAuthorizedLabWithSession(context.Background(), permit,
		[]transport.Grant{{Origin: server.URL, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 30, MaxConcurrent: 1, MaxRedirects: 1,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, loopbackResolver{}, route,
		transport.SessionRoutes{LoginURL: server.URL + "/auth/login", VerifyURL: server.URL + "/private/verify", LoginConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(broker.Close)
	if u.Scheme != "http" {
		t.Fatal("fixture must be HTTP")
	}
	manager, err := NewManager(broker, testSecrets{"alice-ref": []byte("alice-pass"),
		"bob-ref": []byte("bob-pass"), "bad-ref": []byte("bad-pass")})
	if err != nil {
		t.Fatal(err)
	}
	return manager, broker, server.URL
}

func account(id, passwordRef string) Account {
	return Account{ID: id, Username: id, SecretID: passwordRef,
		UsernameField: "username", PasswordField: "password",
		CookieName: "sid", ExpectedBody: id}
}

func TestLoginIsolatesTwoTestIdentities(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != "POST" || r.Header.Get("Cookie") != "" {
				t.Error("login was not a cookie-free POST")
			}
			_ = r.ParseForm()
			id := r.Form.Get("username")
			if r.Form.Get("password") != id+"-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: id + "-token", Path: "/private", HttpOnly: true})
			w.WriteHeader(http.StatusSeeOther)
		case "/private/verify", "/private/data":
			cookie, err := r.Cookie("sid")
			if err != nil || !strings.HasSuffix(cookie.Value, "-token") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			id := strings.TrimSuffix(cookie.Value, "-token")
			mu.Lock()
			seen = append(seen, id)
			mu.Unlock()
			_, _ = io.WriteString(w, id)
		default:
			http.NotFound(w, r)
		}
	})
	manager, broker, base := fixtureManager(t, handler)
	alice, err := manager.Login(context.Background(), account("alice", "alice-ref"))
	if err != nil {
		t.Fatal(err)
	}
	defer alice.Close()
	bob, err := manager.Login(context.Background(), account("bob", "bob-ref"))
	if err != nil {
		t.Fatal(err)
	}
	defer bob.Close()
	for _, item := range []struct {
		s    *Session
		want string
	}{{alice, "alice"}, {bob, "bob"}} {
		if err := item.s.Verify(context.Background()); err != nil {
			t.Fatal(err)
		}
		result, err := item.s.Fetch(context.Background(), base+"/private/data")
		if err != nil || string(result.Body) != item.want || item.s.Identity() != item.want ||
			item.s.ProjectID() != "synthetic-project" || item.s.Revision() != 1 {
			t.Fatalf("identity separation: body=%q identity=%s err=%v", result.Body, item.s.Identity(), err)
		}
	}
	if broker.RequestsUsed() != 10 {
		t.Fatalf("unexpected shared request budget: %d", broker.RequestsUsed())
	}
	if _, err := alice.Fetch(context.Background(), base+"/auth/login"); !errors.Is(err, ErrInconclusive) ||
		broker.RequestsUsed() != 10 {
		t.Fatalf("cookie path widened: err=%v attempts=%d", err, broker.RequestsUsed())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 6 || seen[0] != "alice" || seen[1] != "bob" || seen[5] != "bob" {
		t.Fatalf("unexpected identity sequence: %v", seen)
	}
	alice.Close()
	if _, err := alice.Fetch(context.Background(), base+"/private/data"); !errors.Is(err, ErrExpired) {
		t.Fatalf("closed session remained usable: %v", err)
	}
}

func TestLoginFailureNeverCreatesSession(t *testing.T) {
	var privateHits atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/private/verify" {
			privateHits.Add(1)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	})
	manager, _, _ := fixtureManager(t, handler)
	s, err := manager.Login(context.Background(), account("alice", "bad-ref"))
	if s != nil || !errors.Is(err, ErrLoginInvalid) || privateHits.Load() != 1 {
		t.Fatalf("failed login became a session: session=%v err=%v probes=%d", s, err, privateHits.Load())
	}
}

func TestLoginRejectsDomainCookieAndPublicMarker(t *testing.T) {
	for _, mode := range []string{"domain", "public"} {
		t.Run(mode, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/private/verify" && mode == "public" {
					_, _ = io.WriteString(w, "alice")
					return
				}
				if r.URL.Path == "/auth/login" {
					w.Header().Set("Set-Cookie", "sid=synthetic; Domain=127.0.0.1; Path=/")
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
			})
			manager, _, _ := fixtureManager(t, handler)
			s, err := manager.Login(context.Background(), account("alice", "alice-ref"))
			if s != nil || (mode == "public" && !errors.Is(err, ErrInconclusive)) ||
				(mode == "domain" && !errors.Is(err, ErrLoginInvalid)) {
				t.Fatalf("unsafe login accepted: session=%v err=%v", s, err)
			}
		})
	}
}
