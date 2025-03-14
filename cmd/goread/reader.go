package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/config"
	"github.com/ray-d-song/goread/pkg/epub"
	"github.com/ray-d-song/goread/pkg/parser"
	"github.com/ray-d-song/goread/pkg/ui"
	"github.com/ray-d-song/goread/pkg/utils"
)

// Reader represents the EPUB reader
type Reader struct {
	Book           *epub.Epub
	Config         *config.Config
	FilePath       string
	UI             *ui.UI
	JumpList       map[rune][4]interface{} // [index, width, pos, pctg]
	CurrentChapter int                     // Current chapter index
}

// NewReader creates a new Reader instance
func NewReader(book *epub.Epub, cfg *config.Config, filePath string) *Reader {
	return &Reader{
		Book:           book,
		Config:         cfg,
		FilePath:       filePath,
		UI:             ui.NewUI(),
		JumpList:       make(map[rune][4]interface{}),
		CurrentChapter: 0,
	}
}

var InitialCapture func(event *tcell.EventKey) *tcell.EventKey

// Run runs the reader
func (r *Reader) Run(index int, width int, pos int, pctg float64) {
	// Initialize the UI
	r.UI.Width = width

	// Set up the key handling
	ic := r.UI.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			if r.UI.SearchPattern != "" {
				r.UI.SearchPattern = ""
				r.clearSearchHighlights()
				return nil
			}
			r.saveState(r.CurrentChapter, r.UI.Width, pos, pctg)
			r.UI.App.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				r.saveState(r.CurrentChapter, r.UI.Width, pos, pctg)
				r.UI.App.Stop()
				return nil
			case '?':
				r.UI.ShowHelp()
				return nil
			case 'm':
				r.showMetadata()
				return nil
			case 't', '\t':
				r.showTOC(r.CurrentChapter)
				return nil
			case '/':
				r.search()
				return nil
			case 'n':
				utils.DebugLog("[INFO:searchNext] SearchPattern: '%s', CurrentChapter: %d", r.UI.SearchPattern, r.CurrentChapter)
				if r.UI.SearchPattern != "" {
					r.searchNext()
				} else {
					r.nextChapter(r.CurrentChapter, pos, pctg)
				}
				return nil
			case 'N':
				utils.DebugLog("[INFO:searchPrev] SearchPattern: '%s', CurrentChapter: %d", r.UI.SearchPattern, r.CurrentChapter)
				if r.UI.SearchPattern != "" {
					r.searchPrev()
				} else {
					r.prevChapter(r.CurrentChapter, pos, pctg)
				}
				return nil
			case 'p':
				r.prevChapter(r.CurrentChapter, pos, pctg)
				return nil
			case 'j':
				r.scrollDown(pos)
				return nil
			case 'k':
				r.scrollUp(pos)
				return nil
			case ' ':
				r.pageDown(pos)
				return nil
			case 'g':
				r.goToStart()
				return nil
			case 'G':
				r.goToEnd()
				return nil
			case 'o':
				r.openImage()
				return nil
			case '=':
				// Check if there's a count before the '=' key
				r.toggleWidth(r.CurrentChapter, pos, pctg)
				return nil
			case 'c':
				r.UI.CycleColorScheme()
				return nil
			case 'b':
				r.markPosition(r.CurrentChapter, pos, pctg)
				return nil
			case '`':
				r.jumpToPosition(r.CurrentChapter, pos, pctg)
				return nil
			}
		case tcell.KeyDown:
			r.scrollDown(pos)
			return nil
		case tcell.KeyUp:
			r.scrollUp(pos)
			return nil
		case tcell.KeyPgDn, tcell.KeyRight:
			r.pageDown(pos)
			return nil
		case tcell.KeyPgUp, tcell.KeyLeft:
			r.pageUp(pos)
			return nil
		case tcell.KeyHome:
			r.goToStart()
			return nil
		case tcell.KeyEnd:
			r.goToEnd()
			return nil
		case tcell.KeyCtrlU:
			r.halfPageUp(pos)
			return nil
		case tcell.KeyCtrlD:
			r.halfPageDown(pos)
			return nil
		}
		return event
	})

	InitialCapture = ic.GetInputCapture()
	// Start the reader
	err := r.readChapter(index, pctg)
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
	}

	// Run the application
	if err := r.UI.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

