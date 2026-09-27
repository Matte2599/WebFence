//go:build darwin || linux

package browser

import "golang.org/x/sys/unix"

const (
	helperOpenFilesLimit = 1024
	helperFileSizeLimit  = 64 << 20
)

// ApplyHelperResourceLimits must run in the trusted helper before it starts
// Qt WebEngine or any descendant. Limits are inherited by children, but they
// apply to each process separately, not to the process tree as a whole.
func ApplyHelperResourceLimits() error {
	for _, item := range []struct {
		resource int
		maximum  uint64
	}{
		{unix.RLIMIT_NOFILE, helperOpenFilesLimit},
		{unix.RLIMIT_FSIZE, helperFileSizeLimit},
		{unix.RLIMIT_CORE, 0},
	} {
		var current unix.Rlimit
		if err := unix.Getrlimit(item.resource, &current); err != nil {
			return ErrHelperFailed
		}
		// Never raise an inherited bound, including its hard limit.
		maximum := min(item.maximum, current.Cur, current.Max)
		if err := unix.Setrlimit(item.resource, &unix.Rlimit{Cur: maximum, Max: maximum}); err != nil {
			return ErrHelperFailed
		}
	}
	return nil
}
