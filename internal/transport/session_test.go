package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/scope"
)

func TestSessionBrokerExactLoginAndNoCredentialRedirect(t *testing.T) {
	var outsideHits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		outsideHits.Add(1)
	}))
	defer outside.Close()
	var loginHits, privateHits atomic.Int32
	inside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			loginHits.Add(1)
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" ||
				string(body) != "username=alice&password=synthetic" || r.Header.Get("Cookie") != "" {
				t.Error("login request changed or leaked a cookie")
			}
			w.Header().Set("Set-Cookie", "sid=synthetic-token; Path=/; HttpOnly")
			http.Redirect(w, r, outside.URL+"/stolen", http.StatusFound)
		case "/private/verify":
			privateHits.Add(1)
			if r.Method != "GET" || r.Header.Get("Cookie") != "sid=synthetic-token" {
				t.Error("session request lacked its exact cookie")
			}
			_, _ = io.WriteString(w, "alice")
		default:
			http.NotFound(w, r)
		}
	}))
	defer inside.Close()
	grant := fixtureGrant(t, inside, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(time.Hour))
	lifecycle, cancel := context.WithCancel(context.Background())
	defer cancel()
	permit, err := permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	route, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/private"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	lim := limits()
	lim.MaxConcurrent = 1
	lim.MinRequestInterval = time.Millisecond
	b, err := NewAuthorizedLabWithSession(context.Background(), permit, []Grant{grant}, lim,
		fixed(grant.Addresses[0]), route, SessionRoutes{
			LoginURL: grant.Origin + "/auth/login", VerifyURL: grant.Origin + "/private/verify", LoginConfirmed: true,
		})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := b.LoginForm(context.Background(), grant.Origin+"/auth/other", []byte("x=y")); !errors.Is(err, ErrSessionRoute) || b.RequestsUsed() != 0 {
		t.Fatalf("wrong login path admitted: %v, attempts=%d", err, b.RequestsUsed())
	}
	result, err := b.LoginForm(context.Background(), grant.Origin+"/auth/login",
		[]byte("username=alice&password=synthetic"))
	if err != nil || result.StatusCode != http.StatusFound || len(result.Header.Values("Set-Cookie")) != 1 ||
		outsideHits.Load() != 0 || loginHits.Load() != 1 || b.RequestsUsed() != 1 {
		t.Fatalf("login redirect handling: status=%d err=%v outside=%d login=%d attempts=%d",
			result.StatusCode, err, outsideHits.Load(), loginHits.Load(), b.RequestsUsed())
	}
	if _, err := b.FetchSession(context.Background(), outside.URL+"/stolen", "sid=synthetic-token"); !errors.Is(err, scope.ErrOutOfScope) || outsideHits.Load() != 0 {
		t.Fatalf("cross-origin cookie escaped: %v, outside=%d", err, outsideHits.Load())
	}
	if _, err := b.FetchSession(context.Background(), grant.Origin+"/auth/login", "sid=synthetic-token"); !errors.Is(err, ErrSessionRoute) {
		t.Fatalf("out-of-route cookie admitted: %v", err)
	}
	result, err = b.FetchSession(context.Background(), grant.Origin+"/private/verify", "sid=synthetic-token")
	if err != nil || result.StatusCode != http.StatusOK || string(result.Body) != "alice" ||
		outsideHits.Load() != 0 || privateHits.Load() != 1 || b.RequestsUsed() != 2 {
		t.Fatalf("session verification: status=%d err=%v outside=%d private=%d attempts=%d",
			result.StatusCode, err, outsideHits.Load(), privateHits.Load(), b.RequestsUsed())
	}
}

func TestSessionBrokerRejectsInvalidRoutes(t *testing.T) {
	inside := httptest.NewServer(http.NotFoundHandler())
	defer inside.Close()
	grant := fixtureGrant(t, inside, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(time.Hour))
	route, _ := scope.NewRequestPolicy([]string{"GET"}, []string{"/private"}, nil)
	lim := limits()
	lim.MaxConcurrent = 1
	lim.MinRequestInterval = time.Millisecond
	if b, err := NewAuthorizedLabWithSession(context.Background(), permit, []Grant{grant}, lim,
		fixed(grant.Addresses[0]), route, SessionRoutes{LoginURL: grant.Origin + "/auth", VerifyURL: grant.Origin + "/private", LoginConfirmed: true}); b != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("unmanaged session permit accepted: broker=%v err=%v", b, err)
	}
	lifecycle, cancel := context.WithCancel(context.Background())
	defer cancel()
	permit, err := permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	for name, routes := range map[string]SessionRoutes{
		"login query":   {LoginURL: grant.Origin + "/auth?token=secret", VerifyURL: grant.Origin + "/private", LoginConfirmed: true},
		"encoded path":  {LoginURL: grant.Origin + "/auth%2flogin", VerifyURL: grant.Origin + "/private", LoginConfirmed: true},
		"verify denied": {LoginURL: grant.Origin + "/auth", VerifyURL: grant.Origin + "/outside", LoginConfirmed: true},
		"unconfirmed":   {LoginURL: grant.Origin + "/auth", VerifyURL: grant.Origin + "/private"},
	} {
		t.Run(name, func(t *testing.T) {
			b, err := NewAuthorizedLabWithSession(context.Background(), permit, []Grant{grant}, lim,
				fixed(grant.Addresses[0]), route, routes)
			if b != nil || !errors.Is(err, ErrConfig) {
				t.Fatalf("invalid session routes accepted: broker=%v err=%v", b, err)
			}
		})
	}
}

func TestSessionLoginStopsOnRunRevocation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth" {
			close(started)
			select {
			case <-r.Context().Done():
			case <-release:
			}
		}
	}))
	defer server.Close()
	defer close(release)
	grant := fixtureGrant(t, server, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(time.Hour))
	lifecycle, revoke := context.WithCancel(context.Background())
	defer revoke()
	var err error
	permit, err = permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	route, _ := scope.NewRequestPolicy([]string{"GET"}, []string{"/private"}, nil)
	lim := limits()
	lim.MaxConcurrent = 1
	lim.MinRequestInterval = time.Millisecond
	lim.RequestTimeout = 3 * time.Second
	b, err := NewAuthorizedLabWithSession(context.Background(), permit, []Grant{grant}, lim,
		fixed(grant.Addresses[0]), route, SessionRoutes{LoginURL: grant.Origin + "/auth", VerifyURL: grant.Origin + "/private", LoginConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	done := make(chan error, 1)
	go func() {
		_, err := b.LoginForm(context.Background(), grant.Origin+"/auth", []byte("password=synthetic"))
		done <- err
	}()
	<-started
	revoke()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || b.RequestsUsed() != 1 {
			t.Fatalf("revoked login result: err=%v attempts=%d", err, b.RequestsUsed())
		}
	case <-time.After(time.Second):
		t.Fatal("revoked login stayed in flight")
	}
}
