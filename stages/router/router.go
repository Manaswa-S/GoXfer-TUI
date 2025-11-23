package router

import (
	"goxfer/tui/cipher"
	"goxfer/tui/consts/errs"
	"goxfer/tui/consts/pages"
	"goxfer/tui/core"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/stages/entry"
	entryui "goxfer/tui/stages/entry/tui"
	"goxfer/tui/stages/interaction"
	interactionui "goxfer/tui/stages/interaction/tui"

	"github.com/rivo/tview"
)

type Stages struct {
	errChan  chan *errs.Errorf
	core     *core.Core
	cipher   cipher.Cipher
	settings *auxiliary.Settings

	app          *tview.Application
	pages        *tview.Pages
	constructors map[string]func()
	activePage   string

	entry       *entryui.AppTUI
	interaction *interactionui.AppTUI
}

func NewStages(app *tview.Application, core *core.Core, cipher cipher.Cipher,
	settings *auxiliary.Settings) *Stages {
	s := &Stages{
		core:         core,
		cipher:       cipher,
		settings:     settings,
		app:          app,
		constructors: make(map[string]func()),
	}
	s.registerConstructors()

	return s
}

func (s *Stages) InitStages() (*tview.Pages, error) {
	s.pages = tview.NewPages()
	s.SwitchTo(pages.PAGE_L1_ENTRY)

	return s.pages, nil
}

func (s *Stages) SwitchTo(name string) {
	if s.pages.HasPage(name) {
		s.pages.RemovePage(name)
	}
	constructor, ok := s.constructors[name]
	if !ok {
		panic("internal error: constructor not found")
	}
	constructor()
	s.pages.SwitchToPage(name)
	s.activePage = name
}

func (s *Stages) registerConstructors() {
	s.constructors[pages.PAGE_L1_ENTRY] = func() {
		credsMgr := entry.NewCredsManager()
		entryService := entry.NewService(s.core, credsMgr, s.settings)
		s.entry = entryui.NewAppTUI(s.app, entryService)
		s.entry.SetSwitchStage(s.SwitchTo)
		s.pages.AddPage(pages.PAGE_L1_ENTRY, s.entry.Flex, true, true)
	}

	s.constructors[pages.PAGE_L1_INTERACTION] = func() {
		interactionService := interaction.NewService(s.errChan, s.core, s.cipher, s.settings)
		s.interaction = interactionui.NewAppTUI(s.app, interactionService)
		s.interaction.SetSwitchStage(s.SwitchTo)
		s.pages.AddPage(pages.PAGE_L1_INTERACTION, s.interaction.Flex, true, false)
	}
}

// TODO: handle the errChan
