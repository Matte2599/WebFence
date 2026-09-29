//go:build windows

// This optional lab checks a network-denied Windows AppContainer with a
// synthetic listener on a system-assigned loopback port. An explicit --browser path additionally trials Chromium over inherited pipes.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	securityCapabilitiesAttribute = 0x00020009
	tokenIsAppContainer           = 29
	deniedExit                    = 17
	allowedExit                   = 18
	notSandboxedExit              = 19
	otherNetworkExit              = 20
	timedOutExit                  = 21
)

var (
	userenv                   = windows.NewLazySystemDLL("userenv.dll")
	createAppContainerProfile = userenv.NewProc("CreateAppContainerProfile")
	deleteAppContainerProfile = userenv.NewProc("DeleteAppContainerProfile")
)

type securityCapabilities struct {
	appContainerSID *windows.SID
	capabilities    unsafe.Pointer
	count           uint32
	reserved        uint32
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--sandboxed-child" {
		child(os.Args[2])
	}
	if len(os.Args) != 1 && !(len(os.Args) == 3 && (os.Args[1] == "--browser" || os.Args[1] == "--browser-control")) {
		fmt.Fprintln(os.Stderr, "invalid lab invocation")
		os.Exit(1)
	}
	if err := parent(); err != nil {
		fmt.Fprintln(os.Stderr, "M3 Windows AppContainer lab failed:", err)
		os.Exit(1)
	}
}

func child(address string) {
	if !strings.HasPrefix(address, "127.0.0.1:") || !inAppContainer() {
		os.Exit(notSandboxedExit)
	}
	conn, err := net.DialTimeout("tcp4", address, time.Second)
	if conn != nil {
		_ = conn.Close()
		os.Exit(allowedExit)
	}
	if errors.Is(err, windows.WSAEACCES) {
		os.Exit(deniedExit)
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		os.Exit(timedOutExit)
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && errno != 0 {
		os.Exit(int(errno))
	}
	os.Exit(otherNetworkExit)
}

func inAppContainer() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()
	var value uint32
	var size uint32
	if err := windows.GetTokenInformation(token, tokenIsAppContainer,
		(*byte)(unsafe.Pointer(&value)), uint32(unsafe.Sizeof(value)), &size); err != nil {
		return false
	}
	return size == uint32(unsafe.Sizeof(value)) && value != 0
}

func parent() error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("synthetic listener: %w", err)
	}
	defer listener.Close()
	control, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		return fmt.Errorf("unsandboxed loopback control: %w", err)
	}
	_ = control.Close()
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("profile nonce: %w", err)
	}
	name := "WebFence.M3.Probe." + hex.EncodeToString(nonce[:])
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	display16, _ := windows.UTF16PtrFromString("WebFence M3 AppContainer Probe")
	description16, _ := windows.UTF16PtrFromString("Local synthetic network isolation trial")
	var sid *windows.SID
	hresult, _, _ := createAppContainerProfile.Call(
		uintptr(unsafe.Pointer(name16)), uintptr(unsafe.Pointer(display16)),
		uintptr(unsafe.Pointer(description16)), 0, 0, uintptr(unsafe.Pointer(&sid)))
	if hresult != 0 || sid == nil {
		return fmt.Errorf("CreateAppContainerProfile HRESULT 0x%x", hresult)
	}
	defer func() {
		_, _, _ = deleteAppContainerProfile.Call(uintptr(unsafe.Pointer(name16)))
	}()
	defer windows.FreeSid(sid)

	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return fmt.Errorf("process attributes: %w", err)
	}
	defer attributes.Delete()
	capabilities := securityCapabilities{appContainerSID: sid}
	if err := attributes.Update(securityCapabilitiesAttribute,
		unsafe.Pointer(&capabilities), unsafe.Sizeof(capabilities)); err != nil {
		return fmt.Errorf("security capabilities: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable16, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return err
	}
	command16, err := windows.UTF16PtrFromString(windows.EscapeArg(executable) +
		" --sandboxed-child " + listener.Addr().String())
	if err != nil {
		return err
	}
	startup := windows.StartupInfoEx{
		ProcThreadAttributeList: attributes.List(),
	}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	var process windows.ProcessInformation
	err = windows.CreateProcess(executable16, command16, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_SUSPENDED|
			windows.CREATE_NO_WINDOW, nil, nil, &startup.StartupInfo, &process)
	if err != nil {
		return fmt.Errorf("CreateProcess in AppContainer: %w", err)
	}
	defer windows.CloseHandle(process.Thread)
	defer windows.CloseHandle(process.Process)
	defer windows.TerminateProcess(process.Process, 1)

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("CreateJobObject: %w", err)
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
		windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 4
	limits.JobMemoryLimit = 512 << 20
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return fmt.Errorf("job limits: %w", err)
	}
	if err := windows.AssignProcessToJobObject(job, process.Process); err != nil {
		return fmt.Errorf("job assignment: %w", err)
	}
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		return fmt.Errorf("ResumeThread: %w", err)
	}
	status, err := windows.WaitForSingleObject(process.Process, 5000)
	if err != nil || status != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("sandboxed child did not exit in time: wait=%d error=%v", status, err)
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process.Process, &exitCode); err != nil {
		return fmt.Errorf("child exit code: %w", err)
	}
	if exitCode != deniedExit && exitCode != timedOutExit {
		return fmt.Errorf("sandboxed child returned %d instead of unreachable loopback", exitCode)
	}
	if exitCode == timedOutExit {
		fmt.Println("PASS M3 Windows AppContainer: token confirmed, unsandboxed loopback reachable, sandboxed loopback timed out, process job limited")
	} else {
		fmt.Println("PASS M3 Windows AppContainer: token confirmed, unsandboxed loopback reachable, sandboxed loopback denied, process job limited")
	}
	if len(os.Args) == 3 {
		return browserTrial(os.Args[2], sid, os.Args[1] == "--browser-control")
	}
	return nil
}
