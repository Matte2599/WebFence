//go:build linux && (amd64 || arm64)

package browser

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ApplyHelperNetworkIsolation denies direct network sockets and new connections
// to local services by every thread in the trusted helper. The filter is
// inherited by its descendants. Authorized broker IPC must be connected by
// the parent and inherited before this function is called.
// It must run before Qt or any browser process starts, with no inherited
// INET descriptors. A separate broker holds all authorized target access.
func ApplyHelperNetworkIsolation() error {
	arch := uint32(unix.AUDIT_ARCH_X86_64)
	if runtime.GOARCH == "arm64" {
		arch = unix.AUDIT_ARCH_AARCH64
	}
	const (
		load  = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jeq   = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		jge   = unix.BPF_JMP | unix.BPF_JGE | unix.BPF_K
		band  = unix.BPF_ALU | unix.BPF_AND | unix.BPF_K
		ret   = unix.BPF_RET | unix.BPF_K
		deny  = unix.SECCOMP_RET_ERRNO | uint32(unix.EPERM)
		allow = unix.SECCOMP_RET_ALLOW
		kill  = unix.SECCOMP_RET_KILL_PROCESS
	)
	filters := []unix.SockFilter{
		{Code: load, K: 4},          // seccomp_data.arch
		{Code: jeq, K: arch, Jt: 1}, // reject other syscall ABIs
		{Code: ret, K: kill},
		{Code: load, K: 0},                // seccomp_data.nr
		{Code: jge, K: 0x40000000, Jf: 1}, // reject x32 and high syscall numbers
		{Code: ret, K: kill},
		{Code: jeq, K: unix.SYS_IO_URING_SETUP, Jf: 1}, // no async socket bypass
		{Code: ret, K: deny},
		{Code: jeq, K: unix.SYS_CONNECT, Jf: 1}, // no local pathname or abstract sockets
		{Code: ret, K: deny},
		{Code: jeq, K: unix.SYS_SENDTO, Jf: 6}, // allow only a null destination
		{Code: load, K: 48},                    // args[4], destination pointer low
		{Code: jeq, K: 0, Jf: 3},
		{Code: load, K: 52}, // destination pointer high
		{Code: jeq, K: 0, Jf: 1},
		{Code: ret, K: allow},
		{Code: ret, K: deny},
		{Code: jeq, K: unix.SYS_SENDMMSG, Jf: 1},
		{Code: ret, K: deny},
		{Code: jeq, K: unix.SYS_SOCKET, Jt: 2},
		{Code: jeq, K: unix.SYS_SOCKETPAIR, Jt: 1},
		{Code: ret, K: allow},
		{Code: load, K: 16}, // seccomp_data.args[0], low 32 bits
		{Code: jeq, K: unix.AF_UNIX, Jt: 1},
		{Code: ret, K: deny},
		{Code: load, K: 24},  // seccomp_data.args[1], low 32 bits
		{Code: band, K: 0xf}, // SOCK_CLOEXEC and SOCK_NONBLOCK are outside type bits
		{Code: jeq, K: unix.SOCK_STREAM, Jt: 2},
		{Code: jeq, K: unix.SOCK_SEQPACKET, Jt: 1},
		{Code: ret, K: deny},
		{Code: ret, K: allow},
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return ErrHelperFailed
	}
	program := unix.SockFprog{Len: uint16(len(filters)), Filter: &filters[0]}
	r1, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER,
		unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&program)))
	runtime.KeepAlive(filters)
	if errno != 0 || r1 != 0 {
		return ErrHelperFailed
	}
	return nil
}
