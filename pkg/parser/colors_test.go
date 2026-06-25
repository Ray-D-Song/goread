package parser

import (
	"strings"
	"testing"
)

func TestLightThemeUsesVSCodeColors(t *testing.T) {
	SetHighlightTheme(HighlightLight)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{name: "string", input: `"hello"`, contains: "[#A31515]\"hello\"[-]"},
		{name: "comment", input: "// note", contains: "[#008000]// note[-]"},
		{name: "preprocessor", input: "#include <stdio.h>", contains: "[#795E26]#include <stdio.h>[-]"},
		{name: "number", input: "42", contains: "[#098658]42[-]"},
		{name: "keyword", input: "return", contains: "[#0000FF]return[-]"},
		{name: "type", input: "int", contains: "[#267F99]int[-]"},
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

func TestDarkThemeUsesVSCodeColors(t *testing.T) {
	SetHighlightTheme(HighlightDark)

	got := applyCodeHighlighting(`#include <stdio.h>`)
	if !strings.Contains(got, "[#C586C0]#include <stdio.h>[-]") {
		t.Fatalf("unexpected dark preprocessor color: %q", got)
	}
}