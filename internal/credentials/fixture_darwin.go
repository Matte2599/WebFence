//go:build darwin && cgo && keychainintegration

package credentials

/*
#cgo CFLAGS: -Wno-deprecated-declarations
#include <Security/Security.h>
#include <stdlib.h>
static OSStatus wfFixtureCreate(const char *path, const char *password) {
 SecKeychainRef keychain = NULL;
 OSStatus status = SecKeychainCreate(path, 20, password, false, NULL, &keychain);
 if (keychain) CFRelease(keychain);
 return status;
}
static OSStatus wfFixtureEnd(const char *path, int remove) {
 SecKeychainRef keychain = NULL;
 OSStatus status = SecKeychainOpen(path, &keychain);
 if (status == errSecSuccess) status = remove ? SecKeychainDelete(keychain) : SecKeychainLock(keychain);
 if (keychain) CFRelease(keychain);
 return status;
}
*/
import "C"
import (
	"path/filepath"
	"testing"
	"unsafe"
)

func integrationFactory(t *testing.T, service string) (func() *Store, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "synthetic.keychain-db")
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	password := C.CString("synthetic-test-only!!")
	defer C.free(unsafe.Pointer(password))
	if status := C.wfFixtureCreate(cpath, password); status != C.errSecSuccess {
		t.Fatal("create private test keychain:", int(status))
	}
	t.Cleanup(func() {
		p := C.CString(path)
		defer C.free(unsafe.Pointer(p))
		if status := C.wfFixtureEnd(p, 1); status != C.errSecSuccess {
			t.Error("delete private test keychain:", int(status))
		}
	})
	return func() *Store { return &Store{darwinBackend{service: service, path: path}} }, func() {
		p := C.CString(path)
		defer C.free(unsafe.Pointer(p))
		if status := C.wfFixtureEnd(p, 0); status != C.errSecSuccess {
			t.Fatal("lock private test keychain:", int(status))
		}
	}
}
