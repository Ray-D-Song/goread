package ui

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os/exec"

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
	Container     *tview.Flex
	Horizontal    *tview.Flex
	Content       *tview.Flex
	LeftPanel     *tview.Box
	RightPanel    *tview.Box
	TextArea      *tview.TextView
	StatusBar     *tview.TextView
	SearchInput   *tview.InputField // VIM style search input
	ColorScheme   ColorScheme
	Width         int
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
		SetFieldBackgroundColor(tcell.ColorDeepSkyBlue)

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

	// container, full screen
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetFullScreen(true)

	// horizontal flex
	// use flex to align the content to the center of the screen
	horizontal := tview.NewFlex().SetDirection(tview.FlexColumn)
	_, h := utils.GetTermSize()
	container.AddItem(horizontal, h, 0, true)

	// content
	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.AddItem(textArea, 0, 1, true)
	content.AddItem(statusBar, 1, 0, false)

	leftPanel := tview.NewBox()
	rightPanel := tview.NewBox()
	horizontal.
		AddItem(leftPanel, 0, 1, false).
		AddItem(content, 0, 1, true).
		AddItem(rightPanel, 0, 1, false)

	app.SetRoot(container, true)
	ui.Container = container
	ui.Horizontal = horizontal
	ui.Content = content
	ui.LeftPanel = leftPanel
	ui.RightPanel = rightPanel
	return ui
}

func (ui *UI) ReRender() {
	ui.App.Draw()
}

// SetWidth this func actually sets the width of the goread window
func (ui *UI) SetWidth(width int) {
	if width > 0 {
		ui.Horizontal.ResizeItem(ui.Content, width, 0)
		ui.Width = width
	}
}

// SetCapture sets the input capture function
// return a function to restore the original input capture
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

	ui.StatusBar.SetBackgroundColor(tcell.ColorDeepSkyBlue)
	ui.StatusBar.SetTextColor(tcell.ColorWhite)
	switch scheme {
	case DefaultColorScheme:
		ui.Container.SetBackgroundColor(tcell.ColorDefault)
		ui.Horizontal.SetBackgroundColor(tcell.ColorDefault)
		ui.LeftPanel.SetBackgroundColor(tcell.ColorDefault)
		ui.RightPanel.SetBackgroundColor(tcell.ColorDefault)
		ui.TextArea.SetBackgroundColor(tcell.ColorDefault)
		ui.TextArea.SetTextColor(tcell.ColorDefault)
	case DarkColorScheme:
		ui.Container.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.Horizontal.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.LeftPanel.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.RightPanel.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.TextArea.SetBackgroundColor(tcell.ColorDarkSlateGray)
		ui.TextArea.SetTextColor(tcell.ColorWhite)
	case LightColorScheme:
		ui.Container.SetBackgroundColor(tcell.ColorWhite)
		ui.Horizontal.SetBackgroundColor(tcell.ColorWhite)
		ui.LeftPanel.SetBackgroundColor(tcell.ColorWhite)
		ui.RightPanel.SetBackgroundColor(tcell.ColorWhite)
		ui.TextArea.SetBackgroundColor(tcell.ColorWhite)
		ui.TextArea.SetTextColor(tcell.ColorBlack)
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

// Make text display a norm for the Status Bar
// This method is used to set a temporary new Status Bar
// It will return a restoration method
func (ui *UI) SetTempStatus(views ...tview.Primitive) func() {
	ui.Content.RemoveItem(ui.StatusBar)
	for _, view := range views {
		if v, ok := view.(*tview.InputField); ok {
			v.SetBackgroundColor(tcell.ColorDeepSkyBlue)
		}
		ui.Content.AddItem(view, 1, 0, true)
	}
	// Set focus to the last view (usually the input field)
	if len(views) > 0 {
		ui.App.SetFocus(views[len(views)-1])
	}
	return func() {
		for _, view := range views {
			ui.Content.RemoveItem(view)
		}
		ui.Content.AddItem(ui.StatusBar, 1, 0, false)
	}
}

// SetTempContent sets a temporary content
func (ui *UI) SetTempContent(view tview.Primitive) func() {
	originalContent := ui.Content
	hasLeftPanel := false
	hasRightPanel := false

	// check if the left panel exist
	for i := 0; i < ui.Horizontal.GetItemCount(); i++ {
		if ui.Horizontal.GetItem(i) == ui.LeftPanel {
			hasLeftPanel = true
			break
		}
	}

	// check if the right panel exist
	for i := 0; i < ui.Horizontal.GetItemCount(); i++ {
		if ui.Horizontal.GetItem(i) == ui.RightPanel {
			hasRightPanel = true
			break
		}
	}

	// clear the horizontal layout
	ui.Horizontal.Clear()

	// rebuild the temporary layout
	if hasLeftPanel {
		ui.Horizontal.AddItem(ui.LeftPanel, 0, 1, false)
	}
	ui.Horizontal.AddItem(view, 0, 1, true)
	if hasRightPanel {
		ui.Horizontal.AddItem(ui.RightPanel, 0, 1, false)
	}

	return func() {
		// restore the original layout
		ui.Horizontal.Clear()

		// rebuild the original layout
		if hasLeftPanel {
			ui.Horizontal.AddItem(ui.LeftPanel, 0, 1, false)
		}
		ui.Horizontal.AddItem(originalContent, 0, 1, true)
		if hasRightPanel {
			ui.Horizontal.AddItem(ui.RightPanel, 0, 1, false)
		}
	}
}

// commandExists checks if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
