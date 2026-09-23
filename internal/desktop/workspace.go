// Package desktop implements the offline Qt Widgets workspace.
package desktop

import (
	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/i18n"
	"github.com/Matte2599/WebFence/internal/preferences"
	qt "github.com/mappu/miqt/qt6"
)

type workspace struct {
	copyEvidence                        func(string)
	window                              *qt.QMainWindow
	model                               *qt.QAbstractTableModel
	table                               *qt.QTableView
	search                              *qt.QLineEdit
	language, severity                  *qt.QComboBox
	advanced                            *qt.QCheckBox
	load, clear, copy                   *qt.QPushButton
	intro, summary, count, notice       *qt.QLabel
	languageMenu                        *qt.QMenu
	italianAction, englishAction        *qt.QAction
	preferencePath                      string
	preferenceError                     bool
	evidence                            *qt.QPlainTextEdit
	fileMenu                            *qt.QMenu
	loadAction, clearAction, quitAction *qt.QAction
	rows, visible                       []demo.Record
	selectedID, locale                  string
	variants                            map[string]*qt.QVariant
	emptyVariant                        *qt.QVariant
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
func newWorkspace(locale, preferencePath string, preferenceError bool) *workspace {
	w := &workspace{window: qt.NewQMainWindow2(), locale: i18n.Normalize(locale), preferencePath: preferencePath, preferenceError: preferenceError, variants: make(map[string]*qt.QVariant), emptyVariant: qt.NewQVariant()}
	w.window.SetWindowTitle(w.tr("title"))
	w.window.Resize(1120, 780)
	body := qt.NewQWidget(nil)
	layout := qt.NewQVBoxLayout(body)
	w.window.SetCentralWidget(body)
	w.intro = qt.NewQLabel2()
	w.intro.SetWordWrap(true)
	layout.AddWidget(w.intro.QWidget)
	w.notice = qt.NewQLabel2()
	w.notice.SetWordWrap(true)
	layout.AddWidget(w.notice.QWidget)
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
	w.languageMenu = w.window.MenuBar().AddMenuWithTitle("")
	w.italianAction = w.languageMenu.AddActionWithText("Italiano")
	w.englishAction = w.languageMenu.AddActionWithText("English")
	for _, pair := range []struct {
		action   *qt.QAction
		shortcut string
	}{{w.italianAction, "Ctrl+1"}, {w.englishAction, "Ctrl+2"}} {
		key := qt.NewQKeySequence2(pair.shortcut)
		pair.action.SetShortcut(key)
		key.Delete()
	}
	w.italianAction.OnTriggered(func() { w.language.SetCurrentIndex(0) })
	w.englishAction.OnTriggered(func() { w.language.SetCurrentIndex(1) })
	w.fileMenu = w.window.MenuBar().AddMenuWithTitle("")
	w.loadAction = w.fileMenu.AddActionWithText("")
	key := qt.NewQKeySequence2("Ctrl+O")
	w.loadAction.SetShortcut(key)
	key.Delete()
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
	w.copyEvidence = func(text string) { qt.QGuiApplication_Clipboard().SetText(text) }
	w.copy.OnClicked(func() {
		if r, ok := w.selected(); ok {
			w.copyEvidence(r.Evidence())
		}
	})
	w.search.OnTextChanged(func(string) { w.filter() })
	w.severity.OnCurrentIndexChanged(func(int) { w.filter() })
	w.language.OnCurrentIndexChanged(func(i int) {
		if i < 0 || i > 1 {
			return
		}
		w.locale = []string{"it", "en"}[i]
		w.preferenceError = preferences.Save(w.preferencePath, w.locale) != nil
		w.translate()
	})
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
	w.window.SetWindowTitle(w.tr("title"))
	w.languageMenu.SetTitle(w.tr("language"))
	w.notice.SetText(w.tr("preference_error"))
	w.notice.SetVisible(w.preferenceError)
	w.intro.SetText(w.tr("intro"))
	w.load.SetText(w.tr("load"))
	w.clear.SetText(w.tr("clear"))
	w.copy.SetText(w.tr("copy"))
	w.advanced.SetText(w.tr("advanced"))
	w.search.SetPlaceholderText(w.tr("search"))
	w.search.SetAccessibleName(w.tr("search"))
	w.severity.SetAccessibleName(w.tr("severity"))
	w.language.SetAccessibleName(w.tr("language"))
	w.table.SetAccessibleName(w.tr("results"))
	w.evidence.SetAccessibleName(w.tr("evidence"))
	blocked := w.severity.BlockSignals(true)
	for i, key := range []string{"all", "info", "low", "medium"} {
		w.severity.SetItemText(i, w.tr(key))
	}
	w.severity.BlockSignals(blocked)
	w.fileMenu.SetTitle(w.tr("file"))
	w.loadAction.SetText(w.tr("load"))
	w.clearAction.SetText(w.tr("clear"))
	w.quitAction.SetText(w.tr("quit"))
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
		w.summary.SetText(w.tr("selected", r.ID) + "\n" + r.Path + "\n" + w.tr("example_summary"))
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
