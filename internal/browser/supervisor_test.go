package browser

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func buildHelper(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "helper")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", path, "./testdata/helper")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, out)
	}
	return path
}

func TestRunHelper(t *testing.T) {
	helper := buildHelper(t)
	limits := HelperLimits{MaxRuntime: 10 * time.Second, MaxOutputBytes: 1024}
	got, err := RunHelper(context.Background(), helper, []byte("echo"), limits)
	if err != nil || string(got) != "ok" {
		t.Fatalf("helper output = %q, %v", got, err)
	}
	if _, err := RunHelper(context.Background(), helper, []byte("fail"), limits); !errors.Is(err, ErrHelperFailed) ||
		err.Error() != ErrHelperFailed.Error() {
		t.Fatalf("helper failure = %v", err)
	}
	if _, err := RunHelper(context.Background(), helper, []byte("spam"), limits); !errors.Is(err, ErrHelperOutput) {
		t.Fatalf("output limit = %v", err)
	}
	t.Setenv("WF_BROWSER_PRIVATE_SECRET", "synthetic-secret")
	got, err = RunHelper(context.Background(), helper, []byte("env"), limits)
	if err != nil || string(got) != "isolated" {
		t.Fatalf("helper environment = %q, %v", got, err)
	}
}

func TestRunHelperKillsDescendantsOnTimeout(t *testing.T) {
	helper := buildHelper(t)
	marker := filepath.Join(t.TempDir(), "child-survived")
	_, err := RunHelper(context.Background(), helper, []byte("grandchild:"+marker),
		HelperLimits{MaxRuntime: 150 * time.Millisecond, MaxOutputBytes: 1024})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout = %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("descendant survived timeout: %v", err)
	}
}

func TestRunHelperRejectsInvalidInput(t *testing.T) {
	if _, err := RunHelper(context.Background(), "", []byte("x"), HelperLimits{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("invalid limits = %v", err)
	}
}

func TestHelperEnvironmentKeepsVirtualDisplayAuthorityOnly(t *testing.T) {
	t.Setenv("DISPLAY", ":99")
	t.Setenv("XAUTHORITY", "/tmp/synthetic-xauth")
	t.Setenv("WF_BROWSER_PRIVATE_SECRET", "synthetic-secret")
	env := "\n" + strings.Join(helperEnvironment(t.TempDir()), "\n") + "\n"
	if !strings.Contains(env, "\nDISPLAY=:99\n") || !strings.Contains(env, "\nXAUTHORITY=/tmp/synthetic-xauth\n") {
		t.Fatalf("virtual display environment missing: %q", env)
	}
	if strings.Contains(env, "WF_BROWSER_PRIVATE_SECRET") {
		t.Fatal("unapproved environment variable reached the helper")
	}
}
