package reader

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/epub"
	"github.com/ray-d-song/goread/pkg/ui"
	"github.com/ray-d-song/goread/pkg/utils"
	"github.com/rivo/tview"
)

// showMetadata shows the metadata
func (r *Reader) showMetadata() {
	metadata, err := r.Book.GetMetadata()
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Error getting metadata: %v", err))
		return
	}

	var metadataItems [][]string
	if metadata.Title != "" {
		metadataItems = append(metadataItems, []string{"Title", metadata.Title})
	}
	if metadata.Creator != "" {
		metadataItems = append(metadataItems, []string{"Creator", metadata.Creator})
	}
	if metadata.Publisher != "" {
		metadataItems = append(metadataItems, []string{"Publisher", metadata.Publisher})
	}
	if metadata.Language != "" {
		metadataItems = append(metadataItems, []string{"Language", metadata.Language})
	}
	if metadata.Identifier != "" {
		metadataItems = append(metadataItems, []string{"Identifier", metadata.Identifier})
	}
	if metadata.Date != "" {
		metadataItems = append(metadataItems, []string{"Date", metadata.Date})
	}
	if metadata.Description != "" {
		metadataItems = append(metadataItems, []string{"Description", metadata.Description})
	}
	if metadata.Rights != "" {
		metadataItems = append(metadataItems, []string{"Rights", metadata.Rights})
	}
	for _, item := range metadata.OtherMeta {
		metadataItems = append(metadataItems, item)
	}

	r.UI.ShowMetadata(metadataItems)
}

// showTOC shows the table of contents
func (r *Reader) showTOC(index int) {
	root := tview.NewTreeNode("TOC")
	tree := tview.NewTreeView()
	switch r.UI.ColorScheme {
	case ui.DefaultColorScheme:
		tree.SetBackgroundColor(tcell.ColorDefault)
		tree.SetGraphicsColor(tcell.ColorDefault)
		tree.SetTitleColor(tcell.ColorDefault)
	case ui.DarkColorScheme:
		tree.SetBackgroundColor(tcell.ColorDarkSlateGray)
		tree.SetGraphicsColor(tcell.ColorWhite)
		tree.SetTitleColor(tcell.ColorWhite)
	case ui.LightColorScheme:
		tree.SetBackgroundColor(tcell.ColorWhite)
		tree.SetGraphicsColor(tcell.ColorBlack)
		tree.SetTitleColor(tcell.ColorBlack)
	}
	tree.SetRoot(root)
	tree.SetCurrentNode(root)
	var l0Toc []epub.TOCValue

	var nodes []*tview.TreeNode
	add := func(target *tview.TreeNode, items []epub.TOCValue) {
		for _, item := range items {
			node := tview.NewTreeNode(item.Title)
			node.SetReference(item)
			node.SetSelectable(true)
			node.SetExpanded(false)
			switch r.UI.ColorScheme {
			case ui.DefaultColorScheme:
				node.SetTextStyle(tcell.StyleDefault)
				node.SetSelectedTextStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan))
			case ui.DarkColorScheme:
				node.SetTextStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray))
				node.SetSelectedTextStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue))
			case ui.LightColorScheme:
				node.SetTextStyle(tcell.StyleDefault.Background(tcell.ColorWhite))
				node.SetSelectedTextStyle(tcell.StyleDefault.Background(tcell.ColorLightBlue))
			}
			target.AddChild(node)
			nodes = append(nodes, node)
		}
	}

	var resetCapture func()
	var resetContent func()
	// Set the selected function for the tree
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		utils.DebugLog("[INFO:showTOC] Selected node: %s", node.GetText())
		if node.GetReference() == nil {
			utils.DebugLog("[INFO:showTOC] No reference found for node: %s", node.GetText())
			return
		}
		item, ok := node.GetReference().(epub.TOCValue)
		if !ok {
			utils.DebugLog("[INFO:showTOC] No reference found for node: %s", node.GetText())
			return
		}
		if !item.IsDir {
			resetCapture()
			resetContent()
			index, err := r.Book.GetChapterIndex(item.ID)
			if err != nil {
				utils.DebugLog("[INFO:showTOC] Error getting chapter index: %v", err)
				return
			}
			r.readChapter(index, 0)
			return
		}
		if node.IsExpanded() {
			utils.DebugLog("[INFO:showTOC] Dir is already expanded: %s", node.GetText())
			resetCapture()
			resetContent()
			index, err := r.Book.GetChapterIndex(item.ID)
			if err != nil {
				utils.DebugLog("[INFO:showTOC] Error getting chapter index: %v", err)
				return
			}
			r.readChapter(index, 0)
			return
		}
		var children []epub.TOCValue
		for _, child := range r.Book.TOC.Slice {
			if child.ParentID == item.ID {
				children = append(children, child)
			}
		}
		add(node, children)
		node.SetExpanded(true)
	})

	for _, toc := range r.Book.TOC.Slice {
		if toc.Level == 0 {
			l0Toc = append(l0Toc, toc)
		}
	}
	add(root, l0Toc)
	currentToc := r.Book.TOC.Slice[index]
	var pid string = currentToc.ParentID
	// recursively find currentToc's parent
	for l := currentToc.Level; l >= 0; l-- {
		for _, item := range r.Book.TOC.Slice {
			if item.ID == pid {
				pid = item.ParentID
				utils.DebugLog("[INFO:showTOC] Found parent: %s", item.Title)
				break
			}
		}
	}
	// now pid is the root of the current toc
	// find the root in l0Toc
	utils.DebugLog("[INFO:showTOC] Finding root %s", pid)
	for _, item := range l0Toc {
		utils.DebugLog("[INFO:showTOC] Checking item: %s %s", item.Title, item.ID)
		if item.ID == pid {
			utils.DebugLog("[INFO:showTOC] Found root: %s", item.Title)
		}
	}
	resetContent = r.UI.SetTempContent(tree)

	r.UI.App.SetFocus(tree)

	resetCapture = r.UI.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyTab:
			resetCapture()
			resetContent()
			return nil
		case tcell.KeyEnter:
			return event
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			return event
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				resetContent()
				resetCapture()
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'h':
				return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
			case 'l':
				return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
			}
		}
		return event
	})
}
