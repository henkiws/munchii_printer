package main

import (
	"fmt"
	"sync"
	"time"
)

// ── Polling manager ───────────────────────────────────────────────────────────

type PollerManager struct {
	mu      sync.Mutex
	pollers map[int]*Poller
}

type Poller struct {
	cfg    PrinterConfig
	stop   chan struct{}
	status string
}

var manager = &PollerManager{
	pollers: make(map[int]*Poller),
}

func (m *PollerManager) StartAll() {
	printers, err := loadPrinters()
	if err != nil {
		logStatus("Error loading printers: " + err.Error())
		return
	}
	for _, cfg := range printers {
		m.Start(cfg)
	}
}

func (m *PollerManager) Start(cfg PrinterConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing poller for this ID if any
	if existing, ok := m.pollers[cfg.ID]; ok {
		close(existing.stop)
	}

	p := &Poller{
		cfg:    cfg,
		stop:   make(chan struct{}),
		status: "starting",
	}
	m.pollers[cfg.ID] = p

	go func() {
		interval := time.Duration(cfg.GetPollingSeconds()) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logStatus(fmt.Sprintf("[%s] Polling started (every %ds)", cfg.PrinterName, cfg.GetPollingSeconds()))

		for {
			select {
			case <-ticker.C:
				p.poll()
			case <-p.stop:
				logStatus(fmt.Sprintf("[%s] Polling stopped", cfg.PrinterName))
				return
			}
		}
	}()
}

func (m *PollerManager) Stop(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.pollers[id]; ok {
		close(p.stop)
		delete(m.pollers, id)
	}
}

func (m *PollerManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, p := range m.pollers {
		close(p.stop)
		delete(m.pollers, id)
	}
}

func (m *PollerManager) Restart() {
	m.StopAll()
	time.Sleep(500 * time.Millisecond)
	m.StartAll()
}

func (m *PollerManager) GetStatuses() map[int]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[int]string)
	for id, p := range m.pollers {
		result[id] = p.status
	}
	return result
}

// ── Single poll execution ─────────────────────────────────────────────────────

func (p *Poller) poll() {
	p.status = "polling"

	resp, err := fetchKypesen(p.cfg.ServerURL)
	if err != nil {
		p.status = "error: " + err.Error()
		logStatus(fmt.Sprintf("[%s] API error: %v", p.cfg.PrinterName, err))
		return
	}

	orderCount := len(resp.Response.Data.Orders)
	if orderCount == 0 {
		p.status = "ok (no orders)"
		return
	}

	logStatus(fmt.Sprintf("[%s] Found %d order(s), printing...", p.cfg.PrinterName, orderCount))

	if err := processPrintJob(p.cfg, resp); err != nil {
		p.status = "print error: " + err.Error()
		logStatus(fmt.Sprintf("[%s] Print error: %v", p.cfg.PrinterName, err))
		return
	}

	p.status = fmt.Sprintf("ok (printed %d)", orderCount)
	logStatus(fmt.Sprintf("[%s] Printed %d order(s) successfully", p.cfg.PrinterName, orderCount))
}
