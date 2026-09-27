//go:build m3browserlab && (darwin || linux)

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/transport"
	qt "github.com/mappu/miqt/qt6"
	webengine "github.com/mappu/miqt/qt6/webengine"
)

type schemeResult struct {
	Loaded     bool `json:"loaded"`
	ScriptSeen bool `json:"script_seen"`
	APISeen    bool `json:"api_seen"`
}

// Qt 6.6 added FetchApiAllowed (0x100); MIQT 0.14.0 does not name this flag.
const qtFetchAPIAllowed = webengine.QWebEngineUrlScheme__Flag(0x100)

func runScheme(ctx context.Context, executable, origin string, gate *browser.Gate,
	broker *transport.Broker, targetHits *atomic.Int32) error {
	directory, err := os.MkdirTemp("", "wf-scheme-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	proxy, socketPath, err := browser.NewObservedUnixProxy(ctx, gate, broker, directory, 4)
	if err != nil {
		return err
	}
	defer proxy.Close()
	username, password := proxy.Credentials()
	payload, err := json.Marshal(helperConfig{Mode: "scheme", Origin: origin,
		SocketPath: socketPath, Username: username, Password: password})
	if err != nil {
		return err
	}
	beforeBroker, beforeTarget := broker.RequestsUsed(), targetHits.Load()
	resultJSON, err := browser.RunHelper(ctx, executable, payload, browser.HelperLimits{
		MaxRuntime: 12 * time.Second, MaxOutputBytes: 64 << 10,
	})
	if err != nil {
		return err
	}
	var result schemeResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return errors.New("invalid scheme helper result")
	}
	observed, dropped := proxy.Observations()
	minimum := 2
	if runtime.GOOS == "darwin" {
		minimum = 3
	}
	if !result.Loaded || !result.ScriptSeen || (runtime.GOOS == "darwin" && !result.APISeen) ||
		broker.RequestsUsed()-beforeBroker < minimum || broker.RequestsUsed()-beforeBroker > 3 ||
		targetHits.Load()-beforeTarget != int32(broker.RequestsUsed()-beforeBroker) ||
		dropped != 0 || len(observed) != broker.RequestsUsed()-beforeBroker ||
		observed[0].Path != "/app/scheme" || observed[1].Path != "/app/scheme.js" ||
		(len(observed) == 3 && observed[2].Path != "/app/scheme-api") {
		return fmt.Errorf("unexpected scheme observations: loaded=%t script=%t api=%t broker=%d target=%d observed=%d dropped=%d",
			result.Loaded, result.ScriptSeen, result.APISeen, broker.RequestsUsed()-beforeBroker,
			targetHits.Load()-beforeTarget, len(observed), dropped)
	}
	return nil
}

