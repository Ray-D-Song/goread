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
	IsShadow bool
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

	// check if OEBPS directory exists
	hasOEBPS := false
	for _, file := range e.File.File {
		if strings.HasPrefix(file.Name, "OEBPS/") {
			hasOEBPS = true
			break
		}
	}

	// if OEBPS directory exists but current RootFile is not in OEBPS
	if hasOEBPS && !strings.HasPrefix(e.RootFile, "OEBPS/") {
		// try to find the same file in OEBPS
		oebpsRootFile := "OEBPS/" + filepath.Base(e.RootFile)
		for _, file := range e.File.File {
			if file.Name == oebpsRootFile {
				utils.DebugLog("[INFO:parseContainer] Found rootfile in OEBPS directory: %s", oebpsRootFile)
				e.RootFile = oebpsRootFile
				e.RootDir = filepath.Dir(e.RootFile)
				if e.RootDir != "" {
					e.RootDir += "/"
				}
				break
			}
		}
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
	if err := e.generateTOC(pkg.Spine, manifestItems); err != nil {
		return err
	}

	return nil
}

func (e *Epub) generateTOC(spine []SpineItem, manifestItems map[string]ManifestItem) error {
	utils.DebugLog("[INFO:GenerateTOC] Trying to get contents from TOC file: %s", e.TOCPath)

	// Initialize TOC from spine first (complete list)
	e.TOC = utils.NewDList[TOCValue]()

	// Generate a map to track paths that are in the TOC file
	tocPaths := make(map[string]bool)

	// First, parse the TOC file to know which paths are in the official TOC
	// Remove "./" prefix if present
	tocPath := e.TOCPath
	if strings.HasPrefix(tocPath, "./") {
		tocPath = tocPath[2:]
	}

	// Try to open TOC file
	tocFile, err := e.File.Open(tocPath)
	if err != nil {
		// Try to find the TOC file in OEBPS directory
		if !strings.HasPrefix(tocPath, "OEBPS/") {
			oebpsTocPath := "OEBPS/" + tocPath
			utils.DebugLog("[INFO:GenerateTOC] Trying to find TOC file in OEBPS directory: %s", oebpsTocPath)

			oebpsTocFile, oebpsErr := e.File.Open(oebpsTocPath)
			if oebpsErr == nil {
				utils.DebugLog("[INFO:GenerateTOC] Found TOC file in OEBPS directory")
				tocFile = oebpsTocFile
				tocPath = oebpsTocPath
				err = nil
			} else {
				utils.DebugLog("[ERROR:GenerateTOC] Error opening TOC file: %v", err)
				// Continue with empty tocPaths map - all items will be marked as shadow
			}
		} else {
			utils.DebugLog("[ERROR:GenerateTOC] Error opening TOC file: %v", err)
			// Continue with empty tocPaths map - all items will be marked as shadow
		}
	}

	// If we successfully opened the TOC file, parse it to collect paths
	if err == nil {
		defer tocFile.Close()

		// Determine the TOC file type based on extension or content
		isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")
		utils.DebugLog("[INFO:GenerateTOC] TOC file is NCX: %v", isNCX)

		if isNCX {
			// Parse as NCX file (EPUB 2.0 style)
			var ncx NCX
			decoder := xml.NewDecoder(tocFile)
			err = decoder.Decode(&ncx)
			if err != nil {
				utils.DebugLog("[ERROR:GenerateTOC] Error decoding NCX: %v", err)
				// Continue with empty tocPaths map - all items will be marked as shadow
			} else {
				// Extract all navPoint paths recursively
				collectNavPointPaths(&ncx.NavPoints, tocPaths)
			}
		} else {
			// Parse as navigation document (EPUB 3.0 style)
			var nav Nav
			decoder := xml.NewDecoder(tocFile)
			err = decoder.Decode(&nav)
			if err != nil {
				utils.DebugLog("[ERROR:GenerateTOC] Error decoding Nav: %v", err)
				// Continue with empty tocPaths map - all items will be marked as shadow
			} else {
				// Extract all nav link paths
				for _, link := range nav.NavLinks {
					path, _ := splitPathAndFragment(link.Href)
					tocPaths[path] = true
				}
			}
		}
	}

	// Now generate the full TOC from spine
	var prev *utils.DItem[TOCValue]

	// Create temporary TOC from the official TOC file (if available)
	var tempTOC *utils.DList[TOCValue]
	if err == nil {
		tempTOC = utils.NewDList[TOCValue]()
		tocFile, _ := e.File.Open(tocPath) // Reopen the file
		defer tocFile.Close()

		isNCX := strings.HasSuffix(strings.ToLower(tocPath), ".ncx")

		if isNCX {
			var ncx NCX
			decoder := xml.NewDecoder(tocFile)
			_ = decoder.Decode(&ncx)

			// Process all nav points recursively into tempTOC
			var tempPrev *utils.DItem[TOCValue]
			for i := range ncx.NavPoints {
				tempPrev = processNestedNavPoints(ncx.NavPoints[i], tempTOC, tempPrev, 0, uuid.New().String())
			}
		} else {
			var nav Nav
			decoder := xml.NewDecoder(tocFile)
			_ = decoder.Decode(&nav)

			// Process nav links into tempTOC
			var tempPrev *utils.DItem[TOCValue]
			for _, link := range nav.NavLinks {
				path, fragment := splitPathAndFragment(link.Href)
				newItem := TOCValue{
					ID:       uuid.New().String(),
					Title:    link.Text,
					Path:     path,
					Fragment: fragment,
					Level:    0,
					IsDir:    false,
					ParentID: "",
					IsShadow: false,
				}
				newNode := tempTOC.Add(newItem, tempPrev)
				tempPrev = newNode
			}
		}
	}

	// Create a map of path -> TOCValue from tempTOC for easy lookup
	pathToTOC := make(map[string]TOCValue)
	if tempTOC != nil {
		for _, item := range tempTOC.Slice {
			pathToTOC[item.Path] = item
		}
	}

	// Now process all spine items
	for _, spineItem := range spine {
		if item, ok := manifestItems[spineItem.IDRef]; ok {
			// Get the href from the manifest item
			href := item.Href

			// See if this path is in the official TOC
			path, fragment := splitPathAndFragment(href)
			_, inTOC := tocPaths[path]

			// Get TOC value from the tempTOC if available
			var title string
			var level int
			var isDir bool

			if tocValue, exists := pathToTOC[path]; exists {
				title = tocValue.Title
				level = tocValue.Level
				isDir = tocValue.IsDir
			} else {
				// Use ID as title if not in TOC
				title = spineItem.IDRef
				level = 0
				isDir = false
			}

			// Create TOC entry
			newItem := TOCValue{
				ID:       uuid.New().String(),
				Title:    title,
				Path:     path,
				Fragment: fragment,
				Level:    level,
				IsDir:    isDir,
				ParentID: "",
				IsShadow: !inTOC, // Mark as shadow if not in the official TOC
			}

			newNode := e.TOC.Add(newItem, prev)
			prev = newNode
		}
	}

	return nil
}

