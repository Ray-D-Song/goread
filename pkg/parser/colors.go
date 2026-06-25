package parser

import "strings"

// HighlightTheme selects the syntax highlighting palette.
// Values align with ui.ColorScheme.
type HighlightTheme int

const (
	HighlightDefault HighlightTheme = iota // VS Code Dark+ syntax
	HighlightDark                          // VS Code Dark+ syntax on green background
	HighlightLight                         // VS Code Light+ syntax
)

// ColorRenderMode controls how syntax colors are encoded for tview.
type ColorRenderMode int

const (
	// ColorRenderTrueColor uses [#RRGGBB] tags (needs 24-bit terminal support).
	ColorRenderTrueColor ColorRenderMode = iota
	// ColorRenderNamed uses tcell named colors for better 16/256-color fallback.
	ColorRenderNamed
)

type syntaxColor struct {
	hex  string
	name string
}

type highlightPalette struct {
	string       syntaxColor
	comment      syntaxColor
	preprocessor syntaxColor
	number       syntaxColor
	keyword      syntaxColor
	typeKeyword  syntaxColor
}

var (
	currentHighlightTheme = HighlightDefault
	currentColorMode      = ColorRenderTrueColor
	currentTextResetHex   = "D4D4D4"
)

// SetHighlightTheme sets the palette used for subsequent parsing.
func SetHighlightTheme(theme HighlightTheme) {
	currentHighlightTheme = theme
	switch theme {
	case HighlightLight:
		currentTextResetHex = "000000"
	case HighlightDark:
		currentTextResetHex = "E8E8E8"
	default:
		currentTextResetHex = "D4D4D4"
	}
}

// SetTextResetHex sets the foreground color used after syntax tokens.
func SetTextResetHex(hex string) {
	if hex != "" {
		currentTextResetHex = strings.TrimPrefix(hex, "#")
	}
}

// SetColorRenderMode selects hex vs named-color rendering.
func SetColorRenderMode(mode ColorRenderMode) {
	currentColorMode = mode
}

func vscodeDarkPalette() highlightPalette {
	return highlightPalette{
		string:       syntaxColor{hex: "CE9178", name: "sandybrown"},
		comment:      syntaxColor{hex: "6A9955", name: "darkseagreen"},
		preprocessor: syntaxColor{hex: "C586C0", name: "plum"},
		number:       syntaxColor{hex: "B5CEA8", name: "darkseagreen"},
		keyword:      syntaxColor{hex: "569CD6", name: "deepskyblue"},
		typeKeyword:  syntaxColor{hex: "4EC9B0", name: "mediumturquoise"},
	}
}

func vscodeLightPalette() highlightPalette {
	return highlightPalette{
		string:       syntaxColor{hex: "A31515", name: "maroon"},
		comment:      syntaxColor{hex: "008000", name: "green"},
		preprocessor: syntaxColor{hex: "795E26", name: "saddlebrown"},
		number:       syntaxColor{hex: "098658", name: "teal"},
		keyword:      syntaxColor{hex: "0000FF", name: "blue"},
		typeKeyword:  syntaxColor{hex: "267F99", name: "darkcyan"},
	}
}

func currentPalette() highlightPalette {
	switch currentHighlightTheme {
	case HighlightLight:
		return vscodeLightPalette()
	default:
		return vscodeDarkPalette()
	}
}

func colorTag(color syntaxColor, token string) string {
	reset := resetTag()
	if currentColorMode == ColorRenderNamed {
		return "[" + color.name + "]" + token + reset
	}
	return "[#" + color.hex + "]" + token + reset
}

func resetTag() string {
	return "[#" + currentTextResetHex + "]"
}