package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Theme layout (cycle with c):
//   - Default: VS Code Dark+
//   - Dark:    classic green slate
//   - Light:   VS Code Light+
type themeSpec struct {
	background string
	foreground string
	selection  string
	graphics   string
}

var themeSpecs = map[ColorScheme]themeSpec{
	DefaultColorScheme: {
		background: "1E1E1E",
		foreground: "D4D4D4",
		selection:  "264F78",
		graphics:   "D4D4D4",
	},
	DarkColorScheme: {
		background: "2F4F4F",
		foreground: "E8E8E8",
		selection:  "1A365D",
		graphics:   "E8E8E8",
	},
	LightColorScheme: {
		background: "FFFFFF",
		foreground: "000000",
		selection:  "ADD6FF",
		graphics:   "000000",
	},
}

func specFor(scheme ColorScheme) themeSpec {
	if spec, ok := themeSpecs[scheme]; ok {
		return spec
	}
	return themeSpecs[DefaultColorScheme]
}

func hexColor(value string) tcell.Color {
	return tcell.GetColor("#" + strings.TrimPrefix(value, "#"))
}

func themeColors(scheme ColorScheme) (background, foreground tcell.Color) {
	spec := specFor(scheme)
	return hexColor(spec.background), hexColor(spec.foreground)
}

// BackgroundColor returns the theme background.
func BackgroundColor(scheme ColorScheme) tcell.Color {
	return hexColor(specFor(scheme).background)
}

// ForegroundColor returns the theme foreground.
func ForegroundColor(scheme ColorScheme) tcell.Color {
	return hexColor(specFor(scheme).foreground)
}

// SelectionColor returns the theme selection/highlight background.
func SelectionColor(scheme ColorScheme) tcell.Color {
	return hexColor(specFor(scheme).selection)
}

// GraphicsColor returns the theme color for tree lines and similar UI chrome.
func GraphicsColor(scheme ColorScheme) tcell.Color {
	return hexColor(specFor(scheme).graphics)
}

// ForegroundHex returns the body text color hex (without #) for syntax reset tags.
func ForegroundHex(scheme ColorScheme) string {
	return specFor(scheme).foreground
}