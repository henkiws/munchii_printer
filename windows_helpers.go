//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

func newPSCommand(script string) *exec.Cmd {
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
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

// hideCmdWindow hides console window for a command (used by COM port mode command)
func hideCmdWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// Satisfy unused import
var _ = fmt.Sprintf
