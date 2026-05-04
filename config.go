package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConnType defines how we connect to the printer
const (
	ConnNetwork   = "network"   // TCP/IP via IP address
	ConnBluetooth = "bluetooth" // Bluetooth via COM port (e.g. COM3)
	ConnUSB       = "usb"       // USB via COM port (e.g. COM4) or Windows printer name
)

type PrinterConfig struct {
	ID               int    `json:"id"`
	PrinterName      string `json:"printer_name"`
	ConnType         string `json:"conn_type"`          // "network" | "bluetooth" | "usb"
	PrinterIPAddress string `json:"printer_ip_address"` // used when ConnType == network
	PrinterPort      int    `json:"printer_port"`       // TCP port, default 9100
	COMPort          string `json:"com_port"`           // used when ConnType == bluetooth | usb  e.g. "COM3"
	BaudRate         int    `json:"baud_rate"`          // for COM port, default 9600
	WindowsPrinter   string `json:"windows_printer"`    // Windows printer name for USB (alternative to COM)
	ServerURL        string `json:"server_url"`
	PollingTime      *int   `json:"polling_time"` // null = use default 5s
}

// GetConnType returns conn type with fallback to "network"
func (p PrinterConfig) GetConnType() string {
	switch p.ConnType {
	case ConnBluetooth, ConnUSB, ConnNetwork:
		return p.ConnType
	default:
		return ConnNetwork
	}
}

// GetPort returns TCP port with default 9100
func (p PrinterConfig) GetPort() int {
	if p.PrinterPort > 0 {
		return p.PrinterPort
	}
	return 9100
}

// GetBaudRate returns baud rate with default 9600
func (p PrinterConfig) GetBaudRate() int {
	if p.BaudRate > 0 {
		return p.BaudRate
	}
	return 9600
}

// GetPollingSeconds returns polling interval in seconds, default 5
func (p PrinterConfig) GetPollingSeconds() int {
	if p.PollingTime != nil && *p.PollingTime > 0 {
		return *p.PollingTime
	}
	return 5
}

// ConnSummary returns a short human-readable connection string for the UI
func (p PrinterConfig) ConnSummary() string {
	switch p.GetConnType() {
	case ConnNetwork:
		return fmt.Sprintf("%s:%d", p.PrinterIPAddress, p.GetPort())
	case ConnBluetooth:
		return fmt.Sprintf("BT:%s@%d", p.COMPort, p.GetBaudRate())
	case ConnUSB:
		if p.WindowsPrinter != "" {
			return fmt.Sprintf("USB:%s", p.WindowsPrinter)
		}
		return fmt.Sprintf("USB:%s@%d", p.COMPort, p.GetBaudRate())
	}
	return "unknown"
}

func getConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "printers.json"
	}
	return filepath.Join(filepath.Dir(exe), "printers.json")
}

func loadPrinters() ([]PrinterConfig, error) {
	path := getConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		empty := []PrinterConfig{}
		data, _ := json.MarshalIndent(empty, "", "    ")
		os.WriteFile(path, data, 0644)
		return empty, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var printers []PrinterConfig
	if err := json.Unmarshal(data, &printers); err != nil {
		return nil, err
	}
	return printers, nil
}

func savePrinters(printers []PrinterConfig) error {
	for i := range printers {
		printers[i].ID = i
	}
	data, err := json.MarshalIndent(printers, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}

func addPrinter(p PrinterConfig) error {
	printers, err := loadPrinters()
	if err != nil {
		return err
	}
	p.ID = len(printers)
	printers = append(printers, p)
	return savePrinters(printers)
}

func editPrinter(id int, cfg PrinterConfig) error {
	printers, err := loadPrinters()
	if err != nil {
		return err
	}
	for i, p := range printers {
		if p.ID == id {
			cfg.ID = id
			printers[i] = cfg
			return savePrinters(printers)
		}
	}
	return fmt.Errorf("printer id %d not found", id)
}

func deletePrinter(id int) error {
	printers, err := loadPrinters()
	if err != nil {
		return err
	}
	newList := []PrinterConfig{}
	for _, p := range printers {
		if p.ID != id {
			newList = append(newList, p)
		}
	}
	return savePrinters(newList)
}