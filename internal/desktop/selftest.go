package desktop

import (
	"fmt"
	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/preferences"
	qt "github.com/mappu/miqt/qt6"
	"path/filepath"
	"strings"
	"time"
)

// selfTest exercises the real Qt model/widgets on their owner OS thread.
// It is not a screen-reader test. It never writes to the system clipboard.
func selfTest(w *workspace) int {
	failures := 0
	check := func(ok bool, name string) {
		if !ok {
			fmt.Println("FAIL", name)
			failures++
		} else {
			fmt.Println("PASS", name)
		}
	}
	check(len(w.visible) == 0, "initial empty state")
	start := time.Now()
	w.load.Click()
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("load_10000_ms=%.3f\n", float64(time.Since(start).Microseconds())/1000)
	check(len(w.visible) == 10000, "10000 rows loaded")
	parent := qt.NewQModelIndex()
	defer parent.Delete()
	// Rendering/accessibility can request the same model data repeatedly.
	for i := 0; i < 10000; i++ {
		w.model.Data(w.model.Index(i, 0, parent), int(qt.DisplayRole))
	}
	cached := len(w.variants)
	for i := 0; i < 10000; i++ {
		w.model.Data(w.model.Index(i, 0, parent), int(qt.DisplayRole))
	}
	check(len(w.variants) == cached, "repeated model reads do not grow owned variant cache")
	w.table.SelectRow(9999)
	original := w.evidence.ToPlainText()
	check(w.selectedID == "DEMO-10000" && strings.Contains(original, "<script>"), "last row and inert evidence")
	w.englishAction.Trigger()
	saved, err := preferences.Load(w.preferencePath, "it")
	check(err == nil && saved == "en", "language action persists English")
	w.italianAction.Trigger()
	saved, err = preferences.Load(w.preferencePath, "en")
	check(err == nil && saved == "it", "language action persists Italian")
	check(w.selectedID == "DEMO-10000" && original == w.evidence.ToPlainText(), "language preserves identity and evidence")
	w.advanced.SetChecked(true)
	check(w.evidence.IsVisible(), "advanced detail visible")
	copied := ""
	w.copyEvidence = func(value string) { copied = value }
	w.copy.Click()
	check(copied == original, "explicit copy preserves evidence")
	path := w.preferencePath
	w.preferencePath = filepath.Dir(path)
	w.englishAction.Trigger()
	check(w.locale == "en" && w.notice.IsVisible(), "save error remains visible without blocking language change")
	w.preferencePath = path
	w.italianAction.Trigger()
	check(!w.notice.IsVisible(), "successful save clears preference warning")
	w.severity.SetCurrentIndex(1)
	check(len(w.visible) == 3334 && w.selectedID == "DEMO-10000", "severity preserves matching selection")
	start = time.Now()
	w.search.SetText("DEMO-10000")
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("filter_last_ms=%.3f\n", float64(time.Since(start).Microseconds())/1000)
	check(len(w.visible) == 1 && w.selectedID == "DEMO-10000", "filter last row")
	w.search.SetText("missing")
	check(len(w.visible) == 0 && w.selectedID == "" && w.evidence.ToPlainText() == "" && !w.copy.IsEnabled(), "no stale evidence on empty results")
	w.search.SetText("")
	w.severity.SetCurrentIndex(0)
	w.table.SelectRow(0)
	long := strings.Repeat(demo.Records()[0].Evidence(), 2048)
	w.evidence.SetPlainText(long)
	qt.QCoreApplication_ProcessEvents()
	check(w.evidence.ToPlainText() == long, "long evidence exact text roundtrip")
	fmt.Printf("long_evidence_bytes=%d\n", len(long))
	w.clear.Click()
	check(len(w.visible) == 0 && w.evidence.ToPlainText() == "", "clear dataset")
	if failures > 0 {
		return 1
	}
	return 0
}
