package ui

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
	"github.com/rivo/tview"
)

// ColorScheme represents a color scheme
type ColorScheme int

const (
	// DefaultColorScheme is the default color scheme
	DefaultColorScheme ColorScheme = iota
	// DarkColorScheme is the dark color scheme
	DarkColorScheme
	// LightColorScheme is the light color scheme
	LightColorScheme
)

// UI represents the user interface
type UI struct {
	App           *tview.Application
	TextArea      *tview.TextView
	StatusBar     *tview.TextView
	SearchInput   *tview.InputField // VIM style search input
	ColorScheme   ColorScheme
	Width         int
	Height        int
	JumpList      map[rune][4]interface{} // [index, width, pos, pctg]
	SearchPattern string
	Images        []string // Images in the current chapter
	IsSearchMode  bool     // Mark if the search mode is active
	CountPrefix   int      // Numeric prefix for commands like [count]=
}

// NewUI creates a new UI instance
func NewUI() *UI {
	app := tview.NewApplication()
	textArea := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	searchInput := tview.NewInputField().
		SetLabel("/").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault)

	ui := &UI{
		App:          app,
		TextArea:     textArea,
		StatusBar:    statusBar,
		SearchInput:  searchInput,
		ColorScheme:  DefaultColorScheme,
		JumpList:     make(map[rune][4]interface{}),
		IsSearchMode: false,
		CountPrefix:  0,
	}

	// Set up the layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textArea, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	app.SetRoot(flex, true)

	return ui
}

func (ui *UI) SetCapture(f func(event *tcell.EventKey) *tcell.EventKey) func() {
	originalInputCapture := ui.App.GetInputCapture()
	ui.App.SetInputCapture(f)
	return func() {
		ui.App.SetInputCapture(originalInputCapture)
	}
}

// SetColorScheme sets the color scheme
func (ui *UI) SetColorScheme(scheme ColorScheme) {
	ui.ColorScheme = scheme

	switch scheme {
	case DefaultColorScheme:
		ui.TextArea.SetBackgroundColor(tcell.ColorDefault)
		ui.TextArea.SetTextColor(tcell.ColorDefault)
		ui.StatusBar.SetBackgroundColor(tcell.ColorDefault)
		ui.StatusBar.SetTextColor(tcell.ColorDefault)
		ui.SearchInput.SetBackgroundColor(tcell.ColorDefault)
		ui.SearchInput.SetFieldBackgroundColor(tcell.ColorDefault)
		ui.SearchInput.SetLabelColor(tcell.ColorDefault)
		ui.SearchInput.SetFieldTextColor(tcell.ColorDefault)
	case DarkColorScheme:
		ui.TextArea.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.TextArea.SetTextColor(tcell.ColorWhite)
		ui.StatusBar.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.StatusBar.SetTextColor(tcell.ColorWhite)
		ui.SearchInput.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.SearchInput.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
		ui.SearchInput.SetLabelColor(tcell.ColorWhite)
		ui.SearchInput.SetFieldTextColor(tcell.ColorWhite)
	case LightColorScheme:
		ui.TextArea.SetBackgroundColor(tcell.ColorWhite)
		ui.TextArea.SetTextColor(tcell.ColorBlack)
		ui.StatusBar.SetBackgroundColor(tcell.ColorWhite)
		ui.StatusBar.SetTextColor(tcell.ColorBlack)
		ui.SearchInput.SetBackgroundColor(tcell.ColorWhite)
		ui.SearchInput.SetFieldBackgroundColor(tcell.ColorWhite)
		ui.SearchInput.SetLabelColor(tcell.ColorBlack)
		ui.SearchInput.SetFieldTextColor(tcell.ColorBlack)
	}
}

// CycleColorScheme cycles through the color schemes
func (ui *UI) CycleColorScheme() {
	ui.SetColorScheme((ui.ColorScheme + 1) % 3)
}

// SetStatus sets the status bar text
func (ui *UI) SetStatus(text string) {
	ui.StatusBar.Clear()
	ui.StatusBar.SetText(text)
}

