//go:build m3cdplab && linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	fileRead = unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_READ_FILE |
		unix.LANDLOCK_ACCESS_FS_READ_DIR
	fileWrite = unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_REMOVE_DIR |
		unix.LANDLOCK_ACCESS_FS_REMOVE_FILE | unix.LANDLOCK_ACCESS_FS_MAKE_CHAR |
		unix.LANDLOCK_ACCESS_FS_MAKE_DIR | unix.LANDLOCK_ACCESS_FS_MAKE_REG |
		unix.LANDLOCK_ACCESS_FS_MAKE_SOCK | unix.LANDLOCK_ACCESS_FS_MAKE_FIFO |
		unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK | unix.LANDLOCK_ACCESS_FS_MAKE_SYM |
		unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE
)

// applyCDPFileBoundary is confined to the disposable Linux fixture. It must
// be called on the locked OS thread that starts Chromium. The allowlist grants
// system files read/execute access and grants writes only to the helper's
// private directory. Lack of Landlock support is a trial failure.
func applyCDPFileBoundary(privateDir, deniedFile string) error {
	if !filepath.IsAbs(privateDir) || !filepath.IsAbs(deniedFile) ||
		filepath.Clean(privateDir) == filepath.Clean(deniedFile) {
		return errors.New("invalid CDP file boundary paths")
	}
	abi, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION))
	if errno != 0 || abi < 3 {
		return errors.New("Landlock ABI 3 or newer is required for the CDP trial")
	}
	attr := unix.LandlockRulesetAttr{Access_fs: fileRead | fileWrite}
	ruleset, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr.Access_fs), 0)
	if errno != 0 {
		return fmt.Errorf("create CDP file ruleset: %w", errno)
	}
	defer unix.Close(int(ruleset))
	for _, path := range []string{"/usr", "/lib", "/lib64", "/etc", "/proc", "/dev"} {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := addCDPFileRule(int(ruleset), path, fileRead); err != nil {
			return err
		}
	}
	if err := addCDPFileRule(int(ruleset), privateDir, fileRead|fileWrite); err != nil {
		return err
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("set CDP no-new-privileges: %w", err)
	}
	_, _, errno = unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, ruleset, 0, 0)
	if errno != 0 {
		return fmt.Errorf("enforce CDP file ruleset: %w", errno)
	}
	if _, err := os.ReadFile(deniedFile); !errors.Is(err, syscall.EACCES) {
		return fmt.Errorf("CDP private-file read was not denied: %v", err)
	}
	file, err := os.OpenFile(deniedFile, os.O_WRONLY, 0)
	if file != nil {
		_ = file.Close()
	}
	if !errors.Is(err, syscall.EACCES) {
		return fmt.Errorf("CDP private-file write was not denied: %v", err)
	}
	probe := filepath.Join(privateDir, "landlock-readable")
	if err := os.WriteFile(probe, []byte("allowed"), 0600); err != nil {
		return fmt.Errorf("CDP private directory is not writable: %w", err)
	}
	allowed, err := exec.Command("/usr/bin/cat", probe).Output()
	if err != nil || string(allowed) != "allowed" {
		return errors.New("CDP file boundary blocked an allowed child read")
	}
	if err := exec.Command("/usr/bin/cat", deniedFile).Run(); err == nil {
		return errors.New("CDP file boundary did not reach a child process")
	}
	return nil
}

func addCDPFileRule(ruleset int, path string, access uint64) error {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open CDP file rule path %q: %w", path, err)
	}
	defer unix.Close(fd)
	rule := unix.LandlockPathBeneathAttr{Allowed_access: access, Parent_fd: int32(fd)}
	_, _, errno := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, uintptr(ruleset),
		uintptr(unix.LANDLOCK_RULE_PATH_BENEATH), uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("add CDP file rule %q: %w", path, errno)
	}
	return nil
}
