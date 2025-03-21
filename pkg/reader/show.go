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

	// Create a map to store TOCValues by their ID for faster lookups
	tocMap := make(map[string]epub.TOCValue)
	for _, toc := range r.Book.TOC.Slice {
		tocMap[toc.ID] = toc
	}

	// Get all level 0 TOC entries
	var l0Toc []epub.TOCValue
	for _, toc := range r.Book.TOC.Slice {
		if toc.Level == 0 {
			l0Toc = append(l0Toc, toc)
		}
	}

	// Map to store nodes by their TOC ID for faster lookups
	nodeMap := make(map[string]*tview.TreeNode)

	// Function to add TOC items to the tree
	var add func(target *tview.TreeNode, items []epub.TOCValue)
	add = func(target *tview.TreeNode, items []epub.TOCValue) {
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
			nodeMap[item.ID] = node
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

		// If node is not a directory or is already expanded, read the chapter
		if !item.IsDir || node.IsExpanded() {
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

		// Find and add children if the node is collapsed
		var children []epub.TOCValue
		for _, child := range r.Book.TOC.Slice {
			if child.ParentID == item.ID {
				children = append(children, child)
			}
		}
		if len(children) > 0 {
			add(node, children)
			node.SetExpanded(true)
		}
	})

	// Add root level nodes
	add(root, l0Toc)

	// Get the current TOC entry
	currentToc := r.Book.TOC.Slice[index]
	utils.DebugLog("[INFO:showTOC] Current TOC: %s (level: %d, ID: %s)", currentToc.Title, currentToc.Level, currentToc.ID)

	// Build the path from current node to root
	var path []string
	path = append(path, currentToc.ID)

	// Start from current TOC and go up to build the complete path
	var tempToc = currentToc
	for tempToc.ParentID != "" {
		parentToc, exists := tocMap[tempToc.ParentID]
		if !exists {
			utils.DebugLog("[INFO:showTOC] Warning: Could not find parent with ID: %s", tempToc.ParentID)
			break
		}

		// Add parent ID to the beginning of path
		path = append([]string{parentToc.ID}, path...)
		utils.DebugLog("[INFO:showTOC] Added parent to path: %s (ID: %s, Level: %d)",
			parentToc.Title, parentToc.ID, parentToc.Level)

		tempToc = parentToc
	}

	utils.DebugLog("[INFO:showTOC] Complete path length: %d", len(path))

	if len(path) > 0 {
		// Find the root level node of the path
		rootId := path[0]
		rootNode, exists := nodeMap[rootId]

		if exists {
			utils.DebugLog("[INFO:showTOC] Found root of path: %s", rootNode.GetText())

			// Expand the root node
			rootNode.SetExpanded(true)

			// Add first level children if not already added
			var rootRef epub.TOCValue
			if ref, ok := rootNode.GetReference().(epub.TOCValue); ok {
				rootRef = ref

				// Find children for this node
				var children []epub.TOCValue
				for _, child := range r.Book.TOC.Slice {
					if child.ParentID == rootRef.ID {
						children = append(children, child)
					}
				}

				if len(rootNode.GetChildren()) == 0 && len(children) > 0 {
					utils.DebugLog("[INFO:showTOC] Adding %d children to root node", len(children))
					add(rootNode, children)
				}
			}

			// Navigate the path, expanding nodes as we go
			currentNode := rootNode
			for i := 1; i < len(path); i++ {
				childId := path[i]

				// If the node isn't in our map yet, we need to add and expand its parent
				childNode, exists := nodeMap[childId]
				if !exists {
					utils.DebugLog("[INFO:showTOC] Node not found in map yet, expanding parent: %s", currentNode.GetText())

					// Get the parent's reference
					parentRef, ok := currentNode.GetReference().(epub.TOCValue)
					if !ok {
						utils.DebugLog("[INFO:showTOC] Failed to get reference for parent node")
						break
					}

					// Find all children for the parent
					var children []epub.TOCValue
					for _, child := range r.Book.TOC.Slice {
						if child.ParentID == parentRef.ID {
							children = append(children, child)
							utils.DebugLog("[INFO:showTOC] Found child for %s: %s (ID: %s)",
								parentRef.Title, child.Title, child.ID)
						}
					}

					// Add the children to the parent node and expand it
					if len(children) > 0 {
						add(currentNode, children)
						currentNode.SetExpanded(true)
					}

					// Now try to find our target child again
					childNode, exists = nodeMap[childId]
				}

				if !exists {
					utils.DebugLog("[INFO:showTOC] Failed to find child node with ID: %s", childId)
					break
				}

				// Move to the next node in the path
				currentNode = childNode

				// If this is the last node in the path (our target), select it
				if i == len(path)-1 {
					tree.SetCurrentNode(childNode)
					utils.DebugLog("[INFO:showTOC] Set current node to: %s", childNode.GetText())
				} else {
					// Otherwise expand it to continue down the path
					if !childNode.IsExpanded() {
						childNode.SetExpanded(true)

						// Add children to this node
						childRef, ok := childNode.GetReference().(epub.TOCValue)
						if ok {
							var grandchildren []epub.TOCValue
							for _, child := range r.Book.TOC.Slice {
								if child.ParentID == childRef.ID {
									grandchildren = append(grandchildren, child)
								}
							}

							if len(grandchildren) > 0 {
								utils.DebugLog("[INFO:showTOC] Adding %d children to node: %s",
									len(grandchildren), childNode.GetText())
								add(childNode, grandchildren)
							}
						}
					}
				}
			}
		} else {
			utils.DebugLog("[INFO:showTOC] Could not find root node for path with ID: %s", rootId)
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
