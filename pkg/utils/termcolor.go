package utils

import (
	"os"
	"strings"
)

// EnsureTruecolor enables 24-bit color for tcell when the terminal likely
// supports it but COLORTERM was not set by the shell.
func EnsureTruecolor() {
	if os.Getenv("TCELL_TRUECOLOR") == "disable" {
		return
	}
	if os.Getenv("COLORTERM") != "" {
		return
	}

	term := os.Getenv("TERM")
	if strings.HasSuffix(term, "-truecolor") || strings.Contains(term, "direct") {
		return
	}

	// Modern terminals generally support ISO 8613-6 24-bit color even when
	// COLORTERM is unset (for example macOS Terminal and many SSH sessions).
	os.Setenv("COLORTERM", "truecolor")
}

// SupportsTruecolor reports whether goread should emit 24-bit hex colors.
func SupportsTruecolor() bool {
	if os.Getenv("TCELL_TRUECOLOR") == "disable" {
		return false
	}

	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit", "24-bit":
		return true
	}

	term := os.Getenv("TERM")
	return strings.HasSuffix(term, "-truecolor") || strings.Contains(term, "direct")
}