//go:build linux && (amd64 || arm64)

package browser

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestLinuxNetworkIsolationAndInheritance(t *testing.T) {
	switch os.Getenv("WF_NETWORK_FILTER_TEST_STAGE") {
	case "child":
		const workers = 3
		ready := make(chan struct{}, workers)
		release := make(chan struct{})
		results := make(chan error, workers)
		for range workers {
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				ready <- struct{}{}
				<-release
				conn, err := net.Dial("tcp4", "127.0.0.1:9")
				if conn != nil {
					conn.Close()
				}
				results <- err
			}()
		}
		for range workers {
			<-ready
		}
		if err := ApplyHelperNetworkIsolation(); err != nil {
			t.Fatalf("install network filter: %v", err)
		}
		close(release)
		for range workers {
			if err := <-results; !errors.Is(err, syscall.EPERM) {
				t.Errorf("preexisting Go thread direct socket = %v, want EPERM", err)
			}
		}
		checkDeniedINET(t)
		directory, err := os.MkdirTemp("", "wf-net-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(directory)
		path := filepath.Join(directory, "broker.sock")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatalf("Unix listener blocked: %v", err)
		}
		defer listener.Close()
		conn, err := net.Dial("unix", path)
		if err != nil {
			t.Fatalf("Unix IPC blocked: %v", err)
		}
		conn.Close()
		grandchild := exec.Command(os.Args[0], "-test.run=^TestLinuxNetworkIsolationAndInheritance$")
		grandchild.Env = append(os.Environ(), "WF_NETWORK_FILTER_TEST_STAGE=grandchild")
		if output, err := grandchild.CombinedOutput(); err != nil {
			t.Fatalf("descendant escaped network filter: %v: %s", err, output)
		}
	case "grandchild":
		checkDeniedINET(t)
	default:
		child := exec.Command(os.Args[0], "-test.run=^TestLinuxNetworkIsolationAndInheritance$")
		child.Env = append(os.Environ(), "WF_NETWORK_FILTER_TEST_STAGE=child")
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("network filter trial failed: %v: %s", err, output)
		}
	}
}

func checkDeniedINET(t *testing.T) {
	t.Helper()
	for _, item := range []struct{ network, address string }{
		{"tcp4", "127.0.0.1:9"},
		{"tcp6", "[::1]:9"},
	} {
		conn, err := net.Dial(item.network, item.address)
		if conn != nil {
			conn.Close()
		}
		if !errors.Is(err, syscall.EPERM) {
			t.Errorf("%s direct socket = %v, want EPERM", item.network, err)
		}
	}
}
