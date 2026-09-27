//go:build darwin

package browser

// macOS rejects RLIMIT_AS with EINVAL. Address-space limits are intentionally
// omitted rather than suggesting that the OS enforces one here.
func platformHelperResourceLimits() []struct {
	resource int
	maximum  uint64
} {
	return nil
}
