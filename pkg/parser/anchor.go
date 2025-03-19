package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ExtractBetweenAnchors extracts the content between two anchors in the HTML content.
// startAnchor is the ID or name of the starting anchor.
// nextAnchor is the ID or name of the ending anchor.
// If nextAnchor is empty or not found, it will extract until the end of the document.
func ExtractBetweenAnchors(content string, startAnchor string, nextAnchor string) (string, error) {
	// Validate input parameters
	if content == "" {
		return "", fmt.Errorf("empty content")
	}
	if startAnchor == "" {
		return "", fmt.Errorf("empty start anchor")
	}

	// Try string-based approach first (faster and simpler)
	_, endPos, startTagPos := findAnchorPositions(content, startAnchor, nextAnchor)
	if startTagPos != -1 {
		if endPos == -1 {
			endPos = len(content)
		}
		// Include the start anchor tag by using startTagPos instead of startPos
		extractedContent := content[startTagPos:endPos]

		// Validate the extracted content
		if strings.TrimSpace(extractedContent) != "" {
			return extractedContent, nil
		}
	}

	// If string-based approach failed, try DOM-based approach
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract content between anchors using DOM traversal
	var buf bytes.Buffer
	found, err := extractContentDOM(doc, &buf, startAnchor, nextAnchor, true)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("start anchor '%s' not found", startAnchor)
	}

	result := buf.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("no content found between anchors")
	}

	return result, nil
}

// extractContentDOM extracts content between anchors from the HTML node tree using DOM traversal
func extractContentDOM(n *html.Node, buf *bytes.Buffer, startAnchor, nextAnchor string, includeStartTag bool) (bool, error) {
	// State variables
	var (
		found     bool
		capturing bool
		endFound  bool
	)

	// Function to check if a node has the specified anchor
	hasAnchor := func(node *html.Node, anchor string) bool {
		if node.Type != html.ElementNode || anchor == "" {
			return false
		}
		for _, attr := range node.Attr {
			if (attr.Key == "id" || attr.Key == "name") && attr.Val == anchor {
				return true
			}
		}
		return false
	}

	// Function to render a node's content (not including the node itself)
	renderNodeContent := func(node *html.Node, w io.Writer) error {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if err := html.Render(w, c); err != nil {
				return err
			}
		}
		return nil
	}

	// Function to render a node and all its children
	renderNode := func(node *html.Node, w io.Writer) error {
		return html.Render(w, node)
	}

	// Function to traverse the HTML tree
	var traverse func(*html.Node) bool
	traverse = func(node *html.Node) bool {
		// Check if this is the start anchor
		if !found && hasAnchor(node, startAnchor) {
			found = true
			capturing = true

			// Render the start anchor node itself (include the tag) if requested
			if includeStartTag {
				if err := renderNode(node, buf); err != nil {
					// If there's an error, just continue without capturing this node
					// but still mark it as found
				}
			} else {
				// Render just the content of the start anchor node (not the node itself)
				if err := renderNodeContent(node, buf); err != nil {
					// If there's an error, just continue without capturing this node
					// but still mark it as found
				}
			}

			// If we rendered the node completely, we don't need to traverse its children again
			if includeStartTag {
				return false
			}

			// Continue traversal to find the end anchor
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if traverse(c) {
					return true // End found
				}
			}
			return false
		}

		// Check if this is the end anchor
		if capturing && nextAnchor != "" && hasAnchor(node, nextAnchor) {
			endFound = true
			capturing = false
			return true
		}

		// If we're capturing, add this node to the result
		if capturing {
			if err := renderNode(node, buf); err != nil {
				// If there's an error, try to at least get the text
				if node.Type == html.TextNode {
					buf.WriteString(node.Data)
				}
			}
			return false
		}

		// Continue traversal if we haven't found the end yet
		if !endFound {
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if traverse(c) {
					return true // End found
				}
			}
		}

		return endFound
	}

	// Start traversal
	traverse(n)

	return found, nil
}

// findAnchorPositions finds the positions of the start and end anchors in the HTML content.
// Returns the position after the start anchor tag, the position before the end anchor tag,
// and the position of the start of the start anchor tag.
func findAnchorPositions(content string, startAnchor, nextAnchor string) (int, int, int) {
	startPos := -1
	endPos := -1
	startTagPos := -1

	// Define patterns to look for
	startPatterns := []string{
		fmt.Sprintf(`id="%s"`, startAnchor),
		fmt.Sprintf(`id='%s'`, startAnchor),
		fmt.Sprintf(`name="%s"`, startAnchor),
		fmt.Sprintf(`name='%s'`, startAnchor),
	}

	endPatterns := []string{
		fmt.Sprintf(`id="%s"`, nextAnchor),
		fmt.Sprintf(`id='%s'`, nextAnchor),
		fmt.Sprintf(`name="%s"`, nextAnchor),
		fmt.Sprintf(`name='%s'`, nextAnchor),
	}

	// Find the start anchor
	for _, pattern := range startPatterns {
		pos := strings.Index(content, pattern)
		if pos != -1 {
			// Go back to find the beginning of the tag containing the anchor
			tagStart := pos
			for tagStart > 0 && content[tagStart] != '<' {
				tagStart--
			}
			if tagStart >= 0 {
				startTagPos = tagStart
			}

			// Find the end of the tag containing the anchor
			closePos := strings.Index(content[pos:], ">")
			if closePos != -1 {
				startPos = pos + closePos + 1
				break
			}
		}
	}

	// If start anchor found, find the end anchor
	if startPos != -1 && nextAnchor != "" {
		for _, pattern := range endPatterns {
			pos := strings.Index(content[startPos:], pattern)
			if pos != -1 {
				// Find the beginning of the tag containing the anchor
				// Search backwards from the anchor to find the opening '<'
				subContent := content[startPos : startPos+pos]
				lastOpenTag := strings.LastIndex(subContent, "<")
				if lastOpenTag != -1 {
					endPos = startPos + lastOpenTag
					break
				}
			}
		}
	}

	return startPos, endPos, startTagPos
}
