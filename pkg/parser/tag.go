package parser

import (
	"fmt"
	htmllib "html"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

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
	} else if tag == "code" {
		// If already inside a pre tag, this is a code block
		if p.isPref {
			p.isCode = true
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
