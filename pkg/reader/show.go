package reader

import (
	"fmt"

	"github.com/ray-d-song/goread/pkg/utils"
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
