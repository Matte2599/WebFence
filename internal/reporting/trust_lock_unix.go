//go:build darwin || linux

package reporting

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockTrustFile(f *os.File) error   { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }
func unlockTrustFile(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
