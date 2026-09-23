package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func fetchAsync(b *LabBroker, ctx context.Context, target string) <-chan error {
	done := make(chan error, 1)
	go func() { _, err := b.Fetch(ctx, target); done <- err }()
	return done
}
func awaitError(t *testing.T, done <-chan error, want error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not stop")
	}
}
func dummyGrant() Grant {
	return Grant{Origin: "http://fixture.invalid:8080", Addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}
}

func TestCancelDuringDNSAndDial(t *testing.T) {
	for _, stage := range []string{"dns", "dial"} {
		t.Run(stage, func(t *testing.T) {
			entered, exited := make(chan struct{}), make(chan struct{})
			g := dummyGrant()
			dns := fixed(g.Addresses[0])
			if stage == "dns" {
				dns = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
					close(entered)
					defer close(exited)
					<-ctx.Done()
					return nil, ctx.Err()
				})
			}
			b, dials := newFixtureBroker(t, g, limits(), dns)
			if stage == "dial" {
				b.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
					close(entered)
					defer close(exited)
					<-ctx.Done()
					return nil, ctx.Err()
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := fetchAsync(b, ctx, g.Origin)
			await(t, entered)
			cancel()
			awaitError(t, result, context.Canceled)
			await(t, exited)
			if b.RequestsUsed() != 1 || (stage == "dns" && dials.Load() != 0) {
				t.Fatal("unexpected work after cancellation")
			}
		})
	}
}

func TestCloseCancelsBodyAndWaitingFetch(t *testing.T) {
	entered, exited := make(chan struct{}), make(chan struct{})
	var hits atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) != 1 {
			t.Error("concurrency exceeded")
			return
		}
		_, _ = io.WriteString(w, "partial")
		w.(http.Flusher).Flush()
		close(entered)
		<-r.Context().Done()
		close(exited)
	}))
	defer s.Close()
	g := fixtureGrant(t, s, "")
	l := limits()
	l.MaxConcurrent = 1
	b, _ := newFixtureBroker(t, g, l, fixed(g.Addresses[0]))
	first := fetchAsync(b, context.Background(), g.Origin)
	await(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	queued := fetchAsync(b, ctx, g.Origin)
	cancel()
	awaitError(t, queued, context.Canceled)
	if b.RequestsUsed() != 1 {
		t.Fatal("cancelled waiter consumed an attempt")
	}
	queued = fetchAsync(b, context.Background(), g.Origin)
	b.Close()
	b.Close() // idempotent and permanent
	awaitError(t, first, context.Canceled)
	awaitError(t, queued, context.Canceled)
	await(t, exited)
	if _, err := b.Fetch(context.Background(), g.Origin); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if b.RequestsUsed() != 1 || hits.Load() != 1 {
		t.Fatal("request sent after stop")
	}
}

func TestPreCancelledAndExpiredRun(t *testing.T) {
	g := dummyGrant()
	var lookups atomic.Int32
	dns := resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { lookups.Add(1); return g.Addresses, nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b, err := NewLab(ctx, []Grant{g}, limits(), dns)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := b.Fetch(context.Background(), g.Origin); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if b.RequestsUsed() != 0 || lookups.Load() != 0 {
		t.Fatal("cancelled run performed work")
	}
	l := limits()
	l.RunTimeout = time.Millisecond
	b, err = NewLab(context.Background(), []Grant{g}, l, dns)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	await(t, b.ctx.Done()) // wait for the real deadline, no arbitrary sleep
	if _, err := b.Fetch(context.Background(), g.Origin); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if b.RequestsUsed() != 0 || lookups.Load() != 0 {
		t.Fatal("expired run restarted")
	}
}

func TestRequestDeadlineIncludesDNS(t *testing.T) {
	g := dummyGrant()
	l := limits()
	l.RequestTimeout = 20 * time.Millisecond
	exited := make(chan struct{})
	dns := resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
		defer close(exited)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	b, dials := newFixtureBroker(t, g, l, dns)
	_, err := b.Fetch(context.Background(), g.Origin)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	await(t, exited)
	if dials.Load() != 0 {
		t.Fatal("dial after DNS deadline")
	}
}