func (r *Reader) ResetCapture() {
	r.UI.App.SetInputCapture(InitialCapture)
}

func (r *Reader) SetCapture(f func(event *tcell.EventKey) *tcell.EventKey) {
	r.UI.App.SetInputCapture(f)
}

// readChapter reads a chapter
func (r *Reader) readChapter(index int, pctg float64) error {
	utils.DebugLog("[INFO:readChapter] Reading chapter index: %d, pctg: %f", index, pctg)
	if index < 0 || index >= len(r.Book.Contents) {
		utils.DebugLog("[ERROR:readChapter] Invalid chapter index: %d", index)
		return fmt.Errorf("invalid chapter index: %d", index)
	}

	r.CurrentChapter = index
	r.UI.StatusBar.SetText(fmt.Sprintf("Reading chapter %d of %d", index+1, len(r.Book.Contents)))

	// Get the chapter content
	content, err := r.Book.GetChapterContent(index)
	if err != nil {
		utils.DebugLog("[ERROR:readChapter] Error getting chapter content: %v", err)
		return err
	}

	// Parse the HTML content
	parser := parser.NewHTMLParser()
	err = parser.Parse(content)
	if err != nil {
		utils.DebugLog("[ERROR:readChapter] Error parsing HTML content: %v", err)
		return err
	}

	// Format the lines of text
	lines := parser.FormatLines(r.UI.Width)

	// Store the images for later use
	r.UI.Images = parser.GetImages()

	// Clear the text area and write the formatted lines
	r.UI.TextArea.Clear()
	for _, line := range lines {
		fmt.Fprintln(r.UI.TextArea, line)
	}

	// If there's an active search pattern, highlight the results
	if r.UI.SearchPattern != "" {
		re, err := regexp.Compile(r.UI.SearchPattern)
		if err == nil {
			// Find the first occurrence to highlight it differently
			foundIndex := -1
			for i, line := range lines {
				if re.MatchString(line) {
					foundIndex = i
					break
				}
			}

			if foundIndex >= 0 {
				r.highlightSearchResults(re, foundIndex)
				// Don't automatically scroll to it here, as we want to respect the pctg parameter
			} else {
				// If no matches found, just highlight all (none will be focused)
				r.highlightSearchResults(re, -1)
			}
		} else {
			// If the pattern is invalid, clear it
			r.UI.SearchPattern = ""
		}
	}

	// Scroll to the specified position
	if pctg > 0 {
		// Estimate the line count based on the number of lines we wrote
		lineCount := len(lines)
		if lineCount > 0 {
			r.UI.TextArea.ScrollTo(int(float64(lineCount)*pctg), 0)
		}
	} else {
		r.UI.TextArea.ScrollToBeginning()
	}

	utils.DebugLog("[INFO:readChapter] Successfully read chapter %d", index)
	return nil
}

// saveState saves the reading state
func (r *Reader) saveState(index int, width int, pos int, pctg float64) {
	state := config.State{
		Index:    index,
		Width:    width,
		Pos:      pos,
		Pctg:     pctg,
		LastRead: true,
	}
	r.Config.SetState(r.FilePath, state)
	r.Config.Save()
}

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
	utils.DebugLog("[INFO:showTOC] Showing TOC with current index: %d", index)

	// Combine regular chapters and virtual chapters
	combinedTOCEntries := make([]string, len(r.Book.TOCEntries)+len(r.Book.VirtualTOCEntries))
	copy(combinedTOCEntries, r.Book.TOCEntries)
	copy(combinedTOCEntries[len(r.Book.TOCEntries):], r.Book.VirtualTOCEntries)

	selectedIndex, err := r.UI.ShowTOC(combinedTOCEntries, index)

	if err != nil {
		utils.DebugLog("[ERROR:showTOC] Error showing TOC: %v", err)
		r.UI.SetStatus(fmt.Sprintf("Error showing TOC: %v", err))
		return
	}

	if selectedIndex >= 0 && selectedIndex < len(combinedTOCEntries) {
		// Determine if it's a regular chapter or a virtual chapter
		if selectedIndex < len(r.Book.TOCEntries) {
			// Regular chapter
			utils.DebugLog("[INFO:showTOC] Selected regular chapter at index: %d", selectedIndex)
			err := r.readChapter(selectedIndex, 0)
			if err != nil {
				utils.DebugLog("[ERROR:showTOC] Error reading regular chapter: %v", err)
				r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
			}
		} else {
			// Virtual chapter
			virtualIndex := selectedIndex - len(r.Book.TOCEntries)
			utils.DebugLog("[INFO:showTOC] Selected virtual chapter at index: %d (virtual index: %d)", selectedIndex, virtualIndex)
			err := r.readVirtualChapter(virtualIndex)
			if err != nil {
				utils.DebugLog("[ERROR:showTOC] Error reading virtual chapter: %v", err)
				r.UI.SetStatus(fmt.Sprintf("Error reading virtual chapter: %v", err))
			}
		}
	} else {
		utils.DebugLog("[WARN:showTOC] Invalid selection index: %d", selectedIndex)
	}
}

