//go:build !windows

package main

func registerAutoStart() error   { return nil }
func unregisterAutoStart() error { return nil }
func isAutoStartEnabled() bool   { return false }
func getExePath() string         { return "" }
