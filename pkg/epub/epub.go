package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ray-d-song/goread/pkg/utils"
)

// XML namespaces used in EPUB files
var namespaces = map[string]string{
	"DAISY": "http://www.daisy.org/z3986/2005/ncx/",
	"OPF":   "http://www.idpf.org/2007/opf",
	"CONT":  "urn:oasis:names:tc:opendocument:xmlns:container",
	"XHTML": "http://www.w3.org/1999/xhtml",
	"EPUB":  "http://www.idpf.org/2007/ops",
}

// Epub represents an EPUB book
type Epub struct {
	Path              string
	File              *zip.ReadCloser
	RootFile          string
	RootDir           string
	Version           string
	TOC               string
	Contents          []string
	TOCEntries        []string
	VirtualContents   []VirtualContent // Virtual chapter content, for handling multiple chapters in the same HTML file
	VirtualTOCEntries []string         // Virtual chapter table of contents entries
}

// VirtualContent represents a virtual chapter content
type VirtualContent struct {
	FilePath string // Actual file path
	Fragment string // Anchor in the file
}

// Metadata represents EPUB metadata
type Metadata struct {
	Title       string
	Creator     string
	Publisher   string
	Language    string
	Identifier  string
	Date        string
	Description string
	Rights      string
	OtherMeta   [][]string
}

// Container represents the container.xml file
type Container struct {
	XMLName   xml.Name   `xml:"container"`
	RootFiles []RootFile `xml:"rootfiles>rootfile"`
}

// RootFile represents a rootfile in container.xml
type RootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// Package represents the package element in the OPF file
type Package struct {
	XMLName  xml.Name       `xml:"package"`
	Version  string         `xml:"version,attr"`
	Metadata []MetadataItem `xml:"metadata>*"`
	Manifest []ManifestItem `xml:"manifest>item"`
	Spine    []SpineItem    `xml:"spine>itemref"`
}

// MetadataItem represents a metadata item in the OPF file
type MetadataItem struct {
	XMLName xml.Name
	Content string `xml:",chardata"`
}

// ManifestItem represents an item in the manifest
type ManifestItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// SpineItem represents an itemref in the spine
type SpineItem struct {
	IDRef string `xml:"idref,attr"`
}

// NCX represents the NCX file for EPUB 2.0
type NCX struct {
	XMLName   xml.Name   `xml:"ncx"`
	NavPoints []NavPoint `xml:"navMap>navPoint"`
}

// NavPoint represents a navigation point in the NCX
type NavPoint struct {
	XMLName   xml.Name   `xml:"navPoint"`
	ID        string     `xml:"id,attr"`
	PlayOrder string     `xml:"playOrder,attr"`
	NavLabel  NavLabel   `xml:"navLabel"`
	Content   Content    `xml:"content"`
	NavPoints []NavPoint `xml:"navPoint"`
}

// NavLabel represents a navigation label
type NavLabel struct {
	Text string `xml:"text"`
}

// Content represents content in a navigation point
type Content struct {
	Src string `xml:"src,attr"`
}

// Nav represents the navigation document for EPUB 3.0
type Nav struct {
	XMLName  xml.Name  `xml:"html"`
	NavLinks []NavLink `xml:"body>nav>ol>li>a"`
}

// NavLink represents a navigation link
type NavLink struct {
	XMLName xml.Name `xml:"a"`
	Href    string   `xml:"href,attr"`
	Text    string   `xml:",chardata"`
}

// NewEpub creates a new Epub instance
func NewEpub(filePath string) (*Epub, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.OpenReader(absPath)
	if err != nil {
		return nil, err
	}

	epub := &Epub{
		Path: absPath,
		File: zipReader,
	}

	// Parse container.xml to find the rootfile
	err = epub.parseContainer()
	if err != nil {
		return nil, err
	}

	// Parse the rootfile to get the TOC and spine
	err = epub.parseRootFile()
	if err != nil {
		return nil, err
	}

	// Initialize the content and TOC entries
	err = epub.initialize()
	if err != nil {
		return nil, err
	}

	return epub, nil
}

// parseContainer parses the container.xml file to find the rootfile
func (e *Epub) parseContainer() error {
	var container Container

	containerFile, err := e.File.Open("META-INF/container.xml")
	if err != nil {
		return err
	}
	defer containerFile.Close()

	decoder := xml.NewDecoder(containerFile)
	err = decoder.Decode(&container)
	if err != nil {
		return err
	}

	if len(container.RootFiles) == 0 {
		return fmt.Errorf("no rootfile found in container.xml")
	}

	e.RootFile = container.RootFiles[0].FullPath
	e.RootDir = filepath.Dir(e.RootFile)
	if e.RootDir != "" {
		e.RootDir += "/"
	}

	return nil
}

