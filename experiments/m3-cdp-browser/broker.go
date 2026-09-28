//go:build m3cdplab && linux

package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

const brokerConnections = 8

type loopbackResolver struct{}

func (loopbackResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func runParent() error {
	chrome := os.Getenv("WF_CDP_CHROME")
	if !filepath.IsAbs(chrome) || len(chrome) > 4096 {
		return errors.New("absolute synthetic-lab Chromium path required")
	}
	info, err := os.Stat(chrome)
	if err != nil || info.IsDir() {
		return errors.New("Chromium executable is unavailable")
	}
	for _, scheme := range []string{"http", "https"} {
		if err := runTrial(chrome, scheme); err != nil {
			return fmt.Errorf("%s CDP trial: %w", scheme, err)
		}
	}
	return nil
}

func runTrial(chrome, scheme string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var hits atomic.Int32
	var canaryHits atomic.Int32
	var wrongHost atomic.Bool
	var host string
	canary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		canaryHits.Add(1)
		http.NotFound(w, r)
	}))
	defer canary.Close()
	canaryURL, err := url.Parse(canary.URL)
	if err != nil {
		return err
	}
	outside := scheme + "://outside.test"
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" ||
			r.Header.Get("Proxy-Authorization") != "" {
			wrongHost.Store(true)
		}
		hits.Add(1)
		switch r.URL.Path {
		case "/app/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<html><body><script src="/app/main.js"></script><img src="`+outside+`/x"></body></html>`)
		case "/app/main.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = io.WriteString(w, `console.log('wf-script'); console.log('wf-secure-' + isSecureContext); fetch('/app/api').then(r => r.text()).then(x => console.log('wf-api-' + x)); fetch('/app/redirect').then(r => console.log('wf-redirect-' + r.status))`)
		case "/app/api":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "synthetic")
		case "/app/redirect":
			http.Redirect(w, r, outside+"/secret", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	if scheme == "https" {
		certificate, err := syntheticTLSCertificate()
		if err != nil {
			return err
		}
		target.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
		target.StartTLS()
	} else {
		target.Start()
	}
	defer target.Close()
	targetURL, err := url.Parse(target.URL)
	if err != nil {
		return err
	}
	origin := scheme + "://site.test:" + targetURL.Port()
	host = "site.test:" + targetURL.Port()
	declaration, err := project.New(project.Draft{ID: "m3-cdp-broker-lab", Name: "Synthetic CDP broker lab",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin}})
	if err != nil {
		return err
	}
	permit, err := declaration.BeginRun()
	if err != nil {
		return err
	}
	permit, err = permit.BindLifecycle(ctx)
	if err != nil {
		return err
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	if err != nil {
		return err
	}
	gate, err := browser.NewGate(ctx, permit, policy, browser.Limits{MaxRequests: maxRequests,
		MaxConcurrent: 4, MaxRuntime: 25 * time.Second})
	if err != nil {
		return err
	}
	defer gate.Close()
	grants := []transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}}
	limits := transport.Limits{MaxRequests: maxRequests, MaxConcurrent: 1, MaxRedirects: 2,
		MaxBodyBytes: 16 << 10, RequestTimeout: 5 * time.Second, RunTimeout: 25 * time.Second,
		MinRequestInterval: time.Millisecond}
	var broker *transport.Broker
	if scheme == "https" {
		roots := x509.NewCertPool()
		roots.AddCert(target.Certificate())
		broker, err = transport.NewAuthorizedLabWithPolicyTLSRoots(ctx, permit, grants, limits, loopbackResolver{}, policy, roots)
	} else {
		broker, err = transport.NewAuthorizedLabWithPolicy(ctx, permit, grants, limits, loopbackResolver{}, policy)
	}
	if err != nil {
		return err
	}
	defer broker.Close()
	directory, err := os.MkdirTemp("", "wf-cdp-proxy-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	proxy, socket, err := browser.NewObservedUnixProxy(ctx, gate, broker, directory, maxRequests)
	if err != nil {
		return err
	}
	defer proxy.Close()
	var files []*os.File
	defer func() {
		for _, file := range files {
			_ = file.Close()
		}
	}()
	for range brokerConnections {
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
		if err != nil {
			return err
		}
		file, fileErr := conn.File()
		_ = conn.Close()
		if fileErr != nil {
			return fileErr
		}
		files = append(files, file)
	}
	username, password := proxy.Credentials()
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(helperConfig{Chrome: chrome, Origin: origin, CanaryAddress: canaryURL.Host,
		BrokerFDs: len(files), Username: username, Password: password})
	if err != nil {
		return err
	}
	resultJSON, err := browser.RunHelper(ctx, executable, payload, browser.HelperLimits{
		MaxRuntime: 25 * time.Second, MaxOutputBytes: 4096, InheritedFiles: files})
	if err != nil {
		return err
	}
	var result trialResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return errors.New("invalid CDP helper result")
	}
	observed, dropped := proxy.Observations()
	if !result.Loaded || !result.ScriptSeen || !result.APISeen || !result.RedirectBlocked ||
		result.SecureContext != (scheme == "https") ||
		result.Document != 1 || result.Script != 1 || result.API != 1 || result.Redirect != 1 ||
		result.OutsideImage != 1 || result.OutsideRedirect != 0 || wrongHost.Load() ||
		canaryHits.Load() != 0 ||
		hits.Load() != 4 || broker.RequestsUsed() != 4 || dropped != 0 || len(observed) != 3 ||
		observed[0].Path != "/app/" || observed[1].Path != "/app/main.js" ||
		observed[2].Path != "/app/api" || gate.RequestsUsed() < 5 ||
		gate.RequestsUsed() > maxRequests {
		return fmt.Errorf("unexpected CDP broker result: %+v target=%d canary=%d broker=%d gate=%d observed=%d dropped=%d",
			result, hits.Load(), canaryHits.Load(), broker.RequestsUsed(), gate.RequestsUsed(), len(observed), dropped)
	}
	return nil
}

