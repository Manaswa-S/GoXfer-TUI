package entryui

import (
	"fmt"
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/entry"
	"goxfer/tui/utils"

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
	passFormat := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s",
		"This password is the master key to every file you save.",
		"It should follow the following rules:",
		"	1. 12-64 ASCII characters",
		"	2. Must include uppercase, lowercase, number, and symbol",
		"	3. No spaces, tabs, or control characters",
		"	4. No more than 3 identical characters in a row",
		"	5. Avoid common weak patterns (e.g., password, 123456, qwerty)")

	s.newForm = tview.NewForm().
		AddInputField("Name:", "", 40, nil, nil).
		AddInputField("Password:", "", 20, nil, nil).
		AddTextView("Note:", passFormat, 0, 7, true, false).
		AddCheckbox("Agree to TnC", false, nil).
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
	s.updater.setStatus("Opening ...", -1)
	defer s.updater.setStatus("", -1)

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
	s.updater.setStatus("Creating ...", -1)
	defer s.updater.setStatus("", -1)

	name := s.newForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
	pwd := []byte(s.newForm.GetFormItemByLabel("Password:").(*tview.InputField).GetText())
	chkbox := s.newForm.GetFormItemByLabel("Agree to TnC").(*tview.Checkbox)

	if recommendation := utils.VerifyPassFormat(pwd); recommendation != "" {
		s.updater.setError(fmt.Sprintf("Invalid password: %s", recommendation), -1)
		s.app.SetFocus(s.newForm)
		return
	}

	if !chkbox.IsChecked() {
		s.updater.setError("Please agree to the TnC to continue.", -1)
		s.app.SetFocus(s.newForm)
		return
	}

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
