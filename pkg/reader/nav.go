package reader

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
)

// scrollDown scrolls down
func (r *Reader) scrollDown(pos int) {
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+1, col)
}

// scrollUp scrolls up
func (r *Reader) scrollUp(pos int) {
	row, col := r.UI.TextArea.GetScrollOffset()
	if row > 0 {
		r.UI.TextArea.ScrollTo(row-1, col)
	}
}

// pageDown goes to the next page
func (r *Reader) pageDown(pos int) {
	_, _, height, _ := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+height, col)
}

// pageUp goes to the previous page
func (r *Reader) pageUp(pos int) {
	_, _, height, _ := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	if row-height > 0 {
		r.UI.TextArea.ScrollTo(row-height, col)
	} else {
		r.UI.TextArea.ScrollTo(0, col)
	}
}

// halfPageUp goes up half a page
func (r *Reader) halfPageUp(pos int) {
	_, _, height, _ := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	if row-height/2 > 0 {
		r.UI.TextArea.ScrollTo(row-height/2, col)
	} else {
		r.UI.TextArea.ScrollTo(0, col)
	}
}

// halfPageDown goes down half a page
func (r *Reader) halfPageDown(pos int) {
	_, _, height, _ := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+height/2, col)
}

// goToStart goes to the start of the chapter
func (r *Reader) goToStart() {
	r.UI.TextArea.ScrollTo(0, 0)
}

// goToEnd goes to the end of the chapter
func (r *Reader) goToEnd() {
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")
	r.UI.TextArea.ScrollTo(len(lines)-1, 0)
}

// jumpToPosition jumps to a marked position
func (r *Reader) jumpToPosition(index int, pos int, pctg float64) {
	// Wait for a key
	r.UI.StatusBar.SetText("Jump to position (1-9): ")
	r.UI.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() >= '1' && event.Rune() <= '9' {
			// Jump to the position
			if pos, ok := r.JumpList[event.Rune()]; ok {
				index := pos[0].(int)
				pctg := pos[3].(float64)
				err := r.readChapter(index, pctg)
				if err != nil {
					r.UI.StatusBar.SetText(fmt.Sprintf("Error jumping to position: %v", err))
				} else {
					r.UI.StatusBar.SetText(fmt.Sprintf("Jumped to position %c", event.Rune()))
				}
			} else {
				r.UI.StatusBar.SetText(fmt.Sprintf("No position marked as %c", event.Rune()))
			}
		}
		return event
	})
}

// nextChapter moves to the next chapter
func (r *Reader) nextChapter(index int, pos int, pctg float64) {
	utils.DebugLog("[INFO:nextChapter] Moving to next chapter from index: %d", index)

	// Check if we're currently in a virtual chapter
	virtualIndex, inVirtualChapter := r.getCurrentVirtualChapter()

	// Track the new index or virtual index for preloading
	var newIndex int = index
	var newVirtualIndex int = -1
	var isVirtual bool = false

	if inVirtualChapter {
		// If we're in a virtual chapter, move to the next virtual chapter or the next regular chapter
		if virtualIndex < len(r.Book.VirtualContents)-1 {
			// Move to the next virtual chapter
			utils.DebugLog("[INFO:nextChapter] Moving to next virtual chapter: %d", virtualIndex+1)
			err := r.readVirtualChapter(virtualIndex + 1)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading next virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading next virtual chapter: %v", err))
			} else {
				newVirtualIndex = virtualIndex + 1
				isVirtual = true
			}
		} else if index < len(r.Book.Contents)-1 {
			// Move to the next regular chapter
			utils.DebugLog("[INFO:nextChapter] Moving to next regular chapter: %d", index+1)
			err := r.readChapter(index+1, 0)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading next chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading next chapter: %v", err))
			} else {
				newIndex = index + 1
			}
		} else {
			utils.DebugLog("[INFO:nextChapter] Already at the last chapter")
			r.UI.StatusBar.SetText("Already at the last chapter")
		}
	} else {
		// If we're in a regular chapter
		if index < len(r.Book.Contents)-1 {
			// If there are more regular chapters, move to the next one
			utils.DebugLog("[INFO:nextChapter] Moving to next regular chapter: %d", index+1)
			err := r.readChapter(index+1, 0)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading next chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading next chapter: %v", err))
			} else {
				newIndex = index + 1
			}
		} else if len(r.Book.VirtualContents) > 0 {
			// If there are virtual chapters, move to the first one
			utils.DebugLog("[INFO:nextChapter] Moving to first virtual chapter")
			err := r.readVirtualChapter(0)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading virtual chapter: %v", err))
			} else {
				newVirtualIndex = 0
				isVirtual = true
			}
		} else {
			utils.DebugLog("[INFO:nextChapter] Already at the last chapter")
			r.UI.StatusBar.SetText("Already at the last chapter")
		}
	}

	// If there's an active search pattern, try to find the first occurrence in the new chapter
	if r.UI.SearchPattern != "" {
		re, err := regexp.Compile(r.UI.SearchPattern)
		if err == nil {
			// Find the first occurrence in the new chapter
			text := r.UI.TextArea.GetText(false)
			lines := strings.Split(text, "\n")
			foundIndex := -1

			for i, line := range lines {
				if re.MatchString(line) {
					foundIndex = i
					break
				}
			}

			if foundIndex >= 0 {
				// Highlight all results with the first one focused
				r.highlightSearchResults(re, foundIndex)
				// Scroll to the first occurrence
				r.UI.TextArea.ScrollTo(foundIndex, 0)
				r.UI.SetStatus(fmt.Sprintf("Found: %s", lines[foundIndex]))
			}
		}
	}

	// Preload the next content in background
	go func() {
		if isVirtual {
			// If we moved to a virtual chapter, preload the next virtual chapter
			if newVirtualIndex < len(r.Book.VirtualContents)-1 {
				nextVirtualContent := r.Book.VirtualContents[newVirtualIndex+1]
				filePath := nextVirtualContent.FilePath

				// Only preload if not already cached
				if _, ok := r.HTMLCache[filePath]; !ok {
					utils.DebugLog("[INFO:nextChapter] Preloading next virtual chapter: %d", newVirtualIndex+1)

					// Find the corresponding file index
					fileIndex := -1
					for i, content := range r.Book.Contents {
						if content == filePath {
							fileIndex = i
							break
						}
					}

					if fileIndex != -1 {
						content, err := r.Book.GetChapterContent(fileIndex)
						if err == nil {
							r.HTMLCache[filePath] = content
							utils.DebugLog("[INFO:nextChapter] Completed preloading next virtual chapter: %d", newVirtualIndex+1)
						}
					}
				}
			}
		} else {
			// If we moved to a regular chapter, preload the next regular chapter
			if newIndex < len(r.Book.Contents)-1 {
				r.preloadContent(newIndex)
			}
		}
	}()
}

