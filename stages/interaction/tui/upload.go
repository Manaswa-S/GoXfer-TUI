package interactionui

import (
	"context"
	"fmt"
	"goxfer/tui/consts/pages"
	"goxfer/tui/stages/interaction"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/docker/go-units"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

type SearchResult struct {
	path  string
	entry fs.DirEntry
}

type Upload struct {
	app     *tview.Application
	flex    *tview.Flex
	updater *Updater
	service *interaction.Service

	fileTree      *tview.TreeView
	searchInp     *tview.InputField
	searchTable   *tview.Table
	confirmUpload *tview.Form

	showHidden    bool
	searchDone    context.CancelFunc
	searchResults chan SearchResult
}

func newUpload(app *tview.Application, updater *Updater, service *interaction.Service) *Upload {
	return &Upload{
		app:     app,
		flex:    tview.NewFlex().SetDirection(tview.FlexRow),
		updater: updater,
		service: service,

		searchResults: make(chan SearchResult, 100),
	}
}

func (s *Upload) buildUpload() {
	s.flex.Clear()
	s.addItems()
	s.setFlex()
}

func (s *Upload) addItems() {

	s.searchInp = tview.NewInputField().
		SetPlaceholder(" Search ").
		SetPlaceholderTextColor(tcell.ColorFloralWhite).
		SetFieldTextColor(tcell.ColorWhite)

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	rootDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	root := tview.NewTreeNode(rootDir).
		SetColor(tcell.ColorRed).SetReference(rootDir)
	s.fileTree = tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	s.searchInp.SetLabel(fmt.Sprintf(" Current: %s/ ", rootDir))
	s.fileTree.SetChangedFunc(func(node *tview.TreeNode) {
		ref, ok := node.GetReference().(string)
		if !ok {
			panic("tree node reference is not a string")
		}
		stat, err := os.Stat(ref)
		if err != nil {
			panic(err)
		}

		if stat.IsDir() {
			ref = fmt.Sprintf(" Current: %s/ ", ref)
		} else {
			ref = fmt.Sprintf(" Current: %s ", ref)
		}
		s.searchInp.SetLabel(ref)
	})

	s.addToTree(root, rootDir)
	s.fileTree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref, ok := node.GetReference().(string)
		if !ok {
			panic("tree node reference is not a string")
		}
		if len(node.GetChildren()) == 0 {
			s.addToTree(node, ref)
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
	})
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

	s.searchTable = tview.NewTable().SetSeparator(tview.Borders.Vertical)
	s.searchTable.SetBorder(true).
		SetTitle("Files").
		SetTitleAlign(tview.AlignLeft)
	s.searchTable.SetSelectedFunc(func(row, column int) {
		ref, ok := s.searchTable.GetCell(row, column).GetReference().(string)
		if !ok {
			panic("cell reference not string")
		}
		s.searchInp.SetText("")
		s.uploadFile(ref)
	})

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

	s.searchTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == 0 {
			query := s.searchInp.GetText()
			if event.Key() == tcell.KeyBackspace2 {
				if len(query) > 1 {
					query = query[0 : len(query)-1]
				} else {
					query = ""
				}
				s.processSearch(query)
			} else {
				r := event.Rune()
				if r != 0 && unicode.IsPrint(r) {
					query += string(r)
					s.processSearch(query)
				}
			}
			if query == "" {
				s.flex.RemoveItem(s.searchTable)
				s.flex.AddItem(s.fileTree, 0, 1, true)
				s.app.SetFocus(s.fileTree)
				s.searchInp.SetText("")
			}
		}
		return event
	})

	s.fileTree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == 0 {
			r := event.Rune()
			if r != 0 && unicode.IsPrint(r) {
				query := s.searchInp.GetText()
				query += string(r)
				s.processSearch(query)
			}
		} else {
			if event.Modifiers() == tcell.ModAlt {
				r := event.Rune()
				switch r {
				case 'H', 'h':
					s.showHidden = !s.showHidden
					s.fileTree.GetRoot().ClearChildren()
					s.addToTree(root, rootDir)
				case 'B', 'b':
					s.updater.switchPage(pages.Interaction.FILES)
				}
			}
		}

		if event.Key() == tcell.KeyESC {
			c := s.fileTree.GetRoot().GetChildren()
			for _, child := range c {
				child.CollapseAll()
			}
		}

		return event
	})

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

	s.confirmUpload = tview.NewForm()
	s.confirmUpload.SetTitle("Confirm Upload").SetTitleAlign(tview.AlignCenter)
}

func (s *Upload) setFlex() {
	s.flex.
		AddItem(s.searchInp, 2, 1, false).
		AddItem(s.fileTree, 0, 1, true)
}

