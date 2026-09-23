// Qt/MIQT feasibility experiment. No scanner and no persistent project data.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/i18n"
	qt "github.com/mappu/miqt/qt6"
)

type workspace struct {
	window                              *qt.QMainWindow
	model                               *qt.QAbstractTableModel
	table                               *qt.QTableView
	search                              *qt.QLineEdit
	language, severity                  *qt.QComboBox
	advanced                            *qt.QCheckBox
	load, clear, copy                   *qt.QPushButton
	intro, summary, count               *qt.QLabel
	evidence                            *qt.QPlainTextEdit
	fileMenu                            *qt.QMenu
	loadAction, clearAction, quitAction *qt.QAction
	rows, visible                       []demo.Record
	selectedID, locale                  string
	variants                            map[string]*qt.QVariant
	emptyVariant                        *qt.QVariant
}

func main() {
	runtime.LockOSThread()
	qt.NewQApplication(os.Args)
	w := newWorkspace()
	w.window.Show()
	if len(os.Args) > 1 && os.Args[1] == "--self-test" {
		qt.QCoreApplication_ProcessEvents()
		code := selfTest(w)
		w.dispose()
		os.Exit(code)
	}
	qt.QApplication_Exec()
	w.dispose()
}

// MIQT 0.14 copies callback-returned QVariant values without freeing their
// pointers. Keep owned values alive until the model is destroyed; reuse them
// instead of allocating on every paint/accessibility query. This fixture cache
// is bounded by the finite dataset and the two catalogs, not by repaint count.
func (w *workspace) variant(value string) *qt.QVariant {
	if v, ok := w.variants[value]; ok {
		return v
	}
	v := qt.NewQVariant11(value)
	w.variants[value] = v
	return v
}
func (w *workspace) dispose() {
	w.window.Close()
	w.window.Delete()
	w.model.Delete()
	for _, v := range w.variants {
		v.Delete()
	}
	w.emptyVariant.Delete()
}

func (w *workspace) tr(key string, args ...any) string { return i18n.Text(w.locale, key, args...) }
func (w *workspace) extra(it, en string) string {
	if w.locale == "it" {
		return it
	}
	return en
}

