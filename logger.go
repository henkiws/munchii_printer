package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logMu     sync.Mutex
	logBuffer []string
	maxBuffer = 200
)

// logStatus menulis log ke buffer memori + file, dengan prefix level otomatis.
// Format: [15:04:05] LEVEL [context] pesan
func logStatus(msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)

	logMu.Lock()
	logBuffer = append(logBuffer, line)
	if len(logBuffer) > maxBuffer {
		logBuffer = logBuffer[len(logBuffer)-maxBuffer:]
	}
	logMu.Unlock()

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

// ── File logging ──────────────────────────────────────────────────────────────

func getLogPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "kypesen-printer.log"
	}
	return filepath.Join(filepath.Dir(exe), "kypesen-printer.log")
}

func writeLogFile(line string) {
	f, err := os.OpenFile(getLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}

// ── Auto-cleanup: hapus log lebih dari 1 bulan ────────────────────────────────

// StartLogCleanup menjalankan cleanup setiap hari pukul 00:05.
func StartLogCleanup() {
	go func() {
		// Cleanup pertama saat startup
		cleanOldLogs()

		// Lalu setiap 24 jam
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldLogs()
		}
	}()
}

// cleanOldLogs membaca file log line by line, membuang baris yang lebih tua
// dari 1 bulan, lalu menulis ulang file.
func cleanOldLogs() {
	logPath := getLogPath()
	cutoff := time.Now().AddDate(0, -1, 0) // 1 bulan yang lalu

	f, err := os.Open(logPath)
	if err != nil {
		return // file belum ada, skip
	}

	var kept []string
	removed := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		t := parseLogTimestamp(line)
		if t.IsZero() || t.After(cutoff) {
			kept = append(kept, line)
		} else {
			removed++
		}
	}
	f.Close()

	if removed == 0 {
		return
	}

	// Tulis ulang file dengan baris yang masih valid
	tmp := logPath + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	w := bufio.NewWriter(out)
	for _, line := range kept {
		fmt.Fprintln(w, line)
	}
	w.Flush()
	out.Close()

	os.Rename(tmp, logPath)

	logStatus(fmt.Sprintf("INFO [Logger] Auto-cleanup: %d baris lama dihapus (cutoff: %s)",
		removed, cutoff.Format("2006-01-02")))
}

// parseLogTimestamp mengekstrak timestamp dari format "[2006-01-02 15:04:05] ..."
func parseLogTimestamp(line string) time.Time {
	if len(line) < 21 || line[0] != '[' {
		return time.Time{}
	}
	// Cari penutup ']'
	end := strings.Index(line, "]")
	if end < 1 {
		return time.Time{}
	}
	ts := line[1:end]
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}