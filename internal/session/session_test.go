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

func TestLoginWithCSRFPreSessionKeepsIdentitiesSeparate(t *testing.T) {
	var mu sync.Mutex
	pre := make(map[string]string)
	var posts atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/private/verify":
			cookie, err := r.Cookie("sid")
			if err != nil || !strings.HasSuffix(cookie.Value, "-session") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, strings.TrimSuffix(cookie.Value, "-session"))
		case r.URL.Path == "/auth/login" && r.Method == http.MethodGet:
			if r.Header.Get("Cookie") != "" {
				t.Error("login page received a cookie")
			}
			mu.Lock()
			id := len(pre) + 1
			value := "pre-" + string(rune('0'+id))
			pre[value] = "token-" + value
			mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: value, Path: "/auth", HttpOnly: true})
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<form><input type="hidden" name="csrf" value="token-`+value+`"></form>`)
		case r.URL.Path == "/auth/login" && r.Method == http.MethodPost:
			posts.Add(1)
			_ = r.ParseForm()
			cookie, err := r.Cookie("sid")
			mu.Lock()
			want := pre[cookieValue(cookie, err)]
			mu.Unlock()
			id := r.Form.Get("username")
			if want == "" || r.Form.Get("csrf") != want || r.Form.Get("password") != id+"-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: id + "-session", Path: "/private", HttpOnly: true})
			w.WriteHeader(http.StatusSeeOther)
		default:
			http.NotFound(w, r)
		}
	})
	manager, broker, _ := fixtureManager(t, handler)
	for _, id := range []string{"alice", "bob"} {
		a := account(id, id+"-ref")
		a.CSRFField, a.CSRFCookieName = "csrf", "sid"
		s, err := manager.Login(context.Background(), a)
		if err != nil {
			t.Fatalf("%s login: %v", id, err)
		}
		if err := s.Verify(context.Background()); err != nil || s.Identity() != id {
			t.Fatalf("%s verification: %v", id, err)
		}
		s.Close()
	}
	if posts.Load() != 2 || broker.RequestsUsed() != 10 {
		t.Fatalf("unexpected POSTs or shared budget: %d, %d", posts.Load(), broker.RequestsUsed())
	}
}

func cookieValue(cookie *http.Cookie, err error) string {
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

func TestCSRFLoginFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, page, cookie string
		status                          int
		wantPost                        bool
	}{
		{"missing", "text/html", `<input type="hidden" name="other" value="x">`, "", 200, false},
		{"duplicate", "text/html", `<input type="hidden" name="csrf" value="a"><input type="hidden" name="csrf" value="b">`, "", 200, false},
		{"wrong-type", "text/html", `<input name="csrf" value="a">`, "", 200, false},
		{"wrong-content-type", "text/plain", `<input type="hidden" name="csrf" value="a">`, "", 200, false},
		{"redirect", "text/html", `<input type="hidden" name="csrf" value="a">`, "", 302, false},
		{"broad-cookie", "text/html", `<input type="hidden" name="csrf" value="a">`, "sid=pre; Domain=127.0.0.1; Path=/auth", 200, false},
		{"unrotated-cookie", "text/html", `<input type="hidden" name="csrf" value="a">`, "sid=pre; Path=/", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var posts, external atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/private/verify":
					w.WriteHeader(http.StatusUnauthorized)
				case "/outside":
					external.Add(1)
				case "/auth/login":
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", tc.contentType)
						if tc.cookie != "" {
							w.Header().Set("Set-Cookie", tc.cookie)
						}
						if tc.status == 302 {
							w.Header().Set("Location", "/outside")
						}
						w.WriteHeader(tc.status)
						_, _ = io.WriteString(w, tc.page)
						return
					}
					posts.Add(1)
					if tc.name == "unrotated-cookie" {
						w.Header().Set("Set-Cookie", "sid=pre; Path=/")
					} else {
						w.Header().Set("Set-Cookie", "sid=pre; Path=/private")
					}
					w.WriteHeader(http.StatusSeeOther)
				default:
					http.NotFound(w, r)
				}
			})
			manager, _, _ := fixtureManager(t, handler)
			a := account("alice", "alice-ref")
			a.CSRFField = "csrf"
			if tc.cookie != "" {
				a.CSRFCookieName = "sid"
			}
			s, err := manager.Login(context.Background(), a)
			if s != nil || err == nil || (posts.Load() != 0) != tc.wantPost || external.Load() != 0 ||
				tc.name == "unrotated-cookie" && !errors.Is(err, ErrLoginInvalid) {
				t.Fatalf("unsafe CSRF login: session=%v err=%v posts=%d outside=%d", s, err, posts.Load(), external.Load())
			}
		})
	}
}
