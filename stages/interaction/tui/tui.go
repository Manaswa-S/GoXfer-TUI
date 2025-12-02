package interactionui

import (
	"fmt"
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/interaction"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppTUI struct {
	app       *tview.Application
	pages     *tview.Pages
	Flex      *tview.Flex
	service   *interaction.Service
	updater   *Updater
	buildPage map[string]func()

	layout *Layout
	files  *Files
	upload *Upload
}

// Updater for sub-level pages
type Updater struct {
	switchPage  func(string)
	setStatus   func(string, int)
	setError    func(string, int)
	setConfirm  func(string, func(key tcell.Key))
	switchStage func(string)
}

func NewAppTUI(app *tview.Application, service *interaction.Service) *AppTUI {
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

	ui.files = newFiles(ui.app, ui.updater, ui.service)
	ui.upload = newUpload(ui.app, ui.updater, ui.service)

	ui.buildPage[pages.Interaction.FILES] = ui.files.buildFiles
	ui.buildPage[pages.Interaction.UPLOAD] = ui.upload.buildUpload

	ui.pages.AddPage(pages.Interaction.FILES, ui.files.flex, true, true)
	ui.pages.AddPage(pages.Interaction.UPLOAD, ui.upload.flex, true, false)

	ui.Flex.SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.title, 1, 1, false).
		AddItem(ui.layout.subtitles, 1, 1, false).
		AddItem(tview.NewBox(), 1, 1, false).
		//
		AddItem(ui.pages, 0, 1, true).
		//
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.infoFlex, 1, 1, false).
		AddItem(tview.NewBox(), 1, 1, false).
		AddItem(ui.layout.tips, 2, 1, false)

	ui.SwitchTo(pages.Interaction.FILES)

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

// delay = 0: default time out
// delay = -1: not time out
// delay = +ve: delay time out
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

func (ui *AppTUI) SetConfirm(txt string, proceed func(key tcell.Key)) {
	if txt == "" {
		ui.layout.confirmText.SetText("")
		return
	}

	txt += "(Esc to cancel, Enter to confirm)"
	ui.layout.confirmText.SetText(txt)
	ui.layout.confirmText.SetDoneFunc(proceed)
	ui.app.SetFocus(ui.layout.confirmText)
}

func (ui *AppTUI) SetSwitchStage(fn func(string)) {
	ui.updater.switchStage = fn
}

func (ui *AppTUI) setTips(name string) {
	tips, exists := pages.TipsMap[pages.PAGE_L1_INTERACTION][name]
	if !exists {
		panic("tips dont exist")
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
