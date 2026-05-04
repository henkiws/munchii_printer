package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMu      sync.Mutex
	logBuffer  []string
	maxBuffer  = 100
)

func logStatus(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)

	logMu.Lock()
	logBuffer = append(logBuffer, line)
	if len(logBuffer) > maxBuffer {
		logBuffer = logBuffer[len(logBuffer)-maxBuffer:]
	}
	logMu.Unlock()

	// Also write to log file
	writeLogFile(line)
}

func getRecentLogs(n int) []string {
	logMu.Lock()
	defer logMu.Unlock()
	if n > len(logBuffer) {
		n = len(logBuffer)
	}
	result := make([]string, n)
	copy(result, logBuffer[len(logBuffer)-n:])
	return result
}

func writeLogFile(line string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exe), "kypesen-printer.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}
