//go:build darwin || linux

package browser

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

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
