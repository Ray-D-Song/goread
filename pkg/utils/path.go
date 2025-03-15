package utils

import (
	"path/filepath"
	"strings"
)

// ResolvePath resolves a relative path against a base path
func ResolvePath(basePath, relativePath string) string {
	baseDir := filepath.Dir(basePath)
	if baseDir == "." {
		return relativePath
	}

	// Handle paths with fragments
	fragment := ""
	if idx := strings.LastIndex(relativePath, "#"); idx != -1 {
		fragment = relativePath[idx:]
		relativePath = relativePath[:idx]
	}

	resolved := filepath.Join(baseDir, relativePath)

	return resolved + fragment
}
