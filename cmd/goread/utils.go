package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ray-d-song/goread/pkg/config"
	"github.com/ray-d-song/goread/pkg/epub"
	"github.com/ray-d-song/goread/pkg/parser"
)

// isFile checks if a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// findFileInHistory finds a file in the history
func findFileInHistory(cfg *config.Config, args []string) string {
	// Check if the first argument is a number
	if len(args) == 1 {
		if num, err := strconv.Atoi(args[0]); err == nil {
			// Get the nth file from the history
			var i int
			for file := range cfg.States {
				if i == num-1 {
					return file
				}
				i++
			}
		}
	}

	// Try to match the arguments against the history
	// We don't actually use the pattern variable directly, but we use args in the loop below
	_ = strings.Join(args, " ")
	var bestMatch string
	var bestScore int

	for file := range cfg.States {
		// Calculate a simple match score
		score := 0
		for _, arg := range args {
			if strings.Contains(strings.ToLower(file), strings.ToLower(arg)) {
				score++
			}
		}

		if score > bestScore {
			bestScore = score
			bestMatch = file
		}
	}

	return bestMatch
}

// printHelp prints the help message
func printHelp() {
	fmt.Println(`Usages:
    goread             read last epub
    goread EPUBFILE    read EPUBFILE
    goread STRINGS     read matched STRINGS from history
    goread NUMBER      read file from history
                      with associated NUMBER

Options:
    -r              print reading history
    -d              dump epub
    -h, --help      print short, long help

Key Binding:
    Help             : ?
    Quit             : q
    Scroll down      : DOWN      j
    Scroll up        : UP        k
    Half screen up   : C-u
    Half screen dn   : C-d
    Page down        : PGDN      RIGHT   SPC
    Page up          : PGUP      LEFT
    Next chapter     : n
    Prev chapter     : p
    Beginning of ch  : HOME      g
    End of ch        : END       G
    Open image       : o
    Search           : /
    Next Occurrence  : n
    Prev Occurrence  : N
    Toggle width     : =
    ToC              : TAB       t
    Metadata         : m
    Mark pos to n    : b[n]
    Jump to pos n    : ` + "`" + `[n]
    Switch colorsch  : c`)
}

// printVersion prints the version information
func printVersion() {
	fmt.Printf("goread %s\n", version)
	fmt.Printf("%s License\n", license)
	fmt.Printf("Copyright (c) 2023 %s\n", author)
	fmt.Println(url)
}

// printHistory prints the reading history
func printHistory(cfg *config.Config) {
	fmt.Println("Reading history:")
	var i int
	for file, state := range cfg.States {
		marker := " "
		if state.LastRead {
			marker = "*"
		}
		fmt.Printf("%3d%s %s\n", i+1, marker, file)
		i++
	}
}

// dumpEpub dumps the EPUB content
func dumpEpub(filePath string) {
	// Open the EPUB file
	book, err := epub.NewEpub(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening EPUB file: %v\n", err)
		os.Exit(1)
	}
	defer book.Close()

	// Dump the content
	for i := range book.Contents {
		content, err := book.GetChapterContent(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading chapter: %v\n", err)
			continue
		}

		// Parse the HTML content
		text, err := parser.DumpHTML(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing HTML: %v\n", err)
			continue
		}

		// Print the text
		fmt.Print(text)
	}
}
