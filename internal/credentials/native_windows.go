package credentials

import (
	"context"
	"errors"
	"github.com/danieljoos/wincred"
	"syscall"
)

type windowsBackend struct{ service string }

func nativeBackend(service string) backend { return windowsBackend{service} }
func windowsError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, wincred.ErrElementNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return ErrLocked
	}
	return ErrUnavailable
}
func (b windowsBackend) get(_ context.Context, id string) ([]byte, error) {
	c, err := wincred.GetGenericCredential(b.service + "/" + id)
	if err != nil {
		return nil, windowsError(err)
	}
	return c.CredentialBlob, nil
}
func (b windowsBackend) set(_ context.Context, id string, value []byte) error {
	c := wincred.NewGenericCredential(b.service + "/" + id)
	c.Persist = wincred.PersistLocalMachine // Current user on this computer; not enterprise roaming.
	c.CredentialBlob = value
	return windowsError(c.Write())
}
func (b windowsBackend) delete(_ context.Context, id string) error {
	c := wincred.NewGenericCredential(b.service + "/" + id)
	return windowsError(c.Delete())
}