func newWorkspace() *workspace {
	w := &workspace{window: qt.NewQMainWindow2(), locale: i18n.Normalize(qt.NewQLocale().Name()), variants: make(map[string]*qt.QVariant), emptyVariant: qt.NewQVariant()}
	w.window.SetWindowTitle("WebFence · Qt laboratory")
	w.window.Resize(1120, 780)
	body := qt.NewQWidget(nil)
	layout := qt.NewQVBoxLayout(body)
	w.window.SetCentralWidget(body)
	w.intro = qt.NewQLabel2()
	w.intro.SetWordWrap(true)
	layout.AddWidget(w.intro.QWidget)
	toolbar := qt.NewQWidget(nil)
	tools := qt.NewQHBoxLayout(toolbar)
	w.load = qt.NewQPushButton2()
	w.clear = qt.NewQPushButton2()
	w.advanced = qt.NewQCheckBox2()
	w.language = qt.NewQComboBox2()
	w.language.AddItems([]string{"Italiano", "English"})
	if w.locale == "en" {
		w.language.SetCurrentIndex(1)
	}
	for _, v := range []*qt.QWidget{w.load.QWidget, w.clear.QWidget, w.advanced.QWidget} {
		tools.AddWidget(v)
	}
	tools.AddStretch()
	tools.AddWidget(w.language.QWidget)
	layout.AddWidget(toolbar)
	filters := qt.NewQWidget(nil)
	filterLayout := qt.NewQHBoxLayout(filters)
	w.search = qt.NewQLineEdit2()
	w.severity = qt.NewQComboBox2()
	w.severity.AddItems([]string{"", "", "", ""})
	filterLayout.AddWidget(w.search.QWidget)
	filterLayout.AddWidget(w.severity.QWidget)
	layout.AddWidget(filters)
	w.count = qt.NewQLabel2()
	layout.AddWidget(w.count.QWidget)
	w.model = qt.NewQAbstractTableModel()
	w.model.OnRowCount(func(parent *qt.QModelIndex) int {
		if parent.IsValid() {
			return 0
		}
		return len(w.visible)
	})
	w.model.OnColumnCount(func(parent *qt.QModelIndex) int {
		if parent.IsValid() {
			return 0
		}
		return 4
	})
	w.model.OnData(func(idx *qt.QModelIndex, role int) *qt.QVariant {
		if !idx.IsValid() || idx.Row() < 0 || idx.Row() >= len(w.visible) || idx.Column() < 0 || idx.Column() > 3 {
			return w.emptyVariant
		}
		if role != int(qt.DisplayRole) && role != int(qt.AccessibleTextRole) {
			return w.emptyVariant
		}
		r := w.visible[idx.Row()]
		values := []string{r.ID, w.tr(r.Severity), r.Path, w.tr("synthetic")}
		return w.variant(values[idx.Column()])
	})
	w.model.OnHeaderData(func(super func(int, qt.Orientation, int) *qt.QVariant, section int, orientation qt.Orientation, role int) *qt.QVariant {
		if orientation == qt.Horizontal && role == int(qt.DisplayRole) && section >= 0 && section < 4 {
			return w.variant(w.tr([]string{"id", "severity", "url", "status"}[section]))
		}
		return w.emptyVariant
	})
	w.table = qt.NewQTableView2()
	w.table.SetModel(w.model.QAbstractItemModel)
	w.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	w.table.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	w.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	w.table.SetAlternatingRowColors(true)
	for i, size := range []int{130, 170, 490, 130} {
		w.table.SetColumnWidth(i, size)
	}
	w.table.HorizontalHeader().SetStretchLastSection(true)
	details := qt.NewQWidget(nil)
	detailLayout := qt.NewQVBoxLayout(details)
	w.summary = qt.NewQLabel2()
	w.summary.SetTextFormat(qt.PlainText)
	w.summary.SetWordWrap(true)
	w.summary.SetTextInteractionFlags(qt.TextSelectableByMouse | qt.TextSelectableByKeyboard)
	detailLayout.AddWidget(w.summary.QWidget)
	w.evidence = qt.NewQPlainTextEdit2()
	w.evidence.SetReadOnly(true)
	detailLayout.AddWidget(w.evidence.QWidget)
	w.copy = qt.NewQPushButton2()
	detailLayout.AddWidget(w.copy.QWidget)
	split := qt.NewQSplitter3(qt.Vertical)
	split.AddWidget(w.table.QWidget)
	split.AddWidget(details)
	split.SetSizes([]int{400, 220})
	layout.AddWidget(split.QWidget)
	w.fileMenu = w.window.MenuBar().AddMenuWithTitle("")
	w.loadAction = w.fileMenu.AddActionWithText("")
	w.loadAction.SetShortcut(qt.NewQKeySequence2("Ctrl+O"))
	w.clearAction = w.fileMenu.AddActionWithText("")
	w.fileMenu.AddSeparator()
	w.quitAction = w.fileMenu.AddActionWithText("")
	w.quitAction.SetShortcutsWithShortcuts(qt.QKeySequence__Quit)
	load := func() { w.rows = demo.Records(); w.filter() }
	clear := func() { w.rows = nil; w.filter() }
	w.load.OnClicked(load)
	w.loadAction.OnTriggered(load)
	w.clear.OnClicked(clear)
	w.clearAction.OnTriggered(clear)
	w.quitAction.OnTriggered(qt.QCoreApplication_Quit)
	w.copy.OnClicked(func() {
		if r, ok := w.selected(); ok {
			qt.QGuiApplication_Clipboard().SetText(r.Evidence())
		}
	})
	w.search.OnTextChanged(func(string) { w.filter() })
	w.severity.OnCurrentIndexChanged(func(int) { w.filter() })
	w.language.OnCurrentIndexChanged(func(i int) { w.locale = []string{"it", "en"}[i]; w.translate() })
	w.advanced.OnToggled(func(bool) { w.showEvidence() })
	w.table.SelectionModel().OnCurrentRowChanged(func(current, previous *qt.QModelIndex) {
		if current.IsValid() && current.Row() >= 0 && current.Row() < len(w.visible) {
			w.selectedID = w.visible[current.Row()].ID
		} else {
			w.selectedID = ""
		}
		w.showEvidence()
	})
	w.translate()
	return w
}

