//go:build keychainintegration

package credentials

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"
)

func TestNativeRoundTrip(t *testing.T) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	service := namespace + ".test." + hex.EncodeToString(token)
	factory, lock := integrationFactory(t, service)
	s := factory()
	ctx := context.Background()
	id := "synthetic"
	t.Cleanup(func() {
		err := s.Delete(ctx, id)
		if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrLocked) {
			t.Error("cleanup:", err)
		}
	})
	if _, err := s.Get(ctx, id); err != ErrNotFound {
		t.Fatal("initial get:", err)
	}
	if err := s.Delete(ctx, id); err != ErrNotFound {
		t.Fatal("initial delete:", err)
	}
	first := []byte{0, 255, 1, 128, 0, 42}
	if err := s.Set(ctx, id, first); err != nil {
		t.Fatal("create:", err)
	}
	got, err := factory().Get(ctx, id)
	if err != nil || !bytes.Equal(got, first) {
		t.Fatal("reopen/get:", err)
	}
	clear(got)
	if _, err := s.Get(ctx, "other"); err != ErrNotFound {
		t.Fatal("ID isolation:", err)
	}
	updated := bytes.Repeat([]byte{0, 0xFE}, MaxSecretBytes/2)
	if err := s.Set(ctx, id, updated); err != nil {
		t.Fatal("update:", err)
	}
	got, err = s.Get(ctx, id)
	if err != nil || !bytes.Equal(got, updated) {
		t.Fatal("updated get:", err)
	}
	clear(got)
	if err := s.Delete(ctx, id); err != nil {
		t.Fatal("delete:", err)
	}
	if _, err := s.Get(ctx, id); err != ErrNotFound {
		t.Fatal("deleted read:", err)
	}
	if err := s.Set(ctx, id, first); err != nil {
		t.Fatal(err)
	}
	if lock != nil {
		lock()
		if _, err := s.Get(ctx, id); err != ErrLocked {
			t.Fatal("locked get:", err)
		}
		if err := s.Set(ctx, id, first); err != ErrLocked {
			t.Fatal("locked set:", err)
		}
		if err := s.Delete(ctx, id); err != ErrLocked {
			t.Fatal("locked delete:", err)
		}
	}
}
