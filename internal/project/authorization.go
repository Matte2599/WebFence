// Package project records an operator's authorization claim and produces an
// immutable, origin-only run snapshot. It performs no network I/O and cannot
// prove target ownership or authorize a connection by itself.
package project

import (
	"context"
	"errors"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/scope"
)

const maxOrigins = 32

// Stable errors omit project metadata, target URLs and authorization records.
var (
	ErrInvalidProject       = errors.New("project_invalid_configuration")
	ErrAuthorizationMissing = errors.New("project_authorization_not_confirmed")
	ErrAuthorizationExpired = errors.New("project_authorization_expired")
	ErrAuthorizationRevoked = errors.New("project_authorization_revoked")
)

// Draft is local operator input. TargetOwner and AuthorizationReference are
// descriptive records, not independently verified proof of target ownership.
// This value must never be copied directly into logs or a network request.
type Draft struct {
	ID                     string
	Name                   string
	TargetOwner            string
	AuthorizationReference string
	AuthorizationConfirmed bool
	AuthorizationExpiresAt time.Time
	Origins                []string
}

// AuthorizationDraft replaces all authorization fields in one new revision.
// The project identity and display name remain unchanged.
type AuthorizationDraft struct {
	TargetOwner            string
	AuthorizationReference string
	AuthorizationConfirmed bool
	AuthorizationExpiresAt time.Time
	Origins                []string
}

// Project is immutable after New. The zero value cannot start a run.
type Project struct {
	id                     string
	revision               uint64
	revoked                bool
	name                   string
	targetOwner            string
	authorizationReference string
	expiresAt              time.Time
	origins                []string
	policy                 scope.Policy
}

// RunScope captures one project's authorized origin set. Future project edits
// must create a different snapshot; an existing one never widens implicitly.
// Its zero value denies all checks.
type RunScope struct {
	projectID string
	revision  uint64
	expiresAt time.Time
	policy    scope.Policy
	lifecycle context.Context // optional managed-run cancellation, never replaceable
}

func New(d Draft) (Project, error) { return newAt(d, time.Now()) }

func newAt(d Draft, now time.Time) (Project, error) {
	return build(d, now, true)
}

// Restore validates a previously recorded project, including an expired one.
// Expired authorization remains readable for renewal or deletion, but BeginRun
// still refuses it. Callers must not use Restore to bypass operator confirmation.
func Restore(d Draft) (Project, error) { return RestoreRevision(d, 1) }

// RestoreRevision is for trusted local persistence. Historical revisions must
// remain audit data; callers must load the current revision before a new run.
func RestoreRevision(d Draft, revision uint64) (Project, error) {
	if revision == 0 || revision > math.MaxInt64 {
		return Project{}, ErrInvalidProject
	}
	p, err := build(d, time.Now(), false)
	if err != nil {
		return Project{}, err
	}
	p.revision = revision
	return p, nil
}

// RestoreRevokedRevision retains a revoked declaration for audit and renewal,
// but BeginRun refuses it. Only trusted persistence should restore state.
func RestoreRevokedRevision(d Draft, revision uint64) (Project, error) {
	p, err := RestoreRevision(d, revision)
	if err != nil {
		return Project{}, err
	}
	p.revoked = true
	return p, nil
}

func build(d Draft, now time.Time, requireCurrent bool) (Project, error) {
	if !validID(d.ID) || !validText(d.Name, 128) || !validText(d.TargetOwner, 256) ||
		!validText(d.AuthorizationReference, 256) || d.AuthorizationExpiresAt.IsZero() ||
		len(d.Origins) == 0 || len(d.Origins) > maxOrigins {
		return Project{}, ErrInvalidProject
	}
	if !d.AuthorizationConfirmed {
		return Project{}, ErrAuthorizationMissing
	}
	if requireCurrent && !d.AuthorizationExpiresAt.After(now) {
		return Project{}, ErrAuthorizationExpired
	}
	policy, err := scope.New(d.Origins)
	if err != nil {
		return Project{}, ErrInvalidProject
	}
	canonical := make([]string, 0, len(d.Origins))
	seen := make(map[string]struct{}, len(d.Origins))
	for _, raw := range d.Origins {
		u, err := policy.Check(raw)
		if err != nil {
			return Project{}, ErrInvalidProject
		}
		origin := u.Scheme + "://" + u.Host
		if _, duplicate := seen[origin]; duplicate {
			return Project{}, ErrInvalidProject
		}
		seen[origin] = struct{}{}
		canonical = append(canonical, origin)
	}
	return Project{
		id: d.ID, revision: 1, name: d.Name, targetOwner: d.TargetOwner,
		authorizationReference: d.AuthorizationReference,
		expiresAt:              d.AuthorizationExpiresAt.UTC(), origins: canonical, policy: policy,
	}, nil
}

func (p Project) ID() string       { return p.id }
func (p Project) Name() string     { return p.name }
func (p Project) Revision() uint64 { return p.revision }
func (p Project) Revoked() bool    { return p.revoked }

