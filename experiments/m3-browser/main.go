//go:build m3browserlab

// This optional lab exercises Qt WebEngine against synthetic HTTP fixtures.
// It never accepts a target URL and is not a browser feature of the desktop.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type helperConfig struct {
	Origin        string `json:"origin"`
	ProxyEndpoint string `json:"proxy_endpoint"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type helperResult struct {
	Loaded          bool `json:"loaded"`
	APISeen         bool `json:"api_seen"`
	Navigated       bool `json:"navigated"`
	DynamicSeen     bool `json:"dynamic_seen"`
	RedirectBlocked bool `json:"redirect_blocked"`
	OutsideBlocked  bool `json:"outside_blocked"`
	Denied          int  `json:"denied"`
}

func (loopbackResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--browser-helper" {
		if err := runChild(); err != nil {
			fmt.Fprintln(os.Stderr, "M3 browser helper failed:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "M3 browser lab failed:", err)
		os.Exit(1)
	}
	fmt.Println("PASS M3 Qt browser HTTP fixture: document, DOM navigation, fetch, proxy and out-of-scope block")
}

func run() error {
	if len(os.Args) != 1 {
		return errors.New("the lab accepts no target arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
			_, _ = fmt.Fprint(w, `<html><body><a id="next" href="/app/next">Next</a><script src="/app/main.js"></script><img src="http://outside.test:8080/x"></body></html>`)
		case "/app/main.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = fmt.Fprint(w, `fetch('/app/api').then(r => r.text()).then(x => { if (x === 'synthetic') console.log('wf-synthetic-api-ok') }); fetch('/app/redirect').then(r => { if (r.status === 502) console.log('wf-synthetic-redirect-blocked') })`)
		case "/app/api":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "synthetic")
		case "/app/next":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body><script src="/app/next.js"></script></body></html>`)
		case "/app/next.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = fmt.Fprint(w, `console.log('wf-synthetic-next-loaded'); fetch('/app/dynamic').then(r => r.text()).then(x => { if (x === 'synthetic-dynamic') console.log('wf-synthetic-dynamic-ok') })`)
		case "/app/dynamic":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "synthetic-dynamic")
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
		MaxRequests: 20, MaxConcurrent: 1, MaxRuntime: 30 * time.Second,
	})
	if err != nil {
		return err
	}
	defer gate.Close()
	broker, err := transport.NewAuthorizedLabWithPolicy(ctx, permit,
		[]transport.Grant{{Origin: origin, Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}},
		transport.Limits{MaxRequests: 20, MaxConcurrent: 1, MaxRedirects: 2,
			MaxBodyBytes: 1 << 20, RequestTimeout: 5 * time.Second, RunTimeout: 30 * time.Second,
			MinRequestInterval: time.Millisecond}, loopbackResolver{}, policy)
	if err != nil {
		return err
	}
	defer broker.Close()
	proxy, err := browser.NewObservedProxy(ctx, gate, broker, 16)
	if err != nil {
		return err
	}
	defer proxy.Close()
	username, password := proxy.Credentials()
	payload, err := json.Marshal(helperConfig{Origin: origin, ProxyEndpoint: proxy.Endpoint(),
		Username: username, Password: password})
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	resultJSON, err := browser.RunHelper(ctx, executable, payload, browser.HelperLimits{
		MaxRuntime: 25 * time.Second, MaxOutputBytes: 64 << 10,
	})
	if err != nil {
		return err
	}
	var result helperResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return errors.New("invalid helper result")
	}
	observed, omitted := proxy.Observations()
	apiObserved, dynamicObserved := false, false
	for _, item := range observed {
		if item.Path == "/app/api" && item.Method == http.MethodGet && item.StatusCode == http.StatusOK {
			apiObserved = true
		}
		if item.Path == "/app/dynamic" && item.Method == http.MethodGet && item.StatusCode == http.StatusOK {
			dynamicObserved = true
		}
	}
	if !result.Loaded || !result.APISeen || !result.Navigated || !result.DynamicSeen ||
		!result.RedirectBlocked || !result.OutsideBlocked || wrongHost.Load() || result.Denied < 1 ||
		!apiObserved || !dynamicObserved || omitted != 0 || len(observed) != 6 ||
		targetHits.Load() != 7 || broker.RequestsUsed() != 7 || gate.RequestsUsed() < broker.RequestsUsed() {
		return fmt.Errorf("unexpected synthetic observations: loaded=%t api=%t navigation=%t dynamic=%t redirect=%t outside=%t host=%t denied=%d target=%d gate=%d broker=%d observed=%d omitted=%d",
			result.Loaded, result.APISeen, result.Navigated, result.DynamicSeen,
			result.RedirectBlocked, result.OutsideBlocked, wrongHost.Load(),
			result.Denied, targetHits.Load(), gate.RequestsUsed(), broker.RequestsUsed(), len(observed), omitted)
	}
	return nil
}

