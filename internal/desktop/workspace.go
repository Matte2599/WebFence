// Package desktop implements the offline Qt Widgets workspace.
package desktop

import (
	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/i18n"
	"github.com/Matte2599/WebFence/internal/preferences"
	qt "github.com/mappu/miqt/qt6"
	"runtime"
)

type workspace struct {
	copyEvidence                  func(string)
	window                        *qt.QMainWindow
	model                         *qt.QAbstractTableModel
	table                         *qt.QTableView
	search                        *qt.QLineEdit
	language, severity            *qt.QComboBox
	advanced                      *qt.QCheckBox
	load, clear, copy             *qt.QPushButton
	intro, summary, count, notice *qt.QLabel
	searchLabel, severityLabel    *qt.QLabel
	evidence                      *qt.QPlainTextEdit

	fileMenu, viewMenu, languageMenu    *qt.QMenu
	loadAction, clearAction, quitAction *qt.QAction
	searchAction, resultsAction         *qt.QAction
	evidenceAction, advancedAction      *qt.QAction
	italianAction, englishAction        *qt.QAction

	preferencePath     string
	preferenceError    bool
	rows, visible      []demo.Record
	selectedID, locale string
	variants           map[string]*qt.QVariant
	emptyVariant       *qt.QVariant
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
	tools.SetContentsMargins(0, 0, 0, 0)
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
	filterLayout.SetContentsMargins(0, 0, 0, 0)
	w.search = qt.NewQLineEdit2()
	w.severity = qt.NewQComboBox2()
	w.severity.AddItems([]string{"", "", "", ""})
	w.searchLabel = qt.NewQLabel2()
	w.searchLabel.SetBuddy(w.search.QWidget)
	w.severityLabel = qt.NewQLabel2()
	w.severityLabel.SetBuddy(w.severity.QWidget)
	filterLayout.AddWidget(w.searchLabel.QWidget)
	filterLayout.AddWidget(w.search.QWidget)
	filterLayout.AddWidget(w.severityLabel.QWidget)
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
	w.table.SetTabKeyNavigation(false) // Arrows navigate rows; Tab leaves the table.
	for i, size := range []int{130, 170, 490, 130} {
		w.table.SetColumnWidth(i, size)
	}
	w.table.HorizontalHeader().SetStretchLastSection(true)
	// Keep two complete rows available when high scaling reduces usable height.
	headerHint := w.table.HorizontalHeader().SizeHint()
	scrollHint := w.table.HorizontalScrollBar().SizeHint()
	runtime.SetFinalizer(headerHint, nil)
	runtime.SetFinalizer(scrollHint, nil)
	w.table.SetMinimumHeight(headerHint.Height() + 2*w.table.VerticalHeader().DefaultSectionSize() + scrollHint.Height() + 2*w.table.FrameWidth())
	headerHint.Delete()
	scrollHint.Delete()
	details := qt.NewQWidget(nil)
	detailLayout := qt.NewQVBoxLayout(details)
	detailLayout.SetContentsMargins(0, 0, 0, 0)
	w.summary = qt.NewQLabel2()
	w.summary.SetTextFormat(qt.PlainText)
	w.summary.SetWordWrap(true)
	w.summary.SetTextInteractionFlags(qt.TextSelectableByMouse | qt.TextSelectableByKeyboard)
	w.summary.SetFocusPolicy(qt.StrongFocus)
	detailLayout.AddWidget(w.summary.QWidget)
	w.evidence = qt.NewQPlainTextEdit2()
	w.evidence.SetReadOnly(true)
	w.evidence.SetTabChangesFocus(true)
	detailLayout.AddWidget(w.evidence.QWidget)
	w.copy = qt.NewQPushButton2()
	detailLayout.AddWidget(w.copy.QWidget)
	split := qt.NewQSplitter3(qt.Vertical)
	split.AddWidget(w.table.QWidget)
	split.AddWidget(details)
	split.SetChildrenCollapsible(false)
	split.SetSizes([]int{400, 220})
	layout.AddWidget(split.QWidget)
	// Give spare height to results/evidence, not to the toolbar and filters.
	layout.SetStretchFactor(split.QWidget, 1)
	w.fileMenu = w.window.MenuBar().AddMenuWithTitle("")
	w.viewMenu = w.window.MenuBar().AddMenuWithTitle("")
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
	w.searchAction = w.viewMenu.AddActionWithText("")
	w.resultsAction = w.viewMenu.AddActionWithText("")
	w.evidenceAction = w.viewMenu.AddActionWithText("")
	w.viewMenu.AddSeparator()
	w.advancedAction = w.viewMenu.AddActionWithText("")
	w.advancedAction.SetCheckable(true)
	for _, pair := range []struct {
		action   *qt.QAction
		shortcut string
	}{
		{w.searchAction, "Ctrl+F"}, {w.resultsAction, "F6"},
		{w.evidenceAction, "Ctrl+Shift+E"}, {w.advancedAction, "Ctrl+Shift+D"},
	} {
		key := qt.NewQKeySequence2(pair.shortcut)
		pair.action.SetShortcut(key)
		key.Delete()
	}
	w.searchAction.OnTriggered(func() {
		w.search.SetFocusWithReason(qt.ShortcutFocusReason)
		w.search.SelectAll()
	})
	w.resultsAction.OnTriggered(func() {
		w.table.SetFocusWithReason(qt.ShortcutFocusReason)
		if w.selectedID == "" && len(w.visible) > 0 {
			w.table.SelectRow(0)
		}
	})
	w.evidenceAction.OnTriggered(func() {
		w.advanced.SetChecked(true)
		w.evidence.SetFocusWithReason(qt.ShortcutFocusReason)
	})
	w.advancedAction.OnToggled(func(checked bool) { w.advanced.SetChecked(checked) })
	w.advanced.OnToggled(func(checked bool) {
		// Move focus before hiding the widget that currently owns it.
		if !checked && (w.evidence.HasFocus() || w.copy.HasFocus()) {
			w.advanced.SetFocusWithReason(qt.OtherFocusReason)
		}
		w.advancedAction.SetChecked(checked)
		w.showEvidence()
	})
	chain := []*qt.QWidget{w.load.QWidget, w.clear.QWidget, w.advanced.QWidget,
		w.language.QWidget, w.search.QWidget, w.severity.QWidget, w.table.QWidget,
		w.summary.QWidget, w.evidence.QWidget, w.copy.QWidget}
	for i := 1; i < len(chain); i++ {
		qt.QWidget_SetTabOrder(chain[i-1], chain[i])
	}
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
	w.searchLabel.SetText(w.tr("search_label"))
	w.severityLabel.SetText(w.tr("severity"))
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
	w.viewMenu.SetTitle(w.tr("view"))
	w.searchAction.SetText(w.tr("focus_search"))
	w.resultsAction.SetText(w.tr("focus_results"))
	w.evidenceAction.SetText(w.tr("focus_evidence"))
	w.advancedAction.SetText(w.tr("advanced"))
	// Translation changes data, not row identity or model structure.
	w.model.HeaderDataChanged(qt.Horizontal, 0, 3)
	if len(w.visible) > 0 {
		parent := qt.NewQModelIndex()
		first := w.model.Index(0, 0, parent)
		last := w.model.Index(len(w.visible)-1, 3, parent)
		w.model.DataChanged2(first, last, []int{int(qt.DisplayRole), int(qt.AccessibleTextRole)})
		// Index() returns value wrappers already managed by MIQT finalizers.
		parent.Delete()
	}
	w.updateDetails()
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
	w.updateDetails()
}

func (w *workspace) updateDetails() {
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
		if w.evidence.ToPlainText() != r.Evidence() {
			w.evidence.SetPlainText(r.Evidence())
		}
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
