package project

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/scope"
)

func fixture(now time.Time) Draft {
	return Draft{
		ID: "synthetic-project-1", Name: "Synthetic staging", TargetOwner: "Fixture owner",
		AuthorizationReference: "fixture-approval-1", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: now.Add(time.Hour), Origins: []string{"https://lab.invalid"},
	}
}

func TestPublicProjectFlowStaysOffline(t *testing.T) {
	draft := fixture(time.Now())
	p, err := New(draft)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID() != draft.ID || p.Name() != draft.Name {
		t.Fatalf("project metadata was not preserved")
	}
	run, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	if run.ProjectID() != draft.ID {
		t.Fatalf("run lost project identity")
	}
	if _, err := run.CheckOrigin("https://lab.invalid/fixture"); err != nil {
		t.Fatalf("synthetic origin should be accepted without a socket: %v", err)
	}
}

func TestRunScopeRejectsForeignAndExpiredDestinations(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	p, err := newAt(fixture(now), now)
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.beginAt(now)
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := run.checkAt(now, "https://lab.invalid/fixture?q=1")
	if err != nil || allowed.Host != "lab.invalid:443" {
		t.Fatalf("allowed fixture URL: %v, %v", allowed, err)
	}
	for _, raw := range []string{
		"http://lab.invalid/fixture", "https://lab.invalid:444/fixture",
		"https://other.lab.invalid/fixture", "https://outside.invalid/fixture",
	} {
		if _, err := run.checkAt(now, raw); !errors.Is(err, scope.ErrOutOfScope) {
			t.Errorf("%q: expected out-of-scope, got %v", raw, err)
		}
	}
	if _, err := run.checkAt(now.Add(time.Hour), "https://lab.invalid/fixture"); !errors.Is(err, ErrAuthorizationExpired) {
		t.Fatalf("expiry must be rechecked on every URL: %v", err)
	}
	if _, err := p.beginAt(now.Add(time.Hour)); !errors.Is(err, ErrAuthorizationExpired) {
		t.Fatalf("new run after expiry: %v", err)
	}
	if _, err := (RunScope{}).CheckOrigin("https://lab.invalid"); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("zero run scope must deny: %v", err)
	}
	if _, err := (Project{}).BeginRun(); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("zero project must deny: %v", err)
	}
}

func TestAuthorizationSnapshotCannotBeWidenedByCaller(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	draft := fixture(now)
	p, err := newAt(draft, now)
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.beginAt(now)
	if err != nil {
		t.Fatal(err)
	}
	draft.Origins[0] = "https://outside.invalid"
	origins := p.Origins()
	origins[0] = "https://outside.invalid:443"
	record := p.Record()
	record.Origins[0] = "https://outside.invalid:443"
	if got := p.Origins()[0]; got != "https://lab.invalid:443" {
		t.Fatalf("canonical origin changed: %s", got)
	}
	if _, err := run.checkAt(now, "https://outside.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("mutated caller input widened run scope: %v", err)
	}
	if _, err := run.checkAt(now, "https://lab.invalid"); err != nil {
		t.Fatalf("original origin stopped working: %v", err)
	}
}

func TestDraftRequiresExplicitBoundedClaim(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		edit   func(*Draft)
		wanted error
	}{
		{"no confirmation", func(d *Draft) { d.AuthorizationConfirmed = false }, ErrAuthorizationMissing},
		{"expired", func(d *Draft) { d.AuthorizationExpiresAt = now }, ErrAuthorizationExpired},
		{"no owner", func(d *Draft) { d.TargetOwner = "" }, ErrInvalidProject},
		{"no reference", func(d *Draft) { d.AuthorizationReference = "" }, ErrInvalidProject},
		{"invalid ID", func(d *Draft) { d.ID = "../other" }, ErrInvalidProject},
		{"control in name", func(d *Draft) { d.Name = "fixture\nsecret" }, ErrInvalidProject},
		{"line separator in name", func(d *Draft) { d.Name = "fixture\u2028secret" }, ErrInvalidProject},
		{"path in grant", func(d *Draft) { d.Origins = []string{"https://lab.invalid/admin"} }, ErrInvalidProject},
		{"duplicate canonical origin", func(d *Draft) { d.Origins = []string{"https://lab.invalid", "https://lab.invalid:443"} }, ErrInvalidProject},
		{"too many origins", func(d *Draft) { d.Origins = make([]string, maxOrigins+1) }, ErrInvalidProject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draft := fixture(now)
			tc.edit(&draft)
			_, err := newAt(draft, now)
			if !errors.Is(err, tc.wanted) {
				t.Fatalf("expected %v, got %v", tc.wanted, err)
			}
			if strings.Contains(err.Error(), "lab.invalid") || strings.Contains(err.Error(), "fixture-approval") {
				t.Fatalf("error exposed input: %v", err)
			}
		})
	}
}

