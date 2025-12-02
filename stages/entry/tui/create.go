package entryui

import (
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/entry"

	"github.com/rivo/tview"
)

type Create struct {
	app     *tview.Application
	flex    *tview.Flex
	service *entry.Service
	updater *Updater

	newForm     *tview.Form
	createdForm *tview.Form
}

func newCreate(app *tview.Application, service *entry.Service, updater *Updater) *Create {
	return &Create{
		app:     app,
		flex:    tview.NewFlex().SetDirection(tview.FlexRow),
		service: service,
		updater: updater,
	}
}

func (s *Create) buildCreate() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Create) addItems() {
	s.newForm = tview.NewForm().
		AddInputField("Name:", "", 40, nil, nil).
		AddInputField("Password:", "", 20, nil, nil).
		// TODO:
		// AddTextView("Notes:", "This is just a demo.\nYou can enter whatever you wish.", 40, 2, true, false).
		// AddCheckbox("Agree to Terms and Conditions", false, nil).
		AddButton("Cancel", func() { s.updater.switchPage(pages.Entry.MENU) }).
		AddButton("Save", func() { go s.createBucket() })
	s.newForm.SetBorder(true).SetTitle("Create Bucket").SetTitleAlign(tview.AlignLeft)

	s.createdForm = tview.NewForm().
		AddTextView("Bucket ID:", "", 40, 2, true, false).
		AddButton("Return", func() { s.updater.switchPage(pages.Entry.MENU) }).
		AddButton("Enter Bucket", func() { go s.enterBucket() })
	s.createdForm.SetBorder(true).SetTitle("Bucket Created").SetTitleAlign(tview.AlignLeft)
}

func (s *Create) setFlex() {
	s.flex.AddItem(s.newForm, 0, 1, true)
}

func (s *Create) enterBucket() {
	s.app.QueueUpdateDraw(func() {
		s.app.SetFocus(nil)
	})
	s.updater.setStatus("Opening ...")
	defer s.updater.setStatus("")

	pwd := []byte(s.newForm.GetFormItemByLabel("Password:").(*tview.InputField).GetText())
	bucKey := []byte(s.createdForm.GetFormItemByLabel("Bucket ID:").(*tview.TextView).GetText(true))

	err := s.service.OpenBucket(pwd, bucKey)
	if err != nil {
		s.updater.setError(err.Error(), 0)
		s.updater.switchPage(pages.Entry.MENU)
		return
	}
	s.updater.switchStage(pages.PAGE_L1_INTERACTION)
}

func (s *Create) createBucket() {
	s.app.QueueUpdateDraw(func() {
		s.app.SetFocus(nil)
	})
	s.updater.setStatus("Creating ...")
	defer s.updater.setStatus("")

	name := s.newForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
	pwd := []byte(s.newForm.GetFormItemByLabel("Password:").(*tview.InputField).GetText())

	buc, err := s.service.CreateBucket(pwd, name)
	if err != nil {
		s.updater.setError(err.Error(), 0)
		s.updater.switchPage(pages.Entry.MENU)
		return
	}

	s.createdForm.GetFormItemByLabel("Bucket ID:").(*tview.TextView).SetText(buc.BucketKey)
	s.flex.RemoveItem(s.newForm)
	s.flex.AddItem(s.createdForm, 0, 1, true)
	s.app.SetFocus(s.createdForm)
}
