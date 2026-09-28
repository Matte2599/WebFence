package browser

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	ErrHelperFailed = errors.New("browser_helper_failed")
	ErrHelperOutput = errors.New("browser_helper_output_limit")
)

// HelperLimits bound an untrusted browser helper's lifetime and IPC output.
// The helper receives its configuration on stdin only after the parent has
// installed the platform's process-tree termination boundary.
type HelperLimits struct {
	MaxRuntime     time.Duration
	MaxOutputBytes int
	// InheritedFiles are already connected, trusted IPC descriptors. On Unix,
	// the helper receives them at descriptors 3 onward; Windows rejects them.
	InheritedFiles []*os.File
}

// RunHelper starts a browser helper without a shell. The caller must provide
// an executable that applies resource limits and reads its configuration
// before starting any browser processes. The parent installs process-tree
// supervision before sending that configuration. The payload and helper
// stderr are never included in returned errors.
func RunHelper(ctx context.Context, executable string, payload []byte, limits HelperLimits) ([]byte, error) {
	if ctx == nil || !filepath.IsAbs(executable) || len(payload) == 0 || len(payload) > 64<<10 ||
		limits.MaxRuntime <= 0 || limits.MaxRuntime > 4*time.Hour ||
		limits.MaxOutputBytes <= 0 || limits.MaxOutputBytes > 1<<20 ||
		len(limits.InheritedFiles) > 8 ||
		(runtime.GOOS == "windows" && len(limits.InheritedFiles) != 0) {
		return nil, ErrConfig
	}
	for _, file := range limits.InheritedFiles {
		if file == nil {
			return nil, ErrConfig
		}
	}
	run, cancel := context.WithTimeout(ctx, limits.MaxRuntime)
	defer cancel()
	if err := run.Err(); err != nil {
		return nil, err
	}
	privateDir, err := os.MkdirTemp("", "webfence-browser-")
	if err != nil {
		return nil, ErrHelperFailed
	}
	defer os.RemoveAll(privateDir)
	cmd := exec.Command(executable, "--browser-helper")
	cmd.Env = helperEnvironment(privateDir)
	cmd.Dir = privateDir
	cmd.ExtraFiles = limits.InheritedFiles
	configureHelperProcess(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, ErrHelperFailed
	}
	output := &boundedHelperOutput{max: limits.MaxOutputBytes, exceeded: make(chan struct{})}
	cmd.Stdout = output
	cmd.Stderr = output.stderrWriter()
	if err := cmd.Start(); err != nil {
		return nil, ErrHelperFailed
	}
	boundary, err := bindHelperProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, ErrHelperFailed
	}
	defer boundary.close()
	defer boundary.kill()
	// A separate writer prevents a helper that never reads stdin from blocking
	// cancellation. Closing the pipe is needed for a complete JSON payload.
	wrote := make(chan error, 1)
	go func() {
		_, err := stdin.Write(payload)
		if closeErr := stdin.Close(); err == nil {
			err = closeErr
		}
		wrote <- err
	}()
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err = <-waited:
		if output.isExceeded() {
			boundary.kill()
			return nil, ErrHelperOutput
		}
		if err != nil {
			boundary.kill()
			return nil, ErrHelperFailed
		}
		if err = <-wrote; err != nil {
			return nil, ErrHelperFailed
		}
		return output.bytes(), nil
	case <-output.exceeded:
		boundary.kill()
		<-waited
		return nil, ErrHelperOutput
	case <-run.Done():
		boundary.kill()
		<-waited
		return nil, run.Err()
	}
}

func helperEnvironment(privateDir string) []string {
	values := []string{
		"HOME=" + privateDir, "TMPDIR=" + privateDir, "TMP=" + privateDir,
		"TEMP=" + privateDir, "USERPROFILE=" + privateDir,
		"APPDATA=" + privateDir, "LOCALAPPDATA=" + privateDir,
		"XDG_CONFIG_HOME=" + privateDir, "XDG_CACHE_HOME=" + privateDir,
		"XDG_DATA_HOME=" + privateDir,
	}
	for _, name := range []string{
		"PATH", "SystemRoot", "WINDIR", "QT_QPA_PLATFORM", "QT_PLUGIN_PATH",
		"QT_QPA_PLATFORM_PLUGIN_PATH", "QTWEBENGINEPROCESS_PATH", "DISPLAY",
		"XAUTHORITY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH",
	} {
		if value, ok := os.LookupEnv(name); ok {
			values = append(values, name+"="+value)
		}
	}
	return values
}

type helperBoundary struct {
	kill  func()
	close func()
}

type boundedHelperOutput struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	max      int
	used     int
	tooMuch  bool
	exceeded chan struct{}
}

func (w *boundedHelperOutput) Write(p []byte) (int, error) { return w.add(p, true) }

func (w *boundedHelperOutput) stderrWriter() io.Writer {
	return helperStderr{w}
}

type helperStderr struct{ output *boundedHelperOutput }

func (w helperStderr) Write(p []byte) (int, error) { return w.output.add(p, false) }

func (w *boundedHelperOutput) add(p []byte, keep bool) (int, error) {
	originalSize := len(p)
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.tooMuch {
		remaining := w.max - w.used
		if len(p) > remaining {
			w.tooMuch = true
			close(w.exceeded)
			p = p[:remaining]
		}
		w.used += len(p)
		if keep {
			_, _ = w.buf.Write(p)
		}
	}
	return originalSize, nil
}

func (w *boundedHelperOutput) isExceeded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.tooMuch
}

func (w *boundedHelperOutput) bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return bytes.Clone(w.buf.Bytes())
}