// ShowTOC shows the table of contents
func (ui *UI) ShowTOC(tocEntries []string, currentIndex int) (int, error) {
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)

	for i, entry := range tocEntries {
		list.AddItem(fmt.Sprintf("%d. %s", i+1, entry), "", 0, nil)
	}

	list.SetCurrentItem(currentIndex)

	// Create a frame around the list
	frame := tview.NewFrame(list).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Table of Contents", true, tview.AlignCenter, tcell.ColorWhite).
		AddText("Press Esc to cancel", false, tview.AlignCenter, tcell.ColorWhite)

	// Set up the pages
	pages := tview.NewPages().
		AddPage("toc", frame, true, true)

	// Save the original input capture function
	originalInputCapture := ui.App.GetInputCapture()

	// Set up the input capture at the application level
	var selectedIndex int = -1
	ui.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.App.Stop()
			return nil
		case tcell.KeyEnter:
			selectedIndex = list.GetCurrentItem()
			ui.App.Stop()
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow navigation in the list
			return event
		default:
			// Block all other keys
			return nil
		}
	})

	// Run the application
	ui.App.SetRoot(pages, true)
	if err := ui.App.Run(); err != nil {
		return -1, err
	}

	// Restore the original input capture function
	ui.App.SetInputCapture(originalInputCapture)

	// Restore the original layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.TextArea, 0, 1, true).
		AddItem(ui.StatusBar, 1, 0, false)

	ui.App.SetRoot(flex, true)

	return selectedIndex, nil
}

// ShowMetadata shows the metadata
func (ui *UI) ShowMetadata(metadata [][]string) error {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	for i, item := range metadata {
		table.SetCell(i, 0, tview.NewTableCell(item[0]).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetExpansion(1))
		table.SetCell(i, 1, tview.NewTableCell(item[1]).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft).
			SetExpansion(2))
	}

	// Create a frame around the table
	frame := tview.NewFrame(table).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Metadata", true, tview.AlignCenter, tcell.ColorWhite).
		AddText("Press Esc or Enter to close", false, tview.AlignCenter, tcell.ColorWhite)

	var resetCapture func()
	// Set a new input capture function at the application level
	resetCapture = ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			resetCapture()

			// Restore the original layout
			flex := tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(ui.TextArea, 0, 1, true).
				AddItem(ui.StatusBar, 1, 0, false)

			ui.App.SetRoot(flex, true)
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow scrolling in the table
			return event
		default:
			// Block all other keys
			return nil
		}
	})

	// Run the application
	ui.App.SetRoot(frame, true)

	return nil
}

// ShowHelp shows the help screen
func (ui *UI) ShowHelp() error {
	helpText := `
Goread - EPUB Reader

Key Bindings:
    Help             : ?
    Quit             : q
    Scroll down      : DOWN      j
    Scroll up        : UP        k
    Half screen up   : C-u
    Half screen dn   : C-d
    Page down        : PGDN      RIGHT   SPC
    Page up          : PGUP      LEFT
    Next chapter     : n
    Prev chapter     : p
    Beginning of ch  : HOME      g
    End of ch        : END       G
    Open image       : o
    Search           : /
    Next Occurrence  : n
    Prev Occurrence  : N
    Toggle width     : =
    Set width        : [count]=
    ToC              : TAB       t
    Metadata         : m
    Mark pos to n    : b[n]
    Jump to pos n    : ` + "`" + `[n]
    Switch colorsch  : [default=0, dark=1, light=2]c
`

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(helpText)

	// Create a frame around the text view
	frame := tview.NewFrame(textView).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Help", true, tview.AlignCenter, tcell.ColorWhite).
		AddText("Press Esc or Enter to close", false, tview.AlignCenter, tcell.ColorWhite)

	var resetCapture func()
	// Set a new input capture function at the application level
	resetCapture = ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			// Restore the original input capture function
			resetCapture()

			// Restore the original layout
			flex := tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(ui.TextArea, 0, 1, true).
				AddItem(ui.StatusBar, 1, 0, false)

			ui.App.SetRoot(flex, true)
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow scrolling in the help text
			return event
		default:
			// Block all other keys
			return nil
		}
	})

	// Run the application
	ui.App.SetRoot(frame, true)

	return nil
}

