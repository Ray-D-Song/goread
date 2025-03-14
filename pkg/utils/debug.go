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
	logPath := filepath.Join(os.TempDir(), "goread_debug.log")
	var err error
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
