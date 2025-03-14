package epub

import (
	"encoding/xml"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ray-d-song/goread/pkg/utils"
)

// parseTOC parses the TOC file to get the TOC entries
func (e *Epub) parseTOC() error {
	e.TOCEntries = make([]string, len(e.Contents))
	for i := range e.TOCEntries {
		e.TOCEntries[i] = "-"
	}

	// Add debug log
	utils.DebugLog("[INFO:parseTOC] Trying to open TOC file: %s", e.TOC)

	// Remove "./" prefix if present
	tocPath := e.TOC
	if strings.HasPrefix(tocPath, "./") {
		tocPath = tocPath[2:]
	}
	utils.DebugLog("[INFO:parseTOC] Using TOC path: %s", tocPath)

	tocFile, err := e.File.Open(tocPath)
	if err != nil {
		// Add debug log
		utils.DebugLog("[ERROR:parseTOC] Error opening TOC file: %v", err)
		return err
	}
	defer tocFile.Close()

	// Determine the TOC file type based on extension or content
	isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")

	// Create a map to store all navPoints/navLinks
	// The key is the file path (without fragment), the value is a slice of [fragment, title] pairs
	tocMap := make(map[string][][]string)

	// Create a mapping to store file path to title mapping (without fragments)
	fileToTitle := make(map[string]string)

	if isNCX {
		// Parse as NCX file (EPUB 2.0 style)
		var ncx NCX
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&ncx)
		if err != nil {
			return err
		}

		// Process all navPoints
		for _, navPoint := range ncx.NavPoints {
			decodedSrc, err := url.QueryUnescape(navPoint.Content.Src)
			if err != nil {
				decodedSrc = navPoint.Content.Src
			}

			// Split the src into file path and fragment
			filePath, fragment := splitPathAndFragment(decodedSrc)

			// Add to the map
			if _, ok := tocMap[filePath]; !ok {
				tocMap[filePath] = make([][]string, 0)
				// Store the first encountered title as the main title for the file
				fileToTitle[filePath] = navPoint.NavLabel.Text
			}
			tocMap[filePath] = append(tocMap[filePath], []string{fragment, navPoint.NavLabel.Text})
		}

		// Match contents to navPoints
		for i, content := range e.Contents {
			baseContent := filepath.Base(content)
			matched := false

			// First try exact match
			for filePath, entries := range tocMap {
				if strings.HasSuffix(content, filePath) {
					// Exact match
					if len(entries) > 0 {
						if entries[0][0] == "" {
							// No fragment, use the first title
							e.TOCEntries[i] = entries[0][1]
						} else {
							// Has fragments, use the main title for the file
							e.TOCEntries[i] = fileToTitle[filePath]
						}
					}
					matched = true
					break
				}
			}

			// If exact match fails, try fuzzy matching based on filename
			if !matched {
				for filePath, entries := range tocMap {
					if strings.Contains(filePath, baseContent) || strings.Contains(baseContent, filepath.Base(filePath)) {
						// If there are multiple fragments, use the main title for the file
						if len(entries) > 0 {
							if entries[0][0] == "" {
								// No fragment, use the first title
								e.TOCEntries[i] = entries[0][1]
							} else {
								// Has fragments, use the main title for the file
								e.TOCEntries[i] = fileToTitle[filePath]
							}
						}
						matched = true
						break
					}
				}
			}
		}
	} else {
		// Parse as navigation document (EPUB 3.0 style)
		var nav Nav
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&nav)
		if err != nil {
			return err
		}

		// Process all navLinks
		for _, navLink := range nav.NavLinks {
			decodedHref, err := url.QueryUnescape(navLink.Href)
			if err != nil {
				decodedHref = navLink.Href
			}

			// Split the href into file path and fragment
			filePath, fragment := splitPathAndFragment(decodedHref)

			// Add to the map
			if _, ok := tocMap[filePath]; !ok {
				tocMap[filePath] = make([][]string, 0)
				// Store the first encountered title as the main title for the file
				fileToTitle[filePath] = navLink.Text
			}
			tocMap[filePath] = append(tocMap[filePath], []string{fragment, navLink.Text})
		}

		// Match contents to navLinks
		for i, content := range e.Contents {
			baseContent := filepath.Base(content)
			matched := false

			// First try exact match
			for filePath, entries := range tocMap {
				if strings.HasSuffix(content, filePath) {
					// Exact match
					if len(entries) > 0 {
						if entries[0][0] == "" {
							// No fragment, use the first title
							e.TOCEntries[i] = entries[0][1]
						} else {
							// Has fragments, use the main title for the file
							e.TOCEntries[i] = fileToTitle[filePath]
						}
					}
					matched = true
					break
				}
			}

			// If exact match fails, try fuzzy matching based on filename
			if !matched {
				for filePath, entries := range tocMap {
					if strings.Contains(filePath, baseContent) || strings.Contains(baseContent, filepath.Base(filePath)) {
						// If there are multiple fragments, use the main title for the file
						if len(entries) > 0 {
							if entries[0][0] == "" {
								// No fragment, use the first title
								e.TOCEntries[i] = entries[0][1]
							} else {
								// Has fragments, use the main title for the file
								e.TOCEntries[i] = fileToTitle[filePath]
							}
						}
						matched = true
						break
					}
				}
			}
		}
	}

	return nil
}
