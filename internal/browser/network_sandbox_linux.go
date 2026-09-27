//go:build linux && (amd64 || arm64)

package browser

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ApplyHelperNetworkIsolation denies creation of non-Unix sockets by every
// thread in the trusted helper. The filter is inherited by its descendants.
// It must run before Qt or any browser process starts, with no inherited
// network descriptors. A separate broker holds all authorized target access.
func ApplyHelperNetworkIsolation() error {
	arch := uint32(unix.AUDIT_ARCH_X86_64)
	if runtime.GOARCH == "arm64" {
		arch = unix.AUDIT_ARCH_AARCH64
	}
	const (
		load  = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jeq   = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		jge   = unix.BPF_JMP | unix.BPF_JGE | unix.BPF_K
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
		{Code: jeq, K: unix.SYS_SOCKET, Jt: 4},         // socket(domain, ...)
		{Code: jeq, K: unix.SYS_SOCKETPAIR, Jt: 3},     // socketpair(domain, ...)
		{Code: jeq, K: unix.SYS_IO_URING_SETUP, Jt: 1}, // no async socket bypass
		{Code: ret, K: allow},
		{Code: ret, K: deny},
		{Code: load, K: 16}, // seccomp_data.args[0], low 32 bits
		{Code: jeq, K: unix.AF_UNIX, Jt: 1},
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
