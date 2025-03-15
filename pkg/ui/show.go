package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
	"github.com/rivo/tview"
)

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
	utils.DebugLog("[INFO:ShowSearch] Showing search dialog")
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
