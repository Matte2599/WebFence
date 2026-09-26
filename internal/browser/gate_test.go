package browser

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

func testGate(t *testing.T, maxRequests, maxConcurrent int) *Gate {
	t.Helper()
	p, err := project.New(project.Draft{
		ID: "m3-browser", Name: "Synthetic browser lab", TargetOwner: "Fixture",
		AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins:                []string{"http://site.test:8080"},
	})
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
	policy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/app"}, []string{"/app/logout"})
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGate(context.Background(), permit, policy, Limits{MaxRequests: maxRequests,
		MaxConcurrent: maxConcurrent, MaxRuntime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(g.Close)
	return g
}

func TestGateScopeAndResourceTypes(t *testing.T) {
	g := testGate(t, 12, 1)
	tests := []struct {
		name, method, url string
		kind              RequestType
		want              error
	}{
		{"document", "GET", "http://site.test:8080/app", Document, nil},
		{"script", "GET", "http://site.test:8080/app/main.js", Subresource, nil},
		{"fetch", "HEAD", "http://site.test:8080/app/api", Fetch, nil},
		{"third party", "GET", "http://third.test:8080/app/tracker.js", Subresource, scope.ErrOutOfScope},
		{"redirect target", "GET", "http://site.test:8080/app/logout", Document, scope.ErrPathDenied},
		{"post", "POST", "http://site.test:8080/app/login", Fetch, scope.ErrMethodDenied},
		{"worker", "GET", "http://site.test:8080/app/worker.js", Worker, ErrRequestType},
		{"websocket", "GET", "http://site.test:8080/app/socket", WebSocket, ErrRequestType},
		{"local file", "GET", "file:///etc/passwd", Document, scope.ErrInvalidURL},
		{"encoded path", "GET", "http://site.test:8080/app/%2e%2e/secret", Document, scope.ErrAmbiguousPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, err := g.Admit(context.Background(), tt.method, tt.url, tt.kind)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if err == nil {
				release()
				release()
			}
		})
	}
	if got := g.RequestsUsed(); got != len(tests) {
		t.Fatalf("used %d, want %d", got, len(tests))
	}
}

func TestGateBudgetAndConcurrency(t *testing.T) {
	g := testGate(t, 2, 1)
	first, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/app", Document)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := g.Admit(ctx, http.MethodGet, "http://site.test:8080/app/main.js", Subresource); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting request: %v", err)
	}
	first()
	if _, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/app", Document); !errors.Is(err, ErrBudget) {
		t.Fatalf("budget: %v", err)
	}
}

func TestGateCloseCancelsWaiter(t *testing.T) {
	g := testGate(t, 2, 1)
	first, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/app", Document)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/app/main.js", Subresource)
		result <- err
	}()
	g.Close()
	if err := <-result; !errors.Is(err, ErrClosed) && !errors.Is(err, context.Canceled) {
		t.Fatalf("close: %v", err)
	}
	first()
}

func TestGateRequiresManagedScopeAndStopsOnRevocation(t *testing.T) {
	p, err := project.New(project.Draft{
		ID: "m3-revocation", Name: "Synthetic browser lab", TargetOwner: "Fixture",
		AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins:                []string{"http://site.test:8080"},
	})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	limits := Limits{MaxRequests: 2, MaxConcurrent: 1, MaxRuntime: time.Minute}
	if _, err := NewGate(context.Background(), permit, policy, limits); !errors.Is(err, ErrConfig) {
		t.Fatalf("unmanaged scope: %v", err)
	}
	lifecycle, revoke := context.WithCancelCause(context.Background())
	permit, err = permit.BindLifecycle(lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGate(context.Background(), permit, policy, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	first, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/", Document)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := g.Admit(context.Background(), http.MethodGet, "http://site.test:8080/next", Document)
		result <- err
	}()
	revoke(project.ErrAuthorizationRevoked)
	if err := <-result; !errors.Is(err, project.ErrAuthorizationRevoked) {
		t.Fatalf("revoked waiter: %v", err)
	}
	first()
}
