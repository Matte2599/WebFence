package credentials

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestOnlyLocalBusAddresses(t *testing.T) {
	valid := map[string]string{"unix:path=/tmp/session": "/tmp/session", "unix:abstract=bus-test": "\x00bus-test", "unix:path=/tmp/a%20b,guid=0123456789abcdef0123456789abcdef": "/tmp/a b"}
	for input, want := range valid {
		got, err := unixBusAddress(input)
		if err != nil || got != want {
			t.Fatal("valid local address rejected", err)
		}
	}
	for _, input := range []string{"", "tcp:host=127.0.0.1,port=1", "autolaunch:", "unix:path=relative", "unix:path=/tmp/a;unix:path=/tmp/b", "unix:path=/tmp/a,path=/tmp/b", "unix:path=/tmp/a,abstract=b", "unix:path=/tmp/a%00b", "unix:path=/tmp/a%xx", "unix:abstract=", "unix:guid=x", "unix:path=/tmp/a,unknown=x"} {
		if _, err := unixBusAddress(input); err != ErrUnavailable {
			t.Fatal("unsafe address accepted")
		}
	}
}
func TestUnavailableBusNeverFallsBack(t *testing.T) {
	for _, address := range []string{"", "tcp:host=127.0.0.1,port=1", "unix:path=" + filepath.Join(t.TempDir(), "absent")} {
		t.Setenv("DBUS_SESSION_BUS_ADDRESS", address)
		s := New()
		if _, err := s.Get(context.Background(), "synthetic"); err != ErrUnavailable {
			t.Fatal(err)
		}
		if err := s.Set(context.Background(), "synthetic", []byte{1}); err != ErrUnavailable {
			t.Fatal(err)
		}
		if err := s.Delete(context.Background(), "synthetic"); err != ErrUnavailable {
			t.Fatal(err)
		}
	}
}
func TestCancellationDuringBusAuthentication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bus")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+path)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := New().Get(ctx, "synthetic"); err != context.DeadlineExceeded {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("authentication exceeded cancellation bound")
	}
	select {
	case conn := <-accepted:
		conn.Close()
	case <-time.After(time.Second):
		t.Fatal("socket was not accepted")
	}
}
