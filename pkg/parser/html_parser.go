package parser

import (
	"bytes"
	"fmt"
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

// handleStartTag handles the start of an HTML tag
func (p *HTMLParser) handleStartTag(n *html.Node) {
	tag := n.Data
	p.currentTag = tag

	// Check if this is a heading tag (h1-6)
	isHeading := regexp.MustCompile(`^h[1-6]$`).MatchString(tag)

	if isHeading {
		p.isHead = true
	} else if tag == "q" || tag == "dt" || tag == "dd" || tag == "blockquote" {
		p.isInde = true
	} else if tag == "pre" {
		p.isPref = true
		// For all pre tags, we'll treat them as code blocks
		p.isCode = true

		// Check if there's a class attribute to determine code type
		for _, attr := range n.Attr {
			if attr.Key == "class" {
				// Check if it contains language identifier like "language-go", "lang-python", etc.
				langMatch := regexp.MustCompile(`(?:language|lang)-(\w+)`).FindStringSubmatch(attr.Val)
				if len(langMatch) > 1 {
					p.codeType = langMatch[1]
				}
			}
		}
	} else if tag == "code" {
		// If already inside a pre tag, this is a code block
		if p.isPref {
			p.isCode = true
			// Check if there's a class attribute to determine code type
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					langMatch := regexp.MustCompile(`(?:language|lang)-(\w+)`).FindStringSubmatch(attr.Val)
					if len(langMatch) > 1 {
						p.codeType = langMatch[1]
					}
				}
			}
		}
	} else if tag == "li" {
		p.isBull = true
	} else if tag == "script" || tag == "style" || tag == "head" {
		p.isHidden = true
	} else if tag == "sup" {
		p.buffer += "^{"
	} else if tag == "sub" {
		p.buffer += "_{"
	} else if tag == "img" || tag == "image" {
		// Handle image tags
		var src, alt string
		for _, attr := range n.Attr {
			if (tag == "img" && attr.Key == "src") || (tag == "image" && strings.HasSuffix(attr.Key, "href")) {
				src = attr.Val
			} else if attr.Key == "alt" {
				alt = attr.Val
			}
		}
		if src != "" {
			imgIndex := len(p.images)
			imgText := fmt.Sprintf("[IMG:%d]", imgIndex)
			if alt != "" {
				imgText = fmt.Sprintf("[IMG:%d - %s]", imgIndex, alt)
			}
			p.text = append(p.text, imgText)
			p.images = append(p.images, htmllib.UnescapeString(src))
		}
	} else if tag == "br" {
		p.text = append(p.text, "")
	}
}

