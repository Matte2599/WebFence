// Package headlessfixture is a synthetic parent-side HTTP(S) fixture shared
// by macOS and Windows labs. It accepts no external target configuration.
package headlessfixture

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

type resolver struct{}

func (resolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

type Fixture struct {
	Origin                    string
	Outside                   string
	bridge                    *browser.RequestBridge
	gate                      *browser.Gate
	broker                    *transport.Broker
	target, outside           *httptest.Server
	cancel                    context.CancelFunc
	revoke                    context.CancelCauseFunc
	hits                      atomic.Int32
	outsideHits               atomic.Int32
	wrongHeaders              atomic.Bool
	slowStarted, slowCanceled chan struct{}
	slowOnce                  sync.Once
}

func New(scheme string) (_ *Fixture, err error) {
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("invalid fixture scheme")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	f := &Fixture{cancel: cancel, slowStarted: make(chan struct{}), slowCanceled: make(chan struct{})}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	f.outside = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.outsideHits.Add(1)
		http.NotFound(w, r)
	}))
	outsideURL, err := url.Parse(f.outside.URL)
	if err != nil {
		return nil, err
	}
	f.Outside = scheme + "://outside.test:" + outsideURL.Port()
	f.target = httptest.NewUnstartedServer(http.HandlerFunc(f.serve))
	if scheme == "https" {
		certificate, certErr := certificate()
		if certErr != nil {
			return nil, certErr
		}
		f.target.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
		f.target.StartTLS()
	} else {
		f.target.Start()
	}
	u, err := url.Parse(f.target.URL)
	if err != nil {
		return nil, err
	}
	f.Origin = scheme + "://site.test:" + u.Port()
	declaration, err := project.New(project.Draft{ID: "m3-headless-broker", Name: "Synthetic headless broker",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{f.Origin}})
	if err != nil {
		return nil, err
	}
	permit, err := declaration.BeginRun()
	if err != nil {
		return nil, err
	}
	lifecycle, revoke := context.WithCancelCause(ctx)
	f.revoke = revoke
	permit, err = permit.BindLifecycle(lifecycle)
	if err != nil {
		return nil, err
	}
	policy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/app"}, nil)
	if err != nil {
		return nil, err
	}
	f.gate, err = browser.NewGate(ctx, permit, policy, browser.Limits{MaxRequests: 16, MaxConcurrent: 4, MaxRuntime: 20 * time.Second})
	if err != nil {
		return nil, err
	}
	grants := []transport.Grant{{Origin: f.Origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}}
	limits := transport.Limits{MaxRequests: 16, MaxConcurrent: 1, MaxRedirects: 2, MaxBodyBytes: browser.MaxBridgeBody,
		RequestTimeout: 5 * time.Second, RunTimeout: 20 * time.Second, MinRequestInterval: time.Millisecond}
	if scheme == "https" {
		roots := x509.NewCertPool()
		roots.AddCert(f.target.Certificate())
		f.broker, err = transport.NewAuthorizedLabWithPolicyTLSRoots(ctx, permit, grants, limits, resolver{}, policy, roots)
	} else {
		f.broker, err = transport.NewAuthorizedLabWithPolicy(ctx, permit, grants, limits, resolver{}, policy)
	}
	if err != nil {
		return nil, err
	}
	f.bridge, err = browser.NewRequestBridge(ctx, f.gate, f.broker)
	if err != nil {
		return nil, err
	}
	go func() {
		select {
		case <-f.slowStarted:
			revoke(project.ErrAuthorizationRevoked)
		case <-ctx.Done():
		}
	}()
	return f, nil
}

func (f *Fixture) serve(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(f.Origin)
	if r.Host != u.Host || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Proxy-Authorization") != "" {
		f.wrongHeaders.Store(true)
	}
	f.hits.Add(1)
	switch r.URL.Path {
	case "/app/":
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Set-Cookie", "synthetic_target_secret=must-not-reach-browser; Path=/")
		_, _ = io.WriteString(w, `<html><body><script src="/app/main.js"></script><img src="`+f.Outside+`/outside"><p>mediated fixture</p></body></html>`)
	case "/app/main.js":
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = io.WriteString(w, `window.wfResult=''; (async()=>{try { const api=await fetch('/app/api',{headers:{Authorization:'Bearer synthetic-browser-secret'}}); const text=await api.text(); const redirect=await fetch('/app/redirect'); const revoked=await fetch('/app/slow'); const after=await fetch('/app/after-revoke'); window.wfResult=[location.origin,String(isSecureContext),document.cookie,text,redirect.status,revoked.status,after.status].join('|'); } catch(e){window.wfResult='failed';}})();`)
	case "/app/api":
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "synthetic")
	case "/app/redirect":
		http.Redirect(w, r, f.Outside+"/redirect", http.StatusFound)
	case "/app/slow":
		f.slowOnce.Do(func() { close(f.slowStarted) })
		<-r.Context().Done()
		close(f.slowCanceled)
	default:
		http.NotFound(w, r)
	}
}

func (f *Fixture) Forward(ctx context.Context, method, raw, resourceType string) browser.BridgeReply {
	kind := browser.Subresource
	switch resourceType {
	case "Document":
		kind = browser.Document
	case "Fetch", "XHR":
		kind = browser.Fetch
	case "Worker", "ServiceWorker", "SharedWorker":
		kind = browser.Worker
	case "WebSocket":
		kind = browser.WebSocket
	}
	return f.bridge.Forward(ctx, method, raw, kind)
}

func (f *Fixture) Verify() error {
	observed, dropped := f.bridge.Observations()
	if f.hits.Load() != 5 || f.outsideHits.Load() != 0 || f.wrongHeaders.Load() || f.broker.RequestsUsed() != 5 ||
		f.gate.RequestsUsed() < 7 || len(observed) != 3 || dropped != 0 {
		return errors.New("headless fixture request/identity/observation invariants failed")
	}
	select {
	case <-f.slowCanceled:
		return nil
	case <-time.After(time.Second):
		return errors.New("in-flight revocation failed")
	}
}

func (f *Fixture) Close() {
	if f.cancel != nil {
		f.cancel()
	}
	if f.revoke != nil {
		f.revoke(context.Canceled)
	}
	if f.bridge != nil {
		f.bridge.Close()
	}
	if f.broker != nil {
		f.broker.Close()
	}
	if f.gate != nil {
		f.gate.Close()
	}
	if f.target != nil {
		f.target.Close()
	}
	if f.outside != nil {
		f.outside.Close()
	}
}

func certificate() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"site.test"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, err
}
