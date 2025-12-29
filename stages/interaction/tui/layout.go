package interactionui

import (
	"fmt"
	"goxfer/tui/stages/interaction"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Layout struct {
	app     *tview.Application
	updater *Updater
	service *interaction.Service
	flex    *tview.Flex

	// Elements for the layout
	title     *tview.TextView
	subtitles *tview.TextView
	bucname   *tview.TextView

	infoFlex *tview.Grid

	backBtn     *tview.TextView
	errorText   *tview.TextView
	confirmText *tview.TextView
	statusText  *tview.TextView

	tips *tview.TextView
}

func newLayout(app *tview.Application, updater *Updater, service *interaction.Service) *Layout {
	return &Layout{
		app:     app,
		updater: updater,
		service: service,
		flex:    tview.NewFlex(),
	}
}

// setupLayout sets up all the layout components like title and footer
func (s *Layout) initLayout() {
	// Title
	s.title = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("GoXfer - Stateless Data Transfer").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorWhite)

	// Subtitles
	s.subtitles = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Stateless, Ephemeral, Zero-Overhead Transfers - Just the Payload.").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorLightGray)

	// Subtitles
	s.bucname = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorLightGray)

	// Shows the back button/text
	s.backBtn = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText("< Back (esc) ").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorLightGray)

	// Used to show error messages
	s.errorText = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorIndianRed)

	// Used to show confirmation messages
	s.confirmText = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetLabelWidth(0).
		SetText("").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorLightGray)

	// Used to show status messages
	s.statusText = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText("").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorForestGreen)

	s.tips = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("").
		SetDynamicColors(true).
		SetTextColor(tcell.ColorDarkGreen).SetWrap(true).SetWordWrap(true)

	s.infoFlex = tview.NewGrid().
		SetRows(1).
		SetColumns(0, 0, 0).
		AddItem(s.confirmText, 0, 0, 1, 1, 0, 0, false).
		AddItem(s.errorText, 0, 1, 1, 1, 0, 0, false).
		AddItem(s.statusText, 0, 2, 1, 1, 0, 0, false)

	go s.getBucketData()
}

func (s *Layout) getBucketData() {
	data, err := s.service.GetBucketData()
	if err != nil {
		s.updater.setError(err.Error(), 0)
		return
	}

	s.bucname.SetText(fmt.Sprintf("INTERACTION: %s", data.Name))
	s.app.QueueUpdateDraw(func() {})
}