// readVirtualChapter reads a virtual chapter
func (r *Reader) readVirtualChapter(virtualIndex int) error {
	utils.DebugLog("[INFO:readVirtualChapter] Reading virtual chapter index: %d", virtualIndex)

	if virtualIndex < 0 || virtualIndex >= len(r.Book.VirtualContents) {
		utils.DebugLog("[ERROR:readVirtualChapter] Virtual chapter index out of range: %d", virtualIndex)
		return fmt.Errorf("virtual chapter index out of range")
	}

	virtualContent := r.Book.VirtualContents[virtualIndex]
	utils.DebugLog("[INFO:readVirtualChapter] Virtual content: FilePath=%s, Fragment=%s", virtualContent.FilePath, virtualContent.Fragment)

	// Find the corresponding file index
	fileIndex := -1
	for i, content := range r.Book.Contents {
		if content == virtualContent.FilePath {
			fileIndex = i
			utils.DebugLog("[INFO:readVirtualChapter] Found matching content at index %d", i)
			break
		}
	}

	if fileIndex == -1 {
		utils.DebugLog("[ERROR:readVirtualChapter] File not found for virtual chapter: %s", virtualContent.FilePath)
		return fmt.Errorf("file not found for virtual chapter")
	}

	// Read chapter content
	content, err := r.Book.GetChapterContent(fileIndex)
	if err != nil {
		utils.DebugLog("[ERROR:readVirtualChapter] Error getting chapter content: %v", err)
		return err
	}

	// Parse the HTML content
	htmlParser := parser.NewHTMLParser()
	err = htmlParser.Parse(content)
	if err != nil {
		utils.DebugLog("[ERROR:readVirtualChapter] Error parsing HTML content: %v", err)
		return err
	}

	// Format the lines of text
	lines := htmlParser.FormatLines(r.UI.Width)
	text := strings.Join(lines, "\n")

	// Find anchor position
	utils.DebugLog("[INFO:readVirtualChapter] Looking for anchor: %s", virtualContent.Fragment)
	anchorPattern := fmt.Sprintf(`id="%s"`, virtualContent.Fragment)
	anchorIndex := strings.Index(content, anchorPattern)
	if anchorIndex == -1 {
		// Try other possible anchor formats
		utils.DebugLog("[INFO:readVirtualChapter] Trying alternative anchor format with id='%s'", virtualContent.Fragment)
		anchorPattern = fmt.Sprintf(`id='%s'`, virtualContent.Fragment)
		anchorIndex = strings.Index(content, anchorPattern)
	}
	if anchorIndex == -1 {
		// Try name attribute
		utils.DebugLog("[INFO:readVirtualChapter] Trying name attribute for anchor: %s", virtualContent.Fragment)
		anchorPattern = fmt.Sprintf(`name="%s"`, virtualContent.Fragment)
		anchorIndex = strings.Index(content, anchorPattern)
	}
	if anchorIndex == -1 {
		// Try other possible name attribute formats
		utils.DebugLog("[INFO:readVirtualChapter] Trying alternative name format for anchor: %s", virtualContent.Fragment)
		anchorPattern = fmt.Sprintf(`name='%s'`, virtualContent.Fragment)
		anchorIndex = strings.Index(content, anchorPattern)
	}

	// If anchor is found, calculate the corresponding text position
	var startPos float64 = 0
	if anchorIndex != -1 {
		utils.DebugLog("[INFO:readVirtualChapter] Anchor found at position: %d", anchorIndex)
		// Calculate the proportion of text before the anchor
		beforeAnchor := content[:anchorIndex]
		startPos = float64(len(beforeAnchor)) / float64(len(content))
	} else {
		utils.DebugLog("[WARN:readVirtualChapter] Anchor not found, starting from beginning")
	}

	// Set current chapter
	r.CurrentChapter = fileIndex

	// Display chapter content
	r.UI.TextArea.Clear()
	r.UI.TextArea.SetText(text)

	// Set status bar
	statusText := fmt.Sprintf("Reading chapter %d of %d: %s",
		virtualIndex+1, len(r.Book.VirtualTOCEntries), r.Book.VirtualTOCEntries[virtualIndex])
	r.UI.SetStatus(statusText)

	// Extract images
	r.UI.Images = htmlParser.GetImages()

	// Scroll to anchor position
	lines = strings.Split(text, "\n")
	if startPos > 0 {
		lineCount := len(lines)
		if lineCount > 0 {
			scrollPos := int(float64(lineCount) * startPos)
			utils.DebugLog("[INFO:readVirtualChapter] Scrolling to position: %d of %d lines", scrollPos, lineCount)
			r.UI.TextArea.ScrollTo(scrollPos, 0)
		}
	} else {
		r.UI.TextArea.ScrollToBeginning()
	}

	utils.DebugLog("[INFO:readVirtualChapter] Successfully read virtual chapter %d", virtualIndex)
	return nil
}

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
					r.UI.SetStatus(fmt.Sprintf("Pattern not found: %s", r.UI.SearchPattern))
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

	// Find the next occurrence
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
		r.UI.SetStatus(fmt.Sprintf("Pattern not found: %s", r.UI.SearchPattern))
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
		r.UI.SetStatus(fmt.Sprintf("Pattern not found: %s", r.UI.SearchPattern))
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
				return fmt.Sprintf("[yellow:red]%s[-:-]", match)
			})
			fmt.Fprintln(r.UI.TextArea, highlightedLine)
		}
	}
}

