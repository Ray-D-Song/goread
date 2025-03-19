package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/ray-d-song/goread/pkg/parser"
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

type TOCValue struct {
	ID       string
	ParentID string
	Title    string
	Path     string
	Fragment string
	Level    int
	IsDir    bool
}

// Epub represents an EPUB book
type Epub struct {
	Path     string
	TOCPath  string
	File     *zip.ReadCloser
	RootFile string
	RootDir  string
	Version  string
	TOC      *utils.DList[TOCValue]
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
				e.TOCPath = e.RootDir + item.Href
			} else {
				e.TOCPath = item.Href
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
					e.TOCPath = e.RootDir + item.Href
				} else {
					e.TOCPath = item.Href
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
	if err := e.generateTOC(); err != nil {
		return err
	}

	return nil
}

func (e *Epub) generateTOC() error {
	utils.DebugLog("[INFO:GenerateTOC] Trying to get contents from TOC file: %s", e.TOCPath)

	// Remove "./" prefix if present
	tocPath := e.TOCPath
	if strings.HasPrefix(tocPath, "./") {
		tocPath = tocPath[2:]
	}

	tocFile, err := e.File.Open(tocPath)
	if err != nil {
		utils.DebugLog("[ERROR:GenerateTOC] Error opening TOC file: %v", err)
		return err
	}
	defer tocFile.Close()

	// Determine the TOC file type based on extension or content
	isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")
	utils.DebugLog("[INFO:GenerateTOC] TOC file is NCX: %v", isNCX)

	e.TOC = utils.NewDList[TOCValue]()

	if isNCX {
		// Parse as NCX file (EPUB 2.0 style)
		var ncx NCX
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&ncx)
		if err != nil {
			utils.DebugLog("[ERROR:GenerateTOC] Error decoding NCX: %v", err)
			return err
		}
		// Process all nav points recursively
		var prev *utils.DItem[TOCValue]
		for i := range ncx.NavPoints {
			prev = processNestedNavPoints(ncx.NavPoints[i], e.TOC, prev, 0, uuid.New().String())
		}
	} else {
		// Parse as navigation document (EPUB 3.0 style)
		var nav Nav
		decoder := xml.NewDecoder(tocFile)
		err = decoder.Decode(&nav)
		if err != nil {
			utils.DebugLog("[ERROR:GenerateTOC] Error decoding Nav: %v", err)
			return err
		}

		utils.DebugLog("[INFO:GenerateTOC] Number of navLinks: %d", len(nav.NavLinks))

		// Process nav links
		var prev *utils.DItem[TOCValue]
		for _, link := range nav.NavLinks {
			path, fragment := splitPathAndFragment(link.Href)
			newItem := TOCValue{
				Title:    link.Text,
				Path:     path,
				Fragment: fragment,
				Level:    0, // For now we don't handle nested nav links in EPUB 3.0
				IsDir:    false,
				ParentID: "",
			}
			newNode := e.TOC.Add(newItem, prev)
			prev = newNode
		}
	}

	return nil
}

func processNestedNavPoints(navPoint NavPoint, list *utils.DList[TOCValue], prev *utils.DItem[TOCValue], level int, parentID string) *utils.DItem[TOCValue] {
	path, fragment := splitPathAndFragment(navPoint.Content.Src)
	newItem := TOCValue{
		ID:       uuid.New().String(),
		Title:    navPoint.NavLabel.Text,
		Path:     path,
		Fragment: fragment,
		Level:    level,
		IsDir:    len(navPoint.NavPoints) > 0,
		ParentID: parentID,
	}

	newNode := list.Add(newItem, prev)

	// Recursively process child nav points
	for i := range navPoint.NavPoints {
		newNode = processNestedNavPoints(navPoint.NavPoints[i], list, newNode, level+1, newItem.ID)
	}

	return newNode
}

// splitPathAndFragment splits a path into the file path and fragment
func splitPathAndFragment(path string) (string, string) {
	parts := strings.SplitN(path, "#", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// GetChapterContents returns the content of a chapter
// include text lines and images
type ChapterContent struct {
	Lines  []string
	Text   string
	Images []string
}

func (e *Epub) GetChapterContents(index int) (*ChapterContent, error) {
	if index < 0 || index >= e.TOC.Len() {
		return nil, fmt.Errorf("chapter index out of range")
	}

	tocValue := e.TOC.Slice[index]
	// Remove "./" prefix if present
	chapterPath := tocValue.Path
	if strings.HasPrefix(chapterPath, "./") {
		chapterPath = chapterPath[2:]
	}

	chapterFile, err := e.File.Open(chapterPath)
	if err != nil {
		return nil, err
	}
	defer chapterFile.Close()

	content, err := io.ReadAll(chapterFile)
	if err != nil {
		return nil, err
	}

	parser := parser.NewHTMLParser()
	if err := parser.Parse(string(content)); err != nil {
		return nil, err
	}

	return &ChapterContent{
		Lines:  parser.GetLines(),
		Text:   strings.Join(parser.GetLines(), "\n"),
		Images: parser.GetImages(),
	}, nil
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
