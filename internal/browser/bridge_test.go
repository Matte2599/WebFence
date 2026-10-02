package browser

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

func TestBridgeRejectsUnboundDifferentRunAndBroaderPolicy(t *testing.T) {
	p, _, origin, hits := proxyFixture(t)
	limits := transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxBodyBytes: 1024, RequestTimeout: time.Second, RunTimeout: time.Minute, MinRequestInterval: time.Millisecond}
	grants := []transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}}
	unbound, err := transport.NewLab(context.Background(), grants, limits, net.DefaultResolver)
	if err != nil {
		t.Fatal(err)
	}
	defer unbound.Close()
	declaration, err := project.New(project.Draft{ID: p.gate.ProjectID(), Name: "Fixture", TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: p.gate.permit.ExpiresAt(), Origins: []string{origin}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := declaration.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	other, err = other.BindLifecycle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := transport.NewAuthorizedLabWithPolicy(context.Background(), other, grants, limits, net.DefaultResolver, p.gate.policy)
	if err != nil {
		t.Fatal(err)
	}
	defer foreign.Close()
	broadPolicy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	broad, err := transport.NewAuthorizedLabWithPolicy(context.Background(), p.gate.permit, grants, limits, net.DefaultResolver, broadPolicy)
	if err != nil {
		t.Fatal(err)
	}
	defer broad.Close()
	for _, broker := range []*transport.Broker{unbound, foreign, broad} {
		if bridge, err := NewRequestBridge(context.Background(), p.gate, broker); bridge != nil || !errors.Is(err, ErrConfig) {
			t.Fatal("mismatched broker accepted")
		}
	}
	if hits.Load() != 0 {
		t.Fatal("constructor used network")
	}
	// Equivalent independent policies may reorder methods/prefixes safely.
	equivalent, err := scope.NewRequestPolicy([]string{"HEAD", "GET"}, []string{"/app", "/app"}, []string{"/app/logout"})
	if err != nil {
		t.Fatal(err)
	}
	compatible, err := transport.NewAuthorizedLabWithPolicy(context.Background(), p.gate.permit, grants, limits, net.DefaultResolver, equivalent)
	if err != nil {
		t.Fatal(err)
	}
	defer compatible.Close()
	bridge, err := NewRequestBridge(context.Background(), p.gate, compatible)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	if bridge.Forward(context.Background(), "HEAD", origin+"/app/cookie", Document).Status != 200 {
		t.Fatal("compatible bound broker denied")
	}
}

func TestBridgeAdmissionHeadersAndRedaction(t *testing.T) {
	p, _, origin, hits := proxyFixture(t)
	b, err := NewRequestBridge(context.Background(), p.gate, p.broker)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for _, test := range []struct {
		method, raw string
		kind        RequestType
		status      int
	}{
		{"GET", origin + "/app/cookie?token=synthetic", Document, 200},
		{"HEAD", origin + "/app/script", Subresource, 200},
		{"POST", origin + "/app/cookie", Fetch, 403},
		{"GET", origin + "/app/logout", Document, 403},
		{"GET", "http://outside.test/app", Subresource, 403},
		{"GET", origin + "/app/worker", Worker, 403},
		{"GET", origin + "/app/socket", WebSocket, 403},
		{"GET", origin + "/app/redirect", Fetch, 502},
		{"GET", origin + "/app/allowed-redirect", Fetch, 502},
	} {
		reply := b.Forward(context.Background(), test.method, test.raw, test.kind)
		if reply.Status != test.status {
			t.Fatalf("%s %s: status %d", test.method, test.kind, reply.Status)
		}
		if reply.Header.Get("Set-Cookie") != "" {
			t.Fatal("target cookie reached browser")
		}
		if test.method == "HEAD" && len(reply.Body) != 0 {
			t.Fatal("HEAD returned body")
		}
		if reply.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("response can be cached")
		}
		if test.kind == Document && reply.Status == 200 && reply.Header.Get("Content-Security-Policy") == "" {
			t.Fatal("target restrictions lost")
		}
	}
	if hits.Load() != 5 {
		t.Fatalf("denied request contacted target: %d", hits.Load())
	}
	observed, dropped := b.Observations()
	if len(observed) != 2 || dropped != 0 || observed[0].Path != "/app/cookie" {
		t.Fatalf("observations=%+v dropped=%d", observed, dropped)
	}
	observed[0].Path = "changed"
	again, _ := b.Observations()
	if again[0].Path == "changed" {
		t.Fatal("mutable observations")
	}
	b.Close()
	if b.Forward(context.Background(), "GET", origin+"/app/cookie", Document).Status != 403 || hits.Load() != 5 {
		t.Fatal("closed bridge used target")
	}
}

func newBridgeFixture(t *testing.T, target *httptest.Server, origin string, roots *x509.CertPool, resolver transport.Resolver) *RequestBridge {
	t.Helper()
	p, err := project.New(project.Draft{ID: "bridge", Name: "Fixture", TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Minute), Origins: []string{origin}})
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
	policy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := NewGate(context.Background(), permit, policy, Limits{MaxRequests: 300, MaxConcurrent: 1, MaxRuntime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gate.Close)
	limits := transport.Limits{MaxRequests: 300, MaxConcurrent: 1, MaxBodyBytes: 1 << 20, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute, MinRequestInterval: time.Millisecond}
	grants := []transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}}
	var broker *transport.Broker
	if roots == nil {
		broker, err = transport.NewAuthorizedLabWithPolicy(context.Background(), permit, grants, limits, resolver, policy)
	} else {
		broker, err = transport.NewAuthorizedLabWithPolicyTLSRoots(context.Background(), permit, grants, limits, resolver, policy, roots)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(broker.Close)
	b, err := NewRequestBridge(context.Background(), gate, broker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Close)
	return b
}

func TestBridgeResponseLimitsAndGateCancellation(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/large":
			_, _ = w.Write([]byte(strings.Repeat("x", MaxBridgeBody+1)))
		case "/app/header":
			w.Header().Set("Content-Security-Policy", strings.Repeat("x", 8193))
		case "/app/slow":
			close(started)
			<-r.Context().Done()
			close(canceled)
		}
	}))
	defer target.Close()
	b := newBridgeFixture(t, target, target.URL, nil, net.DefaultResolver)
	for _, path := range []string{"/app/large", "/app/header"} {
		if b.Forward(context.Background(), "GET", target.URL+path, Fetch).Status != 502 {
			t.Fatal("oversized reply accepted")
		}
	}
	done := make(chan BridgeReply, 1)
	go func() { done <- b.Forward(context.Background(), "GET", target.URL+"/app/slow", Fetch) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("target not started")
	}
	b.gate.Close()
	select {
	case reply := <-done:
		if reply.Status != 502 {
			t.Fatal("canceled request succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("gate close did not cancel bridge")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("target still active")
	}
	observed, _ := b.Observations()
	if len(observed) != 0 {
		t.Fatal("failed response observed as successful")
	}
}

type bridgeResolver struct{}

func (bridgeResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func TestBridgeVerifiesTLSHostname(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	u, _ := url.Parse(target.URL)
	roots := x509.NewCertPool()
	roots.AddCert(target.Certificate())
	b := newBridgeFixture(t, target, "https://site.test:"+u.Port(), roots, bridgeResolver{})
	if b.Forward(context.Background(), "GET", "https://site.test:"+u.Port()+"/app/", Document).Status != 502 || hits.Load() != 0 {
		t.Fatal("wrong TLS hostname reached HTTP handler")
	}
}
