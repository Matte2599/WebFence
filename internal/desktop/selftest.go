package desktop

import (
	"fmt"
	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/preferences"
	qt "github.com/mappu/miqt/qt6"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// selfTest exercises the real Qt model/widgets on their owner OS thread.
// It is not a screen-reader test. It never writes to the system clipboard.
func selfTest(w *workspace) int {
	// A headless platform has no screen reader to activate Qt accessibility.
	// Activation is required for model-reset cache invalidation.
	wasActive := qt.QAccessible_IsActive()
	qt.QAccessible_SetActive(true)
	defer qt.QAccessible_SetActive(wasActive)
	failures := 0
	check := func(ok bool, name string) {
		if !ok {
			fmt.Println("FAIL", name)
			failures++
		} else {
			fmt.Println("PASS", name)
		}
	}
	// These are Qt interfaces, not the OS accessibility bridge or a screen reader.
	checkAccessible := func(stage string) {
		qt.QCoreApplication_ProcessEvents()
		iface := qt.QAccessible_QueryAccessibleInterface(w.table.QObject)
		if iface == nil || iface.TableInterface() == nil {
			check(false, "Qt accessible table: "+stage)
			return
		}
		table := iface.TableInterface()
		check(table.RowCount() == len(w.visible) && table.ColumnCount() == 4, "Qt accessible dimensions: "+stage)
		if len(w.visible) > 0 {
			row := len(w.visible) - 1
			cell := table.CellAt(row, 0)
			check(cell != nil && cell.Text(qt.QAccessible__Name) == w.visible[row].ID, "Qt accessible last ID: "+stage)
		}
		// Interfaces/cells are owned by Qt's accessibility cache, never delete them here.
	}
	check(len(w.visible) == 0, "initial empty state")
	checkAccessible("empty")
	start := time.Now()
	w.load.Click()
	qt.QCoreApplication_ProcessEvents()
	fmt.Printf("load_10000_ms=%.3f\n", float64(time.Since(start).Microseconds())/1000)
	check(len(w.visible) == 10000, "10000 rows loaded")
	checkAccessible("loaded")
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
	w.evidenceAction.Trigger()
	check(w.evidence.IsVisible() && w.evidence.HasFocus(), "evidence action reveals and focuses details")
	cursor := w.evidence.TextCursor()
	cursor.SetPosition(0)
	cursor.SetPosition2(10, qt.QTextCursor__KeepAnchor)
	w.evidence.SetTextCursor(cursor)
	selectedText := cursor.SelectedText()
	runtime.SetFinalizer(cursor, nil) // Release QTextCursor on its owner thread.
	cursor.Delete()
	resets := 0
	w.model.OnModelReset(func() { resets++ })
	w.englishAction.Trigger()
	cursor = w.evidence.TextCursor()
	check(resets == 0 && cursor.SelectedText() == selectedText, "translation preserves model and evidence text selection")
	runtime.SetFinalizer(cursor, nil) // Release QTextCursor on its owner thread.
	cursor.Delete()
	checkAccessible("translated")
	saved, err := preferences.Load(w.preferencePath, "it")
	check(err == nil && saved == "en", "language action persists English")
	w.italianAction.Trigger()
	saved, err = preferences.Load(w.preferencePath, "en")
	check(err == nil && saved == "it", "language action persists Italian")
	check(w.selectedID == "DEMO-10000" && original == w.evidence.ToPlainText(), "language preserves identity and evidence")
	w.advancedAction.Trigger()
	check(!w.evidence.IsVisible() && w.advanced.HasFocus() && !w.advanced.IsChecked(), "hiding details returns focus to advanced control")
	w.advancedAction.Trigger()
	check(w.evidence.IsVisible() && w.advanced.IsChecked(), "advanced menu and checkbox stay synchronized")
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
	checkAccessible("filtered")
	w.searchAction.Trigger()
	check(w.search.HasFocus() && w.search.SelectedText() == "DEMO-10000", "search action focuses and selects query")
	w.resultsAction.Trigger()
	check(w.table.HasFocus() && w.selectedID == "DEMO-10000", "results action preserves matching selection")
	w.search.SetText("missing")
	check(len(w.visible) == 0 && w.selectedID == "" && w.evidence.ToPlainText() == "" && !w.copy.IsEnabled(), "no stale evidence on empty results")
	w.resultsAction.Trigger()
	check(w.selectedID == "" && w.table.HasFocus(), "empty results focus does not select stale data")
	checkAccessible("no matches")
	w.search.SetText("")
	w.severity.SetCurrentIndex(0)
	w.resultsAction.Trigger()
	check(w.selectedID == "DEMO-00001", "results action selects first row when none selected")
	checkAccessible("restored")
	long := strings.Repeat(demo.Records()[0].Evidence(), 2048)
	w.evidence.SetPlainText(long)
	qt.QCoreApplication_ProcessEvents()
	check(w.evidence.ToPlainText() == long, "long evidence exact text roundtrip")
	fmt.Printf("long_evidence_bytes=%d\n", len(long))
	w.clear.Click()
	check(len(w.visible) == 0 && w.evidence.ToPlainText() == "", "clear dataset")
	checkAccessible("cleared")
	runtime.GC() // Exercise automatic lifetime management of returned Qt values.
	qt.QCoreApplication_ProcessEvents()
	if failures > 0 {
		return 1
	}
	return 0
}
