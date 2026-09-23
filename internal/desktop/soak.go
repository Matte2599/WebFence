package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	qt "github.com/mappu/miqt/qt6"
)

// soakTest uses the real event loop and production widget callbacks. It is a
// repeatability/stability trial, not a screen-reader or keyboard interaction test.
func soakTest(w *workspace, duration time.Duration) int {
	started := time.Now()
	phase, cycles := 0, 0
	completed := false
	copied := ""
	w.copyEvidence = func(text string) { copied = text }
	timer := qt.NewQTimer()
	defer timer.Delete()
	emit := func(event, detail string) {
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Event     string  `json:"event"`
			Detail    string  `json:"detail,omitempty"`
			Elapsed   float64 `json:"elapsed_seconds"`
			Requested float64 `json:"requested_seconds"`
			Cycles    int     `json:"cycles"`
			Phase     int     `json:"phase"`
			Heap      uint64  `json:"go_heap_alloc_bytes"`
			Sys       uint64  `json:"go_sys_bytes"`
			GC        uint32  `json:"go_gc_count"`
			Variants  int     `json:"variant_cache_entries"`
			Platform  string  `json:"qt_platform"`
		}{event, detail, time.Since(started).Seconds(), duration.Seconds(), cycles, phase,
			memory.HeapAlloc, memory.Sys, memory.NumGC, len(w.variants), qt.QGuiApplication_PlatformName()})
	}
	fail := func(detail string) {
		timer.Stop()
		emit("failed", detail)
		qt.QCoreApplication_ExitWithRetcode(1)
	}
	timer.OnTimeout(func() {
		valid := true
		switch phase {
		case 0:
			w.load.Click()
			valid = len(w.visible) == 10000
		case 1:
			w.table.SelectRow(9999)
			w.advanced.SetChecked(true)
			w.copy.Click()
			record, ok := w.selected()
			valid = ok && w.selectedID == "DEMO-10000" && w.evidence.IsVisible() &&
				w.evidence.ToPlainText() == record.Evidence() && copied == record.Evidence()
		case 2:
			w.englishAction.Trigger()
			valid = w.locale == "en" && !w.preferenceError && w.selectedID == "DEMO-10000"
		case 3:
			w.search.SetText("DEMO-10000")
			valid = len(w.visible) == 1 && w.selectedID == "DEMO-10000"
		case 4:
			w.severity.SetCurrentIndex(1)
			valid = len(w.visible) == 1 && w.selectedID == "DEMO-10000"
		case 5:
			w.search.SetText("")
			valid = len(w.visible) == 3334 && w.selectedID == "DEMO-10000"
		case 6:
			w.severity.SetCurrentIndex(0)
			valid = len(w.visible) == 10000
		case 7:
			w.search.SetText("no-such-fixture")
			valid = len(w.visible) == 0 && w.selectedID == "" && w.evidence.ToPlainText() == "" && !w.copy.IsEnabled()
		case 8:
			w.italianAction.Trigger()
			valid = w.locale == "it" && !w.preferenceError && len(w.visible) == 0
		case 9:
			w.search.SetText("")
			w.table.SelectRow(42)
			record, ok := w.selected()
			valid = len(w.visible) == 10000 && ok && w.selectedID == "DEMO-00043" && w.evidence.ToPlainText() == record.Evidence()
		case 10:
			w.advanced.SetChecked(false)
			valid = !w.evidence.IsVisible() && !w.copy.IsVisible()
		case 11:
			w.clear.Click()
			valid = len(w.rows) == 0 && len(w.visible) == 0 && w.selectedID == "" && w.evidence.ToPlainText() == ""
		}
		if !valid {
			fail(fmt.Sprintf("unexpected workspace state at phase %d", phase))
			return
		}
		phase = (phase + 1) % 12
		if phase == 0 {
			cycles++
			if cycles%5 == 0 {
				emit("sample", "")
			}
			if time.Since(started) >= duration {
				completed = true
				timer.Stop()
				emit("passed", "completed whole cycles")
				qt.QCoreApplication_Quit()
			}
		}
	})
	emit("started", "synthetic fixtures; temporary preferences; clipboard intercepted")
	timer.SetTimerType(qt.PreciseTimer)
	timer.Start(250)
	result := qt.QApplication_Exec()
	if !completed {
		emit("incomplete", "event loop exited before the requested trial completed")
		return 1
	}
	return result
}
