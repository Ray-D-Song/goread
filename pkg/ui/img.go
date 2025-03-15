package ui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
	"github.com/rivo/tview"
)

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
	utils.DebugLog("[INFO:OpenImage] Opening image: %s", imagePath)
	// Check if the image exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		utils.DebugLog("[ERROR:OpenImage] Image not found: %s", imagePath)
		return fmt.Errorf("image not found: %s", imagePath)
	}

	isWSL := false
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		isWSL = true
		utils.DebugLog("[INFO:OpenImage] Running in WSL environment")
	}

	// Open the image using the default image viewer
	var cmd *exec.Cmd
	switch {
	case isWSL:
		// Convert the path to a Windows path
		winPath, err := exec.Command("wslpath", "-w", imagePath).Output()
		if err != nil {
			utils.DebugLog("[ERROR:OpenImage] Failed to convert path: %v", err)
		} else {
			winPathStr := strings.TrimSpace(string(winPath))
			utils.DebugLog("[INFO:OpenImage] Windows path: %s", winPathStr)

			// Try using explorer.exe directly
			utils.DebugLog("[INFO:OpenImage] Trying with explorer.exe %s", winPathStr)
			cmd = exec.Command("explorer.exe", winPathStr)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("[WARN:OpenImage] Explorer.exe failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("[INFO:OpenImage] Explorer.exe succeeded")
				return nil
			}

			// Try using cmd.exe /c start
			utils.DebugLog("[INFO:OpenImage] Trying with cmd.exe /c start")
			cmd = exec.Command("cmd.exe", "/c", "start", winPathStr)
			output, err = cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("[WARN:OpenImage] cmd.exe command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("[INFO:OpenImage] cmd.exe command succeeded")
				return nil
			}

			// Try using PowerShell as a last resort
			psCmd := fmt.Sprintf("Start-Process '%s'", winPathStr)
			utils.DebugLog("[INFO:OpenImage] Trying with powershell: %s", psCmd)
			cmd = exec.Command("powershell.exe", "-c", psCmd)
			output, err = cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("[WARN:OpenImage] PowerShell command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("[INFO:OpenImage] PowerShell command succeeded")
				return nil
			}
		}

		// If all Windows methods failed, try wslview as a fallback
		if commandExists("wslview") {
			utils.DebugLog("[INFO:OpenImage] Trying wslview as fallback")
			cmd = exec.Command("wslview", imagePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("[WARN:OpenImage] wslview command failed: %v, output: %s", err, string(output))
			} else {
				utils.DebugLog("[INFO:OpenImage] wslview command succeeded")
				return nil
			}
		}

		// If all methods failed, try xdg-open
		if commandExists("xdg-open") {
			utils.DebugLog("[INFO:OpenImage] Trying xdg-open as last resort")
			cmd = exec.Command("xdg-open", imagePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.DebugLog("[ERROR:OpenImage] xdg-open command failed: %v, output: %s", err, string(output))
				return fmt.Errorf("all methods to open image failed")
			} else {
				utils.DebugLog("[INFO:OpenImage] xdg-open command succeeded")
				return nil
			}
		}

		return fmt.Errorf("all methods to open image failed")
	case commandExists("xdg-open"):
		utils.DebugLog("[INFO:OpenImage] Using xdg-open %s", imagePath)
		cmd = exec.Command("xdg-open", imagePath)
	case commandExists("open"):
		utils.DebugLog("[INFO:OpenImage] Using open %s", imagePath)
		cmd = exec.Command("open", imagePath)
	case commandExists("start"):
		utils.DebugLog("[INFO:OpenImage] Using start %s", imagePath)
		cmd = exec.Command("start", imagePath)
	default:
		utils.DebugLog("[ERROR:OpenImage] No image viewer found")
		return fmt.Errorf("no image viewer found")
	}

	// For non-WSL environments, execute the command and log results
	if !isWSL {
		output, err := cmd.CombinedOutput()
		if err != nil {
			utils.DebugLog("[ERROR:OpenImage] Command failed: %v, output: %s", err, string(output))
			return fmt.Errorf("failed to open image: %v", err)
		}
		utils.DebugLog("[INFO:OpenImage] Command succeeded")
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
