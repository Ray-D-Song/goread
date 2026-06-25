package parser

import (
	"strings"
	"testing"
)

func TestSnippetCodeHighlighting(t *testing.T) {
	SetColorRenderMode(ColorRenderTrueColor)
	SetHighlightTheme(HighlightLight)
	html := `<p class="snippet code">int main() { return 0; }</p>`
	p := NewHTMLParser()
	if err := p.Parse(html, "", ""); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	lines := p.GetLines()
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected parsed code line")
	}
	if !strings.Contains(lines[0], "[#") {
		t.Fatalf("expected syntax highlighting tags, got %q", lines[0])
	}
	if !p.codeIDs[0] {
		t.Fatal("expected line to be marked as code")
	}
}

func TestSnippetCodePreservesIndentation(t *testing.T) {
	html := `<p class="snippet code">  printf("hello\n");</p>`
	p := NewHTMLParser()
	if err := p.Parse(html, "", ""); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !strings.HasPrefix(p.GetLines()[0], "  printf") {
		t.Fatalf("expected indentation preserved, got %q", p.GetLines()[0])
	}
}

func TestSnippetShellPreservesFormattingWithoutHighlighting(t *testing.T) {
	html := `<p class="snippet shell"><strong>$ ls</strong></p>`
	p := NewHTMLParser()
	if err := p.Parse(html, "", ""); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	line := p.GetLines()[0]
	if line != "$ ls" {
		t.Fatalf("expected shell text preserved, got %q", line)
	}
	if strings.Contains(line, "[#") {
		t.Fatalf("did not expect syntax highlighting for shell snippet, got %q", line)
	}
}

func TestPrefaceSnippetCodeFromBook(t *testing.T) {
	SetColorRenderMode(ColorRenderTrueColor)
	SetHighlightTheme(HighlightLight)
	html := strings.Join([]string{
		`<p class="snippet code">#include &lt;stdio.h&gt;</p>`,
		`<p class="snippet code">  if (ret) {</p>`,
	}, "")

	p := NewHTMLParser()
	if err := p.Parse(html, "", ""); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	lines := p.GetLines()
	if !strings.Contains(lines[0], "[#795E26]#include <stdio.h>[#000000]") {
		t.Fatalf("expected highlighted include line, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  [#") {
		t.Fatalf("expected indented highlighted line, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "if") {
		t.Fatalf("expected if keyword in line, got %q", lines[1])
	}
}