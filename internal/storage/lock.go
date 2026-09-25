package storage

import (
	"errors"
	"os"
	"runtime"
)

// Keep the lock file after close: unlinking it would allow two processes to
// lock different inodes for the same database path.
func acquireStoreLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return nil, ErrInvalidPath
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, ErrUnavailable
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, ErrUnavailable
	}
	if err := lockStoreFile(f); err != nil {
		_ = f.Close()
		if isStoreLockBusy(err) {
			return nil, ErrBusy
		}
		return nil, ErrUnavailable
	}
	return f, nil
}

func unlockStoreFile(f *os.File) error {
	if f == nil {
		return nil
	}
	err := unlockStoreFilePlatform(f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
