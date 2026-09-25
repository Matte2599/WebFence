// Package transport contains a pinned HTTP broker for explicitly granted
// loopback fixtures and public destinations. It is not wired into the desktop
// and does not support private-network scanning, sessions or browser traffic.
package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

var (
	ErrConfig        = errors.New("transport_invalid_configuration")
	ErrAddress       = errors.New("transport_address_not_allowed")
	ErrResolve       = errors.New("transport_resolution_failed")
	ErrBudget        = errors.New("transport_request_budget_exhausted")
	ErrNetwork       = errors.New("transport_network_failed")
	ErrRedirect      = errors.New("transport_invalid_redirect")
	ErrRedirectLimit = errors.New("transport_redirect_limit")
	ErrBodyLimit     = errors.New("transport_body_limit")
	ErrEncoding      = errors.New("transport_encoding_not_supported")
)

// Resolver is trusted infrastructure, must honor ctx and return caller-owned
// addresses. Tests inject it; no public DNS queries are needed by the lab.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Grant binds one exact HTTP(S) origin to explicit IPs. The port is taken
// from Origin. No wildcard, CIDR, default grant or cross-origin IP union.
type Grant struct {
	Origin    string
	Addresses []netip.Addr
}

type Limits struct {
	MaxRequests        int // shared by all Fetch calls, including redirect hops/failures
	MaxConcurrent      int // entire Fetch chains, including response reading
	MaxRedirects       int
	MaxBodyBytes       int64         // per final response; exceeding it returns no partial body
	RequestTimeout     time.Duration // entire Fetch, including waiting, DNS and hops
	RunTimeout         time.Duration // starts at construction, never reset by a Fetch
	MinRequestInterval time.Duration // per origin, including redirect hops; zero only for legacy lab callers
}

type Result struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	// FinalURL is the scope-checked URL that produced this response. It may
	// contain sensitive query data and must not be copied into scan reports.
	FinalURL string
}

// Broker must not be copied. Close cancels in-flight and queued work.
// Configuration is copied; requests cannot supply headers or credentials.
type Broker struct {
	policy     scope.Policy
	route      scope.RequestPolicy
	grants     map[string]map[netip.Addr]struct{}
	limits     Limits
	resolver   Resolver
	ctx        context.Context
	cancel     context.CancelFunc
	slots      chan struct{}
	mu         sync.Mutex
	used       int
	next       map[string]time.Time
	dial       func(context.Context, string, string) (net.Conn, error)
	roots      *x509.CertPool // fixture roots are injected only by same-package tests
	permit     project.RunScope
	authorized bool
	stopPermit func() bool
}

// LabBroker is retained as an alias for the original M0 laboratory API.
type LabBroker = Broker

func NewLab(ctx context.Context, grants []Grant, limits Limits, resolver Resolver) (*Broker, error) {
	return newPinned(ctx, grants, limits, resolver, false)
}

func newPinned(ctx context.Context, grants []Grant, limits Limits, resolver Resolver, public bool) (*Broker, error) {
	if ctx == nil || resolver == nil || len(grants) == 0 || limits.MaxRequests <= 0 ||
		limits.MaxConcurrent <= 0 || limits.MaxConcurrent > 16 || limits.MaxRedirects < 0 || limits.MaxRedirects > 10 ||
		limits.MaxBodyBytes <= 0 || limits.MaxBodyBytes > 8<<20 || limits.RequestTimeout <= 0 || limits.RunTimeout <= 0 ||
		limits.MinRequestInterval < 0 || limits.MinRequestInterval > time.Minute {
		return nil, ErrConfig
	}
	origins := make([]string, 0, len(grants))
	allowed := make(map[string]map[netip.Addr]struct{}, len(grants))
	for _, grant := range grants {
		p, err := scope.New([]string{grant.Origin})
		if err != nil || len(grant.Addresses) == 0 {
			return nil, ErrConfig
		}
		u, err := p.Check(grant.Origin)
		if err != nil {
			return nil, ErrConfig
		}
		key := origin(u)
		if _, exists := allowed[key]; exists {
			return nil, ErrConfig
		}
		ips := make(map[netip.Addr]struct{}, len(grant.Addresses))
		for _, ip := range grant.Addresses {
			if !ip.IsValid() || ip.Is4In6() || ip.Zone() != "" ||
				(public && !publicDestination(ip)) || (!public && !ip.IsLoopback()) {
				return nil, ErrConfig
			}
			ips[ip] = struct{}{}
		}
		allowed[key] = ips
		origins = append(origins, grant.Origin)
	}
	policy, err := scope.New(origins)
	if err != nil {
		return nil, ErrConfig
	}
	run, cancel := context.WithTimeout(ctx, limits.RunTimeout)
	return &Broker{policy: policy, grants: allowed, limits: limits, resolver: resolver,
		ctx: run, cancel: cancel, slots: make(chan struct{}, limits.MaxConcurrent),
		next: make(map[string]time.Time), dial: (&net.Dialer{}).DialContext}, nil
}

