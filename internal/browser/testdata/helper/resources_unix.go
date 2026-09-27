//go:build darwin || linux

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func describeResources() string {
	var addressSpace, openFiles, fileSize, core unix.Rlimit
	if unix.Getrlimit(unix.RLIMIT_AS, &addressSpace) != nil ||
		unix.Getrlimit(unix.RLIMIT_NOFILE, &openFiles) != nil ||
		unix.Getrlimit(unix.RLIMIT_FSIZE, &fileSize) != nil ||
		unix.Getrlimit(unix.RLIMIT_CORE, &core) != nil {
		return "resource-query-failed"
	}
	return fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d", addressSpace.Cur, addressSpace.Max,
		openFiles.Cur, openFiles.Max, fileSize.Cur, fileSize.Max, core.Cur, core.Max)
}
