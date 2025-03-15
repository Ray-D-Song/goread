package epub

import (
	"encoding/xml"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ray-d-song/goread/pkg/utils"
)

// processVirtualChapters processes virtual chapters
func (e *Epub) processVirtualChapters() error {
	// Add debug log
	utils.DebugLog("[INFO:processVirtualChapters] Processing virtual chapters")
	utils.DebugLog("[INFO:processVirtualChapters] TOC file: %s", e.TOC)
	utils.DebugLog("[INFO:processVirtualChapters] Number of contents: %d", len(e.Contents))
	for i, content := range e.Contents {
		utils.DebugLog("[INFO:processVirtualChapters] Content[%d]: %s", i, content)
	}

	// Open the TOC file
	tocPath := e.TOC
	if strings.HasPrefix(tocPath, "./") {
		tocPath = tocPath[2:]
	}

	tocFile, err := e.File.Open(tocPath)
	if err != nil {
		utils.DebugLog("[ERROR:processVirtualChapters] Error opening TOC file: %v", err)
		return err
	}
	defer tocFile.Close()

	// Determine the TOC file type based on extension or content
	isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")
	utils.DebugLog("[INFO:processVirtualChapters] TOC file is NCX: %v", isNCX)

	if isNCX {
		// Parse as NCX file (EPUB 2.0 style)
		var ncx NCX
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&ncx)
		if err != nil {
			utils.DebugLog("[ERROR:processVirtualChapters] Error decoding NCX: %v", err)
			return err
		}

		utils.DebugLog("[INFO:processVirtualChapters] Number of navPoints: %d", len(ncx.NavPoints))
		// Process all navPoints
		for i, navPoint := range ncx.NavPoints {
			decodedSrc, err := url.QueryUnescape(navPoint.Content.Src)
			if err != nil {
				decodedSrc = navPoint.Content.Src
			}

			// Split the src into file path and fragment
			filePath, fragment := splitPathAndFragment(decodedSrc)
			utils.DebugLog("[INFO:processVirtualChapters] NavPoint[%d]: Label=%s, Src=%s, FilePath=%s, Fragment=%s",
				i, navPoint.NavLabel.Text, decodedSrc, filePath, fragment)

			// Skip if no fragment
			if fragment == "" {
				utils.DebugLog("[INFO:processVirtualChapters] Skipping NavPoint[%d] - no fragment", i)
				continue
			}

			// Find the corresponding content file
			contentIndex := -1

			// First try exact suffix match
			for i, content := range e.Contents {
				if strings.HasSuffix(content, filePath) {
					contentIndex = i
					utils.DebugLog("[INFO:processVirtualChapters] Found exact suffix match for %s at index %d", filePath, i)
					break
				}
			}

			// If exact match fails, try partial match
			if contentIndex == -1 {
				baseFilePath := filepath.Base(filePath)
				for i, content := range e.Contents {
					baseContent := filepath.Base(content)
					if strings.Contains(baseContent, baseFilePath) || strings.Contains(baseFilePath, baseContent) {
						contentIndex = i
						utils.DebugLog("[INFO:processVirtualChapters] Found partial match for %s at index %d", filePath, i)
						break
					}
				}
			}

			// Skip if content file not found
			if contentIndex == -1 {
				utils.DebugLog("[WARN:processVirtualChapters] Skipping NavPoint[%d] - content file not found for path: %s", i, filePath)
				continue
			}

			// Add virtual content
			e.VirtualContents = append(e.VirtualContents, VirtualContent{
				FilePath: e.Contents[contentIndex],
				Fragment: fragment,
			})
			e.VirtualTOCEntries = append(e.VirtualTOCEntries, navPoint.NavLabel.Text)
			utils.DebugLog("[INFO:processVirtualChapters] Added virtual chapter: %s (file: %s, fragment: %s)", navPoint.NavLabel.Text, e.Contents[contentIndex], fragment)
		}
	} else {
		// Parse as navigation document (EPUB 3.0 style)
		var nav Nav
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&nav)
		if err != nil {
			utils.DebugLog("[ERROR:processVirtualChapters] Error decoding Nav: %v", err)
			return err
		}

		utils.DebugLog("[INFO:processVirtualChapters] Number of navLinks: %d", len(nav.NavLinks))
		// Process all navLinks
		for i, navLink := range nav.NavLinks {
			decodedHref, err := url.QueryUnescape(navLink.Href)
			if err != nil {
				decodedHref = navLink.Href
			}

			// Split the href into file path and fragment
			filePath, fragment := splitPathAndFragment(decodedHref)
			utils.DebugLog("[INFO:processVirtualChapters] NavLink[%d]: Text=%s, Href=%s, FilePath=%s, Fragment=%s",
				i, navLink.Text, decodedHref, filePath, fragment)

			// Skip if no fragment
			if fragment == "" {
				utils.DebugLog("[INFO:processVirtualChapters] Skipping NavLink[%d] - no fragment", i)
				continue
			}

			// Find the corresponding content file
			contentIndex := -1

			// First try exact suffix match
			for i, content := range e.Contents {
				if strings.HasSuffix(content, filePath) {
					contentIndex = i
					utils.DebugLog("[INFO:processVirtualChapters] Found exact suffix match for %s at index %d", filePath, i)
					break
				}
			}

			// If exact match fails, try partial match
			if contentIndex == -1 {
				baseFilePath := filepath.Base(filePath)
				for i, content := range e.Contents {
					baseContent := filepath.Base(content)
					if strings.Contains(baseContent, baseFilePath) || strings.Contains(baseFilePath, baseContent) {
						contentIndex = i
						utils.DebugLog("[INFO:processVirtualChapters] Found partial match for %s at index %d", filePath, i)
						break
					}
				}
			}

			// Skip if content file not found
			if contentIndex == -1 {
				utils.DebugLog("[WARN:processVirtualChapters] Skipping NavLink[%d] - content file not found for path: %s", i, filePath)
				continue
			}

			// Add virtual content
			e.VirtualContents = append(e.VirtualContents, VirtualContent{
				FilePath: e.Contents[contentIndex],
				Fragment: fragment,
			})
			e.VirtualTOCEntries = append(e.VirtualTOCEntries, navLink.Text)
			utils.DebugLog("[INFO:processVirtualChapters] Added virtual chapter: %s (file: %s, fragment: %s)", navLink.Text, e.Contents[contentIndex], fragment)
		}
	}

	// Add debug log
	utils.DebugLog("[INFO:processVirtualChapters] Found %d virtual chapters", len(e.VirtualContents))
	for i, vc := range e.VirtualContents {
		utils.DebugLog("[INFO:processVirtualChapters] VirtualContent[%d]: Title=%s, FilePath=%s, Fragment=%s",
			i, e.VirtualTOCEntries[i], vc.FilePath, vc.Fragment)
	}

	return nil
}
