//go:build darwin && cgo

package credentials

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestDarwinUnavailableKeychain(t *testing.T) {
	s := &Store{darwinBackend{service: namespace + ".test", path: filepath.Join(t.TempDir(), "absent.keychain-db")}}
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
func TestDarwinCanceledWhileWaiting(t *testing.T) {
	darwinGate <- struct{}{}
	defer func() { <-darwinGate }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := New().Get(ctx, "synthetic"); err != context.DeadlineExceeded {
		t.Fatal(err)
	}
}