// handleEndTag handles the end of an HTML tag
func (p *HTMLParser) handleEndTag(n *html.Node) {
	tag := n.Data

	// Check if this is a heading tag (h1-h6)
	isHeading := regexp.MustCompile(`^h[1-6]$`).MatchString(tag)

	if isHeading {
		p.text = append(p.text, "")
		p.text = append(p.text, "")
		p.isHead = false
	} else if tag == "p" || tag == "div" {
		p.text = append(p.text, "")
	} else if tag == "script" || tag == "style" || tag == "head" {
		p.isHidden = false
	} else if tag == "q" || tag == "dt" || tag == "dd" || tag == "blockquote" {
		if p.text[len(p.text)-1] != "" {
			p.text = append(p.text, "")
		}
		p.isInde = false
	} else if tag == "pre" {
		if p.text[len(p.text)-1] != "" {
			p.text = append(p.text, "")
		}
		p.isPref = false
		p.isCode = false
		p.codeType = ""
	} else if tag == "code" {
		if p.isPref && p.isCode {
			p.isCode = false
		}
	} else if tag == "li" {
		if p.text[len(p.text)-1] != "" {
			p.text = append(p.text, "")
		}
		p.isBull = false
	} else if tag == "sub" || tag == "sup" {
		p.buffer += "}"
	} else if tag == "img" || tag == "image" {
		p.text = append(p.text, "")
	}

	p.currentTag = ""
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

// formatCodeLine formats a code line with syntax highlighting
func formatCodeLine(line string, width int, indent string, codeType string) string {
	if line == "" {
		return ""
	}

	// Apply syntax highlighting
	highlightedLine := applyCodeHighlighting(line, codeType)

	// Handle line wrapping
	var result []string
	lines := strings.Split(highlightedLine, "\n")

	for _, l := range lines {
		// Since we've added color markers, we can't simply split by words
		// Simplified approach: only wrap when exceeding width
		if len(stripColorCodes(l)) <= width {
			result = append(result, indent+l)
		} else {
			// For long lines, simply append as is (this might break color markers)
			// In a real application, more sophisticated handling would be needed
			result = append(result, indent+l)
		}
	}

	return strings.Join(result, "\n")
}

// stripColorCodes removes tview color codes to calculate actual text length
func stripColorCodes(text string) string {
	re := regexp.MustCompile(`\[[^]]*\]`)
	return re.ReplaceAllString(text, "")
}

// applyCodeHighlighting applies syntax highlighting based on code type
func applyCodeHighlighting(code string, codeType string) string {
	// Use universal highlighting for all code blocks
	return highlightUniversal(code)
}

// highlightUniversal adds highlighting for common programming language constructs
func highlightUniversal(code string) string {
	// Use a simpler approach to avoid tview color code issues

	// Split the code into tokens (words, symbols, etc.)
	tokens := tokenizeCode(code)

	// Classify and color each token
	var result strings.Builder
	for _, token := range tokens {
		result.WriteString(colorizeToken(token))
	}

	return result.String()
}

// tokenizeCode splits code into tokens for highlighting
func tokenizeCode(code string) []string {
	// Define a regex pattern to match different code elements
	pattern := regexp.MustCompile(`("[^"]*")|('[^']*')|(\` + "`" + `[^` + "`" + `]*` + "`" + `)|(//.*)|(#.*)|(--.*)|([\d.]+)|([a-zA-Z_]\w*)|(\S)|\s+`)

	matches := pattern.FindAllStringIndex(code, -1)
	var tokens []string

	for _, match := range matches {
		start, end := match[0], match[1]
		tokens = append(tokens, code[start:end])
	}

	return tokens
}

// colorizeToken applies color to a token based on its type
func colorizeToken(token string) string {
	// Check for strings (double quotes, single quotes, backticks)
	if (strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"")) ||
		(strings.HasPrefix(token, "'") && strings.HasSuffix(token, "'")) ||
		(strings.HasPrefix(token, "`") && strings.HasSuffix(token, "`")) {
		return "[#FFFF00]" + token + "[-]"
	}

	// Check for comments
	if strings.HasPrefix(token, "//") || strings.HasPrefix(token, "#") || strings.HasPrefix(token, "--") {
		return "[#00FF00]" + token + "[-]"
	}

	// Check for numbers
	if regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(token) {
		return "[#FF8800]" + token + "[-]"
	}

	// Check for keywords
	if isDataType(token) {
		return "[#00FFFF]" + token + "[-]"
	}

	if isControlFlow(token) {
		return "[#FF00FF]" + token + "[-]"
	}

	if isOtherKeyword(token) {
		return "[#0088FF]" + token + "[-]"
	}

	// Return the token as is if it doesn't match any category
	return token
}

// isDataType checks if a token is a data type keyword
func isDataType(token string) bool {
	dataTypes := map[string]bool{
		"int": true, "float": true, "double": true, "char": true, "string": true,
		"bool": true, "boolean": true, "byte": true, "long": true, "short": true,
		"void": true, "var": true, "let": true, "const": true, "auto": true,
		"static": true, "final": true, "unsigned": true, "signed": true, "uint": true,
		"int8": true, "int16": true, "int32": true, "int64": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "float32": true, "float64": true,
		"object": true, "array": true, "map": true, "set": true, "list": true,
		"vector": true, "dict": true, "tuple": true, "struct": true, "class": true,
		"interface": true, "enum": true, "union": true, "type": true,
	}

	return dataTypes[token]
}

// isControlFlow checks if a token is a control flow keyword
func isControlFlow(token string) bool {
	controlFlow := map[string]bool{
		"if": true, "else": true, "elif": true, "switch": true, "case": true,
		"default": true, "for": true, "while": true, "do": true, "foreach": true,
		"in": true, "of": true, "break": true, "continue": true, "return": true,
		"yield": true, "goto": true, "try": true, "catch": true, "except": true,
		"finally": true, "throw": true, "throws": true, "raise": true,
	}

	return controlFlow[token]
}

// isOtherKeyword checks if a token is another common keyword
func isOtherKeyword(token string) bool {
	otherKeywords := map[string]bool{
		"function": true, "func": true, "def": true, "fn": true, "method": true,
		"import": true, "include": true, "require": true, "from": true, "export": true,
		"package": true, "namespace": true, "module": true, "using": true, "extends": true,
		"implements": true, "override": true, "virtual": true, "abstract": true,
		"public": true, "private": true, "protected": true, "internal": true,
		"async": true, "await": true, "new": true, "delete": true, "this": true,
		"self": true, "super": true, "base": true, "null": true, "nil": true,
		"None": true, "true": true, "false": true, "True": true, "False": true,
		"and": true, "or": true, "not": true, "instanceof": true, "typeof": true,
		"sizeof": true, "lambda": true,
	}

	return otherKeywords[token]
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
