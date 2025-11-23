package interactionui

import (
	"cmp"
	"fmt"
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/interaction"
	"slices"
	"time"

	"github.com/docker/go-units"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Files struct {
	updater *Updater
	service *interaction.Service

	app          *tview.Application
	flex         *tview.Flex
	filesTable   *tview.Table
	confirmModal *tview.Modal
	downloadForm *tview.Form

	files []interaction.FileInfoExtended
}

func newFiles(app *tview.Application, updater *Updater, service *interaction.Service) *Files {
	return &Files{
		app:     app,
		updater: updater,
		service: service,

		flex: tview.NewFlex().SetDirection(tview.FlexRow),
	}
}

func (s *Files) buildFiles() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Files) setFlex() {
	s.flex.
		AddItem(s.filesTable, 0, 1, true)
}

func (s *Files) addItems() {
	s.filesTable = tview.NewTable().SetSeparator(tview.Borders.Vertical)
	s.filesTable.SetBorder(true).
		SetTitle("").
		SetTitleAlign(tview.AlignLeft)

	s.filesTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == tcell.ModAlt {
			switch event.Rune() {
			case 'U', 'u':
				// UPLOAD
				s.updater.switchPage(pages.Interaction.UPLOAD)
			case 'R', 'r':
				// REFRESH
				go s.setFilesList()
			case 'L', 'l':
				// LOGOUT
				s.service.Logout()
				s.updater.switchStage(pages.PAGE_L1_ENTRY)
			case 'D', 'd':
				// DOWNLOAD
				r, _ := s.filesTable.GetSelection()
				idx, ok := s.filesTable.GetCell(r, 0).Reference.(int)
				if !ok {
					break
				}
				s.fileModal(fmt.Sprintf("Want to DOWNLOAD: %s ?", s.files[idx].FileName), func() {
					s.downloadFile(idx)
				})
			case 'X', 'x':
				// DELETE
				r, _ := s.filesTable.GetSelection()
				idx, ok := s.filesTable.GetCell(r, 0).Reference.(int)
				if !ok {
					break
				}
				s.fileModal(fmt.Sprintf("Want to DELETE: %s ?", s.files[idx].FileName), func() {
					go s.deleteFile(idx)
				})
			}
		}
		return event
	})

	go s.setFilesList()
}

func (s *Files) setFilesList() {
	s.app.QueueUpdateDraw(func() {
		s.filesTable.SetTitle("Loading...")
		s.filesTable.Clear()
		headers := []string{"Sr.No.", "Name", "Size", "Created On"}
		for col, h := range headers {
			cell := tview.NewTableCell(h).
				SetSelectable(false).
				SetAttributes(tcell.AttrBold)

			if col > 1 {
				cell.SetAlign(tview.AlignCenter)
			}
			if col == 1 {
				cell.SetExpansion(1)
			} else {
				cell.SetExpansion(0)
			}
			s.filesTable.SetCell(0, col, cell)
		}
	})

	var err error
	clear(s.files)
	if s.files, err = s.service.GetFilesList(); err != nil {
		s.updater.setError(err.Error())
		return
	}

	slices.SortFunc(s.files, func(a, b interaction.FileInfoExtended) int {
		return cmp.Compare(b.CreatedAt.Unix(), a.CreatedAt.Unix())
	})

	s.app.QueueUpdateDraw(func() {
		for i, file := range s.files {
			s.filesTable.SetCell(i+2, 0, tview.NewTableCell(fmt.Sprintf(" %d  ", i+1)).SetExpansion(0).SetReference(i))
			s.filesTable.SetCell(i+2, 1, tview.NewTableCell(fmt.Sprintf(" %s  ", file.FileName)).SetExpansion(1).SetAlign(tview.AlignLeft))
			s.filesTable.SetCell(i+2, 2, tview.NewTableCell(fmt.Sprintf("  %s  ", units.HumanSize(float64(file.FileSize)))).SetAlign(tview.AlignCenter).SetExpansion(0))
			if time.Since(file.CreatedAt).Seconds() > 86400 {
				s.filesTable.SetCell(i+2, 3, tview.NewTableCell(fmt.Sprintf("  %s  ", file.CreatedAt.Format("2006-01-02 15:04:05"))).SetAlign(tview.AlignCenter).SetExpansion(0))
			} else {
				s.filesTable.SetCell(i+2, 3, tview.NewTableCell(fmt.Sprintf("  %s ago ", units.HumanDuration(time.Since(file.CreatedAt)))).SetAlign(tview.AlignCenter).SetExpansion(0))
			}

		}

		s.filesTable.SetFixed(2, 0)
		s.filesTable.SetTitle("Files")

		s.filesTable.
			SetSelectable(true, false).
			SetSelectedStyle(tcell.StyleDefault.
				Background(tcell.ColorWhite).
				Foreground(tcell.ColorBlack)).
			Select(2, 0)
	})
}

