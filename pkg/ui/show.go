package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ray-d-song/goread/pkg/utils"
	"github.com/rivo/tview"
)

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
				SetTextColor(ForegroundColor(ui.ColorScheme)).
				SetAlign(tview.AlignLeft).
				SetExpansion(2))
		}
	}

	table.SetBackgroundColor(BackgroundColor(ui.ColorScheme))

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
			case 'q':
				resetCapture()
				resetContent()
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'c':
				ui.CycleColorScheme()
				table.SetBackgroundColor(BackgroundColor(ui.ColorScheme))
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
    ToC              : t
    Next chapter     : n
    Prev chapter     : N
    Search           : /
    Scroll down      : j
    Scroll up        : k
    Half screen up   : C-u
    Half screen dn   : C-d
    Beginning of ch  : g
    End of ch        : G
    Open image       : o
    Increase width   : +
    Decrease width   : -
    Metadata         : m
    Switch colorsch  : c
		
Press Esc or Enter to close
`
	helpContent := tview.NewTextView().
		SetText(helpText).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	helpContent.SetBackgroundColor(BackgroundColor(ui.ColorScheme))
	helpContent.SetTextColor(ForegroundColor(ui.ColorScheme))

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
			case 'q':
				resetCapture()
				resetContent()
				return nil
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
				helpContent.SetBackgroundColor(BackgroundColor(ui.ColorScheme))
				helpContent.SetTextColor(ForegroundColor(ui.ColorScheme))
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
