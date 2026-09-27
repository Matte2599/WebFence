//go:build darwin || linux

package browser

import (
	"context"
	"fmt"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestRunHelperAppliesInheritedResourceLimits(t *testing.T) {
	var beforeAS, beforeFiles, beforeSize, beforeCore unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_AS, &beforeAS); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &beforeFiles); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_FSIZE, &beforeSize); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_CORE, &beforeCore); err != nil {
		t.Fatal(err)
	}
	got, err := RunHelper(context.Background(), buildHelper(t), []byte("resources"),
		HelperLimits{MaxRuntime: 10 * time.Second, MaxOutputBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d",
		beforeAS.Cur, beforeAS.Max,
		min(beforeFiles.Cur, beforeFiles.Max, uint64(helperOpenFilesLimit)),
		min(beforeFiles.Cur, beforeFiles.Max, uint64(helperOpenFilesLimit)),
		min(beforeSize.Cur, beforeSize.Max, uint64(helperFileSizeLimit)),
		min(beforeSize.Cur, beforeSize.Max, uint64(helperFileSizeLimit)),
		uint64(0), uint64(0))
	if string(got) != want {
		t.Fatalf("helper and descendant resource limits = %q, want %q", got, want)
	}
	var afterAS, afterFiles, afterSize, afterCore unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_AS, &afterAS); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &afterFiles); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_FSIZE, &afterSize); err != nil {
		t.Fatal(err)
	}
	if err := unix.Getrlimit(unix.RLIMIT_CORE, &afterCore); err != nil {
		t.Fatal(err)
	}
	if beforeAS != afterAS || beforeFiles != afterFiles || beforeSize != afterSize || beforeCore != afterCore {
		t.Fatal("resource limits leaked into parent")
	}
}
