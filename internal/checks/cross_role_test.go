package checks

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/transport"
)

type syntheticSecrets struct{}

func (syntheticSecrets) Get(context.Context, string) ([]byte, error) {
	return []byte("synthetic-pass"), nil
}

type accessMode string

const (
	vulnerable accessMode = "vulnerable"
	protected  accessMode = "protected"
	public     accessMode = "public"
	public403  accessMode = "public_403"
	ownerBad   accessMode = "owner_bad"
)

func accessFixture(t *testing.T, mode accessMode) (*transport.Broker, *session.Session, *session.Session, string) {
	t.Helper()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := ""
		if cookie, err := r.Cookie("sid"); err == nil {
			identity = cookie.Value
		}
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != http.MethodPost || identity != "" || r.ParseForm() != nil ||
				r.Form.Get("password") != "synthetic-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			identity = r.Form.Get("username")
			if identity != "alice" && identity != "bob" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: identity, Path: "/app", HttpOnly: true})
		case "/app/verify":
			if identity == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(identity))
		case "/app/private":
			switch identity {
			case "alice":
				if mode == ownerBad {
					_, _ = w.Write([]byte("not the expected private body"))
				} else {
					_, _ = w.Write([]byte("private-alice"))
				}
			case "bob":
				if mode == vulnerable {
					_, _ = w.Write([]byte("private-alice"))
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			default:
				if mode == public || mode == public403 {
					if mode == public403 {
						w.WriteHeader(http.StatusForbidden)
					}
					_, _ = w.Write([]byte("private-alice"))
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(target.Close)
	p, err := project.New(project.Draft{ID: "cross-role", Name: "Synthetic access fixture",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{target.URL}})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	permit, err = permit.BindLifecycle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	broker, err := transport.NewAuthorizedLabWithSession(context.Background(), permit,
		[]transport.Grant{{Origin: target.URL, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 30, MaxConcurrent: 1, MaxRedirects: 1,
			MaxBodyBytes: 4096, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, net.DefaultResolver, policy,
		transport.SessionRoutes{LoginURL: target.URL + "/auth/login",
			VerifyURL: target.URL + "/app/verify", LoginConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(broker.Close)
	manager, err := session.NewManager(broker, syntheticSecrets{})
	if err != nil {
		t.Fatal(err)
	}
	login := func(id string) *session.Session {
		identity, err := manager.Login(context.Background(), session.Account{ID: id, Username: id,
			SecretID: id + "-secret", UsernameField: "username", PasswordField: "password",
			CookieName: "sid", ExpectedBody: id})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(identity.Close)
		return identity
	}
	return broker, login("alice"), login("bob"), target.URL + "/app/private"
}

func TestCrossRolePositiveEvidenceAndInconclusiveCases(t *testing.T) {
	for _, tt := range []struct {
		mode accessMode
		want Outcome
		code string
	}{
		{vulnerable, Finding, "cross_role_private_body_reproduced"},
		{protected, Inconclusive, "other_resource_unverified"},
		{public, Inconclusive, "private_marker_public"},
		{public403, Inconclusive, "private_marker_public"},
		{ownerBad, Inconclusive, "owner_resource_unverified"},
	} {
		t.Run(string(tt.mode), func(t *testing.T) {
			broker, owner, other, resource := accessFixture(t, tt.mode)
			got, err := CheckCrossRole(context.Background(), broker, owner, other,
				CrossRolePlan{ResourceURL: resource, PrivateBody: "private-alice",
					ResourceConfirmed: true, OtherForbiddenConfirmed: true})
			if err != nil || got.RuleID != CrossRoleRuleID || got.RuleRevision != CrossRoleRuleRevision ||
				got.Outcome != tt.want || got.EvidenceCode != tt.code {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
}

func TestCrossRoleRequiresDistinctBoundConfirmedIdentities(t *testing.T) {
	broker, owner, other, resource := accessFixture(t, vulnerable)
	base := CrossRolePlan{ResourceURL: resource, PrivateBody: "private-alice",
		ResourceConfirmed: true, OtherForbiddenConfirmed: true}
	for _, tt := range []struct {
		name         string
		owner, other *session.Session
		plan         CrossRolePlan
	}{
		{"same identity", owner, owner, base},
		{"resource unconfirmed", owner, other, CrossRolePlan{ResourceURL: resource,
			PrivateBody: "private-alice", OtherForbiddenConfirmed: true}},
		{"access policy unconfirmed", owner, other, CrossRolePlan{ResourceURL: resource,
			PrivateBody: "private-alice", ResourceConfirmed: true}},
		{"empty marker", owner, other, CrossRolePlan{ResourceURL: resource,
			ResourceConfirmed: true, OtherForbiddenConfirmed: true}},
		{"outside origin", owner, other, CrossRolePlan{ResourceURL: "http://outside.test/private",
			PrivateBody: "private-alice", ResourceConfirmed: true, OtherForbiddenConfirmed: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CheckCrossRole(context.Background(), broker, tt.owner, tt.other, tt.plan); !errors.Is(err, ErrCrossRolePlan) {
				t.Fatalf("invalid plan: %v", err)
			}
		})
	}
	other.Close()
	got, err := CheckCrossRole(context.Background(), broker, owner, other, base)
	if err != nil || got.Outcome != Inconclusive || got.EvidenceCode != "test_session_unverified" {
		t.Fatalf("closed session: %+v %v", got, err)
	}
}
