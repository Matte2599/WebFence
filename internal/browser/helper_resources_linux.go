//go:build linux

package browser

import "golang.org/x/sys/unix"

const helperAddressSpaceLimit = 16 << 30

func platformHelperResourceLimits() []struct {
	resource int
	maximum  uint64
} {
	return []struct {
		resource int
		maximum  uint64
	}{{unix.RLIMIT_AS, helperAddressSpaceLimit}}
}
