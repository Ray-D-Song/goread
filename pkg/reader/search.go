package reader

import (
	"fmt"
	"regexp"
	"strings"
)

// search searches for a pattern
func (r *Reader) search() {
	r.UI.ShowSearch(func() {
		// When search is completed, highlight all occurrences and find the first one
		if r.UI.SearchPattern != "" {
			re, err := regexp.Compile(r.UI.SearchPattern)
			if err == nil {
				// First find the first occurrence to get its index
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
				} else {
					r.UI.StatusBar.Clear()
					fmt.Fprintf(r.UI.StatusBar, "[red]Pattern not found:[white] %s", r.UI.SearchPattern)
				}
			}
		}
	})
}

// searchNext searches for the next occurrence of the search pattern
func (r *Reader) searchNext() {
	if r.UI.SearchPattern == "" {
		return
	}

	text := r.UI.TextArea.GetText(false)
	re, err := regexp.Compile(r.UI.SearchPattern)
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Invalid search pattern: %v", err))
		return
	}

	// Get the current position
	row, _ := r.UI.TextArea.GetScrollOffset()
	pos := row

	lines := strings.Split(text, "\n")
	found := false
	foundIndex := -1

	for i := pos + 1; i < len(lines); i++ {
		if re.MatchString(lines[i]) {
			foundIndex = i
			found = true
			break
		}
	}

	// If we get here and didn't find anything, wrap around
	if !found {
		for i := 0; i <= pos; i++ {
			if re.MatchString(lines[i]) {
				foundIndex = i
				found = true
				break
			}
		}
	}

	if found {
		// Highlight the search results in the entire text with the current focused one highlighted differently
		r.highlightSearchResults(re, foundIndex)

		// Scroll to the found occurrence
		r.UI.TextArea.ScrollTo(foundIndex, 0)

		// Show status message
		if foundIndex <= pos {
			r.UI.SetStatus(fmt.Sprintf("Found: %s (wrapped)", lines[foundIndex]))
		} else {
			r.UI.SetStatus(fmt.Sprintf("Found: %s", lines[foundIndex]))
		}
	} else {
		r.UI.StatusBar.Clear()
		fmt.Fprintf(r.UI.StatusBar, "[red]Pattern not found:[white] %s", r.UI.SearchPattern)
	}
}

// searchPrev searches for the previous occurrence of the search pattern
func (r *Reader) searchPrev() {
	if r.UI.SearchPattern == "" {
		return
	}

	text := r.UI.TextArea.GetText(false)
	re, err := regexp.Compile(r.UI.SearchPattern)
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Invalid search pattern: %v", err))
		return
	}

	// Get the current position
	row, _ := r.UI.TextArea.GetScrollOffset()
	pos := row

	// Split the text into lines
	lines := strings.Split(text, "\n")
	found := false
	foundIndex := -1

	// Find the previous occurrence (searching backwards)
	for i := pos - 1; i >= 0; i-- {
		if re.MatchString(lines[i]) {
			foundIndex = i
			found = true
			break
		}
	}

	// If we get here and didn't find anything, wrap around to the end
	if !found {
		for i := len(lines) - 1; i >= pos; i-- {
			if re.MatchString(lines[i]) {
				foundIndex = i
				found = true
				break
			}
		}
	}

	if found {
		// Highlight the search results in the entire text with the current focused one highlighted differently
		r.highlightSearchResults(re, foundIndex)

		// Scroll to the found occurrence
		r.UI.TextArea.ScrollTo(foundIndex, 0)

		// Show status message
		if foundIndex >= pos {
			r.UI.SetStatus(fmt.Sprintf("Found: %s (wrapped)", lines[foundIndex]))
		} else {
			r.UI.SetStatus(fmt.Sprintf("Found: %s", lines[foundIndex]))
		}
	} else {
		r.UI.StatusBar.Clear()
		fmt.Fprintf(r.UI.StatusBar, "[red]Pattern not found:[white] %s", r.UI.SearchPattern)
	}
}

// highlightSearchResults highlights all occurrences of the search pattern in the text
// focusedLineIndex is the line index of the currently focused search result
func (r *Reader) highlightSearchResults(re *regexp.Regexp, focusedLineIndex int) {
	// Get the current text
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")

	// Clear the text area
	r.UI.TextArea.Clear()

	// Write the lines back with highlighted search results
	for i, line := range lines {
		if i == focusedLineIndex {
			// For the focused line, highlight matches with a different color
			highlightedLine := re.ReplaceAllStringFunc(line, func(match string) string {
				return fmt.Sprintf("[black:green]%s[-:-]", match)
			})
			fmt.Fprintln(r.UI.TextArea, highlightedLine)
		} else {
			// For other lines, use the standard highlight color
			highlightedLine := re.ReplaceAllStringFunc(line, func(match string) string {
				return fmt.Sprintf("[black:yellow]%s[-:-]", match)
			})
			fmt.Fprintln(r.UI.TextArea, highlightedLine)
		}
	}
}

// clearSearchHighlights clears all search highlights from the text
func (r *Reader) clearSearchHighlights() {
	// Get the current text without color codes
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")

	// Clear the text area
	r.UI.TextArea.Clear()

	// Write the lines back without highlights
	for _, line := range lines {
		fmt.Fprintln(r.UI.TextArea, line)
	}

	r.UI.SetStatus("Search cleared")
}
