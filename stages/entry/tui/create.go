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

	// Elements for the create new
	form    *tview.Form
	created *tview.Form
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
	// clear all stale elements
	s.flex.Clear()
	// add all new elements
	s.addItems()
	// set all elements
	s.setFlex()
}

func (s *Create) addItems() {
	s.form = tview.NewForm().
		AddInputField("Name:", "", 40, nil, nil).
		AddInputField("Password:", "", 20, nil, nil).
		AddTextView("Notes:", "This is just a demo.\nYou can enter whatever you wish.", 40, 2, true, false).
		AddCheckbox("Agree to Terms and Conditions", false, nil).
		AddButton("Save", s.createBucket).
		AddButton("Cancel", func() { s.updater.switchPage(pages.Entry.MENU) })
	s.form.SetBorder(true).SetTitle("Create Bucket").SetTitleAlign(tview.AlignLeft)

	s.created = tview.NewForm().
		AddTextView("Bucket ID:", "", 40, 2, true, false).
		AddButton("Enter Bucket", s.enterBucket).
		AddButton("Return", func() { s.updater.switchPage(pages.Entry.MENU) })
	s.created.SetBorder(true).SetTitle("Bucket Created").SetTitleAlign(tview.AlignLeft)
}

func (s *Create) setFlex() {
	s.flex.AddItem(s.form, 0, 1, true)
}

func (s *Create) enterBucket() {
	pwd := s.form.GetFormItemByLabel("Password:").(*tview.InputField).GetText()
	bucKey := s.created.GetFormItemByLabel("Bucket ID:").(*tview.TextView).GetText(true)

	_, errf := s.service.OpenBucket([]byte(pwd), []byte(bucKey))
	if errf != nil {
		s.updater.setError(errf.Message)
		s.updater.switchPage(pages.Entry.MENU)
		return
	}
	s.updater.switchStage(pages.PAGE_L1_INTERACTION)
}

func (c *Create) createBucket() {
	name := c.form.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
	pwd := c.form.GetFormItemByLabel("Password:").(*tview.InputField).GetText()

	buc, errf := c.service.CreateBucket([]byte(pwd), name)
	if errf != nil {
		c.updater.setError(errf.Message)
		c.updater.switchPage(pages.Entry.MENU)
		return
	}
	c.flex.RemoveItem(c.form)

	c.created.GetFormItemByLabel("Bucket ID:").(*tview.TextView).SetText(buc.BucketKey)
	c.flex.AddItem(c.created, 0, 1, true)
	c.app.SetFocus(c.created)
}