// parseRootFile parses the rootfile to get the TOC and spine
func (e *Epub) parseRootFile() error {
	var pkg Package

	rootFile, err := e.File.Open(e.RootFile)
	if err != nil {
		return err
	}
	defer rootFile.Close()

	decoder := xml.NewDecoder(rootFile)
	err = decoder.Decode(&pkg)
	if err != nil {
		return err
	}

	e.Version = pkg.Version

	// Find the TOC file
	tocFound := false

	// First try to find the NCX file (works for both EPUB 2.0 and some EPUB 3.0)
	for _, item := range pkg.Manifest {
		if item.MediaType == "application/x-dtbncx+xml" {
			// Use the correct path for the TOC file
			if e.RootDir != "" {
				e.TOC = e.RootDir + item.Href
			} else {
				e.TOC = item.Href
			}
			tocFound = true
			break
		}
	}

	// If no NCX file found and it's EPUB 3.0, try to find the navigation document
	if !tocFound && e.Version == "3.0" {
		for _, item := range pkg.Manifest {
			if item.Properties == "nav" {
				// Use the correct path for the TOC file
				if e.RootDir != "" {
					e.TOC = e.RootDir + item.Href
				} else {
					e.TOC = item.Href
				}
				tocFound = true
				break
			}
		}
	}

	return nil
}

// initialize initializes the content and TOC entries
func (e *Epub) initialize() error {
	var pkg Package

	rootFile, err := e.File.Open(e.RootFile)
	if err != nil {
		return err
	}
	defer rootFile.Close()

	decoder := xml.NewDecoder(rootFile)
	err = decoder.Decode(&pkg)
	if err != nil {
		return err
	}

	// Create a map of manifest items
	manifestItems := make(map[string]ManifestItem)
	for _, item := range pkg.Manifest {
		if item.MediaType != "application/x-dtbncx+xml" && item.Properties != "nav" {
			manifestItems[item.ID] = item
		}
	}

	// Try to get chapter information from TOC
	contentsFromTOC, err := e.getContentsFromTOC()
	if err != nil || len(contentsFromTOC) == 0 {
		utils.DebugLog("[INFO:initialize] Failed to get contents from TOC or no contents found, falling back to spine")

		// If getting from TOC fails, use spine
		// Get the spine items
		for _, spineItem := range pkg.Spine {
			if item, ok := manifestItems[spineItem.IDRef]; ok {
				decodedHref, err := url.QueryUnescape(item.Href)
				if err != nil {
					decodedHref = item.Href
				}
				e.Contents = append(e.Contents, e.RootDir+decodedHref)
				delete(manifestItems, spineItem.IDRef)
			}
		}
	} else {
		// Use contents from TOC
		utils.DebugLog("[INFO:initialize] Using contents from TOC, found %d items", len(contentsFromTOC))
		e.Contents = contentsFromTOC
	}

	// Parse the TOC to get the TOC entries
	err = e.parseTOC()
	if err != nil {
		return err
	}

	// Process virtual chapters
	err = e.processVirtualChapters()
	if err != nil {
		return err
	}

	return nil
}

