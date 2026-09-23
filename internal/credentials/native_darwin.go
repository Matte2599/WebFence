//go:build darwin && cgo

package credentials

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#cgo CFLAGS: -Wno-deprecated-declarations
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

// The classic keychain supports unsigned development builds. Its UI switch is
// process-wide: callers serialize this whole operation and restore the old value.
static OSStatus wfKeychain(int op, const char *path, const char *service,
 const char *account, const void *input, UInt32 inputLen, void **output, UInt32 *outputLen) {
 Boolean oldUI;
 OSStatus status = SecKeychainGetUserInteractionAllowed(&oldUI);
 if (status != errSecSuccess) return status;
 status = SecKeychainSetUserInteractionAllowed(false);
 if (status != errSecSuccess) return status;
 SecKeychainRef keychain = NULL;
 SecKeychainItemRef item = NULL;
 status = path ? SecKeychainOpen(path, &keychain) : SecKeychainCopyDefault(&keychain);
 if (status != errSecSuccess) goto done;
 SecKeychainStatus flags;
 status = SecKeychainGetStatus(keychain, &flags);
 if (status != errSecSuccess) goto done;
 if (!(flags & kSecUnlockStateStatus)) { status = errSecInteractionNotAllowed; goto done; }
 status = SecKeychainFindGenericPassword(keychain, (UInt32)strlen(service), service,
  (UInt32)strlen(account), account, op == 0 ? outputLen : NULL,
  op == 0 ? output : NULL, &item);
 if (op == 1) {
  if (status == errSecItemNotFound)
   status = SecKeychainAddGenericPassword(keychain, (UInt32)strlen(service), service,
    (UInt32)strlen(account), account, inputLen, input, NULL);
  else if (status == errSecSuccess)
   status = SecKeychainItemModifyAttributesAndData(item, NULL, inputLen, input);
 } else if (op == 2 && status == errSecSuccess) {
  status = SecKeychainItemDelete(item);
 }
done:
 if (item) CFRelease(item);
 if (keychain) CFRelease(keychain);
 OSStatus restored = SecKeychainSetUserInteractionAllowed(oldUI);
 if (restored != errSecSuccess) return restored;
 return status;
}
*/
import "C"

import (
	"context"
	"unsafe"
)

// This serializes every classic-keychain operation made by this package. Future
// integrations must not add independent in-process Security.framework callers.
var darwinGate = make(chan struct{}, 1)

type darwinBackend struct{ service, path string }

func nativeBackend(service string) backend { return darwinBackend{service: service} }
func darwinError(status C.OSStatus) error {
	switch status {
	case C.errSecSuccess:
		return nil
	case C.errSecItemNotFound:
		return ErrNotFound
	case C.errSecInteractionNotAllowed, C.errSecAuthFailed, C.errSecUserCanceled:
		return ErrLocked
	default:
		return ErrUnavailable
	}
}
func (b darwinBackend) call(ctx context.Context, op int, id string, value []byte) ([]byte, error) {
	select {
	case darwinGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-darwinGate }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	service, account := C.CString(b.service), C.CString(id)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	var path *C.char
	if b.path != "" {
		path = C.CString(b.path)
		defer C.free(unsafe.Pointer(path))
	}
	var input unsafe.Pointer
	if len(value) > 0 {
		input = unsafe.Pointer(&value[0])
	}
	var output unsafe.Pointer
	var size C.UInt32
	status := C.wfKeychain(C.int(op), path, service, account, input, C.UInt32(len(value)), &output, &size)
	if output != nil {
		defer C.SecKeychainItemFreeContent(nil, output)
	}
	if err := darwinError(status); err != nil {
		return nil, err
	}
	if op != 0 {
		return nil, nil
	}
	if size < 1 || size > MaxSecretBytes || output == nil {
		return nil, ErrUnavailable
	}
	return C.GoBytes(output, C.int(size)), nil
}
func (b darwinBackend) get(ctx context.Context, id string) ([]byte, error) {
	return b.call(ctx, 0, id, nil)
}
func (b darwinBackend) set(ctx context.Context, id string, value []byte) error {
	_, err := b.call(ctx, 1, id, value)
	return err
}
func (b darwinBackend) delete(ctx context.Context, id string) error {
	_, err := b.call(ctx, 2, id, nil)
	return err
}
