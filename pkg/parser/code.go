package parser

import (
	"regexp"
	"strings"
)

// formatCodeLine formats a code line with syntax highlighting
func formatCodeLine(line string, indent string) string {
	if line == "" {
		return ""
	}

	// Apply syntax highlighting
	highlightedLine := applyCodeHighlighting(line)

	// Handle line wrapping
	var result []string
	lines := strings.Split(highlightedLine, "\n")

	for _, l := range lines {
		result = append(result, indent+l)
	}

	return strings.Join(result, "\n")
}

// applyCodeHighlighting applies syntax highlighting based on code type
func applyCodeHighlighting(code string) string {
	// Use universal highlighting for all other code blocks
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

	// Check for SQL keywords (case insensitive)
	if isSQLKeyword(token) {
		return "[#FF00AA]" + token + "[-]"
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

// isSQLKeyword checks if a token is an SQL keyword (case insensitive)
func isSQLKeyword(token string) bool {
	// Convert token to uppercase for case-insensitive comparison
	upperToken := strings.ToUpper(token)

	sqlKeywords := map[string]bool{
		// SQL commands
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "CREATE": true,
		"ALTER": true, "DROP": true, "TRUNCATE": true, "GRANT": true, "REVOKE": true,
		"COMMIT": true, "ROLLBACK": true, "SAVEPOINT": true, "SET": true,

		// SQL clauses
		"FROM": true, "WHERE": true, "GROUP": true, "HAVING": true, "ORDER": true,
		"BY": true, "LIMIT": true, "OFFSET": true, "JOIN": true, "INNER": true,
		"OUTER": true, "LEFT": true, "RIGHT": true, "FULL": true, "ON": true,
		"AS": true, "UNION": true, "ALL": true, "INTO": true, "VALUES": true,
		"DISTINCT": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
		"END": true, "WITH": true, "RECURSIVE": true,

		// SQL functions
		"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true,
		"COALESCE": true, "NULLIF": true, "CAST": true, "CONVERT": true,
		"CURRENT_DATE": true, "CURRENT_TIME": true, "CURRENT_TIMESTAMP": true,
		"EXTRACT": true, "SUBSTRING": true, "CONCAT": true, "TRIM": true,
		"UPPER": true, "LOWER": true, "LENGTH": true, "ROUND": true,

		// SQL data types
		"INT": true, "INTEGER": true, "SMALLINT": true, "BIGINT": true,
		"DECIMAL": true, "NUMERIC": true, "FLOAT": true, "REAL": true,
		"DOUBLE": true, "PRECISION": true, "CHAR": true, "VARCHAR": true,
		"TEXT": true, "DATE": true, "TIME": true, "TIMESTAMP": true,
		"BOOLEAN": true, "BLOB": true, "CLOB": true, "BINARY": true,

		// SQL constraints
		"PRIMARY": true, "KEY": true, "FOREIGN": true, "UNIQUE": true,
		"NOT": true, "NULL": true, "CHECK": true, "DEFAULT": true,
		"AUTO_INCREMENT": true, "IDENTITY": true, "REFERENCES": true,
		"CASCADE": true, "RESTRICT": true, "NO": true, "ACTION": true,

		// SQL operators
		"AND": true, "OR": true, "IN": true, "BETWEEN": true,
		"LIKE": true, "IS": true, "EXISTS": true, "ANY": true, "SOME": true,
		"INTERSECT": true, "EXCEPT": true, "MINUS": true,

		// Transaction control
		"BEGIN": true, "TRANSACTION": true, "ISOLATION": true, "LEVEL": true, "READ": true,
		"WRITE": true, "UNCOMMITTED": true, "COMMITTED": true, "REPEATABLE": true,
		"SERIALIZABLE": true,

		// Database objects
		"TABLE": true, "VIEW": true, "INDEX": true, "SEQUENCE": true,
		"TRIGGER": true, "PROCEDURE": true, "FUNCTION": true, "SCHEMA": true,
		"DATABASE": true, "COLUMN": true, "CONSTRAINT": true,

		// Other common SQL keywords
		"IF": true, "WHILE": true, "LOOP": true,
		"FOR": true, "RETURN": true, "DECLARE": true, "EXCEPTION": true,
		"RAISE": true, "HANDLER": true, "CONDITION": true, "SIGNAL": true,
		"RESIGNAL": true, "CALL": true, "EXECUTE": true, "PREPARE": true,
		"DEALLOCATE": true, "ELSEIF": true,
	}

	return sqlKeywords[upperToken]
}
