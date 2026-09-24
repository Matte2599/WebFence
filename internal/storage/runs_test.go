package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

type labResolver struct{ ip netip.Addr }

func (r labResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{r.ip}, nil
}

func labGrant(t *testing.T, server *httptest.Server) transport.Grant {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if err != nil {
		t.Fatal(err)
	}
	return transport.Grant{Origin: server.URL, Addresses: []netip.Addr{ip}}
}

func labLimits() transport.Limits {
	return transport.Limits{
		MaxRequests: 5, MaxConcurrent: 1, MaxRedirects: 2, MaxBodyBytes: 1024,
		RequestTimeout: 5 * time.Second, RunTimeout: 5 * time.Second,
	}
}

func TestManagedRunRevocationStopsInFlightLabAndRequiresCurrentRevision(t *testing.T) {
	started := make(chan struct{})
	oldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer oldServer.Close()
	newServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "new fixture")
	}))
	defer newServer.Close()
	oldGrant, newGrant := labGrant(t, oldServer), labGrant(t, newServer)
	s := openFixture(t, filepath.Join(t.TempDir(), "managed.sqlite"))
	p := syntheticProject(t, "managed", []string{oldServer.URL})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	queued, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer queued.Close()
	// Even an accidentally independent broker context must honor the scope's
	// bound lifecycle; otherwise a caller could bypass store revocation.
	b, err := transport.NewAuthorizedLab(context.Background(), run.Scope(), []transport.Grant{oldGrant, newGrant},
		labLimits(), labResolver{oldGrant.Addresses[0]})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	done := make(chan error, 1)
	go func() { _, fetchErr := b.Fetch(context.Background(), oldServer.URL); done <- fetchErr }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture request did not start")
	}
	queuedFetch := make(chan error, 1)
	go func() { _, fetchErr := b.Fetch(context.Background(), oldServer.URL); queuedFetch <- fetchErr }()
	change := project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "new approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{newServer.URL},
	}
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, change); err != nil {
		t.Fatal(err)
	}
	select {
	case fetchErr := <-done:
		if !errors.Is(fetchErr, project.ErrAuthorizationRevoked) {
			t.Fatalf("in-flight fetch: %v", fetchErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight fetch was not stopped")
	}
	select {
	case fetchErr := <-queuedFetch:
		if !errors.Is(fetchErr, project.ErrAuthorizationRevoked) {
			t.Fatalf("queued fetch: %v", fetchErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("queued fetch was not stopped")
	}
	if !errors.Is(context.Cause(queued.Context()), project.ErrAuthorizationRevoked) {
		t.Fatalf("second active run was not revoked: %v", context.Cause(queued.Context()))
	}
	if _, err := b.Fetch(context.Background(), oldServer.URL); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("revoked broker accepted more work: %v", err)
	}
	if b.RequestsUsed() != 1 {
		t.Fatalf("post-revocation attempt consumed budget: %d", b.RequestsUsed())
	}
	latest, err := s.BeginRun(t.Context(), p.ID())
	if err != nil || latest.Scope().Revision() != 2 {
		t.Fatalf("new managed run: %v", err)
	}
	defer latest.Close()
	newBroker, err := transport.NewAuthorizedLab(latest.Context(), latest.Scope(),
		[]transport.Grant{oldGrant, newGrant}, labLimits(), labResolver{oldGrant.Addresses[0]})
	if err != nil {
		t.Fatal(err)
	}
	defer newBroker.Close()
	if _, err := newBroker.Fetch(context.Background(), oldServer.URL); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("new run retained removed origin: %v", err)
	}
	result, err := newBroker.Fetch(context.Background(), newServer.URL)
	if err != nil || string(result.Body) != "new fixture" {
		t.Fatalf("new origin: %q %v", result.Body, err)
	}
}

func TestManagedRunDeletionAndStoreCloseRevoke(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "delete.sqlite"))
	p := syntheticProject(t, "delete-managed", []string{"https://lab.invalid"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	bad := project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "invalid",
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{"https://lab.invalid"},
	}
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 1, bad); !errors.Is(err, project.ErrAuthorizationMissing) {
		t.Fatalf("invalid revision: %v", err)
	}
	if run.Context().Err() != nil {
		t.Fatal("invalid revision revoked a valid run")
	}
	if err := s.DeleteProject(t.Context(), p.ID()); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(context.Cause(run.Context()), project.ErrAuthorizationRevoked) {
		t.Fatalf("delete did not revoke run: %v", context.Cause(run.Context()))
	}
	if _, err := s.BeginRun(t.Context(), p.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted project started a run: %v", err)
	}
	p2 := syntheticProject(t, "close-managed", []string{"https://lab.invalid"})
	if err := s.CreateProject(t.Context(), p2); err != nil {
		t.Fatal(err)
	}
	run2, err := s.BeginRun(t.Context(), p2.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run2.Close()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(context.Cause(run2.Context()), project.ErrAuthorizationRevoked) {
		t.Fatalf("store close did not revoke run: %v", context.Cause(run2.Context()))
	}
	if _, err := s.BeginRun(t.Context(), p2.ID()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("closed store started a run: %v", err)
	}
}

