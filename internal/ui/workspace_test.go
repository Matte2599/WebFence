package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/Matte2599/WebFence/internal/demo"
)

func TestWorkspaceOfflineFlow(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	w := a.NewWindow("WebFence test")
	t.Cleanup(w.Close)
	clipboard := test.NewClipboard()
	v := NewWorkspace(a.Preferences(), clipboard, "it-IT")
	w.SetContent(v.Content)
	w.Resize(fyne.NewSize(1120, 780))
	w.Show()
	if len(v.visible) != 0 || !v.copy.Disabled() || !v.clear.Disabled() {
		t.Fatal("initial state is not empty")
	}
	test.Tap(v.load)
	if len(v.visible) != demo.RowCount {
		t.Fatal("examples not loaded")
	}
	v.table.Select(widget.TableCellID{Row: 9999, Col: 0})
	before := v.evidence.Text
	test.Tap(v.copy)
	if clipboard.Content() != before || before != v.visible[9999].Evidence() {
		t.Fatal("clipboard changed evidence")
	}
	v.language.SetSelected("English")
	if v.selectedID != "DEMO-10000" || v.evidence.Text != before {
		t.Fatal("locale change altered selection or evidence")
	}
	if a.Preferences().String(languagePreference) != "en" {
		t.Fatal("language not persisted")
	}
	v.severitySelect.SetSelected("Informational")
	if len(v.visible) != 3334 || v.selectedID != "DEMO-10000" {
		t.Fatal("severity filter lost a retained selection")
	}
	v.language.SetSelected("Italiano")
	if v.severity != "info" || len(v.visible) != 3334 {
		t.Fatal("locale change altered severity filter")
	}
	v.search.SetText("no such example")
	if len(v.visible) != 0 || v.selectedID != "" || v.evidence.Text != "" || !v.copy.Disabled() {
		t.Fatal("stale evidence after empty filter")
	}
	v.search.SetText("demo-00001")
	test.Tap(v.next)
	if v.selectedID != "DEMO-00001" || !v.previous.Disabled() || !v.next.Disabled() {
		t.Fatal("single-row navigation failed")
	}
	test.Tap(v.clear)
	if len(v.rows) != 0 || v.evidence.Text != "" || !v.copy.Disabled() {
		t.Fatal("clear retained evidence")
	}
	// Restoring UI state uses the saved language, but does not restore demo records.
	reopened := NewWorkspace(a.Preferences(), w.Clipboard(), "en-US")
	if reopened.locale != "it" || len(reopened.rows) != 0 {
		t.Fatal("unexpected restored state")
	}
}

func TestKeyboardTableSelectionAndNavigation(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	w := a.NewWindow("WebFence keyboard test")
	t.Cleanup(w.Close)
	v := NewWorkspace(a.Preferences(), w.Clipboard(), "en")
	w.SetContent(v.Content)
	w.Resize(fyne.NewSize(1120, 780))
	w.Show()
	test.Tap(v.load)
	test.Tap(v.next)
	if v.selectedID != "DEMO-00001" {
		t.Fatal("first selection failed")
	}
	test.Tap(v.next)
	if v.selectedID != "DEMO-00002" {
		t.Fatal("next selection failed")
	}
	test.Tap(v.previous)
	if v.selectedID != "DEMO-00001" {
		t.Fatal("previous selection failed")
	}
	v.table.FocusGained()
	v.table.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	v.table.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if v.selectedID != "DEMO-00002" {
		t.Fatalf("keyboard selection: %s", v.selectedID)
	}
}