func (s *Files) fileModal(text string, yes func()) {
	s.confirmModal = tview.NewModal().
		SetText(text).
		AddButtons([]string{"Yes", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			s.flex.RemoveItem(s.confirmModal)
			s.flex.AddItem(s.filesTable, 0, 1, true)
			s.app.SetFocus(s.filesTable)
			if buttonIndex == 0 {
				yes()
			}
		})

	s.flex.RemoveItem(s.filesTable)
	s.flex.AddItem(s.confirmModal, 0, 1, true)
	s.app.SetFocus(s.confirmModal)
}

// DOWNLOAD
func (s *Files) downloadFile(idx int) {
	if s.files[idx].HasFilePassword {
		s.showDownloadForm(idx)
	} else {
		go s.processDownload(s.files[idx].FileUUID, []byte{})
	}
}

func (s *Files) showDownloadForm(idx int) {
	s.downloadForm = tview.NewForm().
		AddTextView("File to Download: ", s.files[idx].FileName, 0, 1, true, false).
		AddInputField("File Password:", "", 30, nil, nil).
		AddButton("Cancel", func() {
			s.flex.RemoveItem(s.downloadForm)
			s.flex.AddItem(s.filesTable, 0, 1, true)
			s.app.SetFocus(s.filesTable)
		}).
		AddButton("Confirm", func() {
			pwd := []byte(s.downloadForm.GetFormItemByLabel("File Password:").(*tview.InputField).GetText())
			go s.processDownload(s.files[idx].FileUUID, pwd)
			s.flex.RemoveItem(s.downloadForm)
			s.flex.AddItem(s.filesTable, 0, 1, true)
			s.app.SetFocus(s.filesTable)
		}).
		SetFocus(1).
		SetButtonsAlign(tview.AlignCenter)

	s.flex.RemoveItem(s.filesTable)
	s.flex.AddItem(s.downloadForm, 0, 1, true)
	s.app.SetFocus(s.downloadForm)
}

func (s *Files) processDownload(fileId string, pass []byte) {
	progress := func(done int64) {
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus(fmt.Sprintf("Downloading: %d%%", done))
		})
	}
	fileName, err := s.service.ManageDownload(fileId, pass, progress)
	clear(pass)
	if err != nil {
		s.app.QueueUpdateDraw(func() {
			s.updater.setError(err.Error())
		})
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus("")
		})
		go func() {
			time.Sleep(10 * time.Second)
			s.app.QueueUpdateDraw(func() {
				s.updater.setError("")
			})
		}()
		return
	} else {
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus(fmt.Sprintf("Downloaded: %s", fileName))
		})
		go func() {
			time.Sleep(5 * time.Second)
			s.app.QueueUpdateDraw(func() {
				s.updater.setStatus("")
			})
		}()
	}

}

// DELETE
func (s *Files) deleteFile(idx int) {
	name := s.files[idx].FileName
	s.app.QueueUpdateDraw(func() {
		s.updater.setStatus(fmt.Sprintf("Deleting: %s ...", name[:7]))
	})
	err := s.service.DeleteFile(s.files[idx].FileUUID)
	if err != nil {
		s.app.QueueUpdateDraw(func() {
			s.updater.setError(fmt.Sprintf("Failed to delete: %s ...", name[:7]))
		})
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus("")
		})
		go func() {
			time.Sleep(5 * time.Second)
			s.app.QueueUpdateDraw(func() {
				s.updater.setError("")
			})
		}()
		return
	}
	s.app.QueueUpdateDraw(func() {
		s.updater.setStatus(fmt.Sprintf("Deleted: %s ...", name[:7]))
	})
	go func() {
		time.Sleep(5 * time.Second)
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus("")
		})
	}()

	go s.setFilesList()
}