func TestManagedRunExpiryStopsAndUnregisters(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "deadline.sqlite"))
	p, err := project.New(project.Draft{
		ID: "deadline-managed", Name: "Deadline fixture", TargetOwner: "Fixture owner",
		AuthorizationReference: "fixture approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(150 * time.Millisecond), Origins: []string{"https://lab.invalid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	select {
	case <-run.Context().Done():
	case <-time.After(2 * time.Second):
		t.Fatal("managed run did not stop at expiry")
	}
	if !errors.Is(context.Cause(run.Context()), project.ErrAuthorizationExpired) {
		t.Fatalf("expiry cause: %v", context.Cause(run.Context()))
	}
	deadline := time.After(2 * time.Second)
	for {
		s.mu.Lock()
		count := len(s.runs[p.ID()])
		s.mu.Unlock()
		if count == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("expired run remained registered")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestExplicitRevocationPersistsUntilFreshConfirmedRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "revoked.sqlite")
	s := openFixture(t, path)
	p := syntheticProject(t, "explicit-revoke", []string{"https://lab.invalid"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	revoked, err := s.RevokeAuthorization(t.Context(), p.ID(), 1)
	if err != nil || !revoked.Revoked() || revoked.Revision() != 2 {
		t.Fatalf("revocation: revision=%d revoked=%v err=%v", revoked.Revision(), revoked.Revoked(), err)
	}
	if !errors.Is(context.Cause(run.Context()), project.ErrAuthorizationRevoked) {
		t.Fatalf("active run not stopped: %v", context.Cause(run.Context()))
	}
	if _, err := run.Scope().CheckOrigin("https://lab.invalid"); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("old scope still runnable: %v", err)
	}
	if _, err := s.BeginRun(t.Context(), p.ID()); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("revoked project started a run: %v", err)
	}
	if _, err := s.RevokeAuthorization(t.Context(), p.ID(), 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revocation accepted: %v", err)
	}
	if _, err := s.RevokeAuthorization(t.Context(), p.ID(), 2); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("duplicate revocation accepted: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	loaded, err := s.LoadProject(t.Context(), p.ID())
	if err != nil || !loaded.Revoked() || loaded.Revision() != 2 {
		t.Fatalf("revocation lost on reopen: revision=%d revoked=%v err=%v", loaded.Revision(), loaded.Revoked(), err)
	}
	if _, err := s.BeginRun(t.Context(), p.ID()); !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("reopened revoked project started a run: %v", err)
	}
	history, err := s.ListAuthorizationRevisions(t.Context(), p.ID())
	if err != nil || len(history) != 2 || history[0].Revoked || !history[1].Revoked {
		t.Fatalf("revocation history: %+v %v", history, err)
	}
	change := project.AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "fresh approval", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{"https://new.invalid"},
	}
	change.AuthorizationConfirmed = false
	if _, err := s.ReviseAuthorization(t.Context(), p.ID(), 2, change); !errors.Is(err, project.ErrAuthorizationMissing) {
		t.Fatalf("unconfirmed reactivation accepted: %v", err)
	}
	change.AuthorizationConfirmed = true
	active, err := s.ReviseAuthorization(t.Context(), p.ID(), 2, change)
	if err != nil || active.Revoked() || active.Revision() != 3 {
		t.Fatalf("reactivation: revision=%d revoked=%v err=%v", active.Revision(), active.Revoked(), err)
	}
	newRun, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer newRun.Close()
	if newRun.Scope().Revision() != 3 {
		t.Fatal("reactivated run has stale revision")
	}
	if _, err := newRun.Scope().CheckOrigin("https://lab.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("reactivated run inherited old origin: %v", err)
	}
}
