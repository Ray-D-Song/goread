package parser

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	htmllib "html"

	"golang.org/x/net/html"
)

// HTMLParser parses HTML content and converts it to lines of text
type HTMLParser struct {
	text       []string
	images     []string
	isHead     bool
	isInde     bool
	isBull     bool
	isPref     bool
	isCode     bool // Flag indicating if we're inside a code block
	isHidden   bool
	headIDs    map[int]bool
	indeIDs    map[int]bool
	bullIDs    map[int]bool
	prefIDs    map[int]bool
	codeIDs    map[int]bool // Mark which lines are code
	currentTag string
	buffer     string
}

// NewHTMLParser creates a new HTMLParser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{
		text:    []string{""},
		images:  []string{},
		headIDs: make(map[int]bool),
		indeIDs: make(map[int]bool),
		bullIDs: make(map[int]bool),
		prefIDs: make(map[int]bool),
		codeIDs: make(map[int]bool),
	}
}

// Parse parses HTML content and converts it to lines of text
func (p *HTMLParser) Parse(content string, startAnchor string, nextAnchor string) error {
	var err error
	if startAnchor != "" {
		content, err = ExtractBetweenAnchors(content, startAnchor, nextAnchor)
		if err != nil {
			return err
		}
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return err
	}

	p.parseNode(doc)
	return nil
}

// parseNode parses an HTML node and its children
func (p *HTMLParser) parseNode(n *html.Node) {
	if n.Type == html.ElementNode {
		p.handleStartTag(n)
	} else if n.Type == html.TextNode {
		p.handleText(n.Data)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.parseNode(c)
	}

	if n.Type == html.ElementNode {
		p.handleEndTag(n)
	}
}

// handleText handles text nodes
func (p *HTMLParser) handleText(data string) {
	if data == "" || p.isHidden {
		return
	}

	// Process the text
	if p.text[len(p.text)-1] == "" {
		data = strings.TrimLeftFunc(data, unicode.IsSpace)
	}

	if p.isPref {
		p.buffer += htmllib.UnescapeString(data)
	} else {
		// Replace multiple whitespace with a single space
		re := regexp.MustCompile(`\s+`)
		data = re.ReplaceAllString(data, " ")
		p.buffer += htmllib.UnescapeString(data)
	}

	// If we have accumulated text, add it to the current line
	if p.buffer != "" {
		// Apply code highlighting if in a code block
		if p.isCode {
			p.text[len(p.text)-1] += formatCodeLine(p.buffer, "")
		} else {
			p.text[len(p.text)-1] += p.buffer
		}
		p.buffer = ""

		// Mark the line with the appropriate type
		lineIndex := len(p.text) - 1
		if p.isHead {
			p.headIDs[lineIndex] = true
		} else if p.isBull {
			p.bullIDs[lineIndex] = true
		} else if p.isInde {
			p.indeIDs[lineIndex] = true
		} else if p.isPref {
			p.prefIDs[lineIndex] = true
			if p.isCode {
				p.codeIDs[lineIndex] = true
			}
		}
	}
}

// GetLines returns the parsed lines of text
func (p *HTMLParser) GetLines() []string {
	return p.text
}

// GetImages returns the images found in the HTML
func (p *HTMLParser) GetImages() []string {
	return p.images
}

// DumpHTML dumps the HTML content as plain text
func DumpHTML(content string) (string, error) {
	parser := NewHTMLParser()
	err := parser.Parse(content, "", "")
	if err != nil {
		return "", err
	}

	lines := parser.text
	var buf bytes.Buffer

	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n\n")
	}

	return buf.String(), nil
}
