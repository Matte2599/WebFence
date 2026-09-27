//go:build darwin || linux

package browser

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"github.com/Matte2599/WebFence/internal/transport"
)

// NewObservedUnixProxy exposes the same gate and broker through a private
// Unix-domain socket. The caller owns the mode-0700 directory and removes the
// socket after Close. This listener does not restrict the helper's own network
// access; that requires an independent OS boundary.
func NewObservedUnixProxy(ctx context.Context, gate *Gate, broker *transport.Broker,
	directory string, limit int) (*Proxy, string, error) {
	if ctx == nil || gate == nil || broker == nil || limit <= 0 || limit > MaxObservations || !filepath.IsAbs(directory) {
		return nil, "", ErrConfig
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, "", ErrConfig
	}
	path := filepath.Join(directory, "broker.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, "", ErrConfig
	}
	proxy, err := newProxyOnListener(ctx, gate, broker, nil, limit, listener)
	if err != nil {
		_ = os.Remove(path)
		return nil, "", err
	}
	return proxy, path, nil
}
