package browser

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Matte2599/WebFence/internal/transport"
)

// Proxy is an authenticated, loopback-only HTTP forwarding boundary for a
// future browser adapter. HTTPS CONNECT, WebSocket upgrades, cookies and
// request bodies are deliberately unsupported in this first slice.
type Proxy struct {
	gate   *Gate
	broker *transport.Broker
	server *http.Server
	listen net.Listener
	ctx    context.Context
	cancel context.CancelFunc
	secret string
	done   chan error
	once   sync.Once
}

// NewProxy starts an ephemeral loopback listener. Only requests carrying the
// per-instance Basic proxy secret can reach the gate or broker. The browser
// process must still be configured and independently confined to this proxy.
func NewProxy(ctx context.Context, gate *Gate, broker *transport.Broker) (*Proxy, error) {
	if ctx == nil || gate == nil || broker == nil {
		return nil, ErrConfig
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return nil, ErrConfig
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, ErrConfig
	}
	run, cancel := context.WithCancel(ctx)
	p := &Proxy{gate: gate, broker: broker, listen: listener, ctx: run, cancel: cancel,
		secret: base64.RawURLEncoding.EncodeToString(secret[:]), done: make(chan error, 1)}
	p.server = &http.Server{
		Handler: p, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 32 << 10,
		BaseContext: func(net.Listener) context.Context { return run },
	}
	go func() {
		err := p.server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		p.done <- err
		close(p.done)
		p.cancel()
	}()
	go func() {
		<-run.Done()
		p.Close()
	}()
	return p, nil
}

// Endpoint and Credentials are for the browser process configuration only.
// Do not log or persist the returned password.
func (p *Proxy) Endpoint() string {
	return "http://" + p.listen.Addr().String()
}

func (p *Proxy) Credentials() (username, password string) {
	return "webfence", p.secret
}

// Done reports listener failure or normal shutdown once.
func (p *Proxy) Done() <-chan error { return p.done }

func (p *Proxy) Close() error {
	if p == nil {
		return nil
	}
	var result error
	p.once.Do(func() {
		p.cancel()
		result = p.server.Close()
	})
	return result
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, password := p.Credentials()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	got := r.Header.Get("Proxy-Authorization")
	if len(got) != len(want) || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		w.Header().Set("Proxy-Authenticate", `Basic realm="WebFence"`)
		writeProxyError(w, http.StatusProxyAuthRequired, "browser_proxy_auth_required")
		return
	}
	if r.Method == http.MethodConnect || r.URL == nil || !r.URL.IsAbs() || r.URL.Scheme != "http" ||
		r.URL.Host != r.Host || r.Header.Get("Upgrade") != "" || r.ContentLength > 0 || len(r.TransferEncoding) != 0 {
		writeProxyError(w, http.StatusForbidden, "browser_proxy_request_denied")
		return
	}
	kind := Subresource
	switch strings.ToLower(r.Header.Get("Sec-Fetch-Dest")) {
	case "document", "iframe":
		kind = Document
	case "worker", "sharedworker", "serviceworker":
		kind = Worker
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(p.ctx, cancel)
	defer stop()
	release, err := p.gate.Admit(ctx, r.Method, r.URL.String(), kind)
	if err != nil {
		writeProxyError(w, http.StatusForbidden, "browser_proxy_policy_denied")
		return
	}
	defer release()
	result, err := p.broker.FetchMethod(ctx, r.Method, r.URL.String())
	if err != nil {
		writeProxyError(w, http.StatusBadGateway, "browser_proxy_fetch_failed")
		return
	}
	// Never transfer credential material from the target into a browser profile.
	// These response headers retain basic rendering and target-side restrictions.
	for _, name := range []string{
		"Content-Type", "Content-Security-Policy", "X-Content-Type-Options",
		"X-Frame-Options", "Referrer-Policy", "Cross-Origin-Opener-Policy",
		"Cross-Origin-Embedder-Policy", "Cross-Origin-Resource-Policy",
		"Access-Control-Allow-Origin", "Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers", "Access-Control-Expose-Headers", "Vary",
	} {
		for _, value := range result.Header.Values(name) {
			w.Header().Add(name, value)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(result.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = w.Write(result.Body)
	}
}

func writeProxyError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(code + "\n"))
}