func (w *workspace) translate() {
	w.intro.SetText(w.tr("intro"))
	w.load.SetText(w.tr("load"))
	w.clear.SetText(w.tr("clear"))
	w.copy.SetText(w.tr("copy"))
	w.advanced.SetText(w.extra("Dettagli avanzati", "Advanced details"))
	w.search.SetPlaceholderText(w.tr("search"))
	w.search.SetAccessibleName(w.tr("search"))
	w.severity.SetAccessibleName(w.tr("severity"))
	w.language.SetAccessibleName(w.tr("language"))
	w.table.SetAccessibleName(w.extra("Risultati sintetici", "Synthetic results"))
	w.evidence.SetAccessibleName(w.tr("evidence"))
	blocked := w.severity.BlockSignals(true)
	for i, key := range []string{"all", "info", "low", "medium"} {
		w.severity.SetItemText(i, w.tr(key))
	}
	w.severity.BlockSignals(blocked)
	w.fileMenu.SetTitle(w.extra("File", "File"))
	w.loadAction.SetText(w.tr("load"))
	w.clearAction.SetText(w.tr("clear"))
	w.quitAction.SetText(w.extra("Esci", "Quit"))
	w.filter()
}

func (w *workspace) filter() {
	id := w.selectedID
	w.model.BeginResetModel()
	w.visible = demo.Filter(w.rows, w.search.Text(), []string{"", "info", "low", "medium"}[w.severity.CurrentIndex()])
	w.model.EndResetModel()
	w.selectedID = ""
	for i, r := range w.visible {
		if r.ID == id {
			w.table.SelectRow(i)
			w.selectedID = id
			break
		}
	}
	w.count.SetText(w.tr("count", len(w.visible), len(w.rows)))
	w.clear.SetEnabled(len(w.rows) > 0)
	w.clearAction.SetEnabled(len(w.rows) > 0)
	w.showEvidence()
}
func (w *workspace) selected() (demo.Record, bool) {
	for _, r := range w.visible {
		if r.ID == w.selectedID {
			return r, true
		}
	}
	return demo.Record{}, false
}
func (w *workspace) showEvidence() {
	r, ok := w.selected()
	if ok {
		w.summary.SetText(w.tr("selected", r.ID) + "\n" + r.Path + "\n" + w.extra("Esempio per valutare la GUI. Nessuna vulnerabilità verificata.", "GUI evaluation example. No verified vulnerability."))
		w.evidence.SetPlainText(r.Evidence())
	} else {
		key := "select"
		if len(w.visible) == 0 {
			key = "no_matches"
		}
		if len(w.rows) == 0 {
			key = "empty"
		}
		w.summary.SetText(w.tr(key))
		w.evidence.Clear()
	}
	w.evidence.SetVisible(w.advanced.IsChecked())
	w.copy.SetVisible(w.advanced.IsChecked())
	w.copy.SetEnabled(ok)
}

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
	w.language.SetCurrentIndex(1)
	w.language.SetCurrentIndex(0)
	check(w.selectedID == "DEMO-10000" && original == w.evidence.ToPlainText(), "language preserves identity and evidence")
	w.advanced.SetChecked(true)
	check(w.evidence.IsVisible(), "advanced detail visible")
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
