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
	isCode     bool   // Flag indicating if we're inside a code block
	codeType   string // Code type (language)
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
func (p *HTMLParser) Parse(content string) error {
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
		p.text[len(p.text)-1] += p.buffer
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

// FormatLines formats the lines of text with the given width
func (p *HTMLParser) FormatLines(width int) []string {
	if width <= 0 {
		return p.text
	}

	var formattedLines []string

	for i, line := range p.text {
		if p.headIDs[i] {
			// Center the heading
			padding := (width / 2) - (len(line) / 2)
			if padding < 0 {
				padding = 0
			}
			formattedLines = append(formattedLines, strings.Repeat(" ", padding)+line)
			formattedLines = append(formattedLines, "")
		} else if p.indeIDs[i] {
			// Indent the line
			formattedLines = append(formattedLines, formatIndentedLine(line, width-3, "   "))
			formattedLines = append(formattedLines, "")
		} else if p.bullIDs[i] {
			// Format as a bullet point
			formattedLines = append(formattedLines, formatBulletLine(line, width-3))
			formattedLines = append(formattedLines, "")
		} else if p.prefIDs[i] {
			// Format as preformatted text
			if p.codeIDs[i] {
				// If it's code, add syntax highlighting
				formattedLines = append(formattedLines, formatCodeLine(line, width-6, "   ", p.codeType))
			} else {
				formattedLines = append(formattedLines, formatPreformattedLine(line, width-6, "   "))
			}
			formattedLines = append(formattedLines, "")
		} else {
			// Wrap the line to the given width
			formattedLines = append(formattedLines, formatWrappedLine(line, width)...)
			formattedLines = append(formattedLines, "")
		}
	}

	return formattedLines
}

// formatIndentedLine formats an indented line
func formatIndentedLine(line string, width int, indent string) string {
	if line == "" {
		return ""
	}

	var result []string
	words := strings.Fields(line)
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			result = append(result, indent+currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		result = append(result, indent+currentLine)
	}

	return strings.Join(result, "\n")
}

// formatBulletLine formats a bullet point line
func formatBulletLine(line string, width int) string {
	if line == "" {
		return ""
	}

	var result []string
	words := strings.Fields(line)
	var currentLine string

	for i, word := range words {
		if i == 0 {
			currentLine = " - " + word
		} else if len(currentLine)+len(word)+1 <= width {
			currentLine += " " + word
		} else {
			result = append(result, currentLine)
			currentLine = "   " + word
		}
	}

	if currentLine != "" {
		result = append(result, currentLine)
	}

	return strings.Join(result, "\n")
}

// formatPreformattedLine formats a preformatted line
func formatPreformattedLine(line string, width int, indent string) string {
	if line == "" {
		return ""
	}

	var result []string
	lines := strings.Split(line, "\n")

	for _, l := range lines {
		var currentLine string
		words := strings.Fields(l)

		for _, word := range words {
			if len(currentLine)+len(word)+1 <= width {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				result = append(result, indent+currentLine)
				currentLine = word
			}
		}

		if currentLine != "" {
			result = append(result, indent+currentLine)
		}
	}

	return strings.Join(result, "\n")
}

// formatWrappedLine formats a wrapped line
func formatWrappedLine(line string, width int) []string {
	if line == "" {
		return []string{""}
	}

	var result []string
	var currentLine string
	words := strings.Fields(line)

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			result = append(result, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		result = append(result, currentLine)
	}

	return result
}

// DumpHTML dumps the HTML content as plain text
func DumpHTML(content string) (string, error) {
	parser := NewHTMLParser()
	err := parser.Parse(content)
	if err != nil {
		return "", err
	}

	lines := parser.GetLines()
	var buf bytes.Buffer

	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n\n")
	}

	return buf.String(), nil
}
