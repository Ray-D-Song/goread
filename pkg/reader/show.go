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
	var l0ID string = ""
	// recursively find currentToc's parent
	// for example, if currentToc is level 2, then we need to find the parent of the parent of currentToc
	// so the loop should do currentToc.Level times
	for l := currentToc.Level; l >= 1; l-- {
		for _, item := range r.Book.TOC.Slice {
			if item.ID == pid {
				if item.Level == 0 {
					l0ID = item.ID
				}
				pid = item.ParentID
				utils.DebugLog("[INFO:showTOC] Found parent: %s %s (level: %d)", item.Title, item.ID, item.Level)
				break
			}
		}
	}
	// now l0ID is the root of the current toc
	// find the root in l0Toc
	utils.DebugLog("[INFO:showTOC] Finding root %s", l0ID)
	for _, item := range l0Toc {
		if item.ID == l0ID {
			utils.DebugLog("[INFO:showTOC] Found root: %s", item.Title)
			// Find the TreeNode corresponding to the root
			var rootNode *tview.TreeNode
			for _, node := range nodes {
				if ref, ok := node.GetReference().(epub.TOCValue); ok && ref.ID == item.ID {
					rootNode = node
					utils.DebugLog("[INFO:showTOC] Found root node in tree: %s", ref.Title)
					break
				}
			}

			if rootNode != nil {
				// Expand the root node
				rootNode.SetExpanded(true)
				utils.DebugLog("[INFO:showTOC] Expanded root node")

				// First, let's check what children we have in the root
				var rootRef epub.TOCValue
				if ref, ok := rootNode.GetReference().(epub.TOCValue); ok {
					rootRef = ref
				}

				// Add root's children if not already added
				var children []epub.TOCValue
				for _, child := range r.Book.TOC.Slice {
					if child.ParentID == rootRef.ID {
						children = append(children, child)
						utils.DebugLog("[INFO:showTOC] Root has child: %s (ID: %s)", child.Title, child.ID)
					}
				}
				if len(rootNode.GetChildren()) == 0 && len(children) > 0 {
					utils.DebugLog("[INFO:showTOC] Adding %d children to root node", len(children))
					add(rootNode, children)
				}

				// Recursively find and expand child nodes until the current node is found
				var findCurrentNode func(parent *tview.TreeNode, level int, path []string) *tview.TreeNode
				findCurrentNode = func(parent *tview.TreeNode, level int, path []string) *tview.TreeNode {
					// If reached the level of the current chapter, check if it is the current chapter
					if level == len(path) {
						if ref, ok := parent.GetReference().(epub.TOCValue); ok && ref.ID == currentToc.ID {
							utils.DebugLog("[INFO:showTOC] Found current node: %s", ref.Title)
							return parent
						}
						if ref, ok := parent.GetReference().(epub.TOCValue); ok {
							utils.DebugLog("[INFO:showTOC] Reached path end but node doesn't match. Expected ID: %s, Found ID: %s",
								currentToc.ID, ref.ID)
						} else {
							utils.DebugLog("[INFO:showTOC] Reached path end but node reference is invalid")
						}
						return nil
					}

					// Add all child nodes at this level (if not already added)
					if !parent.IsExpanded() {
						var children []epub.TOCValue
						if ref, ok := parent.GetReference().(epub.TOCValue); ok {
							utils.DebugLog("[INFO:showTOC] Adding children for node: %s (ID: %s)", ref.Title, ref.ID)
							for _, child := range r.Book.TOC.Slice {
								if child.ParentID == ref.ID {
									children = append(children, child)
									utils.DebugLog("[INFO:showTOC] Found child: %s (ID: %s)", child.Title, child.ID)
								}
							}
							add(parent, children)
							parent.SetExpanded(true)
							utils.DebugLog("[INFO:showTOC] Added %d children", len(children))
						}
					}

					// Search for the next node ID in the path among the child nodes
					utils.DebugLog("[INFO:showTOC] Looking for path ID at level %d: %s", level, path[level])
					found := false
					for _, childNode := range parent.GetChildren() {
						if ref, ok := childNode.GetReference().(epub.TOCValue); ok {
							utils.DebugLog("[INFO:showTOC] Checking child node: %s (ID: %s)", ref.Title, ref.ID)
							if ref.ID == path[level] {
								utils.DebugLog("[INFO:showTOC] Following path through: %s", ref.Title)
								found = true
								result := findCurrentNode(childNode, level+1, path)
								if result != nil {
									return result
								}
							}
						}
					}

					if !found {
						utils.DebugLog("[INFO:showTOC] Path node with ID %s not found at level %d", path[level], level)
					}

					return nil
				}

				// Build the path from root node to current node
				var path []string
				temp := currentToc
				utils.DebugLog("[INFO:showTOC] Building path from current: %s (level: %d, ID: %s) to root: %s",
					currentToc.Title, currentToc.Level, currentToc.ID, item.Title)

				// If the current node is a direct child of the root node
				if currentToc.ParentID == l0ID {
					utils.DebugLog("[INFO:showTOC] Current node is direct child of root")
					path = []string{currentToc.ID}
				} else {
					// Otherwise, build the full path
					for temp.ParentID != l0ID && temp.ParentID != "" {
						// Find parent node
						found := false
						for _, toc := range r.Book.TOC.Slice {
							if toc.ID == temp.ParentID {
								path = append([]string{toc.ID}, path...) // Prepend
								utils.DebugLog("[INFO:showTOC] Added to path: %s (ID: %s)", toc.Title, toc.ID)
								temp = toc
								found = true
								break
							}
						}
						if !found {
							utils.DebugLog("[INFO:showTOC] Warning: Could not find parent with ID: %s", temp.ParentID)
							break
						}
					}

					// Add current node to path
					path = append(path, currentToc.ID)
				}

				utils.DebugLog("[INFO:showTOC] Final path length: %d", len(path))

				// Special case: if just one level deep, try direct scan
				if currentToc.Level == 1 {
					utils.DebugLog("[INFO:showTOC] Level 1 node - scanning root's children directly")
					for _, childNode := range rootNode.GetChildren() {
						if ref, ok := childNode.GetReference().(epub.TOCValue); ok {
							utils.DebugLog("[INFO:showTOC] Checking direct child: %s (ID: %s)", ref.Title, ref.ID)
							if ref.ID == currentToc.ID {
								utils.DebugLog("[INFO:showTOC] Found current node directly: %s", ref.Title)
								tree.SetCurrentNode(childNode)
								return
							}
						}
					}
				}

				// Find and select the current node
				currentNode := findCurrentNode(rootNode, 0, path)
				if currentNode != nil {
					tree.SetCurrentNode(currentNode)
					utils.DebugLog("[INFO:showTOC] Selected current node in tree")
				} else {
					utils.DebugLog("[INFO:showTOC] Failed to find current node in tree - dumping path IDs:")
					for i, id := range path {
						utils.DebugLog("[INFO:showTOC] Path[%d]: %s", i, id)
					}
				}
			} else {
				utils.DebugLog("[INFO:showTOC] Root node not found in tree nodes")
			}
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