// prevChapter moves to the previous chapter
func (r *Reader) prevChapter(index int, pos int, pctg float64) {
	utils.DebugLog("[INFO:prevChapter] Moving to previous chapter from index: %d", index)

	// Check if we're currently in a virtual chapter
	virtualIndex, inVirtualChapter := r.getCurrentVirtualChapter()

	if inVirtualChapter {
		// If we're in a virtual chapter, move to the previous virtual chapter or the previous regular chapter
		if virtualIndex > 0 {
			// Move to the previous virtual chapter
			utils.DebugLog("[INFO:prevChapter] Moving to previous virtual chapter: %d", virtualIndex-1)
			err := r.readVirtualChapter(virtualIndex - 1)
			if err != nil {
				utils.DebugLog("[ERROR:prevChapter] Error reading previous virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading previous virtual chapter: %v", err))
			}
		} else if index > 0 {
			// Move to the previous regular chapter
			utils.DebugLog("[INFO:prevChapter] Moving to previous regular chapter: %d", index-1)
			err := r.readChapter(index-1, 0)
			if err != nil {
				utils.DebugLog("[ERROR:prevChapter] Error reading previous chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading previous chapter: %v", err))
			}
		} else {
			utils.DebugLog("[INFO:prevChapter] Already at the first chapter")
			r.UI.StatusBar.SetText("Already at the first chapter")
		}
	} else {
		// If we're in a regular chapter
		if index > 0 {
			// If there are previous regular chapters, move to the previous one
			utils.DebugLog("[INFO:prevChapter] Moving to previous regular chapter: %d", index-1)
			err := r.readChapter(index-1, 0)
			if err != nil {
				utils.DebugLog("[ERROR:prevChapter] Error reading previous chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading previous chapter: %v", err))
			}
		} else if len(r.Book.VirtualContents) > 0 && index == len(r.Book.Contents)-1 {
			// If we're at the last regular chapter and there are virtual chapters, move to the last virtual chapter
			utils.DebugLog("[INFO:prevChapter] Moving to last virtual chapter: %d", len(r.Book.VirtualContents)-1)
			err := r.readVirtualChapter(len(r.Book.VirtualContents) - 1)
			if err != nil {
				utils.DebugLog("[ERROR:prevChapter] Error reading virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading virtual chapter: %v", err))
			}
		} else {
			utils.DebugLog("[INFO:prevChapter] Already at the first chapter")
			r.UI.StatusBar.SetText("Already at the first chapter")
		}
	}

	// If there's an active search pattern, try to find the first occurrence in the new chapter
	if r.UI.SearchPattern != "" {
		re, err := regexp.Compile(r.UI.SearchPattern)
		if err == nil {
			// Find the first occurrence in the new chapter
			text := r.UI.TextArea.GetText(false)
			lines := strings.Split(text, "\n")
			foundIndex := -1

			for i, line := range lines {
				if re.MatchString(line) {
					foundIndex = i
					break
				}
			}

			if foundIndex >= 0 {
				// Highlight all results with the first one focused
				r.highlightSearchResults(re, foundIndex)
				// Scroll to the first occurrence
				r.UI.TextArea.ScrollTo(foundIndex, 0)
				r.UI.SetStatus(fmt.Sprintf("Found: %s", lines[foundIndex]))
			}
		}
	}
}
