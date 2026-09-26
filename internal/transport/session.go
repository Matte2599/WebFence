package transport

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

var ErrSessionRoute = errors.New("transport_session_route_denied")

const maxLoginFormBytes = 4096

// SessionRoutes are exact operator-declared URLs. Login is the only POST
// permitted by this broker; the verification URL must also pass the ordinary
// GET route policy. Neither URL may include query credentials or fragments.
// LoginConfirmed is the caller's explicit confirmation for this POST flow;
// the broker cannot independently prove target ownership.
type SessionRoutes struct {
	LoginURL       string
	VerifyURL      string
	LoginConfirmed bool
}

type sessionRoutes struct {
	origin    string
	loginURL  string
	verifyURL string
}

// NewAuthorizedLabWithSession is a loopback-only M3 authentication laboratory.
// It shares one broker's request budget, pacing and pinned peer checks across
// login and authenticated GETs. No production target traffic is enabled.
func NewAuthorizedLabWithSession(ctx context.Context, permit project.RunScope, grants []Grant,
	limits Limits, resolver Resolver, route scope.RequestPolicy, routes SessionRoutes) (*Broker, error) {
	if permit.Lifecycle() == nil || !routes.LoginConfirmed || limits.MaxRequests > 10000 || limits.RunTimeout > 4*time.Hour {
		return nil, ErrConfig
	}
	b, err := NewAuthorizedLabWithPolicy(ctx, permit, grants, limits, resolver, route)
	if err != nil {
		return nil, err
	}
	configured, err := b.validateSessionRoutes(routes)
	if err != nil {
		b.Close()
		return nil, err
	}
	b.session = configured
	return b, nil
}

func (b *Broker) validateSessionRoutes(routes SessionRoutes) (*sessionRoutes, error) {
	login, err := b.policy.Check(routes.LoginURL)
	if err != nil || login.RawQuery != "" || login.ForceQuery || login.Fragment != "" {
		return nil, ErrConfig
	}
	loginPath, err := scope.NewRequestPolicy([]string{"GET"}, []string{login.EscapedPath()}, nil)
	if err != nil || loginPath.Check("GET", login) != nil {
		return nil, ErrConfig
	}
	verify, err := b.policy.Check(routes.VerifyURL)
	if err != nil || verify.RawQuery != "" || verify.ForceQuery || verify.Fragment != "" ||
		origin(verify) != origin(login) || b.route.Check("GET", verify) != nil {
		return nil, ErrConfig
	}
	if _, err := b.permit.CheckOrigin(login.String()); err != nil {
		return nil, err
	}
	if _, err := b.permit.CheckOrigin(verify.String()); err != nil {
		return nil, err
	}
	return &sessionRoutes{origin: origin(login), loginURL: login.String(), verifyURL: verify.String()}, nil
}

// LoginForm sends one bounded form POST to the exact login URL. Redirects are
// returned as responses, never followed with the form or resulting cookies.
// The caller owns form and must erase it after use. Result headers can contain
// sensitive Set-Cookie values and must not be logged or persisted.
func (b *Broker) LoginForm(ctx context.Context, raw string, form []byte) (Result, error) {
	if b == nil || b.session == nil || len(form) == 0 || len(form) > maxLoginFormBytes {
		return Result{}, ErrSessionRoute
	}
	return b.sessionRequest(ctx, "POST", raw, exchangeOptions{body: form, noRedirect: true})
}

// FetchSession sends a single GET with one caller-owned cookie pair. It never
// follows redirects, which prevents a credential from crossing an origin.
// Only an M3 session broker accepts this method; normal M1 Fetch is unchanged.
func (b *Broker) FetchSession(ctx context.Context, raw, cookie string) (Result, error) {
	if b == nil || b.session == nil || len(cookie) == 0 || len(cookie) > 2048 ||
		strings.ContainsAny(cookie, "\r\n;") || strings.Count(cookie, "=") < 1 {
		return Result{}, ErrSessionRoute
	}
	return b.sessionRequest(ctx, "GET", raw, exchangeOptions{cookie: cookie, noRedirect: true})
}

func (b *Broker) sessionRequest(ctx context.Context, method, raw string, options exchangeOptions) (Result, error) {
	if ctx == nil {
		return Result{}, ErrConfig
	}
	if b.authorized {
		var cancelAuthorization context.CancelFunc
		ctx, cancelAuthorization = context.WithDeadline(ctx, b.permit.ExpiresAt())
		defer cancelAuthorization()
	}
	ctx, cancel := context.WithTimeout(ctx, b.limits.RequestTimeout)
	defer cancel()
	stop := context.AfterFunc(b.ctx, cancel)
	defer stop()
	if err := b.contextError(ctx); err != nil {
		return Result{}, err
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
	case <-ctx.Done():
		return Result{}, b.contextError(ctx)
	}
	if err := b.contextError(ctx); err != nil {
		return Result{}, err
	}
	if _, err := b.permit.CheckOrigin(raw); err != nil {
		return Result{}, err
	}
	u, err := b.policy.Check(raw)
	if err != nil {
		return Result{}, err
	}
	if origin(u) != b.session.origin {
		return Result{}, ErrSessionRoute
	}
	if method == "POST" {
		if u.String() != b.session.loginURL {
			return Result{}, ErrSessionRoute
		}
	} else if method != "GET" || b.route.Check(method, u) != nil {
		return Result{}, ErrSessionRoute
	}
	if err := b.reserve(ctx); err != nil {
		return Result{}, err
	}
	ip, err := b.resolve(ctx, u)
	if err != nil {
		return Result{}, err
	}
	if err := b.pace(ctx, origin(u)); err != nil {
		return Result{}, err
	}
	result, _, err := b.exchange(ctx, method, u, ip, options)
	if err != nil {
		return Result{}, err
	}
	result.FinalURL = u.String()
	return result, nil
}

// VerifyURL exposes only the configured URL for a session validity probe.
// It contains no credential data, but can be sensitive project metadata.
func (b *Broker) VerifyURL() string {
	if b == nil || b.session == nil {
		return ""
	}
	return b.session.verifyURL
}

func (b *Broker) LoginURL() string {
	if b == nil || b.session == nil {
		return ""
	}
	return b.session.loginURL
}

// VerifyAnonymous probes the declared validity URL without credentials and
// without following redirects. A positive authenticated result is ambiguous
// if the anonymous response already has the same expected marker.
func (b *Broker) VerifyAnonymous(ctx context.Context) (Result, error) {
	if b == nil || b.session == nil {
		return Result{}, ErrSessionRoute
	}
	return b.sessionRequest(ctx, "GET", b.session.verifyURL, exchangeOptions{noRedirect: true})
}

// SessionDeadline is fixed by the run authorization and broker lifetime.
func (b *Broker) SessionDeadline() time.Time {
	if b == nil || b.session == nil {
		return time.Time{}
	}
	deadline, _ := b.ctx.Deadline()
	if b.permit.ExpiresAt().Before(deadline) {
		return b.permit.ExpiresAt()
	}
	return deadline
}

// SessionOrigin is used by an in-memory session manager for exact binding.
func (b *Broker) SessionOrigin() string {
	if b == nil || b.session == nil {
		return ""
	}
	return b.session.origin
}

func (b *Broker) SessionProjectID() string {
	if b == nil || b.session == nil {
		return ""
	}
	return b.permit.ProjectID()
}

func (b *Broker) SessionRevision() uint64 {
	if b == nil || b.session == nil {
		return 0
	}
	return b.permit.Revision()
}
