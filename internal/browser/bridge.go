package browser

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/Matte2599/WebFence/internal/transport"
)

// RequestBridge serves intercepted anonymous GET/HEAD requests in the trusted
// controller. It opens no listener and accepts no browser headers or bodies.
// The caller must independently confine the browser and bound its IPC framing.
type RequestBridge struct {
	ctx          context.Context
	cancel       context.CancelFunc
	gate         *Gate
	broker       *transport.Broker
	mu           sync.Mutex
	observations []Observation
	dropped      int
}

const MaxBridgeBody = 16 << 10

// BridgeReply contains only rendering headers and a bounded response body.
type BridgeReply struct {
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

func NewRequestBridge(ctx context.Context, gate *Gate, broker *transport.Broker) (*RequestBridge, error) {
	if ctx == nil || gate == nil || broker == nil {
		return nil, ErrConfig
	}
	run, cancel := context.WithCancel(ctx)
	return &RequestBridge{ctx: run, cancel: cancel, gate: gate, broker: broker}, nil
}

func (b *RequestBridge) Close() {
	if b != nil {
		b.cancel()
	}
}

func bridgeError(status int, code string) BridgeReply {
	return BridgeReply{Status: status, Header: http.Header{"Content-Type": {"text/plain; charset=utf-8"}, "Cache-Control": {"no-store"}}, Body: []byte(code + "\n")}
}

// Forward treats the intercepted URL/method/type as untrusted input. All hops
// use the broker's pinned grants, TLS verification, policy and run lifecycle.
func (b *RequestBridge) Forward(ctx context.Context, method, raw string, kind RequestType) BridgeReply {
	if b == nil || ctx == nil {
		return bridgeError(403, "browser_bridge_denied")
	}
	run, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(b.ctx, cancel)
	defer stop()
	stopGate := context.AfterFunc(b.gate.ctx, cancel)
	defer stopGate()
	if b.ctx.Err() != nil {
		return bridgeError(403, "browser_bridge_denied")
	}
	release, err := b.gate.Admit(run, method, raw, kind)
	if err != nil {
		return bridgeError(403, "browser_bridge_denied")
	}
	defer release()
	result, err := b.broker.FetchMethod(run, method, raw)
	if err != nil || run.Err() != nil || len(result.Body) > MaxBridgeBody {
		return bridgeError(502, "browser_bridge_fetch_failed")
	}
	requested, parseErr := url.Parse(raw)
	// CDP fulfillment does not update the navigation URL. Never render a
	// followed redirect's final body under the original URL/origin.
	if parseErr != nil || requested.String() != result.FinalURL {
		return bridgeError(502, "browser_bridge_redirect_unsupported")
	}
	header := renderingHeaders(result.Header)
	headerBytes := 0
	for name, values := range header {
		for _, value := range values {
			headerBytes += len(name) + len(value)
		}
	}
	if headerBytes > 8<<10 {
		return bridgeError(502, "browser_bridge_fetch_failed")
	}
	b.mu.Lock()
	if len(b.observations) < MaxObservations {
		request, parseErr := http.NewRequest(method, raw, nil)
		var observation Observation
		var ok bool
		if parseErr == nil {
			observation, ok = redactedObservation(request, result, kind)
		}
		if ok {
			b.observations = append(b.observations, observation)
		} else {
			b.dropped++
		}
	} else {
		b.dropped++
	}
	b.mu.Unlock()
	body := result.Body
	if method == http.MethodHead || body == nil {
		body = []byte{}
	}
	return BridgeReply{Status: result.StatusCode, Header: header, Body: body}
}

func (b *RequestBridge) Observations() ([]Observation, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Observation(nil), b.observations...), b.dropped
}
