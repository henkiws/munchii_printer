//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// newPSCommand menulis script ke temp .ps1 lalu jalankan via -File.
// Ini menghindari batas panjang argumen Windows (~32KB).
func newPSCommand(script string) *exec.Cmd {
	tmpFile, err := os.CreateTemp("", "munchii-ps-*.ps1")
	if err != nil {
		// fallback ke -Command jika gagal buat temp file
		cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd
	}

	tmpPath := tmpFile.Name()
	// Tulis dengan UTF-8 BOM agar PowerShell baca encoding dengan benar
	tmpFile.Write([]byte{0xEF, 0xBB, 0xBF})
	tmpFile.WriteString(script)
	tmpFile.Close()

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", tmpPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// listCOMPorts returns available COM ports from Windows registry
func listCOMPorts() []string {
	ports := []string{}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DEVICEMAP\SERIALCOMM`, registry.QUERY_VALUE)
	if err != nil {
		return ports
	}
	defer k.Close()
	names, err := k.ReadValueNames(-1)
	if err != nil {
		return ports
	}
	for _, name := range names {
		val, _, err := k.GetStringValue(name)
		if err == nil && val != "" {
			ports = append(ports, val)
		}
	}
	return ports
}

// listWindowsPrinters returns installed Windows printer names via wmic
func listWindowsPrinters() []string {
	out, err := exec.Command("wmic", "printer", "get", "name", "/format:list").Output()
	if err != nil {
		return []string{}
	}
	var printers []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "Name="))
			if name != "" {
				printers = append(printers, name)
			}
		}
	}
	return printers
}

// hideCmdWindow hides console window for a command
func hideCmdWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// Satisfy unused import
var _ = fmt.Sprintf
