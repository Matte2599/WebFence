//go:build windows

package browser

import (
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

func configureHelperProcess(*exec.Cmd) {}

func bindHelperProcess(cmd *exec.Cmd) (helperBoundary, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return helperBoundary{}, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
		windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 16
	limits.JobMemoryLimit = 1 << 30
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job)
		return helperBoundary{}, err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return helperBoundary{}, err
	}
	err = windows.AssignProcessToJobObject(job, process)
	_ = windows.CloseHandle(process)
	if err != nil {
		_ = windows.CloseHandle(job)
		return helperBoundary{}, err
	}
	return helperBoundary{
		kill:  func() { _ = windows.TerminateJobObject(job, 1) },
		close: func() { _ = windows.CloseHandle(job) },
	}, nil
}