// collectNavPointPaths recursively collects paths from NavPoints
func collectNavPointPaths(navPoints *[]NavPoint, paths map[string]bool) {
	for _, navPoint := range *navPoints {
		path, _ := splitPathAndFragment(navPoint.Content.Src)
		paths[path] = true

		// Process children recursively
		if len(navPoint.NavPoints) > 0 {
			collectNavPointPaths(&navPoint.NavPoints, paths)
		}
	}
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
		IsShadow: false,
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

func (e *Epub) GetChapterIndex(id string) (int, error) {
	for i, toc := range e.TOC.Slice {
		if toc.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("chapter index not found")
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
	var nextTocValue = TOCValue{}
	if index < e.TOC.Len()-1 {
		nextTocValue = e.TOC.Slice[index+1]
	}
	// Remove "./" prefix if present
	chapterPath := tocValue.Path
	if strings.HasPrefix(chapterPath, "./") {
		chapterPath = chapterPath[2:]
	}

	// try to open the chapter file
	chapterFile, err := e.File.Open(chapterPath)
	if err != nil {
		// if failed to open the chapter file, try to find it in OEBPS directory
		if !strings.HasPrefix(chapterPath, "OEBPS/") {
			oebpsPath := "OEBPS/" + chapterPath
			utils.DebugLog("[INFO:GetChapterContents] Trying to find chapter in OEBPS directory: %s", oebpsPath)

			// check if the file exists in OEBPS directory
			oebpsFile, oebpsErr := e.File.Open(oebpsPath)
			if oebpsErr == nil {
				utils.DebugLog("[INFO:GetChapterContents] Found chapter in OEBPS directory")
				chapterFile = oebpsFile
				err = nil
			} else {
				// Try to find the file relative to the RootDir (OPF file's directory)
				if e.RootDir != "" {
					rootDirPath := e.RootDir + chapterPath
					utils.DebugLog("[INFO:GetChapterContents] Trying to find chapter relative to OPF directory: %s", rootDirPath)

					rootDirFile, rootDirErr := e.File.Open(rootDirPath)
					if rootDirErr == nil {
						utils.DebugLog("[INFO:GetChapterContents] Found chapter relative to OPF directory")
						chapterFile = rootDirFile
						err = nil
					} else {
						return nil, err // if not found in RootDir, return the original error
					}
				} else {
					return nil, err // if not found in OEBPS directory, return the original error
				}
			}
		} else {
			// Try to find the file relative to the RootDir (OPF file's directory)
			if e.RootDir != "" && !strings.HasPrefix(chapterPath, e.RootDir) {
				rootDirPath := e.RootDir + strings.TrimPrefix(chapterPath, "OEBPS/")
				utils.DebugLog("[INFO:GetChapterContents] Trying to find chapter relative to OPF directory: %s", rootDirPath)

				rootDirFile, rootDirErr := e.File.Open(rootDirPath)
				if rootDirErr == nil {
					utils.DebugLog("[INFO:GetChapterContents] Found chapter relative to OPF directory")
					chapterFile = rootDirFile
					err = nil
				} else {
					return nil, err // if not found in RootDir, return the original error
				}
			} else {
				return nil, err
			}
		}
	}
	defer chapterFile.Close()

	content, err := io.ReadAll(chapterFile)
	if err != nil {
		return nil, err
	}

	parser := parser.NewHTMLParser()
	// automatically get the next chapter's fragment
	if tocValue.Path == nextTocValue.Path {
		if err := parser.Parse(string(content), tocValue.Fragment, nextTocValue.Fragment); err != nil {
			return nil, err
		}
	} else {
		if err := parser.Parse(string(content), tocValue.Fragment, ""); err != nil {
			return nil, err
		}
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
