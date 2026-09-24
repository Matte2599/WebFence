// Package transport contains an M0 HTTP transport restricted to explicitly
// granted loopback destinations. It is not wired into the desktop or ready for
// public/private-network scanning, authenticated sessions or browser traffic.
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

// Grant binds one exact HTTP(S) origin to explicit loopback IPs. The port is
// taken from Origin. No wildcard, CIDR, default grant or cross-origin IP union.
type Grant struct {
	Origin    string
	Addresses []netip.Addr
}

type Limits struct {
	MaxRequests    int // shared by all Fetch calls, including redirect hops/failures
	MaxConcurrent  int // entire Fetch chains, including response reading
	MaxRedirects   int
	MaxBodyBytes   int64         // per final response; exceeding it returns no partial body
	RequestTimeout time.Duration // entire Fetch, including waiting, DNS and hops
	RunTimeout     time.Duration // starts at construction, never reset by a Fetch
}

type Result struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// LabBroker must not be copied. Close cancels in-flight and queued work.
// Configuration is copied; requests cannot supply headers, credentials or methods.
type LabBroker struct {
	policy     scope.Policy
	grants     map[string]map[netip.Addr]struct{}
	limits     Limits
	resolver   Resolver
	ctx        context.Context
	cancel     context.CancelFunc
	slots      chan struct{}
	mu         sync.Mutex
	used       int
	dial       func(context.Context, string, string) (net.Conn, error)
	roots      *x509.CertPool // fixture roots are injected only by same-package tests
	permit     project.RunScope
	authorized bool
	stopPermit func() bool
}

func NewLab(ctx context.Context, grants []Grant, limits Limits, resolver Resolver) (*LabBroker, error) {
	if ctx == nil || resolver == nil || len(grants) == 0 || limits.MaxRequests <= 0 ||
		limits.MaxConcurrent <= 0 || limits.MaxConcurrent > 16 || limits.MaxRedirects < 0 || limits.MaxRedirects > 10 ||
		limits.MaxBodyBytes <= 0 || limits.MaxBodyBytes > 8<<20 || limits.RequestTimeout <= 0 || limits.RunTimeout <= 0 {
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
			if !ip.IsValid() || !ip.IsLoopback() || ip.Is4In6() || ip.Zone() != "" {
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
	return &LabBroker{policy: policy, grants: allowed, limits: limits, resolver: resolver,
		ctx: run, cancel: cancel, slots: make(chan struct{}, limits.MaxConcurrent),
		dial: (&net.Dialer{}).DialContext}, nil
}

// NewAuthorizedLab adds an operator-declared run scope to the loopback-only
// broker. Network grants may be broader than the project scope; both checks
// apply to every hop. This is still a synthetic laboratory, not a production
// scanner or proof of target ownership.
func NewAuthorizedLab(ctx context.Context, permit project.RunScope, grants []Grant, limits Limits, resolver Resolver) (*LabBroker, error) {
	if err := permit.Validate(); err != nil {
		return nil, err
	}
	b, err := NewLab(ctx, grants, limits, resolver)
	if err != nil {
		return nil, err
	}
	b.permit = permit
	b.authorized = true
	if lifecycle := permit.Lifecycle(); lifecycle != nil {
		b.stopPermit = context.AfterFunc(lifecycle, b.cancel)
	}
	return b, nil
}

func origin(u *url.URL) string { return u.Scheme + "://" + u.Host }
func (b *LabBroker) Close() {
	b.cancel()
	if b.stopPermit != nil {
		b.stopPermit()
	}
}
func (b *LabBroker) RequestsUsed() int { b.mu.Lock(); defer b.mu.Unlock(); return b.used }

func (b *LabBroker) contextError(ctx context.Context) error {
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

func (b *LabBroker) reserve(ctx context.Context) error {
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
func (b *LabBroker) Fetch(ctx context.Context, raw string) (Result, error) {
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
		if err = b.reserve(ctx); err != nil {
			return Result{}, err
		}
		ip, err := b.resolve(ctx, u)
		if err != nil {
			return Result{}, err
		}
		result, next, err := b.exchange(ctx, u, ip)
		if err != nil {
			return Result{}, err
		}
		if next == "" {
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

func (b *LabBroker) resolve(ctx context.Context, u *url.URL) (netip.Addr, error) {
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

func (b *LabBroker) exchange(ctx context.Context, u *url.URL, ip netip.Addr) (Result, string, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
