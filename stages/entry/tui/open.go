package entryui

import (
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/entry"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Open struct {
	app     *tview.Application // the app
	flex    *tview.Flex        // the root flex canvas
	service *entry.Service
	updater *Updater

	// Elements for the search key
	form *tview.Form
}

func newOpen(app *tview.Application, service *entry.Service, updater *Updater) *Open {
	return &Open{
		app:     app,
		flex:    tview.NewFlex().SetDirection(tview.FlexRow),
		service: service,
		updater: updater,
	}
}

func (s *Open) buildOpen() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Open) addItems() {
	s.form = tview.NewForm().
		AddInputField("Bucket Key :", "", 30, nil, nil).
		AddInputField("Password :", "", 30, nil, nil).
		AddCheckbox("Remember? :", false, nil).
		AddButton("Cancel", func() { s.updater.switchPage(pages.Entry.MENU) }).
		AddButton("Open", func() {
			s.app.SetFocus(nil)
			s.updater.setStatus("Opening ...")
			go s.openBucket()
		})
	s.form.SetBorder(true).SetTitle("Open Bucket").SetTitleAlign(tview.AlignLeft)

	creds := s.service.GetSavedCreds()
	if len(creds) > 0 {
		options := make([]string, len(creds))
		for i, cred := range creds {
			options[i] = cred.Key
		}
		s.form.AddFormItem(tview.NewDropDown().SetLabel("Saved :").SetOptions(options, func(option string, optionIndex int) {
			s.form.GetFormItemByLabel("Bucket Key :").(*tview.InputField).SetText(creds[optionIndex].Key)
			s.form.GetFormItemByLabel("Password :").(*tview.InputField).SetText(creds[optionIndex].Pass)

			s.service.UsedCreds(creds[optionIndex].Key)
		}).SetCurrentOption(0))
	}

	s.setShortcuts()
}

func (s *Open) setShortcuts() {
	s.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == tcell.ModAlt {
			switch event.Rune() {
			case 'O', 'o':
				s.app.SetFocus(nil)
				s.updater.setStatus("Opening ...")
				go s.openBucket()
			case 'C', 'c':
				s.updater.switchPage(pages.Entry.MENU)
			}
		}
		return event
	})
}

func (c *Open) setFlex() {
	c.flex.
		AddItem(c.form, 0, 1, true)
}

func (s *Open) openBucket() {
	s.app.SetFocus(nil)
	s.updater.setStatus("Opening ...")
	bucKey := []byte(s.form.GetFormItemByLabel("Bucket Key :").(*tview.InputField).GetText())
	pwd := []byte(s.form.GetFormItemByLabel("Password :").(*tview.InputField).GetText())
	remember := s.form.GetFormItemByLabel("Remember? :").(*tview.Checkbox).IsChecked()
	if remember {
		// TODO: needs to be []byte()
		s.service.SaveCreds(entry.Remember{
			Key:  string(bucKey),
			Pass: string(pwd),
		})
	}

	_, errf := s.service.OpenBucket(pwd, bucKey)
	if errf != nil {
		panic(errf.Message + errf.Error.Error())
		s.updater.setError(errf.Message)
		s.updater.setStatus(errf.Error.Error())
		s.updater.switchPage(pages.Entry.MENU)
		return
	}
	s.updater.switchStage(pages.PAGE_L1_INTERACTION)
}
