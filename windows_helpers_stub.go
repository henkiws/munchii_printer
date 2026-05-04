//go:build !windows

package main

import "os/exec"

func newPSCommand(script string) *exec.Cmd {
	return exec.Command("echo", "not supported")
}

func listCOMPorts() []string        { return []string{} }
func listWindowsPrinters() []string { return []string{} }
func hideCmdWindow(cmd *exec.Cmd)   {}
