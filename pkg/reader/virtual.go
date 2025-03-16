package reader

import (
	"fmt"
	"strings"

	"github.com/ray-d-song/goread/pkg/parser"
	"github.com/ray-d-song/goread/pkg/utils"
)

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

	// Set current chapter
	r.CurrentChapter = fileIndex

	// Create cache keys
	fragment := virtualContent.Fragment

	fileContent, err := r.Book.GetChapterContent(fileIndex)
	if err != nil {
		utils.DebugLog("[ERROR:readVirtualChapter] Error getting chapter content: %v", err)
		return err
	}

	// Initialize HTML parser
	htmlParser := parser.NewHTMLParser()

	utils.DebugLog("[INFO:readVirtualChapter] Looking for anchor: %s", fragment)
	// Get next anchor
	var nextAnchor string
	if virtualIndex < len(r.Book.VirtualContents)-1 {
		nextVirtualContent := r.Book.VirtualContents[virtualIndex+1]
		// Only use the next anchor if it's in the same file
		if nextVirtualContent.FilePath == virtualContent.FilePath {
			nextAnchor = nextVirtualContent.Fragment
		}
	}
	utils.DebugLog("[INFO:readVirtualChapter] Next anchor: %s", nextAnchor)

	// Extract content between anchors
	extractedContent, err := parser.ExtractBetweenAnchors(fileContent, fragment, nextAnchor)
	if err == nil && extractedContent != "" {
		utils.DebugLog("[INFO:readVirtualChapter] Successfully extracted content between anchors")

		// Parse the extracted content to get images
		err = htmlParser.Parse(extractedContent)
		if err != nil {
			utils.DebugLog("[WARN:readVirtualChapter] Error parsing extracted content: %v", err)
			// If parsing the extracted content fails, try parsing the entire file content
			err = htmlParser.Parse(fileContent)
			if err != nil {
				utils.DebugLog("[ERROR:readVirtualChapter] Error parsing file content: %v", err)
				// Continue execution even if parsing fails, just might not have images
			}
		}
	} else {
		utils.DebugLog("[WARN:readVirtualChapter] Failed to extract content between anchors %s and %s, error: %v", fragment, nextAnchor, err)
		// If extraction fails, use the entire file content
		extractedContent = fileContent

		// Parse the entire file content
		err = htmlParser.Parse(fileContent)
		if err != nil {
			utils.DebugLog("[ERROR:readVirtualChapter] Error parsing file content: %v", err)
			// Continue execution even if parsing fails, just might not have images
		}
	}

	// Format the extracted content (optional)
	formattedLines := htmlParser.FormatLines(r.UI.Width)
	text := strings.Join(formattedLines, "\n")

	// If formatted content is empty, use the raw extracted content
	if strings.TrimSpace(text) == "" {
		text = extractedContent
	}

	// Display chapter content
	r.UI.TextArea.Clear()
	r.UI.TextArea.SetText(text)

	// Set status bar
	statusText := fmt.Sprintf("Reading chapter %d of %d: %s",
		virtualIndex+1, len(r.Book.VirtualTOCEntries), r.Book.VirtualTOCEntries[virtualIndex])
	r.UI.SetStatus(statusText)

	// Extract images
	r.UI.Images = htmlParser.GetImages()

	r.UI.TextArea.ScrollToBeginning()

	utils.DebugLog("[INFO:readVirtualChapter] Successfully read virtual chapter %d", virtualIndex)
	return nil
}