// ShowSearch shows the search dialog in VIM style
func (ui *UI) ShowSearch(cb func()) error {
	utils.DebugLog("showing search")
	// Save the current search pattern
	originalSearchPattern := ui.SearchPattern
	// Save the current status bar content
	originalStatusText := ui.StatusBar.GetText(false)

	// Set the initial search text
	ui.SearchInput.SetText(ui.SearchPattern)

	// Create a new Flex layout, replacing the status bar with the search input
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.TextArea, 0, 1, false).
		AddItem(ui.SearchInput, 1, 0, true)

	ui.App.SetRoot(flex, true)
	ui.IsSearchMode = true

	// Set the input capture function at the application level
	resetCapture := ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEscape:
			// Let these keys be handled by the input field's DoneFunc
			return event
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete,
			tcell.KeyLeft, tcell.KeyRight, tcell.KeyHome, tcell.KeyEnd:
			// Allow these keys for text editing
			return event
		case tcell.KeyRune:
			// Allow text input
			return event
		default:
			// Block all other keys
			return nil
		}
	})

	// Set the completion function for the search input
	ui.SearchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// Search completed, save the search pattern
			ui.SearchPattern = ui.SearchInput.GetText()
			ui.IsSearchMode = false

			// Restore the original input capture function
			resetCapture()

			// Restore the original layout
			restoreFlex := tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(ui.TextArea, 0, 1, true).
				AddItem(ui.StatusBar, 1, 0, false)

			ui.App.SetRoot(restoreFlex, true)

			// Restore the status bar content
			ui.StatusBar.SetText(originalStatusText)

			// Call the callback function to perform the search
			if cb != nil {
				cb()
			}
		} else if key == tcell.KeyEscape {
			// Cancel search, restore the original search pattern
			ui.SearchPattern = originalSearchPattern
			ui.IsSearchMode = false

			// Restore the original input capture function
			resetCapture()

			// Restore the original layout
			restoreFlex := tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(ui.TextArea, 0, 1, true).
				AddItem(ui.StatusBar, 1, 0, false)

			ui.App.SetRoot(restoreFlex, true)

			// Restore the status bar content
			ui.StatusBar.SetText(originalStatusText)
		}
	})

	return nil
}

// ShowImageSelect shows an input dialog for selecting an image by number
func (ui *UI) ShowImageSelect(images []string, callback func(string)) error {
	// Save the current status bar content
	originalStatusText := ui.StatusBar.GetText(false)

	// Get image descriptions from the text
	text := ui.TextArea.GetText(false)
	lines := strings.Split(text, "\n")

	// Find image descriptions using regex
	imgDescriptions := make([]string, len(images))
	for _, line := range lines {
		re := regexp.MustCompile(`\[IMG:(\d+)(?:\s*-\s*([^\]]+))?\]`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				idx := 0
				fmt.Sscanf(match[1], "%d", &idx)
				if idx < len(imgDescriptions) && len(match) > 2 && match[2] != "" {
					imgDescriptions[idx] = match[2]
				}
			}
		}
	}

	// Create a text view to display image descriptions
	descView := tview.NewTextView()
	descView.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetBorder(true).
		SetTitle(" Available Images ")

	// Build the text content for image descriptions
	var descContent strings.Builder
	for i, desc := range imgDescriptions {
		if desc != "" {
			descContent.WriteString(fmt.Sprintf("[yellow]%d[white]: %s\n", i, desc))
		} else {
			descContent.WriteString(fmt.Sprintf("[yellow]%d[white]: Image %d\n", i, i))
		}
	}

	// Set the text content
	descView.SetText(descContent.String())

	// Create an input field for image selection
	imageInput := tview.NewInputField().
		SetLabel(fmt.Sprintf("Enter image number (0-%d): ", len(images)-1)).
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetAcceptanceFunc(tview.InputFieldInteger)

	// Create a new Flex layout with the description view and input field
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.TextArea, 0, 3, false).
		AddItem(descView, 0, 1, false).
		AddItem(imageInput, 1, 0, true)

	ui.App.SetRoot(flex, true)

	// Variable to track if we're currently focused on the description view
	var focusOnDesc bool = false

	// Save the original input capture function
	resetCapture := ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			if focusOnDesc {
				// If focused on description, switch focus back to input
				focusOnDesc = false
				ui.App.SetFocus(imageInput)
				return nil
			}
			// Otherwise, let the input field handle it
			return event
		case tcell.KeyEscape:
			if focusOnDesc {
				// If focused on description, switch focus back to input
				focusOnDesc = false
				ui.App.SetFocus(imageInput)
				return nil
			}
			// Otherwise, let the input field handle it
			return event
		case tcell.KeyTab:
			// Toggle focus between input and description view
			focusOnDesc = !focusOnDesc
			if focusOnDesc {
				ui.App.SetFocus(descView)
			} else {
				ui.App.SetFocus(imageInput)
			}
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
			if focusOnDesc {
				// Pass navigation keys to the description view when it's focused
				return event
			} else if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
				// Switch focus to description view on up/down when in input field
				focusOnDesc = true
				ui.App.SetFocus(descView)
				return event
			}
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete,
			tcell.KeyLeft, tcell.KeyRight:
			// Allow these keys for text editing when input is focused
			if !focusOnDesc {
				return event
			}
			return nil
		case tcell.KeyRune:
			// Allow text input (only numbers) when input is focused
			if !focusOnDesc && event.Rune() >= '0' && event.Rune() <= '9' {
				return event
			}
			return nil
		default:
			// Block all other keys
			return nil
		}
	})

	// Set the completion function for the image input
	imageInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// Get the selected image number
			numStr := imageInput.GetText()
			if numStr != "" {
				num := 0
				fmt.Sscanf(numStr, "%d", &num)

				// Check if the number is valid
				if num >= 0 && num < len(images) {
					// Call the callback with the selected image
					callback(images[num])
				} else {
					ui.SetStatus(fmt.Sprintf("Invalid image number: %d (valid range: 0-%d)", num, len(images)-1))
				}
			}
		} else if key == tcell.KeyEscape {
			// Cancel image selection
			callback("") // Call callback with empty string to indicate cancellation
		}

		// Restore the original input capture function
		resetCapture()

		// Restore the original layout
		restoreFlex := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(ui.TextArea, 0, 1, true).
			AddItem(ui.StatusBar, 1, 0, false)

		ui.App.SetRoot(restoreFlex, true)

		// Restore the status bar content
		ui.StatusBar.SetText(originalStatusText)
	})

	return nil
}

