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
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

func fixturePermit(t *testing.T, origin string, expiresAt time.Time) project.RunScope {
	t.Helper()
	p, err := project.New(project.Draft{
		ID: "synthetic-project", Name: "Synthetic staging", TargetOwner: "Fixture owner",
		AuthorizationReference: "fixture-approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: expiresAt, Origins: []string{origin},
	})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	return permit
}

func TestAuthorizedLabChecksEachRedirectAgainstProject(t *testing.T) {
	var outsideHits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outsideHits.Add(1)
		_, _ = io.WriteString(w, "must not be read")
	}))
	defer outside.Close()
	inside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, outside.URL+"/forbidden", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "fixture")
	}))
	defer inside.Close()
	insideGrant := fixtureGrant(t, inside, "")
	outsideGrant := fixtureGrant(t, outside, "")
	permit := fixturePermit(t, insideGrant.Origin, time.Now().Add(time.Hour))
	b, err := NewAuthorizedLab(context.Background(), permit, []Grant{insideGrant, outsideGrant}, limits(), fixed(insideGrant.Addresses[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	// A second, test-only boundary prevents accidental dialing beyond the two
	// owned loopback fixtures even if the broker under test regresses.
	endpoints := map[string]bool{}
	for _, grant := range []Grant{insideGrant, outsideGrant} {
		u, _ := url.Parse(grant.Origin)
		endpoints[net.JoinHostPort(grant.Addresses[0].String(), u.Port())] = true
	}
	var dials atomic.Int32
	b.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		dials.Add(1)
		if !endpoints[address] {
			return nil, errors.New("fixture_dial_denied")
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	got, err := b.Fetch(context.Background(), inside.URL+"/allowed")
	if err != nil || string(got.Body) != "fixture" {
		t.Fatalf("authorized fixture fetch: body=%q err=%v", got.Body, err)
	}
	_, err = b.Fetch(context.Background(), inside.URL+"/redirect")
	if !errors.Is(err, scope.ErrOutOfScope) || outsideHits.Load() != 0 || dials.Load() != 2 || b.RequestsUsed() != 2 {
		t.Fatalf("project scope failed on redirect: err=%v outside=%d dials=%d budget=%d", err, outsideHits.Load(), dials.Load(), b.RequestsUsed())
	}
}

func TestAuthorizedLabDeniesZeroPermit(t *testing.T) {
	g := dummyGrant()
	b, err := NewAuthorizedLab(context.Background(), project.RunScope{}, []Grant{g}, limits(), fixed(netip.MustParseAddr("127.0.0.1")))
	if b != nil || !errors.Is(err, project.ErrInvalidProject) {
		t.Fatalf("zero permit accepted: broker=%v err=%v", b, err)
	}
}

func TestAuthorizedLabCancelsInFlightAtAuthorizationExpiry(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	grant := fixtureGrant(t, server, "")
	permit := fixturePermit(t, grant.Origin, time.Now().Add(2*time.Second))
	l := limits()
	l.RequestTimeout = 5 * time.Second
	b, err := NewAuthorizedLab(context.Background(), permit, []Grant{grant}, l, fixed(grant.Addresses[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	_, err = b.Fetch(context.Background(), grant.Origin)
	if !errors.Is(err, project.ErrAuthorizationExpired) {
		t.Fatalf("in-flight request was not stopped by authorization expiry: %v", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("fixture request never started; expiry cancellation was not exercised")
	}
	if b.RequestsUsed() != 1 {
		t.Fatalf("started request did not consume one attempt: %d", b.RequestsUsed())
	}
}
