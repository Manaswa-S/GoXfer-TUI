package entryui

import (
	"github.com/rivo/tview"
)

type Settings struct {
	app     *tview.Application
	flex    *tview.Flex
	updater *Updater

	settingsMenu *tview.List
}

func newSettings(app *tview.Application, updater *Updater) *Settings {
	return &Settings{
		app:     app,
		flex:    tview.NewFlex(),
		updater: updater,
	}
}

func (s *Settings) buildSettings() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Settings) addItems() {
	s.settingsMenu = tview.NewList().
		AddItem("Open Existing Bucket", "Open any of your existing bucket", '1', nil)

	s.settingsMenu.SetSelectedFunc(func(idx int, mainText, secondaryText string, shortcut rune) {
		switch shortcut {
		case 'q':
			s.app.Stop()
		}
	})
}

func (s *Settings) setFlex() {
	s.flex.SetDirection(tview.FlexRow).
		AddItem(s.settingsMenu, (s.settingsMenu.GetItemCount() * 2), 1, true)
}
