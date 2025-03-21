package main

import (
	"fmt"
	"os"
	"sort"
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
			// Get the files in the same order as displayed in history
			files := getOrderedHistoryFiles(cfg)

			// The number is 1-indexed (as shown in the history output)
			if num > 0 && num <= len(files) {
				return files[num-1]
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

// getOrderedHistoryFiles returns files from config in a consistent order
func getOrderedHistoryFiles(cfg *config.Config) []string {
	var files []string
	for file := range cfg.States {
		files = append(files, file)
	}

	// Go's map iteration is random, so ensure consistent ordering
	// Here we sort by the file path for consistency
	// Note: we could add more sorting logic here if needed
	sort.Strings(files)

	return files
}

// printHelp prints the help message
func printHelp() {
	fmt.Println(`
Usages:
    goread             read last epub
    goread EPUBFILE    read EPUBFILE
    goread STRINGS     read matched STRINGS from history
    goread NUMBER      read file from history
                      with associated NUMBER

Options:
    -r              print reading history
    -d              dump epub
    -h, --help      print short, long help

Key Bindings:
    Help             : ?
    Quit             : q
    ToC              : t
    Next chapter     : n
    Prev chapter     : N
    Search           : /
    Scroll down      : j
    Scroll up        : k
    Half screen up   : C-u
    Half screen dn   : C-d
    Beginning of ch  : g
    End of ch        : G
    Open image       : o
    Increase width   : +
    Decrease width   : -
    Metadata         : m
    Switch colorsch  : c

Press Esc or Enter to close
`)
}

// printVersion prints the version information
func printVersion() {
	fmt.Printf("goread %s\n", version)
	fmt.Printf("%s License\n", license)
	fmt.Printf("Copyright (c) 2025 %s\n", author)
	fmt.Println(url)
}

// printHistory prints the reading history
func printHistory(cfg *config.Config) {
	fmt.Println("Reading history:")

	// Get files in consistent order
	files := getOrderedHistoryFiles(cfg)

	// Print each file with its index
	for i, file := range files {
		state := cfg.States[file]
		marker := " "
		if state.LastRead {
			marker = "*"
		}
		fmt.Printf("%3d %s %s\n", i+1, marker, file)
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
	for i := range book.TOC.Slice {
		content, err := book.GetChapterContents(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading chapter: %v\n", err)
			continue
		}

		// Parse the HTML content
		text, err := parser.DumpHTML(content.Text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing HTML: %v\n", err)
			continue
		}

		// Print the text
		fmt.Print(text)
	}
}
