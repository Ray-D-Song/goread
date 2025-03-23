package reader

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

	// Cache fields
	TempDir string // Temporary directory for image files
}

// NewReader creates a new Reader instance
func NewReader(book *epub.Epub, cfg *config.Config, filePath string) *Reader {
	// Create a temporary directory for image files
	tempDir, err := os.MkdirTemp("", "goread-images-*")
	if err != nil {
		utils.DebugLog("[ERROR:NewReader] Failed to create temp directory: %v", err)
		// Continue without a dedicated temp directory
		tempDir = ""
	} else {
		// Register cleanup on program exit
		utils.DebugLog("[INFO:NewReader] Created temp directory: %s", tempDir)
	}

	return &Reader{
		Book:           book,
		Config:         cfg,
		FilePath:       filePath,
		UI:             ui.NewUI(),
		JumpList:       make(map[rune][4]interface{}),
		CurrentChapter: 0,
		TempDir:        tempDir,
	}
}

var InitialCapture func(event *tcell.EventKey) *tcell.EventKey

// Run runs the reader
func (r *Reader) Run(index int, width int, pos int, pctg float64) {
	// Initialize the UI
	r.UI.SetWidth(width)

	// Clean up temp directory when the function returns
	if r.TempDir != "" {
		defer func() {
			utils.DebugLog("[INFO:Run] Cleaning up temp directory: %s", r.TempDir)
			os.RemoveAll(r.TempDir)
		}()
	}

	r.UI.ReadChapter = r.readChapter
	// First load the regular chapter
	err := r.readChapter(index, pctg)
	if err != nil {
		utils.DebugLog("[ERROR:Run] Error reading chapter: %v", err)
		r.UI.StatusBar.SetText(fmt.Sprintf("Error reading chapter: %v", err))
	}

	// Set up the key handling
	ic := r.UI.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			if r.UI.SearchPattern != "" {
				r.UI.SearchPattern = ""
				r.clearSearchHighlights()
				return nil
			} else {
				// Only exit if not in search mode
				r.saveState(r.CurrentChapter, r.UI.Width, pos, pctg)
				r.UI.App.Stop()
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				if r.UI.SearchPattern != "" {
					r.UI.SearchPattern = ""
					r.clearSearchHighlights()
					return nil
				}
				r.saveState(r.CurrentChapter, r.UI.Width, pos, pctg)
				r.UI.App.Stop()
				return nil
			case '?':
				r.UI.ShowHelp()
				return nil
			case 'm':
				r.showMetadata()
				return nil
			// NEED FIX: not work in some books
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
					r.nextChapter(pos, pctg)
				}
				return nil
			case 'N':
				utils.DebugLog("[INFO:searchPrev] SearchPattern: '%s', CurrentChapter: %d", r.UI.SearchPattern, r.CurrentChapter)
				if r.UI.SearchPattern != "" {
					r.searchPrev()
				} else {
					r.prevChapter(pos, pctg)
				}
				return nil
			case 'j':
				r.scrollDown()
				return nil
			case 'k':
				r.scrollUp()
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
			case '+':
				r.increaseWidth()
				return nil
			case '-':
				r.decreaseWidth()
				return nil
			case 'c':
				r.UI.CycleColorScheme()
				return nil
			case 'C':
				r.UI.SetStatus("All caches cleared")
				return nil
				// currently not supported
				// case 'b':
				// 	r.markPosition(r.CurrentChapter, pos, pctg)
				// 	return nil
				// case '`':
				// 	r.jumpToPosition(r.CurrentChapter, pos, pctg)
				// 	return nil
			}
		case tcell.KeyDown:
			r.scrollDown()
			return nil
		case tcell.KeyUp:
			r.scrollUp()
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

	// Run the application
	if err := r.UI.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// readChapter reads a chapter