// nextChapter moves to the next chapter
func (r *Reader) nextChapter(index int, pos int, pctg float64) {
	utils.DebugLog("[INFO:nextChapter] Moving to next chapter from index: %d", index)

	// Check if we're currently in a virtual chapter
	virtualIndex, inVirtualChapter := r.getCurrentVirtualChapter()

	if inVirtualChapter {
		// If we're in a virtual chapter, move to the next virtual chapter or the next regular chapter
		if virtualIndex < len(r.Book.VirtualContents)-1 {
			// Move to the next virtual chapter
			utils.DebugLog("[INFO:nextChapter] Moving to next virtual chapter: %d", virtualIndex+1)
			err := r.readVirtualChapter(virtualIndex + 1)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading next virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading next virtual chapter: %v", err))
			}
		} else if index < len(r.Book.Contents)-1 {
			// Move to the next regular chapter
			utils.DebugLog("[INFO:nextChapter] Moving to next regular chapter: %d", index+1)
			err := r.readChapter(index+1, 0)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading next chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading next chapter: %v", err))
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
			}
		} else if len(r.Book.VirtualContents) > 0 {
			// If there are virtual chapters, move to the first one
			utils.DebugLog("[INFO:nextChapter] Moving to first virtual chapter")
			err := r.readVirtualChapter(0)
			if err != nil {
				utils.DebugLog("[ERROR:nextChapter] Error reading virtual chapter: %v", err)
				r.UI.StatusBar.SetText(fmt.Sprintf("Error reading virtual chapter: %v", err))
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

// getCurrentVirtualChapter gets the current virtual chapter index if we're in a virtual chapter
func (r *Reader) getCurrentVirtualChapter() (int, bool) {
	// Check if we're in a virtual chapter by examining the status bar text
	status := r.UI.StatusBar.GetText(false)

	// Look for a pattern like "Reading chapter X of Y: Chapter Title"
	re := regexp.MustCompile(`Reading chapter (\d+) of (\d+): (.+)`)
	matches := re.FindStringSubmatch(status)

	if len(matches) < 4 {
		return -1, false
	}

	_, err := strconv.Atoi(matches[1])
	if err != nil {
		utils.DebugLog("[ERROR:getCurrentVirtualChapter] Error parsing chapter number: %v", err)
		return -1, false
	}

	_, err = strconv.Atoi(matches[2])
	if err != nil {
		utils.DebugLog("[ERROR:getCurrentVirtualChapter] Error parsing total chapters: %v", err)
		return -1, false
	}

	chapterTitle := matches[3]

	// Check if this matches any virtual chapter title
	for i, title := range r.Book.VirtualTOCEntries {
		if title == chapterTitle {
			utils.DebugLog("[INFO:getCurrentVirtualChapter] Found matching virtual chapter at index %d", i)
			return i, true
		}
	}

	return -1, false
}

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

// openImage opens an image
func (r *Reader) openImage() {
	// Get the current chapter
	index, err := r.getCurrentChapter()
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
		return
	}

	// Check if we have images in the UI
	if len(r.UI.Images) == 0 {
		// Fallback: parse the chapter content to get images
		content, err := r.Book.GetChapterContent(index)
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
			return
		}

		// Parse the HTML content
		htmlParser := parser.NewHTMLParser()
		err = htmlParser.Parse(content)
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error parsing HTML: %v", err))
			return
		}

		// Get the images
		r.UI.Images = htmlParser.GetImages()
	}

	// Check if we have any images
	if len(r.UI.Images) == 0 {
		r.UI.SetStatus("No images found in this chapter")
		return
	}

	// Use the ShowImageSelect function to let the user select an image by number
	r.UI.ShowImageSelect(r.UI.Images, func(imagePath string) {
		if imagePath == "" {
			r.UI.SetStatus("No image selected")
			return
		}

		// Resolve the image path
		if index < 0 || index >= len(r.Book.Contents) {
			r.UI.SetStatus(fmt.Sprintf("Invalid chapter index: %d", index))
			return
		}

		chapterPath := r.Book.Contents[index]
		chapterDir := filepath.Dir(chapterPath)
		resolvedPath := filepath.Join(chapterDir, imagePath)

		// Extract the image to a temporary file
		tempFile, err := extractImage(r.Book, resolvedPath)
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error extracting image: %v", err))
			return
		}
		defer os.Remove(tempFile)

		// Open the image using the system's default image viewer
		err = r.UI.OpenImage(tempFile)
		if err != nil {
			utils.DebugLog("[ERROR:openImage] Error opening image: %v", err)
			r.UI.SetStatus(fmt.Sprintf("Error opening image: %v", err))
			return
		}
	})
}

