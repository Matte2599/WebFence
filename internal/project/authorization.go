// Package project records an operator's authorization claim and produces an
// immutable, origin-only run snapshot. It performs no network I/O and cannot
// prove target ownership or authorize a connection by itself.
package project

import (
	"errors"
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

// Project is immutable after New. The zero value cannot start a run.
type Project struct {
	id                     string
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
	expiresAt time.Time
	policy    scope.Policy
}

func New(d Draft) (Project, error) { return newAt(d, time.Now()) }

func newAt(d Draft, now time.Time) (Project, error) {
	if !validID(d.ID) || !validText(d.Name, 128) || !validText(d.TargetOwner, 256) ||
		!validText(d.AuthorizationReference, 256) || len(d.Origins) == 0 || len(d.Origins) > maxOrigins {
		return Project{}, ErrInvalidProject
	}
	if !d.AuthorizationConfirmed {
		return Project{}, ErrAuthorizationMissing
	}
	if !d.AuthorizationExpiresAt.After(now) {
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
		id: d.ID, name: d.Name, targetOwner: d.TargetOwner,
		authorizationReference: d.AuthorizationReference,
		expiresAt:              d.AuthorizationExpiresAt.UTC(), origins: canonical, policy: policy,
	}, nil
}

func (p Project) ID() string   { return p.id }
func (p Project) Name() string { return p.name }

// Origins returns a copy; callers cannot widen the stored policy.
func (p Project) Origins() []string { return append([]string(nil), p.origins...) }

// BeginRun rechecks expiry before creating an immutable run scope. Calling it
// does not start a scan or grant permission to open a socket.
func (p Project) BeginRun() (RunScope, error) { return p.beginAt(time.Now()) }

func (p Project) beginAt(now time.Time) (RunScope, error) {
	if p.id == "" {
		return RunScope{}, ErrInvalidProject
	}
	if !now.Before(p.expiresAt) {
		return RunScope{}, ErrAuthorizationExpired
	}
	return RunScope{projectID: p.id, expiresAt: p.expiresAt, policy: p.policy}, nil
}

func (r RunScope) ProjectID() string { return r.projectID }

// CheckOrigin enforces expiry and exact HTTP(S) origin for each planned URL.
// It does not enforce methods, paths, IP egress, budgets or target ownership;
// those checks must precede every actual network request in later M1 blocks.
func (r RunScope) CheckOrigin(rawURL string) (*url.URL, error) {
	return r.checkAt(time.Now(), rawURL)
}

func (r RunScope) checkAt(now time.Time, rawURL string) (*url.URL, error) {
	if r.projectID == "" {
		return nil, ErrInvalidProject
	}
	if !now.Before(r.expiresAt) {
		return nil, ErrAuthorizationExpired
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