// TREE FUNCTIONALITY
func (s *Upload) addToTree(target *tview.TreeNode, path string) {
	info, err := os.Stat(path)
	if err != nil {
		panic(err)
	}

	if !info.IsDir() {
		s.uploadFile(path)
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	if len(entries) == 0 {
		emptyNode := tview.NewTreeNode("empty").SetSelectable(false)
		target.AddChild(emptyNode)
		go func(emptyNode *tview.TreeNode) {
			time.AfterFunc(5*time.Second, func() {
				target.CollapseAll()
			})
		}(emptyNode)
	}

	for _, entry := range entries {
		name := entry.Name()
		if !s.showHidden {
			if strings.HasPrefix(name, ".") {
				continue
			}
		}

		node := tview.NewTreeNode(name).
			SetReference(filepath.Join(path, name))
		if entry.IsDir() {
			node.SetColor(tcell.ColorGreen)
		}
		target.AddChild(node)
	}
}

// SEARCH FUNCTIONALITY

func (s *Upload) setSearchTableHeaders() {
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
		s.searchTable.SetCell(0, col, cell)
		s.searchTable.SetCell(1, col, tview.NewTableCell("").SetSelectable(false))
	}
}

func (s *Upload) processSearch(query string) {
	// cancel any previous ongoing search
	if s.searchDone != nil {
		s.searchDone()
	}

	if query == "" {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.searchDone = cancel

	// set UI
	s.searchInp.SetText(query)
	s.flex.RemoveItem(s.flex.GetItem(1))
	s.searchTable.Clear()
	s.setSearchTableHeaders()
	s.flex.AddItem(s.searchTable, 0, 1, true)
	s.app.SetFocus(s.searchTable)
	s.searchTable.
		SetFixed(2, 0).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorWhite).
			Foreground(tcell.ColorBlack)).
		Select(2, 0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-s.searchResults:
				s.app.QueueUpdateDraw(func() {
					s.appendSearchResult(result)
				})
			}
		}
	}()

	currNode := s.fileTree.GetCurrentNode()
	if currNode == nil {
		currNode = s.fileTree.GetRoot()
	}
	currRoot := currNode.GetReference().(string)

	go s.searched(ctx, currRoot, query)
}

func (s *Upload) appendSearchResult(result SearchResult) {
	i := s.searchTable.GetRowCount()
	currRow, _ := s.searchTable.GetSelection()
	info, err := result.entry.Info()
	if err != nil {
		return
	}
	s.searchTable.SetCell(i, 0, tview.NewTableCell(fmt.Sprintf(" %d  ", i-1)).SetExpansion(0).SetReference(result.path))
	s.searchTable.SetCell(i, 1, tview.NewTableCell(fmt.Sprintf(" %s  ", info.Name())).SetExpansion(1).SetAlign(tview.AlignLeft))
	s.searchTable.SetCell(i, 2, tview.NewTableCell(fmt.Sprintf("  %s  ", units.HumanSize(float64(info.Size())))).SetAlign(tview.AlignCenter).SetExpansion(0))
	s.searchTable.SetCell(i, 3, tview.NewTableCell(fmt.Sprintf("  %s  ", units.HumanDuration(time.Since(info.ModTime())))).SetAlign(tview.AlignCenter).SetExpansion(0))
	s.searchTable.Select(currRow, 0)
}

func (s *Upload) searched(ctx context.Context, root, query string) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		if err != nil || path == root {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		if !s.showHidden && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if fuzzy.MatchNormalizedFold(query, d.Name()) {
			s.searchResults <- SearchResult{
				path:  path,
				entry: d,
			}
		}

		return nil
	})
}

// UPLOAD
func (s *Upload) uploadFile(path string) {
	s.flex.RemoveItem(s.searchTable)
	s.flex.RemoveItem(s.fileTree)

	s.confirmUpload.
		AddTextView("File to Upload: ", path, 0, 1, true, false).
		AddInputField("File Password:", "", 30, nil, nil).
		AddButton("Cancel", func() {
			s.confirmUpload.Clear(true)
			s.flex.RemoveItem(s.confirmUpload)
			s.flex.AddItem(s.fileTree, 0, 1, true)
			s.app.SetFocus(s.fileTree)
		}).SetButtonsAlign(tview.AlignCenter).
		AddButton("Confirm", func() {
			item := s.confirmUpload.GetFormItemByLabel("File Password:")
			pwd := []byte(item.(*tview.InputField).GetText())
			go s.processUpload(pwd, path)
			s.confirmUpload.Clear(true)
			s.updater.switchPage(pages.Interaction.FILES)
		}).SetFocus(1).
		SetCancelFunc(func() {
			s.confirmUpload.Clear(true)
			s.flex.RemoveItem(s.confirmUpload)
			s.flex.AddItem(s.fileTree, 0, 1, true)
			s.app.SetFocus(s.fileTree)
		})

	s.flex.AddItem(s.confirmUpload, 0, 1, true)
	s.app.SetFocus(s.confirmUpload)
}

func (s *Upload) processUpload(pwd []byte, path string) {
	name := filepath.Base(path)
	progress := func(stage string, done int64) {
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus(fmt.Sprintf("Uploading %s... : %s %d%%", name[:7], stage, done))
		})
	}
	s.service.ManageUpload(pwd, path, progress)
	clear(pwd)

	s.app.QueueUpdateDraw(func() {
		s.updater.setStatus(fmt.Sprintf("Uploaded %s... successfully", name[:7]))
	})

	time.AfterFunc(5*time.Second, func() {
		s.app.QueueUpdateDraw(func() {
			s.updater.setStatus("")
		})
	})
}
