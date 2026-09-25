package storage

import (
	"context"
	"os"
	"path/filepath"
)

// Backup writes a consistent SQLite snapshot to a new private file. The
// caller chooses a trusted local destination and retains responsibility for
// its lifetime. Existing files are never overwritten.
func (s *Store) Backup(ctx context.Context, destination string) error {
	if s == nil || s.db == nil || ctx == nil || !filepath.IsAbs(destination) || filepath.Base(destination) == "." {
		return ErrInvalidPath
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrUnavailable
	}
	if s.activeScanID != "" {
		return ErrBusy
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ErrInvalidPath
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(destination)
		return ErrUnavailable
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(destination)
		}
	}()
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return storageError(ctx, err)
	}
	// FlushFileBuffers requires a write-capable handle on Windows.
	f, err = os.OpenFile(destination, os.O_RDWR, 0)
	if err != nil {
		return ErrUnavailable
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return ErrUnavailable
	}
	if err := f.Close(); err != nil {
		return ErrUnavailable
	}
	keep = true
	return nil
}
