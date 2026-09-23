package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Matte2599/WebFence/internal/demo"
	"github.com/Matte2599/WebFence/internal/i18n"
)

const languagePreference = "ui.language"

// Workspace is the M0 offline GUI experiment. All callbacks run on Fyne's UI
// goroutine. No scanner, target networking or persistent findings are attached.
type Workspace struct {
	Content fyne.CanvasObject

	locale      string
	preferences fyne.Preferences
	clipboard   fyne.Clipboard
	rows        []demo.Record
	visible     []demo.Record
	selectedID  string
	severity    string

	table          *widget.Table
	search         *widget.Entry
	language       *widget.Select
	severitySelect *widget.Select

	load     *widget.Button
	clear    *widget.Button
	copy     *widget.Button
	previous *widget.Button
	next     *widget.Button

	title         *widget.Label
	intro         *widget.Label
	languageLabel *widget.Label
	severityLabel *widget.Label
	count         *widget.Label
	empty         *widget.Label
	evidenceTitle *widget.Label
	selected      *widget.Label
	evidence      *widget.Label
	footer        *widget.Label
}

func NewWorkspace(preferences fyne.Preferences, clipboard fyne.Clipboard, systemLocale string) *Workspace {
	v := &Workspace{
		locale:      i18n.Normalize(preferences.StringWithFallback(languagePreference, systemLocale)),
		preferences: preferences,
		clipboard:   clipboard,
	}
	v.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.title.SizeName = theme.SizeNameHeadingText
	v.intro = widget.NewLabel("")
	v.intro.Wrapping = fyne.TextWrapWord
	v.languageLabel = widget.NewLabel("")
	v.severityLabel = widget.NewLabel("")
	v.count = widget.NewLabel("")
	v.empty = widget.NewLabel("")
	v.empty.Wrapping = fyne.TextWrapWord
	v.evidenceTitle = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.selected = widget.NewLabel("")
	v.selected.Wrapping = fyne.TextWrapWord
	v.evidence = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	v.evidence.Selectable = true
	v.evidence.Wrapping = fyne.TextWrapBreak
	v.footer = widget.NewLabel("")
	v.footer.Wrapping = fyne.TextWrapWord
	v.search = widget.NewEntry()
	v.search.OnChanged = func(string) { v.filter() }
	v.language = widget.NewSelect([]string{"Italiano", "English"}, func(value string) {
		v.locale = "en"
		if value == "Italiano" {
			v.locale = "it"
		}
		v.preferences.SetString(languagePreference, v.locale)
		v.translate()
	})
	v.severitySelect = widget.NewSelect(nil, func(value string) {
		v.severity = ""
		for _, code := range []string{"info", "low", "medium"} {
			if value == v.text(code) {
				v.severity = code
			}
		}
		v.filter()
	})
	v.load = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		v.rows = demo.Records()
		v.filter()
	})
	v.clear = widget.NewButton("", func() {
		v.rows = nil
		v.filter()
	})
	v.copy = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if row, ok := v.selectedRecord(); ok {
			v.clipboard.SetContent(row.Evidence())
		}
	})
	v.previous = widget.NewButton("", func() { v.move(-1) })
	v.next = widget.NewButton("", func() { v.move(1) })
	v.table = widget.NewTable(
		func() (int, int) { return len(v.visible), 4 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, object fyne.CanvasObject) {
			row := v.visible[id.Row]
			values := [...]string{row.ID, v.text(row.Severity), row.Path, v.text("synthetic")}
			object.(*widget.Label).SetText(values[id.Col])
		},
	)
	v.table.ShowHeaderRow = true
	v.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	v.table.UpdateHeader = func(id widget.TableCellID, object fyne.CanvasObject) {
		keys := [...]string{"id", "severity", "url", "status"}
		if id.Col >= 0 {
			object.(*widget.Label).SetText(v.text(keys[id.Col]))
		}
	}
	for col, width := range []float32{140, 180, 470, 130} {
		v.table.SetColumnWidth(col, width)
	}
	v.table.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Row < len(v.visible) {
			v.selectedID = v.visible[id.Row].ID
			v.showEvidence()
		}
	}

	toolbar := container.NewBorder(nil, nil,
		container.NewHBox(v.load, v.clear),
		container.NewHBox(v.languageLabel, v.language), nil)
	filters := container.NewBorder(nil, nil, nil,
		container.NewHBox(v.severityLabel, v.severitySelect), v.search)
	results := container.NewBorder(container.NewVBox(filters, v.count), nil, nil, nil,
		container.NewStack(v.table, container.NewCenter(v.empty)))
	details := container.NewBorder(
		container.NewVBox(v.evidenceTitle, v.selected),
		container.NewHBox(v.previous, v.next, v.copy), nil, nil,
		container.NewVScroll(v.evidence))
	split := container.NewVSplit(results, details)
	split.Offset = 0.56
	v.Content = container.NewPadded(container.NewBorder(
		container.NewVBox(v.title, v.intro, toolbar, widget.NewSeparator()),
		v.footer, nil, nil, split))
	v.translate()
	return v
}