// OpenImage opens an image using the default image viewer
func (ui *UI) OpenImage(imagePath string) error {
	utils.DebugLog("opening image: %s", imagePath)
	// Check if the image exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		utils.DebugLog("image not found: %s", imagePath)
		return fmt.Errorf("image not found: %s", imagePath)
	}

	isWSL := false
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		isWSL = true
	}

	// Open the image using the default image viewer
	var cmd *exec.Cmd
	switch {
	case isWSL:
		utils.DebugLog("running in WSL")

		// Convert the path to a Windows path
		winPath, err := exec.Command("wslpath", "-w", imagePath).Output()
		if err != nil {
			utils.DebugLog("failed to convert path: %v", err)
		} else {
			winPathStr := strings.TrimSpace(string(winPath))
			utils.DebugLog("windows path: %s", winPathStr)

			// Try using explorer.exe directly
			utils.DebugLog("trying with explorer.exe %s", winPathStr)
			cmd = exec.Command("explorer.exe", winPathStr)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("explorer.exe failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("explorer.exe succeeded")
				return nil
			}

			// Try using cmd.exe /c start
			utils.DebugLog("trying with cmd.exe /c start")
			cmd = exec.Command("cmd.exe", "/c", "start", winPathStr)
			output, err = cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("cmd.exe command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("cmd.exe command succeeded")
				return nil
			}

			// Try using PowerShell as a last resort
			psCmd := fmt.Sprintf("Start-Process '%s'", winPathStr)
			utils.DebugLog("trying with powershell: %s", psCmd)
			cmd = exec.Command("powershell.exe", "-c", psCmd)
			output, err = cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("powershell command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("powershell command succeeded")
				return nil
			}
		}

		// If all Windows methods failed, try wslview as a fallback
		if commandExists("wslview") {
			utils.DebugLog("trying wslview as fallback")
			cmd = exec.Command("wslview", imagePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("wslview command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("wslview command succeeded")
				return nil
			}
		}

		// If all methods failed, try xdg-open
		if commandExists("xdg-open") {
			utils.DebugLog("trying xdg-open as last resort")
			cmd = exec.Command("xdg-open", imagePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("xdg-open command failed: %v, output: %s", err, string(output))
				return fmt.Errorf("all methods to open image failed")
			} else {
				utils.DebugLog("xdg-open command succeeded")
				return nil
			}
		}

		return fmt.Errorf("all methods to open image failed")
	case commandExists("xdg-open"):
		utils.DebugLog("using xdg-open %s", imagePath)
		cmd = exec.Command("xdg-open", imagePath)
	case commandExists("open"):
		utils.DebugLog("using open %s", imagePath)
		cmd = exec.Command("open", imagePath)
	case commandExists("start"):
		utils.DebugLog("using start %s", imagePath)
		cmd = exec.Command("start", imagePath)
	default:
		utils.DebugLog("no image viewer found")
		return fmt.Errorf("no image viewer found")
	}

	// For non-WSL environments, execute the command and log results
	if !isWSL {
		output, err := cmd.CombinedOutput()
		if err != nil {
			utils.DebugLog("command failed: %v, output: %s", err, string(output))
			return fmt.Errorf("failed to open image: %v", err)
		}
		utils.DebugLog("command succeeded")
		return nil
	}

	return nil
}