// getContentsFromTOC attempts to get content files from the TOC
func (e *Epub) getContentsFromTOC() ([]string, error) {
	utils.DebugLog("[INFO:getContentsFromTOC] Trying to get contents from TOC file: %s", e.TOC)

	// Remove "./" prefix if present
	tocPath := e.TOC
	if strings.HasPrefix(tocPath, "./") {
		tocPath = tocPath[2:]
	}

	tocFile, err := e.File.Open(tocPath)
	if err != nil {
		utils.DebugLog("[ERROR:getContentsFromTOC] Error opening TOC file: %v", err)
		return nil, err
	}
	defer tocFile.Close()

	// Determine the TOC file type based on extension or content
	isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")
	utils.DebugLog("[INFO:getContentsFromTOC] TOC file is NCX: %v", isNCX)

	var contents []string
	var contentPaths []string
	var contentMap = make(map[string]bool) // Used for deduplication

	if isNCX {
		// Parse as NCX file (EPUB 2.0 style)
		var ncx NCX
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&ncx)
		if err != nil {
			utils.DebugLog("[ERROR:getContentsFromTOC] Error decoding NCX: %v", err)
			return nil, err
		}

		utils.DebugLog("[INFO:getContentsFromTOC] Number of navPoints: %d", len(ncx.NavPoints))
		// Process all navPoints
		for i, navPoint := range ncx.NavPoints {
			decodedSrc, err := url.QueryUnescape(navPoint.Content.Src)
			if err != nil {
				decodedSrc = navPoint.Content.Src
			}

			// Split the src into file path and fragment
			filePath, _ := splitPathAndFragment(decodedSrc)
			utils.DebugLog("[INFO:getContentsFromTOC] NavPoint[%d]: Label=%s, Src=%s, FilePath=%s",
				i, navPoint.NavLabel.Text, decodedSrc, filePath)

			// Add to content paths list, ensuring no duplicates
			if filePath != "" && !contentMap[filePath] {
				contentPaths = append(contentPaths, filePath)
				contentMap[filePath] = true
			}
		}
	} else {
		// Parse as navigation document (EPUB 3.0 style)
		var nav Nav
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&nav)
		if err != nil {
			utils.DebugLog("[ERROR:getContentsFromTOC] Error decoding Nav: %v", err)
			return nil, err
		}

		utils.DebugLog("[INFO:getContentsFromTOC] Number of navLinks: %d", len(nav.NavLinks))
		// Process all navLinks
		for i, navLink := range nav.NavLinks {
			decodedHref, err := url.QueryUnescape(navLink.Href)
			if err != nil {
				decodedHref = navLink.Href
			}

			// Split the href into file path and fragment
			filePath, _ := splitPathAndFragment(decodedHref)
			utils.DebugLog("[INFO:getContentsFromTOC] NavLink[%d]: Text=%s, Href=%s, FilePath=%s",
				i, navLink.Text, decodedHref, filePath)

			// Add to content paths list, ensuring no duplicates
			if filePath != "" && !contentMap[filePath] {
				contentPaths = append(contentPaths, filePath)
				contentMap[filePath] = true
			}
		}
	}

	// Convert relative paths to full paths
	for _, path := range contentPaths {
		fullPath := e.RootDir + path
		contents = append(contents, fullPath)
		utils.DebugLog("[INFO:getContentsFromTOC] Added content file: %s", fullPath)
	}

	// If we found content files, return them
	if len(contents) > 0 {
		utils.DebugLog("[INFO:getContentsFromTOC] Successfully extracted %d content files from TOC", len(contents))
		return contents, nil
	}

	// If no content files were found, return an empty slice
	utils.DebugLog("[WARN:getContentsFromTOC] No content files found in TOC")
	return []string{}, nil
}

// splitPathAndFragment splits a path into the file path and fragment
func splitPathAndFragment(path string) (string, string) {
	parts := strings.SplitN(path, "#", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// GetMetadata returns the metadata of the EPUB
func (e *Epub) GetMetadata() (*Metadata, error) {
	var pkg Package

	rootFile, err := e.File.Open(e.RootFile)
	if err != nil {
		return nil, err
	}
	defer rootFile.Close()

	decoder := xml.NewDecoder(rootFile)
	err = decoder.Decode(&pkg)
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{}

	for _, item := range pkg.Metadata {
		tagName := item.XMLName.Local

		switch tagName {
		case "title":
			metadata.Title = item.Content
		case "creator":
			metadata.Creator = item.Content
		case "publisher":
			metadata.Publisher = item.Content
		case "language":
			metadata.Language = item.Content
		case "identifier":
			metadata.Identifier = item.Content
		case "date":
			metadata.Date = item.Content
		case "description":
			metadata.Description = item.Content
		case "rights":
			metadata.Rights = item.Content
		default:
			metadata.OtherMeta = append(metadata.OtherMeta, []string{tagName, item.Content})
		}
	}

	return metadata, nil
}

// GetChapterContent returns the content of a chapter
func (e *Epub) GetChapterContent(index int) (string, error) {
	if index < 0 || index >= len(e.Contents) {
		return "", fmt.Errorf("chapter index out of range")
	}

	// Remove "./" prefix if present
	chapterPath := e.Contents[index]
	if strings.HasPrefix(chapterPath, "./") {
		chapterPath = chapterPath[2:]
	}

	chapterFile, err := e.File.Open(chapterPath)
	if err != nil {
		return "", err
	}
	defer chapterFile.Close()

	content, err := io.ReadAll(chapterFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// Close closes the EPUB file
func (e *Epub) Close() error {
	return e.File.Close()
}

// ResolvePath resolves a relative path against a base path
func ResolvePath(basePath, relativePath string) string {
	baseDir := filepath.Dir(basePath)
	if baseDir == "." {
		return relativePath
	}

	// Handle paths with fragments
	fragment := ""
	if idx := strings.LastIndex(relativePath, "#"); idx != -1 {
		fragment = relativePath[idx:]
		relativePath = relativePath[:idx]
	}

	resolved := filepath.Join(baseDir, relativePath)

	return resolved + fragment
}