func runSchemeChild(config helperConfig) error {
	if err := validateSchemeConfig(config); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		if err := applyLinuxSchemeNetworkIsolation(); err != nil {
			return err
		}
		for _, probe := range []struct{ network, address string }{
			{"tcp4", "127.0.0.1:9"}, {"tcp6", "[::1]:9"},
		} {
			conn, err := net.DialTimeout(probe.network, probe.address, 100*time.Millisecond)
			if conn != nil {
				conn.Close()
			}
			if !errors.Is(err, syscall.EPERM) {
				return errors.New("browser helper direct network is not blocked")
			}
		}
	}
	if err := os.Setenv("QTWEBENGINE_CHROMIUM_FLAGS",
		"--disable-background-networking --disable-component-update --disable-sync --disable-extensions --disable-default-apps --renderer-process-limit=4"); err != nil {
		return err
	}
	scheme := webengine.NewQWebEngineUrlScheme2([]byte("wfsite"))
	scheme.SetSyntax(webengine.QWebEngineUrlScheme__Host)
	flags := webengine.QWebEngineUrlScheme__CorsEnabled
	if runtime.GOOS == "darwin" {
		flags |= qtFetchAPIAllowed
	}
	scheme.SetFlags(flags)
	webengine.QWebEngineUrlScheme_RegisterScheme(scheme)
	defer scheme.Delete()

	runtime.LockOSThread()
	qt.NewQApplication([]string{os.Args[0]})
	profile := webengine.NewQWebEngineProfile()
	defer profile.Delete()
	if !profile.IsOffTheRecord() {
		return errors.New("scheme browser profile is persistent")
	}
	proxyURL := &url.URL{Scheme: "http", Host: "127.0.0.1"}
	proxyURL.User = url.UserPassword(config.Username, config.Password)
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", config.SocketPath)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	handler := webengine.NewQWebEngineUrlSchemeHandler()
	defer handler.Delete()
	handler.OnRequestStarted(func(job *webengine.QWebEngineUrlRequestJob) {
		requested, err := url.Parse(job.RequestUrl().ToString())
		if err != nil || requested.Scheme != "wfsite" || requested.Host != "site.test" ||
			requested.RawQuery != "" || requested.Fragment != "" || string(job.RequestMethod()) != http.MethodGet ||
			(requested.Path != "/app/scheme" && requested.Path != "/app/scheme.js" &&
				requested.Path != "/app/scheme-api") {
			job.Fail(webengine.QWebEngineUrlRequestJob__RequestDenied)
			return
		}
		response, err := client.Get(config.Origin + requested.EscapedPath())
		if err != nil {
			job.Fail(webengine.QWebEngineUrlRequestJob__RequestFailed)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		response.Body.Close()
		if readErr != nil || len(body) > 1<<20 || response.StatusCode != http.StatusOK {
			job.Fail(webengine.QWebEngineUrlRequestJob__RequestFailed)
			return
		}
		buffer := qt.NewQBuffer2(job.QObject)
		buffer.SetData(body)
		if !buffer.Open(qt.QIODeviceBase__ReadOnly) {
			buffer.Delete()
			job.Fail(webengine.QWebEngineUrlRequestJob__RequestFailed)
			return
		}
		job.Reply([]byte(response.Header.Get("Content-Type")), buffer.QIODevice)
	})
	profile.InstallUrlSchemeHandler([]byte("wfsite"), handler)
	page := webengine.NewQWebEnginePage2(profile)
	defer page.Delete()
	var loaded, scriptSeen, apiSeen atomic.Bool
	page.OnLoadFinished(func(ok bool) { loaded.Store(ok) })
	page.OnJavaScriptConsoleMessage(func(super func(webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, string, int, string),
		level webengine.QWebEnginePage__JavaScriptConsoleMessageLevel, message string, line int, source string) {
		if message == "wf-scheme-script-ok" {
			scriptSeen.Store(true)
		} else if message == "wf-scheme-api-ok" {
			apiSeen.Store(true)
		}
	})
	timer := qt.NewQTimer()
	defer timer.Delete()
	started := time.Now()
	timer.OnTimeout(func() {
		complete := loaded.Load() && scriptSeen.Load()
		if runtime.GOOS == "darwin" {
			complete = loaded.Load() && apiSeen.Load()
		}
		if complete || time.Since(started) > 9*time.Second {
			qt.QCoreApplication_Quit()
		}
	})
	timer.Start(100)
	page.Load(qt.NewQUrl3("wfsite://site.test/app/scheme"))
	qt.QApplication_Exec()
	return json.NewEncoder(os.Stdout).Encode(schemeResult{Loaded: loaded.Load(),
		ScriptSeen: scriptSeen.Load(), APISeen: apiSeen.Load()})
}

func validateSchemeConfig(c helperConfig) error {
	if c.Mode != "scheme" || c.ProxyEndpoint != "" || c.Username != "webfence" ||
		!filepath.IsAbs(c.SocketPath) || filepath.Base(c.SocketPath) != "broker.sock" {
		return errors.New("invalid scheme helper configuration")
	}
	u, err := url.Parse(c.Origin)
	if err != nil || u.Scheme != "http" || u.Hostname() != "site.test" || u.Port() == "" ||
		u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("invalid scheme helper origin")
	}
	if _, err := scope.New([]string{c.Origin}); err != nil {
		return errors.New("invalid scheme helper origin")
	}
	secret, err := base64.RawURLEncoding.DecodeString(c.Password)
	if err != nil || len(secret) != 32 {
		return errors.New("invalid scheme helper proxy")
	}
	info, err := os.Lstat(c.SocketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return errors.New("invalid scheme helper socket")
	}
	directory, err := os.Lstat(filepath.Dir(c.SocketPath))
	if err != nil || !directory.IsDir() || directory.Mode().Perm() != 0700 {
		return errors.New("invalid scheme helper socket directory")
	}
	return nil
}
