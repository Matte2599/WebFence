package browser

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/transport"
	"golang.org/x/net/netutil"
)

// Proxy is an authenticated, loopback-only HTTP forwarding boundary for a
// future browser adapter. HTTPS CONNECT, WebSocket upgrades, cookies and
// request bodies are deliberately unsupported in this first slice.
type Proxy struct {
	gate    *Gate
	broker  *transport.Broker
	session *session.Session
	server  *http.Server
	listen  net.Listener
	ctx     context.Context
	cancel  context.CancelFunc
	secret  string
	done    chan error
	once    sync.Once
	obsMu   sync.Mutex
	obsMax  int
	obs     []Observation
	obsDrop int
}

// Observation is an ephemeral record of a request actually admitted and
// fetched through the proxy. Paths omit queries, headers and response bodies.
type Observation struct {
	Method     string
	Path       string
	FinalPath  string
	Kind       RequestType
	StatusCode int
}

const MaxObservations = 256

// NewProxy starts an ephemeral loopback listener. Only requests carrying the
// per-instance Basic proxy secret can reach the gate or broker. The browser
// process must still be configured and independently confined to this proxy.
func NewProxy(ctx context.Context, gate *Gate, broker *transport.Broker) (*Proxy, error) {
	return newProxy(ctx, gate, broker, nil, 0)
}

// NewObservedProxy retains at most limit in-memory request observations.
// It never schedules requests from observed paths.
func NewObservedProxy(ctx context.Context, gate *Gate, broker *transport.Broker, limit int) (*Proxy, error) {
	if limit <= 0 || limit > MaxObservations {
		return nil, ErrConfig
	}
	return newProxy(ctx, gate, broker, nil, limit)
}

// NewProxyWithSession binds exactly one verified test identity to this proxy
// instance. Browser-supplied cookies remain ignored; the in-memory Session
// attaches its own cookie only to exact-origin allowlisted GET requests.
func NewProxyWithSession(ctx context.Context, gate *Gate, broker *transport.Broker, identity *session.Session) (*Proxy, error) {
	return newProxyWithSession(ctx, gate, broker, identity, 0)
}

// NewObservedProxyWithSession adds bounded in-memory observations to one
// verified identity's proxy. It does not expose the target cookie to Qt.
func NewObservedProxyWithSession(ctx context.Context, gate *Gate, broker *transport.Broker, identity *session.Session, limit int) (*Proxy, error) {
	if limit <= 0 || limit > MaxObservations {
		return nil, ErrConfig
	}
	return newProxyWithSession(ctx, gate, broker, identity, limit)
}

func newProxyWithSession(ctx context.Context, gate *Gate, broker *transport.Broker, identity *session.Session, observationLimit int) (*Proxy, error) {
	if identity == nil || gate == nil || broker == nil || !identity.BoundTo(broker) || identity.ProjectID() == "" ||
		identity.ProjectID() != gate.ProjectID() || identity.ProjectID() != broker.SessionProjectID() ||
		identity.Revision() != gate.Revision() || identity.Revision() != broker.SessionRevision() {
		return nil, ErrConfig
	}
	if err := identity.Verify(ctx); err != nil {
		return nil, err
	}
	return newProxy(ctx, gate, broker, identity, observationLimit)
}

func newProxy(ctx context.Context, gate *Gate, broker *transport.Broker, identity *session.Session, observationLimit int) (*Proxy, error) {
	if ctx == nil || gate == nil || broker == nil {
		return nil, ErrConfig
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, ErrConfig
	}
	return newProxyOnListener(ctx, gate, broker, identity, observationLimit, listener)
}

func newProxyOnListener(ctx context.Context, gate *Gate, broker *transport.Broker,
	identity *session.Session, observationLimit int, listener net.Listener) (*Proxy, error) {
	if ctx == nil || gate == nil || broker == nil || listener == nil {
		if listener != nil {
			_ = listener.Close()
		}
		return nil, ErrConfig
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		_ = listener.Close()
		return nil, ErrConfig
	}
	listener = netutil.LimitListener(listener, 32)
	run, cancel := context.WithCancel(ctx)
	p := &Proxy{gate: gate, broker: broker, session: identity, listen: listener, ctx: run, cancel: cancel,
		obsMax: observationLimit,
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
	if p.listen.Addr().Network() != "tcp" {
		return ""
	}
	return "http://" + p.listen.Addr().String()
}

func (p *Proxy) Credentials() (username, password string) {
	return "webfence", p.secret
}

// Done reports listener failure or normal shutdown once.
func (p *Proxy) Done() <-chan error { return p.done }

// Observations returns a copy. The paths are untrusted target data and must
// not be persisted or re-fetched without a separate redaction/selection step.
func (p *Proxy) Observations() ([]Observation, int) {
	if p == nil {
		return nil, 0
	}
	p.obsMu.Lock()
	defer p.obsMu.Unlock()
	return append([]Observation(nil), p.obs...), p.obsDrop
}

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
	case "empty":
		kind = Fetch
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
	var result transport.Result
	if p.session != nil {
		if r.Method != http.MethodGet {
			writeProxyError(w, http.StatusForbidden, "browser_proxy_request_denied")
			return
		}
		result, err = p.session.Fetch(ctx, r.URL.String())
	} else {
		result, err = p.broker.FetchMethod(ctx, r.Method, r.URL.String())
	}
	if err != nil {
		writeProxyError(w, http.StatusBadGateway, "browser_proxy_fetch_failed")
		return
	}
	p.observe(r, result, kind)
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

func (p *Proxy) observe(r *http.Request, result transport.Result, kind RequestType) {
	if p.obsMax == 0 {
		return
	}
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	final, err := url.Parse(result.FinalURL)
	if err != nil || final == nil || !final.IsAbs() || len(path) > 1024 {
		p.obsMu.Lock()
		p.obsDrop++
		p.obsMu.Unlock()
		return
	}
	finalPath := final.EscapedPath()
	if finalPath == "" {
		finalPath = "/"
	}
	if len(finalPath) > 1024 {
		p.obsMu.Lock()
		p.obsDrop++
		p.obsMu.Unlock()
		return
	}
	p.obsMu.Lock()
	defer p.obsMu.Unlock()
	if len(p.obs) >= p.obsMax {
		p.obsDrop++
		return
	}
	p.obs = append(p.obs, Observation{Method: r.Method, Path: path,
		FinalPath: finalPath, Kind: kind, StatusCode: result.StatusCode})
}

func writeProxyError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(code + "\n"))
}
