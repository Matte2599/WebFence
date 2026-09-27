//go:build m3cdplab && linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var hits atomic.Int32
	var wrongHost atomic.Bool
	var host string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" ||
			r.Header.Get("Proxy-Authorization") != "" {
			wrongHost.Store(true)
		}
		hits.Add(1)
		switch r.URL.Path {
		case "/app/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<html><body><script src="/app/main.js"></script><img src="http://outside.test/x"></body></html>`)
		case "/app/main.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = io.WriteString(w, `console.log('wf-script'); fetch('/app/api').then(r => r.text()).then(x => console.log('wf-api-' + x)); fetch('/app/redirect').then(r => console.log('wf-redirect-' + r.status))`)
		case "/app/api":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "synthetic")
		case "/app/redirect":
			http.Redirect(w, r, "http://outside.test/secret", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer target.Close()
	targetURL, err := url.Parse(target.URL)
	if err != nil {
		return err
	}
	origin := "http://site.test:" + targetURL.Port()
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
	broker, err := transport.NewAuthorizedLabWithPolicy(ctx, permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: maxRequests, MaxConcurrent: 1, MaxRedirects: 2,
			MaxBodyBytes: 16 << 10, RequestTimeout: 5 * time.Second, RunTimeout: 25 * time.Second,
			MinRequestInterval: time.Millisecond}, loopbackResolver{}, policy)
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
	payload, err := json.Marshal(helperConfig{Chrome: chrome, Origin: origin,
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
		result.Document != 1 || result.Script != 1 || result.API != 1 || result.Redirect != 1 ||
		result.OutsideImage != 1 || result.OutsideRedirect != 0 || wrongHost.Load() ||
		hits.Load() != 4 || broker.RequestsUsed() != 4 || dropped != 0 || len(observed) != 3 ||
		observed[0].Path != "/app/" || observed[1].Path != "/app/main.js" ||
		observed[2].Path != "/app/api" || gate.RequestsUsed() < 5 ||
		gate.RequestsUsed() > maxRequests {
		return fmt.Errorf("unexpected CDP broker result: %+v target=%d broker=%d gate=%d observed=%d dropped=%d",
			result, hits.Load(), broker.RequestsUsed(), gate.RequestsUsed(), len(observed), dropped)
	}
	return nil
}

func brokerClient(config helperConfig, inherited chan net.Conn) *http.Client {
	proxyURL := &url.URL{Scheme: "http", Host: "127.0.0.1"}
	proxyURL.User = url.UserPassword(config.Username, config.Password)
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			select {
			case conn, ok := <-inherited:
				if !ok {
					return nil, errors.New("broker IPC exhausted")
				}
				return conn, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}}
	return &http.Client{Transport: transport, Timeout: 6 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
