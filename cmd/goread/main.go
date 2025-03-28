package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ray-d-song/goread/pkg/config"
	"github.com/ray-d-song/goread/pkg/epub"
	"github.com/ray-d-song/goread/pkg/reader"
	"github.com/ray-d-song/goread/pkg/utils"
)

const (
	version = "0.1.0"
	license = "MIT"
	author  = "Ray-D-Song"
	url     = "https://github.com/ray-d-song/goread"
)

var (
	helpFlag     = flag.Bool("h", false, "Print help message")
	helpLongFlag = flag.Bool("help", false, "Print help message")
	versionFlag  = flag.Bool("v", false, "Print version information")
	historyFlag  = flag.Bool("r", false, "Print reading history")
	dumpFlag     = flag.Bool("d", false, "Dump EPUB content")
)

func main() {
	// Initialize debug logger
	utils.InitDebugLogger()
	defer utils.CloseDebugLogger()

	flag.Parse()

	// Handle help and version flags first (no config needed)
	if *helpFlag || *helpLongFlag {
		printHelp()
		os.Exit(0)
	}

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	// Load configuration once for all subsequent operations
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Handle history flag separately
	if *historyFlag {
		printHistory(cfg)
		os.Exit(0)
	}

	// Get the file to read
	var filePath string
	args := flag.Args()

	if len(args) == 0 {
		// No arguments, try to get the last read file
		lastRead, ok := cfg.GetLastRead()
		if !ok {
			printHelp()
			fmt.Fprintf(os.Stderr, "Error: No last read file found\n")
			os.Exit(1)
		}
		filePath = lastRead
	} else if len(args) == 1 && isFile(args[0]) {
		// Single argument is a file
		filePath = args[0]
		// Convert to absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to absolute path: %v\n", err)
			os.Exit(1)
		}
		filePath = absPath
	} else {
		// Try to match the arguments against the history
		filePath = findFileInHistory(cfg, args)
		if filePath == "" {
			printHelp()
			fmt.Fprintf(os.Stderr, "Error: No matching file found in history\n")
			os.Exit(1)
		}
	}

	// Check if we should dump the EPUB content
	if *dumpFlag {
		dumpEpub(filePath)
		os.Exit(0)
	}

	// Read the EPUB file
	book, err := epub.NewEpub(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening EPUB file: %v\n", err)
		os.Exit(1)
	}
	defer book.Close()

	// Get the reading state
	state, ok := cfg.GetState(filePath)
	if !ok {
		// No state, start from the beginning
		state = config.State{
			Index: 0,
			Width: 80,
			Pos:   0,
			Pctg:  0,
		}

		// Only set and save state if it's a new file
		cfg.SetState(filePath, state)
	}

	// Set the last read file
	cfg.SetLastRead(filePath)
	cfg.Save()

	// Start the reader
	reader := reader.NewReader(book, cfg, filePath)

	reader.UI.SetColorScheme(state.ColorScheme)

	reader.Run(state.Index, state.Width, state.Pos, state.Pctg)
}