// NewAuthorizedLab adds an operator-declared run scope to the loopback-only
// broker. Network grants may be broader than the project scope; both checks
// apply to every hop. This is still a synthetic laboratory, not a production
// scanner or proof of target ownership.
func NewAuthorizedLab(ctx context.Context, permit project.RunScope, grants []Grant, limits Limits, resolver Resolver) (*Broker, error) {
	if err := permit.Validate(); err != nil {
		return nil, err
	}
	b, err := NewLab(ctx, grants, limits, resolver)
	if err != nil {
		return nil, err
	}
	b.bindPermit(permit)
	return b, nil
}

// NewAuthorizedLabWithPolicy applies method/path checks and pacing to an
// owned loopback fixture. Unlike NewAuthorizedLab, zero rate is not allowed.
func NewAuthorizedLabWithPolicy(ctx context.Context, permit project.RunScope, grants []Grant, limits Limits, resolver Resolver, route scope.RequestPolicy) (*Broker, error) {
	if !route.Valid() || limits.MinRequestInterval <= 0 || limits.MaxConcurrent != 1 {
		return nil, ErrConfig
	}
	b, err := NewAuthorizedLab(ctx, permit, grants, limits, resolver)
	if err != nil {
		return nil, err
	}
	b.route = route
	return b, nil
}

// NewAuthorizedPublic allows only explicitly pinned public IPs for exact
// operator-declared origins. No implicit DNS trust, private target exception,
// proxy, credentials or TLS bypass is available. Caller must retain proof of
// target authorization; an operator declaration is not independently verified.
func NewAuthorizedPublic(ctx context.Context, permit project.RunScope, grants []Grant, limits Limits, resolver Resolver, route scope.RequestPolicy) (*Broker, error) {
	if err := permit.Validate(); err != nil {
		return nil, err
	}
	if permit.Lifecycle() == nil || !route.Valid() || limits.MinRequestInterval < 100*time.Millisecond || limits.MaxConcurrent != 1 || limits.MaxRequests > 10000 ||
		limits.RunTimeout > 4*time.Hour || len(grants) == 0 || len(grants) > 32 {
		return nil, ErrConfig
	}
	for _, grant := range grants {
		if _, err := permit.CheckOrigin(grant.Origin); err != nil {
			return nil, err
		}
		if len(grant.Addresses) == 0 || len(grant.Addresses) > 16 {
			return nil, ErrConfig
		}
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	b, err := newPinned(ctx, grants, limits, resolver, true)
	if err != nil {
		return nil, err
	}
	b.route = route
	b.bindPermit(permit)
	return b, nil
}

func (b *Broker) bindPermit(permit project.RunScope) {
	b.permit = permit
	b.authorized = true
	if lifecycle := permit.Lifecycle(); lifecycle != nil {
		b.stopPermit = context.AfterFunc(lifecycle, b.cancel)
	}
}

func origin(u *url.URL) string { return u.Scheme + "://" + u.Host }
func (b *Broker) Close() {
	b.cancel()
	if b.stopPermit != nil {
		b.stopPermit()
	}
}
func (b *Broker) RequestsUsed() int { b.mu.Lock(); defer b.mu.Unlock(); return b.used }

func (b *Broker) contextError(ctx context.Context) error {
	if b.authorized {
		if err := b.permit.Validate(); err != nil {
			return err
		}
	}
	if err := b.ctx.Err(); err != nil {
		if errors.Is(context.Cause(b.ctx), project.ErrAuthorizationRevoked) {
			return project.ErrAuthorizationRevoked
		}
		return err
	}
	return ctx.Err()
}

func (b *Broker) reserve(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.contextError(ctx); err != nil {
		return err
	}
	if b.used >= b.limits.MaxRequests {
		return ErrBudget
	}
	b.used++ // Failure consumes an attempt; refunds could allow unbounded retries.
	return nil
}

// Fetch issues GET only, with manual redirects and no cookies/auth/referer.
// All error values are redacted codes (or context/scope sentinels), never raw
// url.Error, TLS, DNS or socket errors containing target or certificate data.
func (b *Broker) Fetch(ctx context.Context, raw string) (Result, error) {
	return b.FetchMethod(ctx, http.MethodGet, raw)
}

// FetchMethod accepts only GET or HEAD. The route policy, when configured,
// is rechecked on every redirect before DNS or a socket is opened.
func (b *Broker) FetchMethod(ctx context.Context, method, raw string) (Result, error) {
	if b == nil || ctx == nil || (method != http.MethodGet && method != http.MethodHead) {
		return Result{}, scope.ErrMethodDenied
	}
	if method == http.MethodHead && !b.route.Valid() {
		return Result{}, scope.ErrMethodDenied // legacy laboratory remains GET-only
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
	for hops := 0; ; hops++ {
		if err := b.contextError(ctx); err != nil {
			return Result{}, err
		}
		if b.authorized {
			if _, err := b.permit.CheckOrigin(raw); err != nil {
				return Result{}, err
			}
		}
		u, err := b.policy.Check(raw)
		if err != nil {
			return Result{}, err
		}
		if b.route.Valid() {
			if err := b.route.Check(method, u); err != nil {
				return Result{}, err
			}
		}
		if err = b.reserve(ctx); err != nil {
			return Result{}, err
		}
		ip, err := b.resolve(ctx, u)
		if err != nil {
			return Result{}, err
		}
		if err := b.pace(ctx, origin(u)); err != nil {
			return Result{}, err
		}
		result, next, err := b.exchange(ctx, method, u, ip)
		if err != nil {
			return Result{}, err
		}
		if next == "" {
			result.FinalURL = u.String()
			return result, nil
		}
		if hops >= b.limits.MaxRedirects {
			return Result{}, ErrRedirectLimit
		}
		ref, err := url.Parse(next)
		if err != nil {
			return Result{}, ErrRedirect
		}
		raw = u.ResolveReference(ref).String()
	}
}

// pace assigns non-bursting per-origin slots, including redirect hops. Policy
// constructors require one concurrent chain so the next attempt cannot pass
// an earlier chain while it connects or reads. A canceled slot is not refunded;
// over-throttling is safer than allowing a burst after cancellation.
func (b *Broker) pace(ctx context.Context, key string) error {
	if b.limits.MinRequestInterval == 0 {
		return b.contextError(ctx)
	}
	b.mu.Lock()
	now := time.Now()
	start := now
	if next := b.next[key]; next.After(start) {
		start = next
	}
	b.next[key] = start.Add(b.limits.MinRequestInterval)
	b.mu.Unlock()
	if wait := time.Until(start); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return b.contextError(ctx)
		case <-b.ctx.Done():
			return b.contextError(ctx)
		}
	}
	return b.contextError(ctx)
}

