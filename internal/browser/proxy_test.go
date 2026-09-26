package browser

import (
	"context"
	"encoding/base64"
	"io"
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

func proxyFixture(t *testing.T) (*Proxy, *http.Client, string, *atomic.Int32) {
	t.Helper()
	hits := new(atomic.Int32)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("proxy or browser credentials reached the target")
		}
		switch r.URL.Path {
		case "/app/redirect":
			http.Redirect(w, r, "http://outside.test:8080/app/secret", http.StatusFound)
		case "/app/cookie":
			w.Header().Set("Set-Cookie", "session=synthetic; HttpOnly")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<p>local</p>"))
		default:
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte("window.fixture = true"))
		}
	}))
	t.Cleanup(target.Close)
	origin := target.URL
	p, err := project.New(project.Draft{ID: "browser-proxy", Name: "Synthetic browser lab", TargetOwner: "Fixture",
		AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin}})
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
	gate, err := NewGate(context.Background(), permit, policy,
		Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRuntime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gate.Close)
	broker, err := transport.NewAuthorizedLabWithPolicy(context.Background(), permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRedirects: 2,
			MaxBodyBytes: 1 << 20, RequestTimeout: 3 * time.Second, RunTimeout: time.Minute,
			MinRequestInterval: time.Millisecond}, net.DefaultResolver, policy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(broker.Close)
	proxy, err := NewProxy(context.Background(), gate, broker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	proxyURL, err := url.Parse(proxy.Endpoint())
	if err != nil {
		t.Fatal(err)
	}
	username, password := proxy.Credentials()
	proxyURL.User = url.UserPassword(username, password)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	t.Cleanup(client.CloseIdleConnections)
	return proxy, client, origin, hits
}

func TestProxyForwardsOnlyAdmittedHTTP(t *testing.T) {
	proxy, client, origin, hits := proxyFixture(t)
	if !strings.HasPrefix(proxy.Endpoint(), "http://127.0.0.1:") {
		t.Fatal("proxy not bound to IPv4 loopback")
	}
	response, err := client.Get(origin + "/app/cookie")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "<p>local</p>" ||
		response.Header.Get("Set-Cookie") != "" || response.Header.Get("Content-Security-Policy") != "default-src 'self'" ||
		response.Header.Get("Cache-Control") != "no-store" || hits.Load() != 1 {
		t.Fatalf("unexpected proxy result: status=%d body=%q hits=%d", response.StatusCode, body, hits.Load())
	}
	response, err = client.Head(origin + "/app/script.js")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || hits.Load() != 2 {
		t.Fatalf("HEAD result: status=%d hits=%d", response.StatusCode, hits.Load())
	}
}

func TestProxyDeniesAlternateDestinationsAndMethods(t *testing.T) {
	_, client, origin, hits := proxyFixture(t)
	for _, tt := range []struct {
		name, method, target string
		header               string
		upgrade              bool
		status               int
	}{
		{"outside origin", "GET", "http://outside.test:8080/app", "", false, http.StatusForbidden},
		{"excluded path", "GET", origin + "/app/logout", "", false, http.StatusForbidden},
		{"worker", "GET", origin + "/app/worker.js", "worker", false, http.StatusForbidden},
		{"websocket upgrade", "GET", origin + "/app/socket", "", true, http.StatusForbidden},
		{"post", "POST", origin + "/app/login", "", false, http.StatusForbidden},
		{"redirect outside", "GET", origin + "/app/redirect", "", false, http.StatusBadGateway},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.target, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.header != "" {
				req.Header.Set("Sec-Fetch-Dest", tt.header)
			}
			if tt.upgrade {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}
			response, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != tt.status {
				t.Fatalf("got status %d, want %d", response.StatusCode, tt.status)
			}
		})
	}
	if hits.Load() != 1 { // Only the in-scope redirect source was contacted.
		t.Fatalf("unexpected target requests: %d", hits.Load())
	}
}

func TestProxyRejectsUnauthenticatedAndConnect(t *testing.T) {
	proxy, _, origin, hits := proxyFixture(t)
	proxyURL, err := url.Parse(proxy.Endpoint())
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	defer client.CloseIdleConnections()
	response, err := client.Get(origin + "/app")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusProxyAuthRequired || hits.Load() != 0 {
		t.Fatalf("unauthenticated request: status=%d hits=%d", response.StatusCode, hits.Load())
	}
	username, password := proxy.Credentials()
	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(proxy.Endpoint(), "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, err = io.WriteString(conn, "CONNECT "+strings.TrimPrefix(origin, "http://")+" HTTP/1.1\r\nHost: "+strings.TrimPrefix(origin, "http://")+"\r\nProxy-Authorization: Basic "+
		base64.StdEncoding.EncodeToString([]byte(username+":"+password))+"\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(buffer[:n]), "403 Forbidden") || hits.Load() != 0 {
		t.Fatalf("CONNECT was not rejected: %q, hits=%d", buffer[:n], hits.Load())
	}
}