// toggleWidth toggles the width between 80 and the terminal width
func (r *Reader) toggleWidth(index int, pos int, pctg float64) {
	if r.UI.Width == 80 {
		r.UI.Width = pos
		utils.DebugLog("[INFO:toggleWidth] Toggled to terminal width")
	} else {
		r.UI.Width = 80
		utils.DebugLog("[INFO:toggleWidth] Toggled to 80 columns")
	}

	// Re-read the chapter
	err := r.readChapter(index, 0)
	if err != nil {
		r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
	}
}

// markPosition marks the current position
func (r *Reader) markPosition(index int, pos int, pctg float64) {
	// Get the current position
	row, _ := r.UI.TextArea.GetScrollOffset()
	text := r.UI.TextArea.GetText(false)
	lines := strings.Split(text, "\n")
	pctg = float64(row) / float64(len(lines))

	// Wait for a key
	r.UI.SetStatus("Mark position (1-9): ")
	r.UI.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() >= '1' && event.Rune() <= '9' {
			// Mark the position
			r.JumpList[event.Rune()] = [4]interface{}{index, r.UI.Width, row, pctg}
			r.UI.SetStatus(fmt.Sprintf("Position marked as %c", event.Rune()))
			return nil
		}
		return event
	})
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

// getCurrentChapter gets the current chapter
func (r *Reader) getCurrentChapter() (int, error) {
	// Use the CurrentChapter field directly
	if r.CurrentChapter >= 0 && r.CurrentChapter < len(r.Book.Contents) {
		return r.CurrentChapter, nil
	}

	// Fallback: try to get the current chapter from the status bar
	status := r.UI.StatusBar.GetText(false)
	re := regexp.MustCompile(`Reading chapter (\d+) of`)
	matches := re.FindStringSubmatch(status)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not determine current chapter")
	}

	// Parse the chapter number
	chapter, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	return chapter - 1, nil
}

