package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/i18n"
	"github.com/Matte2599/WebFence/internal/preferences"
	"github.com/Matte2599/WebFence/internal/storage"
	qt "github.com/mappu/miqt/qt6"
)

// Run must be called from main: Qt owns the main OS thread.
func Run(args []string) int {
	selfTesting := len(args) == 2 && args[1] == "--self-test"
	soakDuration, valid := parseSoakArgs(args)
	if !valid || (len(args) > 1 && !selfTesting && soakDuration == 0) {
		fmt.Fprintln(os.Stderr, "Usage: webfence [--self-test | --soak-test=10s..1h]")
		return 2
	}
	runtime.LockOSThread()
	qt.NewQApplication(args)
	// Keep WebFence's explicit tab order reachable on Cocoa too. This override
	// changes only this QStyleHints instance, never the user's OS preferences.
	// The setter is public in Qt headers/MIQT but documented internal by Qt;
	// keep native regression coverage when changing the pinned Qt baseline.
	qt.QGuiApplication_StyleHints().SetTabFocusBehavior(qt.TabFocusAllControls)
	systemLocale := qt.NewQLocale()
	locale := i18n.Normalize(systemLocale.Name())
	systemLocale.Delete()
	dir, err := os.UserConfigDir()
	path := ""
	if err == nil {
		path = filepath.Join(dir, "WebFence", "ui-language")
	}
	if selfTesting || soakDuration > 0 {
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
	var scanStore *storage.Store
	var scanErr error
	if dir != "" {
		dataDir := filepath.Join(dir, "WebFence")
		if mkErr := os.MkdirAll(dataDir, 0o700); mkErr != nil {
			scanErr = storage.ErrUnavailable
		} else {
			scanStore, scanErr = storage.Open(context.Background(), filepath.Join(dataDir, "projects.sqlite"))
		}
	} else {
		scanErr = storage.ErrUnavailable
	}
	defer func() {
		if scanStore != nil {
			_ = scanStore.Close()
		}
	}()
	w.scan = newScannerUI(w, scanStore, scanErr)
	defer w.dispose()
	w.window.Show()
	if soakDuration > 0 {
		return soakTest(w, soakDuration)
	}
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

// Reject malformed/bounded trial arguments before creating QApplication.
func parseSoakArgs(args []string) (time.Duration, bool) {
	if len(args) == 2 && strings.HasPrefix(args[1], "--soak-test=") {
		duration, err := time.ParseDuration(strings.TrimPrefix(args[1], "--soak-test="))
		return duration, err == nil && duration >= 10*time.Second && duration <= time.Hour
	}
	return 0, true
}
