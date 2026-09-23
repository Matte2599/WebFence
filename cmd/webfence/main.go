// WebFence's M0 desktop prototype uses synthetic, offline data only.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"

	"github.com/Matte2599/WebFence/internal/ui"
)

func main() {
	a := app.NewWithID("io.github.Matte2599.WebFence")
	w := a.NewWindow("WebFence · M0")
	view := ui.NewWorkspace(a.Preferences(), w.Clipboard(), string(lang.SystemLocale()))
	w.SetContent(view.Content)
	w.Resize(fyne.NewSize(1120, 780))
	w.ShowAndRun()
}
