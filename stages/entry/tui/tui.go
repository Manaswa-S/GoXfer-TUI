package entryui

import (
	"fmt"
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/entry"
	"strings"
	"time"

	"github.com/rivo/tview"
)

type AppTUI struct {
	app       *tview.Application
	pages     *tview.Pages
	Flex      *tview.Flex
	service   *entry.Service
	updater   *Updater
	buildPage map[string]func()

	layout   *Layout
	menu     *Menu
	create   *Create
	open     *Open
	settings *Settings
}

// Updater for sub-level pages
type Updater struct {
	switchPage  func(string)
	setStatus   func(string, int)
	setError    func(string, int)
	setConfirm  func(string)
	switchStage func(string)
}

func NewAppTUI(app *tview.Application, service *entry.Service) *AppTUI {
	ui := &AppTUI{
		app:       app,
		pages:     tview.NewPages(),
		Flex:      tview.NewFlex(),
		service:   service,
		buildPage: make(map[string]func()),
	}
	ui.updater = &Updater{
		switchPage: ui.SwitchTo,
		setStatus:  ui.SetStatus,
		setError:   ui.SetError,
		setConfirm: ui.SetConfirm,
	}

	ui.layout = newLayout(ui.app)
	ui.layout.initLayout()

	ui.menu = newMenu(ui.app, ui.updater)
	ui.create = newCreate(ui.app, service, ui.updater)
	ui.open = newOpen(ui.app, ui.service, ui.updater)
	ui.settings = newSettings(ui.app, ui.updater)

	ui.buildPage[pages.Entry.MENU] = ui.menu.buildMenu
	ui.buildPage[pages.Entry.CREATE] = ui.create.buildCreate
	ui.buildPage[pages.Entry.OPEN] = ui.open.buildOpen
	ui.buildPage[pages.Entry.SETTINGS] = ui.settings.buildSettings

	ui.pages.AddPage(pages.Entry.MENU, ui.menu.flex, true, true)
	ui.pages.AddPage(pages.Entry.CREATE, ui.create.flex, true, false)
	ui.pages.AddPage(pages.Entry.OPEN, ui.open.flex, true, false)
	ui.pages.AddPage(pages.Entry.SETTINGS, ui.settings.flex, true, false)

	ui.Flex.SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.title, 1, 1, false).
		AddItem(ui.layout.subtitles, 1, 1, false).
		AddItem(tview.NewBox(), 2, 1, false).
		//
		AddItem(ui.pages, 0, 1, true).
		//
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.infoFlex, 1, 1, false).
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.tips, 2, 1, false)

	ui.SwitchTo(pages.Entry.MENU)

	return ui
}

func (ui *AppTUI) SwitchTo(name string) {
	build, exists := ui.buildPage[name]
	if !exists {
		panic("build not found")
	}
	build()
	ui.pages.SwitchToPage(name)
	ui.app.SetFocus(ui.pages.GetPage(name))
	ui.setTips(name)
}

func (ui *AppTUI) SetError(errTxt string, delay int) {
	f := func() {
		ui.layout.errorText.SetText(errTxt)
	}
	ui.app.QueueUpdateDraw(f)
	if errTxt == "" || delay == -1 {
		return
	}
	go func() {
		if delay < 1 {
			delay = 5
		}
		time.Sleep(time.Duration(delay) * time.Second)
		ui.app.QueueUpdateDraw(func() {
			ui.layout.errorText.SetText("")
		})
	}()
}

func (ui *AppTUI) SetStatus(txt string, delay int) {
	f := func() {
		ui.layout.statusText.SetText(txt)
	}
	ui.app.QueueUpdateDraw(f)
	if txt == "" || delay == -1 {
		return
	}
	// TODO: this is very bad
	go func() {
		if delay < 1 {
			delay = 5
		}
		time.Sleep(time.Duration(delay) * time.Second)
		ui.app.QueueUpdateDraw(func() {
			ui.layout.errorText.SetText("")
		})
	}()
}

func (ui *AppTUI) SetConfirm(txt string) {
	ui.layout.confirmText.SetText(txt)
}

func (ui *AppTUI) SetSwitchStage(fn func(string)) {
	ui.updater.switchStage = fn
}

func (ui *AppTUI) setTips(name string) {
	tips, exists := pages.TipsMap[pages.PAGE_L1_ENTRY][name]
	if !exists {
		tips = make([]*pages.Tip, 0)
	}

	text := "       "
	new := ""
	for _, tip := range tips {
		if tip.ShortCut == "" {
			new = strings.ReplaceAll(fmt.Sprintf("[yellow]~ [esc]%s", tip.Label), " ", "\u00A0")
		} else {
			new = strings.ReplaceAll(fmt.Sprintf("[yellow]<%s> [esc]%s", tip.ShortCut, tip.Label), " ", "\u00A0")
		}
		text += new + "       "
	}
	ui.layout.tips.SetText(text)
}
