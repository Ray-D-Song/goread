package utils

import (
	"os"
	"testing"
)

func TestEnsureTruecolorSetsCOLORTERM(t *testing.T) {
	t.Setenv("TCELL_TRUECOLOR", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")

	EnsureTruecolor()

	if os.Getenv("COLORTERM") != "truecolor" {
		t.Fatalf("expected COLORTERM=truecolor, got %q", os.Getenv("COLORTERM"))
	}
	if !SupportsTruecolor() {
		t.Fatal("expected truecolor support after EnsureTruecolor")
	}
}

func TestSupportsTruecolorDisabled(t *testing.T) {
	t.Setenv("TCELL_TRUECOLOR", "disable")
	t.Setenv("COLORTERM", "truecolor")

	if SupportsTruecolor() {
		t.Fatal("expected truecolor to be disabled")
	}
}