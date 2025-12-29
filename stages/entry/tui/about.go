package entryui

// import (
// 	"github.com/rivo/tview"
// )

// type About struct {
// 	app     *tview.Application
// 	flex    *tview.Flex
// 	// updater *Updater

// 	about *tview.Form
// }

// func newAbout(app *tview.Application, updater *Updater) *About {
// 	return &About{
// 		app:     app,
// 		flex:    tview.NewFlex().SetDirection(tview.FlexRow),
// 		updater: updater,
// 	}
// }

// func (a *About) initAbout() {
// 	defer a.setFlex()

// 	a.about = tview.NewForm().
// 		AddTextView("About:", helpContent, 0, 0, true, true).
// 		AddButton("Back", func() {
// 			a.updater.switchPage(PAGE_MENU)
// 		})
// 	a.about.SetBorder(true).SetTitle("About").SetTitleAlign(tview.AlignLeft)
// }

// func (a *About) setFlex() {
// 	a.flex.AddItem(a.about, 0, 1, true)
// }