func TestAuthorizationRevisionKeepsPreviousRunSnapshot(t *testing.T) {
	base := fixture(time.Now())
	old, err := New(base)
	if err != nil {
		t.Fatal(err)
	}
	run, err := old.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	change := AuthorizationDraft{
		TargetOwner: "New fixture owner", AuthorizationReference: "renewal-2",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(2 * time.Hour),
		Origins: []string{"https://new.invalid"},
	}
	next, err := old.ReviseAuthorization(change)
	if err != nil {
		t.Fatal(err)
	}
	change.Origins[0] = "https://mutated.invalid"
	if old.Revision() != 1 || run.Revision() != 1 || next.Revision() != 2 {
		t.Fatal("revision identity changed or was lost")
	}
	if _, err := run.CheckOrigin("https://new.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("prior run acquired new origin: %v", err)
	}
	if _, err := run.CheckOrigin("https://lab.invalid"); err != nil {
		t.Fatalf("prior run lost its original origin: %v", err)
	}
	nextRun, err := next.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nextRun.CheckOrigin("https://new.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := nextRun.CheckOrigin("https://lab.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("new revision kept removed origin: %v", err)
	}
	change.AuthorizationConfirmed = false
	if _, err := next.ReviseAuthorization(change); !errors.Is(err, ErrAuthorizationMissing) {
		t.Fatalf("unconfirmed revision accepted: %v", err)
	}
	if _, err := RestoreRevision(base, 0); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("invalid revision restored: %v", err)
	}
}

func TestBoundRunScopeCannotDiscardRevocation(t *testing.T) {
	p, err := New(fixture(time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	ctx, revoke := context.WithCancelCause(context.Background())
	bound, err := run.BindLifecycle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bound.BindLifecycle(context.Background()); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("bound lifecycle replaced: %v", err)
	}
	revoke(ErrAuthorizationRevoked)
	if _, err := bound.CheckOrigin("https://lab.invalid"); !errors.Is(err, ErrAuthorizationRevoked) {
		t.Fatalf("revoked scope accepted origin: %v", err)
	}
	if _, err := run.CheckOrigin("https://lab.invalid"); err != nil {
		t.Fatalf("original unmanaged snapshot unexpectedly changed: %v", err)
	}
}

func TestRevokedRevisionCannotStartUntilFreshDeclaration(t *testing.T) {
	p, err := New(fixture(time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := p.RevokeAuthorization()
	if err != nil || !revoked.Revoked() || revoked.Revision() != 2 {
		t.Fatalf("revoke: %+v %v", revoked.Record(), err)
	}
	if _, err := revoked.BeginRun(); !errors.Is(err, ErrAuthorizationRevoked) {
		t.Fatalf("revoked revision started a run: %v", err)
	}
	if _, err := revoked.RevokeAuthorization(); !errors.Is(err, ErrAuthorizationRevoked) {
		t.Fatalf("duplicate revoke: %v", err)
	}
	change := AuthorizationDraft{
		TargetOwner: "Fixture owner", AuthorizationReference: "fresh approval",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins: []string{"https://new.invalid"},
	}
	active, err := revoked.ReviseAuthorization(change)
	if err != nil || active.Revoked() || active.Revision() != 3 {
		t.Fatalf("fresh declaration: %+v %v", active.Record(), err)
	}
	if _, err := active.BeginRun(); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreRevokedRevision(revoked.Record(), 2)
	if err != nil || !restored.Revoked() {
		t.Fatalf("restore revoked: %v", err)
	}
	if _, err := restored.BeginRun(); !errors.Is(err, ErrAuthorizationRevoked) {
		t.Fatalf("restored revoked revision started: %v", err)
	}
}
