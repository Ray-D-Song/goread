package ui

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os/exec"

	"github.com/gdamore/tcell/v2"
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

// commandExists checks if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
