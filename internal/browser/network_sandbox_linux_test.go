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

	"golang.org/x/sys/unix"
)

func TestLinuxNetworkIsolationAndInheritance(t *testing.T) {
	switch os.Getenv("WF_NETWORK_FILTER_TEST_STAGE") {
	case "child":
		directory := t.TempDir()
		path := filepath.Join(directory, "broker.sock")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		broker, err := net.Dial("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		defer broker.Close()
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
		checkDeniedUnix(t, path)
		if _, err := broker.Write([]byte("ok")); err != nil {
			t.Fatalf("inherited broker IPC blocked: %v", err)
		}
		accepted, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer accepted.Close()
		message := make([]byte, 2)
		if _, err := accepted.Read(message); err != nil || string(message) != "ok" {
			t.Fatalf("inherited broker IPC read = %q, %v", message, err)
		}
		paired, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
		if err != nil {
			t.Fatalf("local process IPC socketpair blocked: %v", err)
		}
		defer unix.Close(paired[0])
		defer unix.Close(paired[1])
		if _, err := unix.SendmsgN(paired[0], []byte("i"), nil, nil, 0); err != nil {
			t.Fatalf("connected process IPC sendmsg blocked: %v", err)
		}
		grandchild := exec.Command(os.Args[0], "-test.run=^TestLinuxNetworkIsolationAndInheritance$")
		grandchild.Env = append(os.Environ(), "WF_NETWORK_FILTER_TEST_STAGE=grandchild",
			"WF_NETWORK_FILTER_SOCKET="+path)
		if output, err := grandchild.CombinedOutput(); err != nil {
			t.Fatalf("descendant escaped network filter: %v: %s", err, output)
		}
	case "grandchild":
		checkDeniedINET(t)
		checkDeniedUnix(t, os.Getenv("WF_NETWORK_FILTER_SOCKET"))
	default:
		child := exec.Command(os.Args[0], "-test.run=^TestLinuxNetworkIsolationAndInheritance$")
		child.Env = append(os.Environ(), "WF_NETWORK_FILTER_TEST_STAGE=child")
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("network filter trial failed: %v: %s", err, output)
		}
	}
}

func checkDeniedUnix(t *testing.T, path string) {
	t.Helper()
	conn, err := net.Dial("unix", path)
	if conn != nil {
		_ = conn.Close()
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("Unix service connection = %v, want EPERM", err)
	}
	if fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_DGRAM, 0); !errors.Is(err, syscall.EPERM) {
		if err == nil {
			_ = unix.Close(fd)
		}
		t.Errorf("Unix datagram socket = %v, want EPERM", err)
	}
	for _, kind := range []int{unix.SOCK_STREAM, unix.SOCK_SEQPACKET} {
		fd, err := unix.Socket(unix.AF_UNIX, kind, 0)
		if err != nil {
			t.Fatalf("Unix local IPC socket blocked: %v", err)
		}
		if err := unix.Sendto(fd, []byte("x"), 0, &unix.SockaddrUnix{Name: path}); !errors.Is(err, syscall.EPERM) {
			t.Errorf("Unix sendto = %v, want EPERM", err)
		}
		if _, err := unix.SendmsgN(fd, []byte("x"), nil, &unix.SockaddrUnix{Name: path}, 0); err == nil {
			t.Error("unconnected Unix IPC socket sent to service without connect")
		}
		_ = unix.Close(fd)
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
