// Package credentials provides a bounded, fail-closed OS credential store.
// It is not yet connected to the GUI or to report signing.
package credentials

import (
	"context"
	"errors"
)

const MaxSecretBytes = 2048
const namespace = "io.github.Matte2599.WebFence"

var (
	ErrInvalid     = errors.New("credentials: invalid input")
	ErrNotFound    = errors.New("credentials: not found")
	ErrLocked      = errors.New("credentials: locked or interaction required")
	ErrUnavailable = errors.New("credentials: unavailable")
	ErrConflict    = errors.New("credentials: ambiguous item")
)

type backend interface {
	get(context.Context, string) ([]byte, error)
	set(context.Context, string, []byte) error
	delete(context.Context, string) error
}

// Store never falls back to plaintext files or an in-memory substitute.
// Its zero value is unavailable. IDs are opaque lowercase ASCII identifiers.
// Callers own returned bytes and should clear them after use; Go/OS copies may remain.
// Calls are synchronous. Only Linux supports cancellation during native I/O.
type Store struct{ backend backend }

func New() *Store { return &Store{backend: nativeBackend(namespace)} }

func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range []byte(id) {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func (s *Store) check(ctx context.Context, id string) error {
	if ctx == nil || !validID(id) {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	return nil
}

// sanitize deliberately drops native diagnostics, which can contain sensitive data.
func sanitize(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case errors.Is(err, ErrLocked):
		return ErrLocked
	case errors.Is(err, ErrConflict):
		return ErrConflict
	default:
		return ErrUnavailable
	}
}

func (s *Store) Get(ctx context.Context, id string) ([]byte, error) {
	if err := s.check(ctx, id); err != nil {
		return nil, err
	}
	value, err := s.backend.get(ctx, id)
	if err != nil {
		clear(value)
		return nil, sanitize(err)
	}
	if len(value) < 1 || len(value) > MaxSecretBytes {
		clear(value)
		return nil, ErrUnavailable
	}
	return value, nil
}

// Set creates or replaces exactly one ID within the application's namespace.
func (s *Store) Set(ctx context.Context, id string, value []byte) error {
	if len(value) < 1 || len(value) > MaxSecretBytes {
		return ErrInvalid
	}
	if err := s.check(ctx, id); err != nil {
		return err
	}
	owned := append([]byte(nil), value...)
	defer clear(owned)
	return sanitize(s.backend.set(ctx, id, owned))
}

// Delete reports ErrNotFound when the exact item does not exist.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := s.check(ctx, id); err != nil {
		return err
	}
	return sanitize(s.backend.delete(ctx, id))
}
