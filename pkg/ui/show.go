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
		ShowSecondaryText(false)

	for i, entry := range tocEntries {
		list.AddItem(fmt.Sprintf("%d. %s", i+1, entry), "", 0, nil)
	}

	list.SetCurrentItem(currentIndex)

	// Apply theme colors to the list
	switch ui.ColorScheme {
	case DefaultColorScheme:
		list.SetMainTextColor(tcell.ColorDefault)
		list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorDefault))
		list.SetBackgroundColor(tcell.ColorDefault)
		list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
		list.SetSelectedTextColor(tcell.ColorWhite)
	case DarkColorScheme:
		list.SetMainTextColor(tcell.ColorWhite)
		list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray))
		list.SetBackgroundColor(tcell.ColorDarkSlateGray)
		list.SetSelectedBackgroundColor(tcell.ColorDarkBlue)
		list.SetSelectedTextColor(tcell.ColorWhite)
	case LightColorScheme:
		list.SetMainTextColor(tcell.ColorBlack)
		list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorWhite))
		list.SetBackgroundColor(tcell.ColorWhite)
		list.SetSelectedBackgroundColor(tcell.ColorLightBlue)
		list.SetSelectedTextColor(tcell.ColorBlack)
	}

	resetContent := ui.SetTempContent(list)

	ui.App.SetFocus(list)

	var resetCapture func()
	// Set up the input capture at the application level
	var selectedIndex int = -1
	resetCapture = ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, 'q':
			resetContent()
			resetCapture()
			return nil
		case tcell.KeyEnter:
			selectedIndex = list.GetCurrentItem()
			// get the selected item
			resetContent()
			ui.ReadChapter(selectedIndex, 0)
			resetCapture()
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow navigation in the list
			return event
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				resetContent()
				resetCapture()
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'c':
				// Cycle through color schemes
				ui.CycleColorScheme()
				// Update list colors based on the new color scheme
				switch ui.ColorScheme {
				case DefaultColorScheme:
					list.SetBackgroundColor(tcell.ColorDefault)
					list.SetMainTextColor(tcell.ColorDefault)
					list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorDefault))
					list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
					list.SetSelectedTextColor(tcell.ColorWhite)
				case DarkColorScheme:
					list.SetBackgroundColor(tcell.ColorDarkSlateGray)
					list.SetMainTextColor(tcell.ColorWhite)
					list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray))
					list.SetSelectedBackgroundColor(tcell.ColorDarkBlue)
					list.SetSelectedTextColor(tcell.ColorWhite)
				case LightColorScheme:
					list.SetMainTextColor(tcell.ColorBlack)
					list.SetMainTextStyle(tcell.StyleDefault.Background(tcell.ColorWhite))
					list.SetBackgroundColor(tcell.ColorWhite)
					list.SetSelectedBackgroundColor(tcell.ColorLightBlue)
					list.SetSelectedTextColor(tcell.ColorBlack)
				}
				return nil
			}
		default:
			// Block all other keys
			return nil
		}
		return event
	})

	return selectedIndex, nil
}

// ShowMetadata shows the metadata
func (ui *UI) ShowMetadata(metadata [][]string) error {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	if len(metadata) == 0 {
		table.SetCell(0, 0, tview.NewTableCell("No metadata found").
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter).
			SetExpansion(1))
		table.SetCell(0, 1, tview.NewTableCell("").
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter).
			SetExpansion(2))
	} else {

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
	}

	switch ui.ColorScheme {
	case DefaultColorScheme:
		table.SetBackgroundColor(tcell.ColorDefault)
	case DarkColorScheme:
		table.SetBackgroundColor(tcell.ColorDarkSlateGray)
	case LightColorScheme:
		table.SetBackgroundColor(tcell.ColorWhite)
	}

	resetContent := ui.SetTempContent(table)
	var resetCapture func()
	// Set a new input capture function at the application level
	resetCapture = ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			resetCapture()

			resetContent()
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow scrolling in the table
			return event
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'c':
				ui.CycleColorScheme()
				switch ui.ColorScheme {
				case DefaultColorScheme:
					table.SetBackgroundColor(tcell.ColorDefault)
				case DarkColorScheme:
					table.SetBackgroundColor(tcell.ColorDarkSlateGray)
				case LightColorScheme:
					table.SetBackgroundColor(tcell.ColorWhite)
				}
				return nil
			}
		default:
			// Block all other keys
			return nil
		}
		return event
	})

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
    ToC              : TAB       t
    Metadata         : m
    Mark pos to n    : b[n]
    Jump to pos n    : ` + "`" + `[n]
    Switch colorsch  : c
		
Press Esc or Enter to close
`
	helpContent := tview.NewTextView().
		SetText(helpText).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	switch ui.ColorScheme {
	case DefaultColorScheme:
		helpContent.SetBackgroundColor(tcell.ColorDefault)
		helpContent.SetTextColor(tcell.ColorDefault)
	case DarkColorScheme:
		helpContent.SetBackgroundColor(tcell.ColorDarkSlateGray)
		helpContent.SetTextColor(tcell.ColorWhite)
	case LightColorScheme:
		helpContent.SetBackgroundColor(tcell.ColorWhite)
		helpContent.SetTextColor(tcell.ColorBlack)
	}

	resetContent := ui.SetTempContent(helpContent)

	ui.App.SetFocus(helpContent)

	var resetCapture func()
	// Set a new input capture function at the application level
	resetCapture = ui.SetCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			// Restore the original input capture function
			resetCapture()
			resetContent()
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			// Allow scrolling in the help text
			return event
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				row, col := helpContent.GetScrollOffset()
				helpContent.ScrollTo(row+1, col)
				return nil
			case 'k':
				row, col := helpContent.GetScrollOffset()
				if row > 0 {
					helpContent.ScrollTo(row-1, col)
				}
				return nil
			case 'c':
				ui.CycleColorScheme()
				switch ui.ColorScheme {
				case DefaultColorScheme:
					helpContent.SetBackgroundColor(tcell.ColorDefault)
					helpContent.SetTextColor(tcell.ColorDefault)
				case DarkColorScheme:
					helpContent.SetBackgroundColor(tcell.ColorDarkSlateGray)
					helpContent.SetTextColor(tcell.ColorWhite)
				case LightColorScheme:
					helpContent.SetBackgroundColor(tcell.ColorWhite)
					helpContent.SetTextColor(tcell.ColorBlack)
				}
				return nil
			}
		default:
			// Block all other keys
			return nil
		}
		return event
	})

	return nil
}

// ShowSearch shows the search dialog in VIM style
func (ui *UI) ShowSearch(cb func()) error {
	utils.DebugLog("[INFO:ShowSearch] Showing search dialog")
	// Save the current search pattern
	originalSearchPattern := ui.SearchPattern

	// Set the initial search text
	ui.SearchInput.SetText(ui.SearchPattern)

	resetStatus := ui.SetTempStatus(ui.SearchInput)

	// Explicitly set focus to the search input
	ui.App.SetFocus(ui.SearchInput)

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

			resetStatus()

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

			resetStatus()
		}
	})

	return nil
}