func (v *Workspace) text(key string, args ...any) string { return i18n.Text(v.locale, key, args...) }

func (v *Workspace) translate() {
	for label, key := range map[*widget.Label]string{
		v.title: "title", v.intro: "intro", v.languageLabel: "language",
		v.severityLabel: "severity", v.evidenceTitle: "evidence", v.footer: "footer",
	} {
		label.SetText(v.text(key))
	}
	for button, key := range map[*widget.Button]string{
		v.load: "load", v.clear: "clear", v.copy: "copy", v.previous: "previous", v.next: "next",
	} {
		button.SetText(v.text(key))
	}
	v.search.SetPlaceHolder(v.text("search"))
	// Re-label controls without firing callbacks that could reset canonical filters.
	v.language.Selected = "English"
	if v.locale == "it" {
		v.language.Selected = "Italiano"
	}
	v.language.Refresh()
	v.severitySelect.Options = []string{v.text("all"), v.text("info"), v.text("low"), v.text("medium")}
	v.severitySelect.Selected = v.text("all")
	if v.severity != "" {
		v.severitySelect.Selected = v.text(v.severity)
	}
	v.severitySelect.Refresh()
	v.filter()
}

func (v *Workspace) filter() {
	v.visible = demo.Filter(v.rows, v.search.Text, v.severity)
	// Clear table coordinates before replacing the selection by stable ID.
	v.table.UnselectAll()
	v.table.Refresh()
	selected := -1
	for i, row := range v.visible {
		if row.ID == v.selectedID {
			selected = i
			break
		}
	}
	if selected >= 0 {
		v.table.Select(widget.TableCellID{Row: selected, Col: 0})
	} else {
		v.selectedID = ""
	}
	v.count.SetText(v.text("count", len(v.visible), len(v.rows)))
	if len(v.visible) == 0 {
		key := "no_matches"
		if len(v.rows) == 0 {
			key = "empty"
		}
		v.empty.SetText(v.text(key))
		v.empty.Show()
		v.table.Hide()
	} else {
		v.empty.Hide()
		v.table.Show()
	}
	if len(v.rows) == 0 {
		v.clear.Disable()
	} else {
		v.clear.Enable()
	}
	v.showEvidence()
}

func (v *Workspace) selectedRecord() (demo.Record, bool) {
	for _, row := range v.visible {
		if row.ID == v.selectedID {
			return row, true
		}
	}
	return demo.Record{}, false
}

func (v *Workspace) showEvidence() {
	v.previous.Disable()
	v.next.Disable()
	if row, ok := v.selectedRecord(); ok {
		v.selected.SetText(v.text("selected", row.ID))
		v.evidence.SetText(row.Evidence())
		v.copy.Enable()
		for i, item := range v.visible {
			if item.ID != row.ID {
				continue
			}
			if i > 0 {
				v.previous.Enable()
			}
			if i+1 < len(v.visible) {
				v.next.Enable()
			}
			break
		}
	} else {
		v.selected.SetText(v.text("select"))
		v.evidence.SetText("")
		v.copy.Disable()
		if len(v.visible) > 0 {
			v.next.Enable()
		}
	}
}

func (v *Workspace) move(delta int) {
	i := -1
	for n, row := range v.visible {
		if row.ID == v.selectedID {
			i = n
			break
		}
	}
	next := i + delta
	if next >= 0 && next < len(v.visible) {
		v.table.Select(widget.TableCellID{Row: next, Col: 0})
		v.table.ScrollTo(widget.TableCellID{Row: next, Col: 0})
	}
}
