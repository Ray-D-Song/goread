package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var debugLogger *log.Logger
var debugFile *os.File

func InitDebugLogger() {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
		return
	}

	// Create .goread directory in home directory
	logDir := filepath.Join(homeDir, ".goread")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return
	}

	logPath := filepath.Join(logDir, "debug.log")
	debugFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
		return
	}

	debugLogger = log.New(debugFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger.Println("=== Debug Session Start ===", time.Now())
}

func CloseDebugLogger() {
	if debugFile != nil {
		debugLogger.Println("=== Debug Session End ===", time.Now())
		debugFile.Close()
	}
}

func DebugLog(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}
