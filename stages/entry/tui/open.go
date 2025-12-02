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

	openForm *tview.Form
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
	s.openForm = tview.NewForm().
		AddInputField("Bucket Key :", "", 30, nil, nil).
		AddInputField("Password :", "", 30, nil, nil).
		AddCheckbox("Remember? :", false, nil).
		AddButton("Cancel", func() { s.updater.switchPage(pages.Entry.MENU) }).
		AddButton("Open", func() {
			go s.openBucket()
		})
	s.openForm.SetBorder(true).SetTitle("Open Bucket").SetTitleAlign(tview.AlignLeft)

	creds := s.service.GetSavedCreds()
	if len(creds) > 0 {
		options := make([]string, len(creds))
		for i, cred := range creds {
			options[i] = string(cred.Key)
		}
		s.openForm.AddFormItem(tview.NewDropDown().SetLabel("Saved :").SetOptions(options, func(option string, optionIndex int) {
			s.openForm.GetFormItemByLabel("Bucket Key :").(*tview.InputField).SetText(string(creds[optionIndex].Key))
			s.openForm.GetFormItemByLabel("Password :").(*tview.InputField).SetText(string(creds[optionIndex].Pass))
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
		AddItem(c.openForm, 0, 1, true)
}

func (s *Open) openBucket() {
	s.app.QueueUpdateDraw(func() {
		s.app.SetFocus(nil)
	})
	s.updater.setStatus("Opening ...")
	defer s.updater.setStatus("")

	bucKey := []byte(s.openForm.GetFormItemByLabel("Bucket Key :").(*tview.InputField).GetText())
	pwd := []byte(s.openForm.GetFormItemByLabel("Password :").(*tview.InputField).GetText())
	remember := s.openForm.GetFormItemByLabel("Remember? :").(*tview.Checkbox).IsChecked()
	if remember {
		s.service.SaveCreds(&entry.Remember{
			Key:  bucKey,
			Pass: pwd,
		})
	}

	if err := s.service.OpenBucket(pwd, bucKey); err != nil {
		s.updater.setError(err.Error(), 0)
		s.updater.switchPage(pages.Entry.MENU)
		return
	}
	s.updater.switchStage(pages.PAGE_L1_INTERACTION)
}
