//go:build keychainintegration

package credentials

import (
	"context"
	"github.com/godbus/dbus/v5"
	"os"
	"testing"
	"time"
)

func integrationFactory(t *testing.T, service string) (func() *Store, func()) {
	t.Helper()
	if os.Getenv("WEBFENCE_ISOLATED_SECRET_SERVICE") != "1" {
		t.Fatal("native Linux integration requires the isolated test launcher")
	}
	return func() *Store { return &Store{nativeBackend(service)} }, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conn, err := connectBus(ctx, os.Getenv("DBUS_SESSION_BUS_ADDRESS"))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		object := conn.Object(secretService, secretRoot)
		var collection dbus.ObjectPath
		if err := object.CallWithContext(ctx, secretInterface+"Service.ReadAlias", 0, "default").Store(&collection); err != nil {
			t.Fatal(err)
		}
		var locked []dbus.ObjectPath
		var prompt dbus.ObjectPath
		if err := object.CallWithContext(ctx, secretInterface+"Service.Lock", 0, []dbus.ObjectPath{collection}).Store(&locked, &prompt); err != nil {
			t.Fatal(err)
		}
		if prompt != "/" || len(locked) != 1 || locked[0] != collection {
			t.Fatal("isolated collection did not lock")
		}
	}
}

func TestSessionWithoutSecretService(t *testing.T) {
	if os.Getenv("WEBFENCE_EXPECT_NO_SECRET_SERVICE") != "1" {
		t.Skip("requires an empty isolated session bus")
	}
	s := New()
	ctx := context.Background()
	if _, err := s.Get(ctx, "synthetic"); err != ErrUnavailable {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "synthetic", []byte{1}); err != ErrUnavailable {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "synthetic"); err != ErrUnavailable {
		t.Fatal(err)
	}
}
