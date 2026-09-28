package checks

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
)

type runSecrets map[string]string

type runResolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f runResolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func (s runSecrets) Get(_ context.Context, id string) ([]byte, error) {
	if secret := s[id]; secret != "" {
		return []byte(secret), nil
	}
	return nil, errors.New("missing synthetic test credential")
}

func crossRoleRunFixture(t *testing.T, vulnerable bool, rotate ...bool) (*storage.Store, CrossRoleRunPlan, *atomic.Int32) {
	t.Helper()
	rotating := len(rotate) != 0 && rotate[0]
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		identity, stage := "", ""
		if cookie, err := r.Cookie("sid"); err == nil {
			identity = cookie.Value
			if rotating {
				identity, stage, _ = strings.Cut(cookie.Value, "-")
			}
		}
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != http.MethodPost || identity != "" || r.ParseForm() != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			name := r.Form.Get("username")
			if (name != "alice" && name != "bob") || r.Form.Get("password") != name+"-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			value := name
			if rotating {
				value += "-login"
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: value, Path: "/app", HttpOnly: true})
		case "/app/verify":
			if identity == "" || rotating && stage != "login" && stage != "ready" && stage != "data" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if rotating && stage == "login" {
				http.SetCookie(w, &http.Cookie{Name: "sid", Value: identity + "-ready", Path: "/app", HttpOnly: true})
			}
			_, _ = w.Write([]byte(identity))
		case "/app/private":
			if rotating {
				if stage != "ready" && stage != "data" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if stage == "ready" {
					http.SetCookie(w, &http.Cookie{Name: "sid", Value: identity + "-data", Path: "/app", HttpOnly: true})
				}
			}
			switch identity {
			case "alice":
				_, _ = w.Write([]byte("private-alice"))
			case "bob":
				if vulnerable {
					_, _ = w.Write([]byte("private-alice"))
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	store, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "auth-run.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	p, err := project.New(project.Draft{ID: "cross-role-run", Name: "Synthetic cross-role run",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic local permission",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	account := func(name string) session.Account {
		return session.Account{ID: name, Username: name, SecretID: name + "-ref",
			UsernameField: "username", PasswordField: "password", CookieName: "sid", ExpectedBody: name}
	}
	return store, CrossRoleRunPlan{ProjectID: p.ID(), Origin: server.URL,
		Grant: transport.Grant{Origin: server.URL, Addresses: []netip.Addr{ip}}, Policy: policy,
		Limits: transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRedirects: 0,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, Resolver: net.DefaultResolver,
		Routes: transport.SessionRoutes{LoginURL: server.URL + "/auth/login", VerifyURL: server.URL + "/app/verify", LoginConfirmed: true},
		Owner:  account("alice"), Other: account("bob"),
		Check: CrossRolePlan{ResourceURL: server.URL + "/app/private", PrivateBody: "private-alice",
			ResourceConfirmed: true, OtherForbiddenConfirmed: true}}, &hits
}

func TestRunCrossRoleManagedCookieRotation(t *testing.T) {
	store, plan, hits := crossRoleRunFixture(t, true, true)
	got, err := RunCrossRole(t.Context(), store,
		runSecrets{"alice-ref": "alice-pass", "bob-ref": "bob-pass"}, plan)
	if err != nil || got.Outcome != Finding || got.EvidenceCode != "cross_role_private_body_reproduced" ||
		hits.Load() != 15 {
		t.Fatalf("rotated identities in managed check: %+v hits=%d err=%v", got, hits.Load(), err)
	}
}

func TestRunCrossRoleManagedPositiveAndDenied(t *testing.T) {
	for _, tc := range []struct {
		vulnerable bool
		want       Outcome
	}{
		{true, Finding}, {false, Inconclusive},
	} {
		store, plan, hits := crossRoleRunFixture(t, tc.vulnerable)
		got, err := RunCrossRole(t.Context(), store, runSecrets{"alice-ref": "alice-pass", "bob-ref": "bob-pass"}, plan)
		if err != nil || got.Outcome != tc.want || got.RuleID != CrossRoleRuleID || hits.Load() != 11 {
			t.Fatalf("managed check: %+v hits=%d err=%v", got, hits.Load(), err)
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{plan.Origin, "private-alice", "alice-pass", "bob-pass", "alice-ref"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("result retained private input: %s", encoded)
			}
		}
	}
}

func TestRunCrossRoleFailsClosedBeforeOrDuringLogin(t *testing.T) {
	store, plan, hits := crossRoleRunFixture(t, true)
	for _, alter := range []func(*CrossRoleRunPlan){
		func(p *CrossRoleRunPlan) { p.Check.OtherForbiddenConfirmed = false },
		func(p *CrossRoleRunPlan) { p.Other.ExpectedBody = p.Owner.ExpectedBody },
		func(p *CrossRoleRunPlan) { p.Other.SecretID = p.Owner.SecretID },
		func(p *CrossRoleRunPlan) { p.Check.ResourceURL = "http://outside.test/private" },
	} {
		changed := plan
		alter(&changed)
		if _, err := RunCrossRole(t.Context(), store, runSecrets{}, changed); err == nil || hits.Load() != 0 {
			t.Fatalf("invalid plan reached fixture: hits=%d err=%v", hits.Load(), err)
		}
	}
	got, err := RunCrossRole(t.Context(), store, runSecrets{"alice-ref": "alice-pass"}, plan)
	if err != nil || got.Outcome != Inconclusive || got.EvidenceCode != "other_login_unverified" || hits.Load() != 4 {
		t.Fatalf("missing other identity became a result: %+v hits=%d err=%v", got, hits.Load(), err)
	}
}

func TestRunCrossRoleBudgetCannotProduceFinding(t *testing.T) {
	store, plan, hits := crossRoleRunFixture(t, true)
	plan.Limits.MaxRequests = 8
	got, err := RunCrossRole(t.Context(), store, runSecrets{"alice-ref": "alice-pass", "bob-ref": "bob-pass"}, plan)
	if err != nil || got.Outcome != Inconclusive || got.EvidenceCode != "anonymous_baseline_unavailable" || hits.Load() != 8 {
		t.Fatalf("budget exhaustion: %+v hits=%d err=%v", got, hits.Load(), err)
	}
}

func TestRunCrossRolePublicModeRequiresExplicitGrantAndConfirmation(t *testing.T) {
	origin := "https://public.fixture.test:443"
	store, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "public-auth-run.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	p, err := project.New(project.Draft{ID: "public-auth-run", Name: "Synthetic public authorization",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic local permission",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var lookups atomic.Int32
	resolver := runResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		lookups.Add(1)
		return nil, errors.New("synthetic DNS failure")
	})
	account := func(name string) session.Account {
		return session.Account{ID: name, Username: name, SecretID: name + "-ref",
			UsernameField: "username", PasswordField: "password", CookieName: "sid", ExpectedBody: name}
	}
	plan := CrossRoleRunPlan{ProjectID: p.ID(), Mode: CrossRoleRunPinnedPublic, Origin: origin,
		Grant:  transport.Grant{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}},
		Policy: policy, Limits: transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRedirects: 0,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: 500 * time.Millisecond}, Resolver: resolver,
		Routes: transport.SessionRoutes{LoginURL: origin + "/auth/login", VerifyURL: origin + "/app/verify",
			LoginConfirmed: true, PublicConfirmed: true},
		Owner: account("alice"), Other: account("bob"),
		Check: CrossRolePlan{ResourceURL: origin + "/app/private", PrivateBody: "private-alice",
			ResourceConfirmed: true, OtherForbiddenConfirmed: true}}
	for _, alter := range []func(*CrossRoleRunPlan){
		func(p *CrossRoleRunPlan) { p.Routes.PublicConfirmed = false },
		func(p *CrossRoleRunPlan) { p.Mode = "unknown" },
		func(p *CrossRoleRunPlan) { p.Mode = CrossRoleRunLoopback },
		func(p *CrossRoleRunPlan) { p.Origin = "http://public.fixture.test:443" },
		func(p *CrossRoleRunPlan) { p.Grant.Addresses = []netip.Addr{netip.MustParseAddr("127.0.0.1")} },
	} {
		invalid := plan
		alter(&invalid)
		if _, err := RunCrossRole(t.Context(), store, runSecrets{}, invalid); !errors.Is(err, ErrCrossRoleRunPlan) || lookups.Load() != 0 {
			t.Fatalf("invalid public plan reached DNS: err=%v lookups=%d", err, lookups.Load())
		}
	}
	invalidPin := plan
	invalidPin.Grant.Addresses = []netip.Addr{netip.MustParseAddr("192.0.2.1")}
	if _, err := RunCrossRole(t.Context(), store, runSecrets{}, invalidPin); !errors.Is(err, transport.ErrConfig) || lookups.Load() != 0 {
		t.Fatalf("special IP reached DNS: err=%v lookups=%d", err, lookups.Load())
	}
	got, err := RunCrossRole(t.Context(), store, runSecrets{}, plan)
	if err != nil || got.Outcome != Inconclusive || got.EvidenceCode != "owner_login_unverified" || lookups.Load() != 1 {
		t.Fatalf("public managed mode did not fail closed on DNS: %+v err=%v lookups=%d", got, err, lookups.Load())
	}
}