func syntheticTLSCertificate() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"site.test"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func brokerClient(config helperConfig, inherited chan net.Conn) *http.Client {
	authorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Username+":"+config.Password))
	return &http.Client{Transport: &brokerIPCTransport{inherited: inherited, authorization: authorization}, Timeout: 6 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// brokerIPCTransport sends absolute HTTP(S) URLs through already connected
// Unix sockets. The helper cannot dial a target, establish CONNECT or decide
// TLS trust; the parent-side broker validates each HTTPS peer and hostname.
type brokerIPCTransport struct {
	inherited     <-chan net.Conn
	authorization string
}

func (t *brokerIPCTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || (request.URL.Scheme != "http" && request.URL.Scheme != "https") ||
		(request.Method != http.MethodGet && request.Method != http.MethodHead) || request.Body != nil {
		return nil, errors.New("unsupported broker IPC request")
	}
	var conn net.Conn
	select {
	case conn = <-t.inherited:
		if conn == nil {
			return nil, errors.New("broker IPC exhausted")
		}
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}
	deadline := time.Now().Add(6 * time.Second)
	if earlier, ok := request.Context().Deadline(); ok && earlier.Before(deadline) {
		deadline = earlier
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, err
	}
	stop := context.AfterFunc(request.Context(), func() { _ = conn.Close() })
	fail := func(err error) (*http.Response, error) {
		stop()
		_ = conn.Close()
		return nil, err
	}
	forward := request.Clone(request.Context())
	forward.Close = true
	forward.Header.Set("Proxy-Authorization", t.authorization)
	if err := forward.WriteProxy(conn); err != nil {
		return fail(err)
	}
	response, err := http.ReadResponse(bufio.NewReaderSize(conn, 32<<10), forward)
	if err != nil {
		return fail(err)
	}
	response.Body = &ipcResponseBody{ReadCloser: response.Body, conn: conn, stop: stop}
	return response, nil
}

type ipcResponseBody struct {
	io.ReadCloser
	conn net.Conn
	stop func() bool
}

func (b *ipcResponseBody) Close() error {
	b.stop()
	err := b.ReadCloser.Close()
	_ = b.conn.Close()
	return err
}
