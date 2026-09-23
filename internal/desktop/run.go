package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Matte2599/WebFence/internal/i18n"
	"github.com/Matte2599/WebFence/internal/preferences"
	qt "github.com/mappu/miqt/qt6"
)

// Run must be called from main: Qt owns the main OS thread.
func Run(args []string) int {
	selfTesting := len(args) == 2 && args[1] == "--self-test"
	if len(args) > 1 && !selfTesting {
		fmt.Fprintln(os.Stderr, "Usage: webfence [--self-test]")
		return 2
	}
	runtime.LockOSThread()
	qt.NewQApplication(args)
	systemLocale := qt.NewQLocale()
	locale := i18n.Normalize(systemLocale.Name())
	systemLocale.Delete()
	dir, err := os.UserConfigDir()
	path := ""
	if err == nil {
		path = filepath.Join(dir, "WebFence", "ui-language")
	}
	if selfTesting {
		dir, err = os.MkdirTemp("", "webfence-selftest-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer os.RemoveAll(dir)
		path = filepath.Join(dir, "ui-language")
		locale = "it"
	}
	preferenceError := err != nil
	if !preferenceError {
		locale, err = preferences.Load(path, locale)
		preferenceError = err != nil
	}
	w := newWorkspace(locale, path, preferenceError)
	defer w.dispose()
	w.window.Show()
	if selfTesting {
		// Native window managers deliver map/activation asynchronously. Wait for
		// that before testing keyboard focus; do not change OS focus preferences.
		w.window.ActivateWindow()
		deadline := time.Now().Add(3 * time.Second)
		for {
			qt.QCoreApplication_ProcessEvents()
			if w.window.IsActiveWindow() || time.Now().After(deadline) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		return selfTest(w)
	}
	return qt.QApplication_Exec()
}
