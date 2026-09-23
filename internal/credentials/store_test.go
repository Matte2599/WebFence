package credentials

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeBackend struct {
	calls int
	value []byte
	err   error
}

func (b *fakeBackend) get(context.Context, string) ([]byte, error) { b.calls++; return b.value, b.err }
func (b *fakeBackend) set(_ context.Context, _ string, value []byte) error {
	b.calls++
	b.value = value
	return b.err
}
func (b *fakeBackend) delete(context.Context, string) error { b.calls++; return b.err }

func TestInvalidInputsNeverReachBackend(t *testing.T) {
	b := &fakeBackend{}
	s := &Store{b}
	ctx := context.Background()
	for _, id := range []string{"", "A", "a/b", "https://host", "caffè", "a\x00b", strings.Repeat("a", 65)} {
		if _, err := s.Get(ctx, id); err != ErrInvalid {
			t.Fatal("get accepted invalid ID")
		}
		if err := s.Set(ctx, id, []byte("synthetic")); err != ErrInvalid {
			t.Fatal("set accepted invalid ID")
		}
		if err := s.Delete(ctx, id); err != ErrInvalid {
			t.Fatal("delete accepted invalid ID")
		}
	}
	for _, value := range [][]byte{nil, {}, make([]byte, MaxSecretBytes+1)} {
		if err := s.Set(ctx, "valid", value); err != ErrInvalid {
			t.Fatal("invalid size accepted")
		}
	}
	if _, err := s.Get(nil, "valid"); err != ErrInvalid {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Get(ctx, "valid"); err != context.Canceled {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "valid", []byte{1}); err != context.Canceled {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "valid"); err != context.Canceled {
		t.Fatal(err)
	}
	if b.calls != 0 {
		t.Fatal("invalid/canceled request reached backend")
	}
}
func TestZeroStoreFailsClosed(t *testing.T) {
	for _, s := range []*Store{nil, {}} {
		if _, err := s.Get(context.Background(), "valid"); err != ErrUnavailable {
			t.Fatal(err)
		}
		if err := s.Set(context.Background(), "valid", []byte{1}); err != ErrUnavailable {
			t.Fatal(err)
		}
		if err := s.Delete(context.Background(), "valid"); err != ErrUnavailable {
			t.Fatal(err)
		}
	}
}
func TestErrorsDoNotExposeNativeDataOrReturnSecrets(t *testing.T) {
	for _, want := range []error{ErrLocked, ErrNotFound, ErrConflict, context.Canceled, context.DeadlineExceeded, ErrUnavailable} {
		native := fmt.Errorf("synthetic sensitive diagnostic: %w", want)
		b := &fakeBackend{err: native, value: []byte("synthetic partial secret")}
		s := &Store{b}
		value, err := s.Get(context.Background(), "valid")
		if err != want || value != nil || !bytes.Equal(b.value, make([]byte, len(b.value))) {
			t.Fatal("unsafe failed read")
		}
		if err := s.Set(context.Background(), "valid", []byte{1}); err != want {
			t.Fatal(err)
		}
		if err := s.Delete(context.Background(), "valid"); err != want {
			t.Fatal(err)
		}
		if b.calls != 3 {
			t.Fatal("unexpected retries/fallback")
		}
	}
	if sanitize(errors.New("secret native message")) != ErrUnavailable {
		t.Fatal("unredacted native error")
	}
}
func TestSizeAndBufferOwnership(t *testing.T) {
	for _, size := range []int{1, MaxSecretBytes} {
		input := bytes.Repeat([]byte{0x81}, size)
		original := bytes.Clone(input)
		b := &fakeBackend{}
		s := &Store{b}
		if err := s.Set(context.Background(), strings.Repeat("a", 64), input); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(input, original) || !bytes.Equal(b.value, make([]byte, size)) {
			t.Fatal("input modified or owned buffer not cleared")
		}
		b.value = bytes.Clone(original)
		got, err := s.Get(context.Background(), "a-z_09")
		if err != nil || !bytes.Equal(got, original) {
			t.Fatal("binary round trip failed")
		}
	}
	for _, size := range []int{0, MaxSecretBytes + 1} {
		b := &fakeBackend{value: bytes.Repeat([]byte{1}, size)}
		got, err := (&Store{b}).Get(context.Background(), "valid")
		if got != nil || err != ErrUnavailable || !bytes.Equal(b.value, make([]byte, size)) {
			t.Fatal("invalid native value exposed")
		}
	}
}