// extractImage extracts an image from the EPUB file to a temporary file
func extractImage(book *epub.Epub, imagePath string) (string, error) {
	// Validate inputs
	if book == nil || book.File == nil {
		return "", fmt.Errorf("invalid book or zip file")
	}

	if imagePath == "" {
		return "", fmt.Errorf("empty image path")
	}

	// Open the image file
	imageFile, err := book.File.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("image file not found in EPUB: %v", err)
	}
	defer imageFile.Close()

	// Check if running in WSL
	isWSL := false
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		isWSL = true
	}

	var tempFile *os.File
	if isWSL {
		// In WSL, use a Windows-accessible temp directory
		// First try to use the Windows temp directory
		winTempDir := "/mnt/c/Windows/Temp"
		if _, err := os.Stat(winTempDir); err == nil {
			utils.DebugLog("[INFO:extractImage] Using Windows temp directory: %s", winTempDir)
			tempFile, err = os.CreateTemp(winTempDir, "goread-image-*.png")
			if err != nil {
				utils.DebugLog("[WARN:extractImage] Failed to create temp file in Windows temp directory: %v", err)
				// Fall back to default temp directory
				tempFile, err = os.CreateTemp("", "goread-image-*.png")
				if err != nil {
					return "", fmt.Errorf("failed to create temp file: %v", err)
				}
			}
		} else {
			// Try user's home directory in Windows
			homeDir := "/mnt/c/Users"
			if _, err := os.Stat(homeDir); err == nil {
				// Try to find a user directory
				entries, err := os.ReadDir(homeDir)
				if err == nil && len(entries) > 0 {
					// Use the first user directory found
					for _, entry := range entries {
						if entry.IsDir() && entry.Name() != "Public" && entry.Name() != "Default" && entry.Name() != "All Users" {
							userTempDir := filepath.Join(homeDir, entry.Name(), "AppData", "Local", "Temp")
							if _, err := os.Stat(userTempDir); err == nil {
								utils.DebugLog("[INFO:extractImage] Using Windows user temp directory: %s", userTempDir)
								tempFile, err = os.CreateTemp(userTempDir, "goread-image-*.png")
								if err == nil {
									break
								}
								utils.DebugLog("[WARN:extractImage] Failed to create temp file in Windows user temp directory: %v", err)
							}
						}
					}
				}
			}

			// If still no temp file, fall back to default
			if tempFile == nil {
				utils.DebugLog("[INFO:extractImage] Falling back to default temp directory")
				tempFile, err = os.CreateTemp("", "goread-image-*.png")
				if err != nil {
					return "", fmt.Errorf("failed to create temp file: %v", err)
				}
			}
		}
	} else {
		// Create a temporary file in the default location
		tempFile, err = os.CreateTemp("", "goread-image-*.png")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %v", err)
		}
	}
	defer tempFile.Close()

	utils.DebugLog("[INFO:extractImage] Created temp file: %s", tempFile.Name())

	// Copy the image to the temporary file
	n, err := io.Copy(tempFile, imageFile)
	if err != nil {
		os.Remove(tempFile.Name()) // Clean up on error
		return "", fmt.Errorf("failed to copy image data: %v", err)
	}

	if n == 0 {
		os.Remove(tempFile.Name()) // Clean up on error
		return "", fmt.Errorf("no data copied from image file")
	}

	return tempFile.Name(), nil
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