// RevokeAuthorization returns a new, non-runnable revision while retaining the
// descriptive record and origins. A fresh confirmed revision may reactivate it.
func (p Project) RevokeAuthorization() (Project, error) {
	if p.id == "" || p.revision == 0 || p.revision >= math.MaxInt64 {
		return Project{}, ErrInvalidProject
	}
	if p.revoked {
		return Project{}, ErrAuthorizationRevoked
	}
	p.revision++
	p.revoked = true
	return p, nil
}

// ReviseAuthorization validates a fresh operator declaration and returns a
// distinct immutable project. Persistence must compare-and-swap the revision.
func (p Project) ReviseAuthorization(change AuthorizationDraft) (Project, error) {
	if p.id == "" || p.revision == 0 || p.revision >= math.MaxInt64 {
		return Project{}, ErrInvalidProject
	}
	next, err := New(Draft{
		ID: p.id, Name: p.name, TargetOwner: change.TargetOwner,
		AuthorizationReference: change.AuthorizationReference,
		AuthorizationConfirmed: change.AuthorizationConfirmed,
		AuthorizationExpiresAt: change.AuthorizationExpiresAt,
		Origins:                change.Origins,
	})
	if err != nil {
		return Project{}, err
	}
	next.revision = p.revision + 1
	return next, nil
}

// Record returns a detached copy of the descriptive project fields for local
// persistence. The authorization reference is sensitive metadata: do not log it.
func (p Project) Record() Draft {
	if p.id == "" {
		return Draft{}
	}
	return Draft{
		ID: p.id, Name: p.name, TargetOwner: p.targetOwner,
		AuthorizationReference: p.authorizationReference,
		AuthorizationConfirmed: true, AuthorizationExpiresAt: p.expiresAt,
		Origins: p.Origins(),
	}
}

// Origins returns a copy; callers cannot widen the stored policy.
func (p Project) Origins() []string { return append([]string(nil), p.origins...) }

// BeginRun rechecks expiry and revocation before creating an immutable scope.
// It does not register the run for later store changes; use Store.BeginRun for
// managed work. It does not start a scan or grant permission to open a socket.
func (p Project) BeginRun() (RunScope, error) { return p.beginAt(time.Now()) }

func (p Project) beginAt(now time.Time) (RunScope, error) {
	if p.id == "" {
		return RunScope{}, ErrInvalidProject
	}
	if p.revoked {
		return RunScope{}, ErrAuthorizationRevoked
	}
	if !now.Before(p.expiresAt) {
		return RunScope{}, ErrAuthorizationExpired
	}
	return RunScope{projectID: p.id, revision: p.revision, expiresAt: p.expiresAt, policy: p.policy}, nil
}

func (r RunScope) ProjectID() string { return r.projectID }
func (r RunScope) Revision() uint64  { return r.revision }

// BindLifecycle returns a scope that also denies work when ctx is canceled.
// A bound lifecycle cannot be replaced or removed from a copied RunScope.
func (r RunScope) BindLifecycle(ctx context.Context) (RunScope, error) {
	if r.projectID == "" || ctx == nil || r.lifecycle != nil {
		return RunScope{}, ErrInvalidProject
	}
	r.lifecycle = ctx
	return r, nil
}

// Lifecycle is nil for an unmanaged snapshot. Authorized brokers watch it to
// interrupt in-flight I/O as well as checking the scope before every hop.
func (r RunScope) Lifecycle() context.Context { return r.lifecycle }

// ExpiresAt is the fixed deadline of this run snapshot. The returned time
// cannot extend it; CheckOrigin and Validate always use the stored deadline.
func (r RunScope) ExpiresAt() time.Time { return r.expiresAt }

// Validate checks that a run snapshot exists and has not expired. It does not
// prove the operator's claim or authorize network access by itself.
func (r RunScope) Validate() error { return r.validateAt(time.Now()) }

func (r RunScope) validateAt(now time.Time) error {
	if r.projectID == "" {
		return ErrInvalidProject
	}
	if r.lifecycle != nil && r.lifecycle.Err() != nil {
		cause := context.Cause(r.lifecycle)
		if errors.Is(cause, ErrAuthorizationRevoked) {
			return ErrAuthorizationRevoked
		}
		if errors.Is(cause, ErrAuthorizationExpired) {
			return ErrAuthorizationExpired
		}
		return r.lifecycle.Err()
	}
	if !now.Before(r.expiresAt) {
		return ErrAuthorizationExpired
	}
	return nil
}

// CheckOrigin enforces expiry and exact HTTP(S) origin for each planned URL.
// It does not enforce methods, paths, IP egress, budgets or target ownership;
// those checks must precede every actual network request in later M1 blocks.
func (r RunScope) CheckOrigin(rawURL string) (*url.URL, error) {
	return r.checkAt(time.Now(), rawURL)
}

func (r RunScope) checkAt(now time.Time, rawURL string) (*url.URL, error) {
	if err := r.validateAt(now); err != nil {
		return nil, err
	}
	return r.policy.Check(rawURL)
}

func validID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func validText(value string, maxBytes int) bool {
	if len(value) == 0 || len(value) > maxBytes || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if unicode.IsControl(c) || unicode.IsSpace(c) && c != ' ' {
			return false
		}
	}
	return true
}