func (r *Reader) readChapter(index int, pctg float64) error {
	utils.DebugLog("[INFO:readChapter] Reading chapter index: %d, pctg: %f", index, pctg)
	if index < 0 || index >= r.Book.TOC.Len() {
		utils.DebugLog("[ERROR:readChapter] Invalid chapter index: %d", index)
		return fmt.Errorf("invalid chapter index: %d", index)
	}

	r.CurrentChapter = index
	r.UI.StatusBar.SetText(fmt.Sprintf("Reading chapter %d of %d", index+1, r.Book.TOC.Len()))

	// Step 1: Get HTML content (from cache if available)
	// Get the chapter content
	var err error
	chapterContent, err := r.Book.GetChapterContents(index)
	if err != nil {
		utils.DebugLog("[ERROR:readChapter] Error getting chapter content: %v", err)
		return err
	}

	// Store the images for later use
	r.UI.Images = chapterContent.Images

	// Clear the text area and write the formatted lines
	r.UI.TextArea.Clear()
	fmt.Fprintln(r.UI.TextArea, chapterContent.Text)

	// If there's an active search pattern, highlight the results
	if r.UI.SearchPattern != "" {
		re, err := regexp.Compile(r.UI.SearchPattern)
		if err == nil {
			// Find the first occurrence to highlight it differently
			foundIndex := -1
			for i, line := range chapterContent.Lines {
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
		lineCount := len(chapterContent.Lines)
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
		Index:       index,
		Width:       width,
		Pos:         pos,
		Pctg:        pctg,
		LastRead:    true,
		ColorScheme: r.UI.ColorScheme,
	}
	r.Config.SetState(r.FilePath, state)
	r.Config.Save()
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
		chapterContent, err := r.Book.GetChapterContents(index)
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error reading chapter: %v", err))
			return
		}

		// Parse the HTML content
		htmlParser := parser.NewHTMLParser()
		err = htmlParser.Parse(chapterContent.Text, "", "")
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error parsing HTML: %v", err))
			return
		}

		// Get the images
		r.UI.Images = chapterContent.Images
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
		if index < 0 || index >= r.Book.TOC.Len() {
			r.UI.SetStatus(fmt.Sprintf("Invalid chapter index: %d", index))
			return
		}

		chapterPath := r.Book.TOC.Slice[index].Path
		chapterDir := filepath.Dir(chapterPath)
		resolvedPath := filepath.Join(chapterDir, imagePath)

		// Extract the image to a temporary file
		tempFile, err := extractImage(r.Book, resolvedPath, r.TempDir)
		if err != nil {
			r.UI.SetStatus(fmt.Sprintf("Error extracting image: %v", err))
			return
		}

		// Open the image using the system's default image viewer
		err = r.UI.OpenImage(tempFile)
		if err != nil {
			utils.DebugLog("[ERROR:openImage] Error opening image: %v", err)
			r.UI.SetStatus(fmt.Sprintf("Error opening image: %v", err))
			return
		}
	})
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

// getCurrentChapter gets the current chapter
func (r *Reader) getCurrentChapter() (int, error) {
	// Use the CurrentChapter field directly
	if r.CurrentChapter >= 0 && r.CurrentChapter < r.Book.TOC.Len() {
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
func extractImage(book *epub.Epub, imagePath string, tempDir string) (string, error) {
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
		// Try to find the image in OEBPS directory
		if !strings.HasPrefix(imagePath, "OEBPS/") {
			oebpsImagePath := "OEBPS/" + imagePath
			utils.DebugLog("[INFO:extractImage] Trying to find image in OEBPS directory: %s", oebpsImagePath)

			oebpsImageFile, oebpsErr := book.File.Open(oebpsImagePath)
			if oebpsErr == nil {
				utils.DebugLog("[INFO:extractImage] Found image in OEBPS directory")
				imageFile = oebpsImageFile
				err = nil
			} else {
				// Try to find the image relative to the RootDir (OPF file's directory)
				if book.RootDir != "" {
					rootDirImagePath := book.RootDir + imagePath
					utils.DebugLog("[INFO:extractImage] Trying to find image relative to OPF directory: %s", rootDirImagePath)

					rootDirImageFile, rootDirErr := book.File.Open(rootDirImagePath)
					if rootDirErr == nil {
						utils.DebugLog("[INFO:extractImage] Found image relative to OPF directory")
						imageFile = rootDirImageFile
						err = nil
					} else {
						return "", fmt.Errorf("image file not found in EPUB: %v", err)
					}
				} else {
					return "", fmt.Errorf("image file not found in EPUB: %v", err)
				}
			}
		} else {
			// Try to find the image relative to the RootDir (OPF file's directory)
			if book.RootDir != "" && !strings.HasPrefix(imagePath, book.RootDir) {
				rootDirImagePath := book.RootDir + strings.TrimPrefix(imagePath, "OEBPS/")
				utils.DebugLog("[INFO:extractImage] Trying to find image relative to OPF directory: %s", rootDirImagePath)

				rootDirImageFile, rootDirErr := book.File.Open(rootDirImagePath)
				if rootDirErr == nil {
					utils.DebugLog("[INFO:extractImage] Found image relative to OPF directory")
					imageFile = rootDirImageFile
					err = nil
				} else {
					return "", fmt.Errorf("image file not found in EPUB: %v", err)
				}
			} else {
				return "", fmt.Errorf("image file not found in EPUB: %v", err)
			}
		}
	}
	defer imageFile.Close()

	// Check if running in WSL
	isWSL := false
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		isWSL = true
	}

	var tempFile *os.File

	// If we have a temp directory and it's accessible, use it
	if tempDir != "" && isDirectoryWritable(tempDir) {
		// Use the provided temp directory
		utils.DebugLog("[INFO:extractImage] Using provided temp directory: %s", tempDir)
		tempFile, err = os.CreateTemp(tempDir, "goread-image-*.png")
		if err != nil {
			utils.DebugLog("[WARN:extractImage] Failed to create temp file in provided directory: %v", err)
			// Fall through to other methods
		} else {
			// Successfully created temp file in the provided directory
			defer tempFile.Close()
			utils.DebugLog("[INFO:extractImage] Created temp file: %s", tempFile.Name())
			goto COPY_IMAGE
		}
	}

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

COPY_IMAGE:
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

// isDirectoryWritable checks if a directory is writable
func isDirectoryWritable(dir string) bool {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Try to create a temporary file in the directory
	testFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}

	// Clean up
	f.Close()
	os.Remove(testFile)

	return true
}
