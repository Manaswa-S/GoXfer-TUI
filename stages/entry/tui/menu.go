package entryui

import (
	"goxfer/tui/consts/pages"

	"github.com/rivo/tview"
)

type Menu struct {
	app     *tview.Application
	flex    *tview.Flex
	updater *Updater

	entryMenu *tview.List
}

func newMenu(app *tview.Application, updater *Updater) *Menu {
	return &Menu{
		app:     app,
		flex:    tview.NewFlex(),
		updater: updater,
	}
}

func (s *Menu) buildMenu() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Menu) addItems() {
	s.entryMenu = tview.NewList().
		AddItem("Open Existing Bucket", "Open any of your existing bucket", '1', nil).
		AddItem("Create New Bucket", "Start by creating a new bucket", '2', nil).
		AddItem("Help", "Instructions and usage tips", '3', nil).
		AddItem("About", "App information and version", '4', nil).
		AddItem("Settings", "Configure preferences", '5', nil).
		AddItem("Quit", "Exit goXfer", 'q', nil)

	s.entryMenu.SetSelectedFunc(func(idx int, mainText, secondaryText string, shortcut rune) {
		switch shortcut {
		case '1':
			s.updater.switchPage(pages.Entry.OPEN)
		case '2':
			s.updater.switchPage(pages.Entry.CREATE)
		case '3':
		case '4':
		case '5':
			s.updater.switchPage(pages.Entry.SETTINGS)
		case 'q':
			s.app.Stop()
		}
	})
}

func (s *Menu) setFlex() {
	s.flex.SetDirection(tview.FlexRow).
		AddItem(s.entryMenu, (s.entryMenu.GetItemCount() * 2), 1, true)
}
