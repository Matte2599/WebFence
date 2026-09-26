// Package browser contains policy for a future isolated browser adapter.
// A Gate does not launch a browser or provide network containment by itself.
package browser

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

var (
	ErrConfig      = errors.New("browser_invalid_configuration")
	ErrBudget      = errors.New("browser_request_budget_exhausted")
	ErrRequestType = errors.New("browser_request_type_denied")
	ErrClosed      = errors.New("browser_gate_closed")
)

// RequestType identifies a browser-initiated request. The initial policy
// allows document, subresource, and fetch requests. Workers and WebSockets
// remain disabled until their containment is proven by an adapter.
type RequestType string

const (
	Document    RequestType = "document"
	Subresource RequestType = "subresource"
	Fetch       RequestType = "fetch"
	Worker      RequestType = "worker"
	WebSocket   RequestType = "websocket"
)

type Limits struct {
	// MaxRequests counts every request considered, including denied requests.
	MaxRequests   int
	MaxConcurrent int
	MaxRuntime    time.Duration
}

type Gate struct {
	permit     project.RunScope
	policy     scope.RequestPolicy
	limits     Limits
	ctx        context.Context
	cancel     context.CancelFunc
	stopPermit func() bool
	slots      chan struct{}
	mu         sync.Mutex
	used       int
	closed     bool
}

// NewGate binds browser request admission to a managed run's immutable scope.
// The caller must separately ensure every browser network path passes this
// gate and the pinned transport; an interceptor alone is not an egress fence.
func NewGate(ctx context.Context, permit project.RunScope, policy scope.RequestPolicy, limits Limits) (*Gate, error) {
	if ctx == nil || permit.Validate() != nil || permit.Lifecycle() == nil || !policy.Valid() || limits.MaxRequests <= 0 ||
		limits.MaxRequests > 10000 || limits.MaxConcurrent <= 0 || limits.MaxConcurrent > 16 ||
		limits.MaxRuntime <= 0 || limits.MaxRuntime > 4*time.Hour {
		return nil, ErrConfig
	}
	deadline := time.Now().Add(limits.MaxRuntime)
	if permit.ExpiresAt().Before(deadline) {
		deadline = permit.ExpiresAt()
	}
	run, cancel := context.WithDeadline(ctx, deadline)
	return &Gate{permit: permit, policy: policy, limits: limits, ctx: run, cancel: cancel,
		stopPermit: context.AfterFunc(permit.Lifecycle(), cancel),
		slots:      make(chan struct{}, limits.MaxConcurrent)}, nil
}

func (g *Gate) Close() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()
	g.cancel()
	g.stopPermit()
}

func (g *Gate) RequestsUsed() int {
	if g == nil {
		return 0
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.used
}

// Admit validates every navigation, redirect and resource before the adapter
// permits it to reach the network. A successful caller must release the lease
// after the request completes. Errors contain no URL or credential material.
func (g *Gate) Admit(ctx context.Context, method, raw string, kind RequestType) (release func(), err error) {
	if g == nil || ctx == nil {
		return nil, ErrConfig
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil, ErrClosed
	}
	if g.used >= g.limits.MaxRequests {
		g.mu.Unlock()
		return nil, ErrBudget
	}
	g.used++
	g.mu.Unlock()
	if err := g.active(ctx); err != nil {
		return nil, err
	}
	switch kind {
	case Document, Subresource, Fetch:
	case Worker, WebSocket:
		return nil, ErrRequestType
	default:
		return nil, ErrRequestType
	}
	if method != http.MethodGet && method != http.MethodHead {
		return nil, scope.ErrMethodDenied
	}
	u, err := g.permit.CheckOrigin(raw)
	if err != nil {
		return nil, err
	}
	if err := g.policy.Check(method, u); err != nil {
		return nil, err
	}
	select {
	case g.slots <- struct{}{}:
		if err := g.active(ctx); err != nil {
			<-g.slots
			return nil, err
		}
		var once sync.Once
		return func() { once.Do(func() { <-g.slots }) }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-g.ctx.Done():
		return nil, g.active(ctx)
	}
}

func (g *Gate) active(ctx context.Context) error {
	if err := g.permit.Validate(); err != nil {
		return err
	}
	if err := g.ctx.Err(); err != nil {
		return err
	}
	return ctx.Err()
}
