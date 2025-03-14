package epub

import (
	"testing"
)

func TestResolvePath(t *testing.T) {
	tests := []struct {
		basePath     string
		relativePath string
		expected     string
	}{
		{
			basePath:     "dir/file.html",
			relativePath: "image.jpg",
			expected:     "dir/image.jpg",
		},
		{
			basePath:     "file.html",
			relativePath: "image.jpg",
			expected:     "image.jpg",
		},
		{
			basePath:     "dir/file.html",
			relativePath: "../image.jpg",
			expected:     "image.jpg",
		},
		{
			basePath:     "dir/subdir/file.html",
			relativePath: "../image.jpg",
			expected:     "dir/image.jpg",
		},
		{
			basePath:     "dir/file.html",
			relativePath: "subdir/image.jpg",
			expected:     "dir/subdir/image.jpg",
		},
		{
			basePath:     "dir/file.html",
			relativePath: "image.jpg#fragment",
			expected:     "dir/image.jpg#fragment",
		},
	}

	for _, test := range tests {
		result := ResolvePath(test.basePath, test.relativePath)
		if result != test.expected {
			t.Errorf("ResolvePath(%q, %q) = %q, expected %q",
				test.basePath, test.relativePath, result, test.expected)
		}
	}
}