func (b *Broker) resolve(ctx context.Context, u *url.URL) (netip.Addr, error) {
	ip, err := netip.ParseAddr(u.Hostname())
	ips := []netip.Addr{ip}
	if err != nil {
		ips, err = b.resolver.LookupNetIP(ctx, "ip", u.Hostname())
		if ctxErr := b.contextError(ctx); ctxErr != nil {
			return netip.Addr{}, ctxErr
		}
		if err != nil || len(ips) == 0 {
			return netip.Addr{}, ErrResolve
		}
	}
	var selected netip.Addr
	for _, candidate := range ips {
		if _, ok := b.grants[origin(u)][candidate]; !ok {
			return netip.Addr{}, ErrAddress
		}
		if !selected.IsValid() || candidate.Compare(selected) < 0 {
			selected = candidate
		}
	}
	return selected, nil
}

func (b *Broker) exchange(ctx context.Context, method string, u *url.URL, ip netip.Addr) (Result, string, error) {
	port, _ := strconv.ParseUint(u.Port(), 10, 16) // scope already checked it
	pinned := netip.AddrPortFrom(ip, uint16(port))
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	tr := &http.Transport{
		Protocols: protocols,
		Proxy:     nil, DisableKeepAlives: true, DisableCompression: true,
		MaxResponseHeaderBytes: 32 << 10,
		// One new HTTP/1 connection per hop: no pooled connection, automatic retry,
		// alternate-IP fallback or HTTP/2 connection coalescing outside this budget.
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname(), RootCAs: b.roots},
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			if err := b.contextError(ctx); err != nil {
				return nil, err
			}
			if address != u.Host {
				return nil, ErrAddress
			}
			// Go transport may detach its dial context from the request. Bind the actual
			// socket operation to Fetch so cancellation also interrupts connection setup.
			conn, err := b.dial(ctx, "tcp", pinned.String())
			if err != nil {
				return nil, err
			}
			peer, err := netip.ParseAddrPort(conn.RemoteAddr().String())
			if err != nil || peer.Addr().Unmap() != ip || peer.Port() != pinned.Port() {
				conn.Close()
				return nil, ErrAddress
			}
			if err := b.contextError(ctx); err != nil {
				conn.Close()
				return nil, err
			}
			return conn, nil
		},
	}
	defer tr.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return Result{}, "", scope.ErrInvalidURL
	}
	req.Header.Set("Accept-Encoding", "identity")
	// RoundTrip never follows redirects and adds no client cookie jar or referer.
	resp, err := tr.RoundTrip(req)
	if err != nil {
		if ctxErr := b.contextError(ctx); ctxErr != nil {
			return Result{}, "", ctxErr
		}
		if errors.Is(err, ErrAddress) {
			return Result{}, "", ErrAddress
		}
		return Result{}, "", ErrNetwork
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 301, 302, 303, 307, 308:
		location := resp.Header.Get("Location")
		if location == "" || len(location) > scope.MaxURLBytes {
			return Result{}, "", ErrRedirect
		}
		return Result{}, location, nil
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return Result{}, "", ErrEncoding
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, b.limits.MaxBodyBytes+1))
	if ctxErr := b.contextError(ctx); ctxErr != nil {
		return Result{}, "", ctxErr
	}
	if err != nil {
		return Result{}, "", ErrNetwork
	}
	if int64(len(body)) > b.limits.MaxBodyBytes {
		return Result{}, "", ErrBodyLimit
	}
	return Result{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: body}, "", nil
}