func runChild() error {
	if err := browser.ApplyHelperResourceLimits(); err != nil {
		return err
	}
	var config helperConfig
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return errors.New("invalid helper configuration")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("invalid helper configuration")
	}
	if err := validateHelperConfig(config); err != nil {
		return err
	}
	if err := os.Setenv("QTWEBENGINE_CHROMIUM_FLAGS",
		"--disable-background-networking --disable-component-update --disable-sync --disable-extensions --disable-default-apps --renderer-process-limit=4"); err != nil {
		return err
	}

	runtime.LockOSThread()
	qt.NewQApplication([]string{os.Args[0]})
	endpoint, err := url.Parse(config.ProxyEndpoint)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(endpoint.Port())
	if err != nil || port <= 0 || port > 65535 {
		return errors.New("invalid helper proxy port")
	}
	qtProxy := network.NewQNetworkProxy7(network.QNetworkProxy__HttpProxy,
		"127.0.0.1", uint16(port), config.Username, config.Password)
	defer qtProxy.Delete()
	network.QNetworkProxy_SetApplicationProxy(qtProxy)
	profile := webengine.NewQWebEngineProfile()
	defer profile.Delete()
	if !profile.IsOffTheRecord() {
		return errors.New("browser profile is persistent")
	}
	originPolicy, _ := scope.New([]string{config.Origin})
	pathPolicy, _ := scope.NewRequestPolicy([]string{"GET"}, []string{"/app"}, nil)
	var denied atomic.Int32
	var outsideBlocked atomic.Bool
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
		u, err := originPolicy.Check(info.RequestUrl().ToString())
		if err == nil {
			err = pathPolicy.Check(string(info.RequestMethod()), u)
		}
		if err != nil {
			info.Block(true)
			denied.Add(1)
			if info.RequestUrl().ToString() == "http://outside.test:8080/x" {
				outsideBlocked.Store(true)
			}
		}
	})
	profile.SetUrlRequestInterceptor(interceptor)
	page := webengine.NewQWebEnginePage2(profile)
	defer page.Delete()
	var loaded atomic.Bool
	var apiSeen atomic.Bool
	var navigated atomic.Bool
	var dynamicSeen atomic.Bool
	var clickStarted atomic.Bool
	var redirectBlocked atomic.Bool
	page.OnLoadFinished(func(ok bool) { loaded.Store(ok) })
	page.OnJavaScriptConsoleMessage(func(super func(webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, string, int, string),
		level webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, message string, line int, source string) {
		if message == "wf-synthetic-api-ok" {
			apiSeen.Store(true)
		} else if message == "wf-synthetic-next-loaded" {
			navigated.Store(true)
		} else if message == "wf-synthetic-dynamic-ok" {
			dynamicSeen.Store(true)
		} else if message == "wf-synthetic-redirect-blocked" {
			redirectBlocked.Store(true)
		}
	})
	timer := qt.NewQTimer()
	defer timer.Delete()
	started := time.Now()
	timer.OnTimeout(func() {
		if loaded.Load() && apiSeen.Load() && redirectBlocked.Load() && outsideBlocked.Load() &&
			clickStarted.CompareAndSwap(false, true) {
			page.RunJavaScriptWithScriptSource("document.getElementById('next').click()")
		}
		if (navigated.Load() && dynamicSeen.Load()) ||
			time.Since(started) >= 20*time.Second {
			qt.QCoreApplication_Quit()
		}
	})
	timer.Start(100)
	page.Load(qt.NewQUrl3(config.Origin + "/app"))
	qt.QApplication_Exec()
	return json.NewEncoder(os.Stdout).Encode(helperResult{Loaded: loaded.Load(), APISeen: apiSeen.Load(),
		Navigated: navigated.Load(), DynamicSeen: dynamicSeen.Load(),
		RedirectBlocked: redirectBlocked.Load(), OutsideBlocked: outsideBlocked.Load(), Denied: int(denied.Load())})
}

func validateHelperConfig(c helperConfig) error {
	u, err := url.Parse(c.Origin)
	if err != nil || u.Scheme != "http" || u.Hostname() != "site.test" || u.Port() == "" ||
		u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("invalid helper origin")
	}
	if _, err := scope.New([]string{c.Origin}); err != nil {
		return errors.New("invalid helper origin")
	}
	p, err := url.Parse(c.ProxyEndpoint)
	if err != nil || p.Scheme != "http" || p.Hostname() != "127.0.0.1" || p.Port() == "" ||
		p.Path != "" || p.RawQuery != "" || p.ForceQuery || p.Fragment != "" || p.User != nil ||
		c.Username != "webfence" {
		return errors.New("invalid helper proxy")
	}
	port, err := strconv.Atoi(p.Port())
	if err != nil || port <= 0 || port > 65535 {
		return errors.New("invalid helper proxy")
	}
	secret, err := base64.RawURLEncoding.DecodeString(c.Password)
	if err != nil || len(secret) != 32 {
		return errors.New("invalid helper proxy")
	}
	return nil
}
