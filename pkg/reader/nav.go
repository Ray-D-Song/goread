package reader

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
)

// scrollDown scrolls down
func (r *Reader) scrollDown() {
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+1, col)
}

// scrollUp scrolls up
func (r *Reader) scrollUp() {
	row, col := r.UI.TextArea.GetScrollOffset()
	if row > 0 {
		r.UI.TextArea.ScrollTo(row-1, col)
	}
}

// pageDown goes to the next page
func (r *Reader) pageDown(pos int) {
	_, _, _, height := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+height, col)
}

// pageUp goes to the previous page
func (r *Reader) pageUp(pos int) {
	_, _, _, height := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	if row-height > 0 {
		r.UI.TextArea.ScrollTo(row-height, col)
	} else {
		r.UI.TextArea.ScrollTo(0, col)
	}
}

// halfPageUp goes up half a page
func (r *Reader) halfPageUp(pos int) {
	_, _, _, height := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	if row-height/2 > 0 {
		r.UI.TextArea.ScrollTo(row-height/2, col)
	} else {
		r.UI.TextArea.ScrollTo(0, col)
	}
}

// halfPageDown goes down half a page
func (r *Reader) halfPageDown(pos int) {
	_, _, _, height := r.UI.TextArea.GetInnerRect()
	row, col := r.UI.TextArea.GetScrollOffset()
	r.UI.TextArea.ScrollTo(row+height/2, col)
}

// goToStart goes to the start of the chapter
func (r *Reader) goToStart() {
	r.UI.TextArea.ScrollToBeginning()
}

// goToEnd goes to the end of the chapter
func (r *Reader) goToEnd() {
	r.UI.TextArea.ScrollToEnd()
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
func (r *Reader) nextChapter(pos int, pctg float64) {
	utils.DebugLog("[INFO:nextChapter] Moving to next chapter from index: %d", r.CurrentChapter)

	// Save current chapter state before moving to next chapter
	row, _ := r.UI.TextArea.GetScrollOffset()
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")
	currentPctg := float64(0)
	if len(lines) > 0 {
		currentPctg = float64(row) / float64(len(lines))
	}
	r.saveState(r.CurrentChapter, r.UI.Width, row, currentPctg)

	r.CurrentChapter++
	err := r.readChapter(r.CurrentChapter, 0) // Start at the beginning of the new chapter
	if err != nil {
		r.UI.StatusBar.SetText(fmt.Sprintf("Error reading chapter: %v", err))
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

// prevChapter moves to the previous chapter
func (r *Reader) prevChapter(pos int, pctg float64) {
	utils.DebugLog("[INFO:prevChapter] Moving to previous chapter from index: %d", r.CurrentChapter)

	// Save current chapter state before moving to previous chapter
	row, _ := r.UI.TextArea.GetScrollOffset()
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")
	currentPctg := float64(0)
	if len(lines) > 0 {
		currentPctg = float64(row) / float64(len(lines))
	}
	r.saveState(r.CurrentChapter, r.UI.Width, row, currentPctg)

	r.CurrentChapter--
	err := r.readChapter(r.CurrentChapter, 0) // Start at the beginning of the new chapter
	if err != nil {
		r.UI.StatusBar.SetText(fmt.Sprintf("Error reading chapter: %v", err))
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
