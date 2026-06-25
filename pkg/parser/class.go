package parser

import (
	"strings"

	"golang.org/x/net/html"
)

// hasClasses reports whether the node has a class attribute containing all tokens.
func hasClasses(n *html.Node, classes ...string) bool {
	classAttr := ""
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classAttr = attr.Val
			break
		}
	}
	if classAttr == "" {
		return false
	}

	tokens := strings.Fields(classAttr)
	tokenSet := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		tokenSet[token] = true
	}

	for _, class := range classes {
		if !tokenSet[class] {
			return false
		}
	}

	return true
}