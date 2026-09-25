//go:build darwin || linux

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockStoreFile(f *os.File) error           { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }
func unlockStoreFilePlatform(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
func isStoreLockBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}
