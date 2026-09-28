//go:build darwin || linux

package browser

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
)

type fixedTLSResolver struct{}

func (fixedTLSResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func TestUnixProxyMediatesAbsoluteHTTPSAndRejectsWrongHostname(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("proxy credentials or browser cookie reached TLS target")
		}
		_, _ = io.WriteString(w, "tls-fixture")
	}))
	t.Cleanup(target.Close)
	targetURL, err := url.Parse(target.URL)
	if err != nil || len(target.Certificate().DNSNames) == 0 {
		t.Fatal("TLS fixture has no DNS name")
	}
	origin := "https://" + net.JoinHostPort(target.Certificate().DNSNames[0], targetURL.Port())
	wrongOrigin := "https://wrong.test:" + targetURL.Port()
	p, err := project.New(project.Draft{ID: "browser-https-lab", Name: "Synthetic HTTPS browser lab",
		TargetOwner: "Fixture", AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin, wrongOrigin}})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	permit, err = permit.BindLifecycle(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := NewGate(t.Context(), permit, policy, Limits{MaxRequests: 4, MaxConcurrent: 1, MaxRuntime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gate.Close)
	grants := []transport.Grant{
		{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
		{Origin: wrongOrigin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
	}
	roots := x509.NewCertPool()
	roots.AddCert(target.Certificate())
	broker, err := transport.NewAuthorizedLabWithPolicyTLSRoots(t.Context(), permit, grants,
		transport.Limits{MaxRequests: 4, MaxConcurrent: 1, MaxRedirects: 0,
			MaxBodyBytes: 1024, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, fixedTLSResolver{}, policy, roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(broker.Close)
	directory, err := os.MkdirTemp("", "wf-https-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	proxy, socket, err := NewObservedUnixProxy(t.Context(), gate, broker, directory, 4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	username, password := proxy.Credentials()
	request := func(targetURL string) (int, string) {
		t.Helper()
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
		if err := req.WriteProxy(conn); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, string(body)
	}
	if status, body := request(origin + "/app/"); status != http.StatusOK || body != "tls-fixture" || hits.Load() != 1 {
		t.Fatalf("trusted HTTPS: status=%d body=%q hits=%d", status, body, hits.Load())
	}
	if status, _ := request(wrongOrigin + "/app/"); status != http.StatusBadGateway || hits.Load() != 1 {
		t.Fatalf("wrong TLS hostname: status=%d hits=%d", status, hits.Load())
	}
	if status, _ := request("https://outside.test/app/"); status != http.StatusForbidden || hits.Load() != 1 {
		t.Fatalf("outside HTTPS origin: status=%d hits=%d", status, hits.Load())
	}
	observed, dropped := proxy.Observations()
	if len(observed) != 1 || dropped != 0 || observed[0].Path != "/app/" ||
		observed[0].StatusCode != http.StatusOK {
		t.Fatalf("HTTPS observations=%+v dropped=%d", observed, dropped)
	}
}

func TestUnixProxyUsesGateBrokerAndPrivateSocket(t *testing.T) {
	fixture, _, origin, hits := proxyFixture(t)
	directory, err := os.MkdirTemp("", "wf-unix-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	proxy, socketPath, err := NewObservedUnixProxy(context.Background(), fixture.gate, fixture.broker, directory, 8)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	if socketPath != filepath.Join(directory, "broker.sock") || proxy.Endpoint() != "" {
		t.Fatal("Unix proxy exposed a TCP endpoint or unexpected socket path")
	}
	username, password := proxy.Credentials()
	proxyURL := &url.URL{Scheme: "http", Host: "127.0.0.1"}
	proxyURL.User = url.UserPassword(username, password)
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DialContext: func(context.Context, string, string) (net.Conn, error) {
		return net.Dial("unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	response, err := client.Get(origin + "/app/script.js")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || string(body) != "window.fixture = true" {
		t.Fatalf("Unix proxy response = %d %q, %v", response.StatusCode, body, err)
	}
	response, err = client.Get("http://outside.test:8080/app")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden || hits.Load() != 1 {
		t.Fatalf("outside request status=%d hits=%d", response.StatusCode, hits.Load())
	}
	observed, dropped := proxy.Observations()
	if len(observed) != 1 || dropped != 0 || observed[0].Path != "/app/script.js" {
		t.Fatalf("Unix proxy observations = %+v dropped=%d", observed, dropped)
	}
}

func TestUnixProxyRejectsNonprivateDirectory(t *testing.T) {
	fixture, _, _, _ := proxyFixture(t)
	directory := t.TempDir()
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewObservedUnixProxy(context.Background(), fixture.gate, fixture.broker, directory, 8); err == nil {
		t.Fatal("nonprivate socket directory accepted")
	}
}