// SelectImage shows a dialog to select an image
func (ui *UI) SelectImage(images []string, startLine int, endLine int) (string, error) {
	// Find images in the visible area
	var visibleImages []string
	var visibleIndices []int
	var visibleDescriptions []string

	text := ui.TextArea.GetText(false)
	lines := strings.Split(text, "\n")

	for i := startLine; i <= endLine && i < len(lines); i++ {
		line := lines[i]
		re := regexp.MustCompile(`\[IMG:(\d+)(?:\s*-\s*([^\]]+))?\]`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				index := match[1]
				idx := 0
				fmt.Sscanf(index, "%d", &idx)
				if idx < len(images) {
					visibleImages = append(visibleImages, images[idx])
					visibleIndices = append(visibleIndices, idx)

					// Get description if available
					description := ""
					if len(match) > 2 && match[2] != "" {
						description = match[2]
					}
					visibleDescriptions = append(visibleDescriptions, description)
				}
			}
		}
	}

	if len(visibleImages) == 0 {
		// If no images in visible area, show all images
		if len(images) > 0 {
			for idx, img := range images {
				visibleImages = append(visibleImages, img)
				visibleIndices = append(visibleIndices, idx)
				visibleDescriptions = append(visibleDescriptions, "")
			}
		} else {
			return "", fmt.Errorf("no images found in the chapter")
		}
	}

	if len(visibleImages) == 1 {
		return visibleImages[0], nil
	}

	// Show a dialog to select an image
	list := tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)

	for i, idx := range visibleIndices {
		description := ""
		if i < len(visibleDescriptions) && visibleDescriptions[i] != "" {
			description = visibleDescriptions[i]
		}
		list.AddItem(fmt.Sprintf("Image %d", idx), description, 0, nil)
	}

	// Create a frame around the list
	frame := tview.NewFrame(list).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Select an image", true, tview.AlignCenter, tcell.ColorWhite).
		AddText("Press Enter to select, Esc to cancel", false, tview.AlignCenter, tcell.ColorWhite)

	// Create a new application for the image selection
	selectApp := tview.NewApplication().SetRoot(frame, true)

	// Set up the input capture
	var selectedIndex = -1
	selectApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			selectApp.Stop()
			return nil
		case tcell.KeyEnter:
			selectedIndex = list.GetCurrentItem()
			selectApp.Stop()
			return nil
		}
		// Only pass navigation keys to the list
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			return event
		}
		// Ignore all other keys
		return nil
	})

	// Run the application
	if err := selectApp.Run(); err != nil {
		return "", err
	}

	// Check if an image was selected
	if selectedIndex < 0 || selectedIndex >= len(visibleImages) {
		return "", fmt.Errorf("no image selected")
	}

	return visibleImages[selectedIndex], nil
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

// commandExists checks if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// DisplayImageWithTViewImage displays an image using the system's default image viewer
// This method is kept for backward compatibility but now directly calls OpenImage
func (ui *UI) DisplayImageWithTViewImage(imagePath string) error {
	// Simply call OpenImage instead of trying to render in terminal
	return ui.OpenImage(imagePath)
}
