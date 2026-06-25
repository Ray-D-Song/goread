package parser

// HighlightTheme selects the syntax highlighting palette.
type HighlightTheme int

const (
	HighlightDefault HighlightTheme = iota
	HighlightDark
	HighlightLight
)

type highlightPalette struct {
	string       string
	comment      string
	preprocessor string
	number       string
	keyword      string
	typeKeyword  string
}

var currentHighlightTheme = HighlightDefault

// SetHighlightTheme sets the palette used for subsequent parsing.
func SetHighlightTheme(theme HighlightTheme) {
	currentHighlightTheme = theme
}

func currentPalette() highlightPalette {
	switch currentHighlightTheme {
	case HighlightDark:
		// VS Code Dark+
		return highlightPalette{
			string:       "CE9178",
			comment:      "6A9955",
			preprocessor: "C586C0",
			number:       "B5CEA8",
			keyword:      "569CD6",
			typeKeyword:  "4EC9B0",
		}
	case HighlightLight:
		// VS Code Light+
		return highlightPalette{
			string:       "A31515",
			comment:      "008000",
			preprocessor: "795E26",
			number:       "098658",
			keyword:      "0000FF",
			typeKeyword:  "267F99",
		}
	default:
		// Default to the light palette for readability on bright terminals.
		return highlightPalette{
			string:       "A31515",
			comment:      "008000",
			preprocessor: "795E26",
			number:       "098658",
			keyword:      "0000FF",
			typeKeyword:  "267F99",
		}
	}
}

func colorTag(hex string, token string) string {
	return "[#" + hex + "]" + token + "[-]"
}