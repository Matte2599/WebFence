//go:build keychainintegration

package credentials

import "testing"

func integrationFactory(t *testing.T, service string) (func() *Store, func()) {
	t.Helper()
	return func() *Store { return &Store{nativeBackend(service)} }, nil
}
