package parser

import (
	"strings"
	"testing"
)

func TestLightThemeUsesVSCodeColors(t *testing.T) {
	SetColorRenderMode(ColorRenderTrueColor)
	SetHighlightTheme(HighlightLight)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{name: "string", input: `"hello"`, contains: "[#A31515]\"hello\"[#000000]"},
		{name: "comment", input: "// note", contains: "[#008000]// note[#000000]"},
		{name: "preprocessor", input: "#include <stdio.h>", contains: "[#795E26]#include <stdio.h>[#000000]"},
		{name: "number", input: "42", contains: "[#098658]42[#000000]"},
		{name: "keyword", input: "return", contains: "[#0000FF]return[#000000]"},
		{name: "type", input: "int", contains: "[#267F99]int[#000000]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := applyCodeHighlighting(tc.input)
			if !strings.Contains(got, tc.contains) {
				t.Fatalf("expected %q in %q", tc.contains, got)
			}
		})
	}
}

func TestNamedColorFallback(t *testing.T) {
	SetColorRenderMode(ColorRenderNamed)
	SetHighlightTheme(HighlightLight)

	got := applyCodeHighlighting(`return`)
	if got != "[blue]return[#000000]" {
		t.Fatalf("unexpected named keyword color: %q", got)
	}
}

func TestVSCodeDarkThemeUsesVSCodeColors(t *testing.T) {
	SetColorRenderMode(ColorRenderTrueColor)
	SetHighlightTheme(HighlightDefault)

	got := applyCodeHighlighting(`#include <stdio.h>`)
	if !strings.Contains(got, "[#C586C0]#include <stdio.h>[#D4D4D4]") {
		t.Fatalf("unexpected vscode dark preprocessor color: %q", got)
	}
}

func TestGreenThemeUsesVSCodeDarkSyntax(t *testing.T) {
	SetColorRenderMode(ColorRenderTrueColor)
	SetHighlightTheme(HighlightDark)

	got := applyCodeHighlighting(`#include <stdio.h>`)
	if !strings.Contains(got, "[#C586C0]#include <stdio.h>[#E8E8E8]") {
		t.Fatalf("unexpected green theme preprocessor color: %q", got)
	}
}