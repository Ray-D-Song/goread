package reader

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/epub"
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
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	var l1Toc []epub.TOCValue

	add := func(target *tview.TreeNode, items []epub.TOCValue) {
		for _, item := range items {
			node := tview.NewTreeNode(item.Title)
			node.SetReference(item)
			node.SetSelectable(true)
			target.AddChild(node)
		}
	}

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		item := node.GetReference().(epub.TOCValue)
		var children []epub.TOCValue
		for _, child := range r.Book.TOC.Slice {
			if child.ParentID == item.ID {
				children = append(children, child)
			}
		}
		add(node, children)
		node.SetExpanded(!node.IsExpanded())
	})

	for _, toc := range r.Book.TOC.Slice {
		if toc.Level == 1 {
			l1Toc = append(l1Toc, toc)
		}
	}
	add(root, l1Toc)

	r.UI.SetTempContent(tree)

	r.UI.App.SetFocus(tree)

	var resetCapture func()
	resetCapture = r.UI.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyTab:
			resetCapture()
			return nil
		case tcell.KeyEnter:
			selectedNode := tree.GetCurrentNode()
			if selectedNode == nil {
				return nil
			}

			if selectedNode.GetReference() == nil {
				return nil
			}

			item := selectedNode.GetReference().(epub.TOCValue)

			if item.IsDir && selectedNode.IsExpanded() {
				var children []epub.TOCValue
				for _, child := range r.Book.TOC.Slice {
					if child.ParentID == item.ID {
						children = append(children, child)
					}
				}
				add(selectedNode, children)
				selectedNode.SetExpanded(!selectedNode.IsExpanded())
			} else {
				for i, toc := range r.Book.TOC.Slice {
					if toc.ID == item.ID {
						resetCapture()
						r.readChapter(i, 0)
						return nil
					}
				}
			}
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			return event
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
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
