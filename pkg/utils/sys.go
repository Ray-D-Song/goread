package utils

import (
	"os"

	"golang.org/x/term"
)

func GetTermSize() (int, int) {
	fd := os.Stdout.Fd()
	width, height, err := term.GetSize(int(fd))
	if err != nil {
		return 0, 0
	}
	return width, height
}
