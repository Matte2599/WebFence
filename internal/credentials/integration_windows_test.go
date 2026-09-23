//go:build keychainintegration

package credentials

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func integrationFactory(t *testing.T, service string) (func() *Store, func()) {
	t.Helper()
	return func() *Store { return &Store{nativeBackend(service)} }, nil
}

// Run impersonation in a dedicated test subprocess, on one pinned OS thread.
// No personal session is locked; all operations target our random synthetic item.
func TestWindowsUnavailableCredentialSession(t *testing.T) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	service := namespace + ".test." + hex.EncodeToString(token)
	s := &Store{nativeBackend(service)}
	ctx := context.Background()
	expected := []byte("synthetic sentinel")
	if err := s.Set(ctx, "synthetic", expected); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Delete(ctx, "synthetic"); err != nil {
			t.Error("cleanup:", err)
		}
	})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	child := exec.CommandContext(childCtx, executable, "-test.run=^TestAnonymousCredentialSessionHelper$", "-test.v")
	child.Env = append(os.Environ(), "WEBFENCE_TEST_ANONYMOUS_SERVICE="+service)
	output, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("anonymous session subprocess: %v\n%s", err, output)
	}
	got, err := s.Get(ctx, "synthetic")
	if err != nil || !bytes.Equal(got, expected) {
		t.Fatal("original credential changed during unavailable-session trial", err)
	}
	clear(got)
	t.Log("anonymous native read/write/delete denied; original credential preserved")
}

func TestAnonymousCredentialSessionHelper(t *testing.T) {
	service := os.Getenv("WEBFENCE_TEST_ANONYMOUS_SERVICE")
	if service == "" {
		t.Skip("only invoked by the isolated parent test")
	}
	suffix := strings.TrimPrefix(service, namespace+".test.")
	if !strings.HasPrefix(service, namespace+".test.") || len(suffix) != 32 || strings.Trim(suffix, "0123456789abcdef") != "" {
		t.Fatal("invalid test namespace")
	}
	impersonate := windows.NewLazySystemDLL("advapi32.dll").NewProc("ImpersonateAnonymousToken")
	if err := impersonate.Find(); err != nil {
		t.Fatal(err)
	}
	runtime.LockOSThread()
	// Resolve and verify the revert operation before changing this thread's token.
	if err := windows.RevertToSelf(); err != nil {
		t.Fatal(err)
	}
	thread, err := windows.OpenThread(windows.THREAD_IMPERSONATE, false, windows.GetCurrentThreadId())
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(thread)
	result, _, callErr := impersonate.Call(uintptr(thread))
	if result == 0 {
		t.Fatal("anonymous impersonation unavailable:", callErr)
	}
	defer func() {
		if err := windows.RevertToSelf(); err != nil {
			t.Error("cannot restore subprocess thread identity")
			return
		}
		runtime.UnlockOSThread()
	}()
	s := &Store{nativeBackend(service)}
	ctx := context.Background()
	got, readErr := s.Get(ctx, "synthetic")
	writeErr := s.Set(ctx, "synthetic", []byte("synthetic replacement"))
	deleteErr := s.Delete(ctx, "synthetic")
	if got != nil {
		clear(got)
		t.Error("anonymous read returned a value")
	}
	for op, err := range map[string]error{"read": readErr, "write": writeErr, "delete": deleteErr} {
		if err != ErrUnavailable && err != ErrLocked {
			t.Errorf("%s must report unavailable/denied, got %v", op, err)
		}
	}
}
