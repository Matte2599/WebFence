//go:build m3browserlab

// This optional lab exercises Qt WebEngine against synthetic HTTP fixtures.
// It never accepts a target URL and is not a browser feature of the desktop.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
	qt "github.com/mappu/miqt/qt6"
	network "github.com/mappu/miqt/qt6/network"
	webengine "github.com/mappu/miqt/qt6/webengine"
)

type loopbackResolver struct{}

func (loopbackResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "M3 browser lab failed:", err)
		os.Exit(1)
	}
	fmt.Println("PASS M3 Qt browser HTTP fixture: document, script, fetch, proxy and out-of-scope block")
}

func run() error {
	if len(os.Args) != 1 {
		return errors.New("the lab accepts no target arguments")
	}
	if err := os.Setenv("QTWEBENGINE_CHROMIUM_FLAGS",
		"--disable-background-networking --disable-component-update --disable-sync --disable-extensions --disable-default-apps"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var targetHits atomic.Int32
	var wrongHost atomic.Bool
	var targetHost string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != targetHost || r.Header.Get("Proxy-Authorization") != "" ||
			r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			wrongHost.Store(true)
		}
		targetHits.Add(1)
		switch r.URL.Path {
		case "/app":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body><script src="/app/main.js"></script><img src="http://outside.test:8080/x"></body></html>`)
		case "/app/main.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = fmt.Fprint(w, `fetch('/app/api').then(r => r.text()).then(x => { if (x === 'synthetic') console.log('wf-synthetic-api-ok') }); fetch('/app/redirect')`)
		case "/app/api":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "synthetic")
		case "/app/redirect":
			http.Redirect(w, r, "http://outside.test:8080/secret", http.StatusFound)
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
	targetHost = "site.test:" + targetURL.Port()
	declaration, err := project.New(project.Draft{
		ID: "m3-qt-browser-lab", Name: "Synthetic Qt browser lab", TargetOwner: "Fixture",
		AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour), Origins: []string{origin},
	})
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
	gate, err := browser.NewGate(ctx, permit, policy, browser.Limits{
		MaxRequests: 20, MaxConcurrent: 1, MaxRuntime: 20 * time.Second,
	})
	if err != nil {
		return err
	}
	defer gate.Close()
	broker, err := transport.NewAuthorizedLabWithPolicy(ctx, permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRedirects: 2,
			MaxBodyBytes: 1 << 20, RequestTimeout: 5 * time.Second, RunTimeout: 20 * time.Second,
			MinRequestInterval: time.Millisecond}, loopbackResolver{}, policy)
	if err != nil {
		return err
	}
	defer broker.Close()
	proxy, err := browser.NewProxy(ctx, gate, broker)
	if err != nil {
		return err
	}
	defer proxy.Close()

	runtime.LockOSThread()
	qt.NewQApplication(os.Args)
	endpoint, err := url.Parse(proxy.Endpoint())
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(endpoint.Port())
	if err != nil {
		return err
	}
	username, password := proxy.Credentials()
	qtProxy := network.NewQNetworkProxy7(network.QNetworkProxy__HttpProxy,
		"127.0.0.1", uint16(port), username, password)
	defer qtProxy.Delete()
	network.QNetworkProxy_SetApplicationProxy(qtProxy)
	profile := webengine.NewQWebEngineProfile()
	defer profile.Delete()
	if !profile.IsOffTheRecord() {
		return errors.New("browser profile is persistent")
	}
	var denied atomic.Int32
	interceptor := webengine.NewQWebEngineUrlRequestInterceptor()
	defer interceptor.Delete()
	interceptor.OnInterceptRequest(func(info *webengine.QWebEngineUrlRequestInfo) {
		kind := info.ResourceType()
		if kind == webengine.QWebEngineUrlRequestInfo__ResourceTypeWorker ||
			kind == webengine.QWebEngineUrlRequestInfo__ResourceTypeSharedWorker ||
			kind == webengine.QWebEngineUrlRequestInfo__ResourceTypeServiceWorker ||
			kind == webengine.QWebEngineUrlRequestInfo__ResourceTypeWebSocket {
			info.Block(true)
			denied.Add(1)
			return
		}
		u, err := permit.CheckOrigin(info.RequestUrl().ToString())
		if err == nil {
			err = policy.Check(string(info.RequestMethod()), u)
		}
		if err != nil {
			info.Block(true)
			denied.Add(1)
		}
	})
	profile.SetUrlRequestInterceptor(interceptor)
	page := webengine.NewQWebEnginePage2(profile)
	defer page.Delete()
	var loaded atomic.Bool
	var apiSeen atomic.Bool
	page.OnLoadFinished(func(ok bool) { loaded.Store(ok) })
	page.OnJavaScriptConsoleMessage(func(super func(webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, string, int, string),
		level webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, message string, line int, source string) {
		if message == "wf-synthetic-api-ok" {
			apiSeen.Store(true)
		}
	})
	timer := qt.NewQTimer()
	defer timer.Delete()
	timer.OnTimeout(qt.QCoreApplication_Quit)
	timer.Start(5000)
	page.Load(qt.NewQUrl3(origin + "/app"))
	qt.QApplication_Exec()
	if !loaded.Load() || !apiSeen.Load() || wrongHost.Load() || denied.Load() < 1 ||
		targetHits.Load() != 4 || gate.RequestsUsed() != 4 || broker.RequestsUsed() != 4 {
		return fmt.Errorf("unexpected synthetic observations: loaded=%t api=%t host=%t denied=%d target=%d gate=%d broker=%d",
			loaded.Load(), apiSeen.Load(), wrongHost.Load(), denied.Load(), targetHits.Load(), gate.RequestsUsed(), broker.RequestsUsed())
	}
	return nil
}
