package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"bytes"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	wsReconnectBaseDelay = 3 * time.Second
	wsReconnectMaxDelay  = 60 * time.Second
	wsPingInterval       = 30 * time.Second
	wsPongTimeout        = 10 * time.Second
	printTCPTimeout      = 10 * time.Second
)

// ── Manager ───────────────────────────────────────────────────────────────────

// WSManager mengelola satu WebSocket connection per printer UUID.
// Setiap printer config punya server_url yang mengandung UUID → dipakai
// sebagai client_id ke Go Hub.
type WSManager struct {
	mu      sync.Mutex
	clients map[int]*WSClient // key: printer config ID
}

type WSClient struct {
	cfg      PrinterConfig
	hubURL   string // ws://host:port/ws?client_id=UUID
	stopCh   chan struct{}
	status   string
	mu       sync.Mutex
}

var wsManager = &WSManager{
	clients: make(map[int]*WSClient),
}

// StartAll membaca semua printer config dan mulai koneksi WS masing-masing.
func (m *WSManager) StartAll() {
	printers, err := loadPrinters()
	if err != nil {
		logStatus("ERROR [WSManager] Gagal load printers: " + err.Error())
		return
	}
	for _, cfg := range printers {
		m.Start(cfg)
	}
}

func (m *WSManager) Start(cfg PrinterConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop client lama jika ada
	if old, ok := m.clients[cfg.ID]; ok {
		close(old.stopCh)
		delete(m.clients, cfg.ID)
	}

	hubURL := buildHubWSURL(cfg.ServerURL)
	if hubURL == "" {
		logStatus(fmt.Sprintf("ERROR [%s] Tidak bisa build WS URL dari: %s", cfg.PrinterName, cfg.ServerURL))
		return
	}

	c := &WSClient{
		cfg:    cfg,
		hubURL: hubURL,
		stopCh: make(chan struct{}),
		status: "starting",
	}
	m.clients[cfg.ID] = c
	go c.run()
}

func (m *WSManager) Stop(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[id]; ok {
		close(c.stopCh)
		delete(m.clients, id)
	}
}

func (m *WSManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		close(c.stopCh)
		delete(m.clients, id)
	}
}

func (m *WSManager) Restart() {
	m.StopAll()
	time.Sleep(500 * time.Millisecond)
	m.StartAll()
}

func (m *WSManager) GetStatuses() map[int]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[int]string)
	for id, c := range m.clients {
		c.mu.Lock()
		result[id] = c.status
		c.mu.Unlock()
	}
	return result
}

func (c *WSClient) setStatus(s string) {
	c.mu.Lock()
	c.status = s
	c.mu.Unlock()
}

// ── URL Builder ───────────────────────────────────────────────────────────────

// buildHubWSURL mengekstrak host dari server_url Laravel lalu membuat
// WebSocket URL ke Go Hub.
// Contoh: "http://192.168.1.10:8001/api/v1/print/UUID-XXX"
//      → "ws://192.168.1.10:8080/ws?client_id=UUID-XXX"
func buildHubWSURL(serverURL string) string {
	// Ekstrak UUID dari akhir path
	parts := strings.Split(strings.TrimRight(serverURL, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	uuid := parts[len(parts)-1]
	if uuid == "" {
		return ""
	}

	// Ekstrak host (tanpa port Laravel)
	// serverURL = "http://host:laravelPort/..."
	noScheme := strings.TrimPrefix(serverURL, "https://")
	noScheme = strings.TrimPrefix(noScheme, "http://")
	hostPart := strings.SplitN(noScheme, "/", 2)[0]
	host := strings.SplitN(hostPart, ":", 2)[0] // buang port Laravel

	// Go Hub selalu di port 8080
	return fmt.Sprintf("ws://%s:8080/ws?client_id=%s", host, uuid)
}

// ── WebSocket run loop (auto-reconnect) ───────────────────────────────────────

func (c *WSClient) run() {
	delay := wsReconnectBaseDelay
	attempt := 0

	for {
		select {
		case <-c.stopCh:
			logStatus(fmt.Sprintf("INFO [%s] WebSocket client dihentikan", c.cfg.PrinterName))
			return
		default:
		}

		attempt++
		if attempt > 1 {
			logStatus(fmt.Sprintf("INFO [%s] Reconnect attempt #%d dalam %v...", c.cfg.PrinterName, attempt, delay))
			select {
			case <-time.After(delay):
			case <-c.stopCh:
				return
			}
			delay = min(delay*2, wsReconnectMaxDelay)
		}

		c.setStatus(fmt.Sprintf("connecting (attempt #%d)", attempt))
		logStatus(fmt.Sprintf("INFO [%s] Menghubungkan ke Hub: %s", c.cfg.PrinterName, c.hubURL))

		err := c.connect()
		if err != nil {
			c.setStatus("disconnected: " + err.Error())
			logStatus(fmt.Sprintf("ERROR [%s] Koneksi WS gagal: %v", c.cfg.PrinterName, err))
			continue
		}

		// Koneksi sukses, reset delay
		delay = wsReconnectBaseDelay
		attempt = 0
	}
}

func (c *WSClient) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, resp, err := dialer.Dial(c.hubURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("dial gagal (HTTP %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("dial gagal: %w", err)
	}
	defer conn.Close()

	c.setStatus("connected")
	logStatus(fmt.Sprintf("OK [%s] Terhubung ke Hub ✓", c.cfg.PrinterName))

	// Ping/pong keepalive
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongTimeout + wsPingInterval))
		return nil
	})

	pingTicker := time.NewTicker(wsPingInterval)
	defer pingTicker.Stop()

	// Read loop di goroutine terpisah
	msgCh := make(chan []byte, 8)
	errCh := make(chan error, 1)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			msgCh <- msg
		}
	}()

	for {
		select {
		case <-c.stopCh:
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return nil

		case err := <-errCh:
			c.setStatus("disconnected")
			return fmt.Errorf("read error: %w", err)

		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("ping gagal: %w", err)
			}

		case raw := <-msgCh:
			conn.SetReadDeadline(time.Now().Add(wsPongTimeout + wsPingInterval))
			go c.handleMessage(raw)
		}
	}
}

// ── Message handler ───────────────────────────────────────────────────────────

func (c *WSClient) handleMessage(raw []byte) {
	// Cek apakah ini ACK koneksi dari Hub
	var ack map[string]string
	if json.Unmarshal(raw, &ack) == nil {
		if ack["status"] == "connected" {
			logStatus(fmt.Sprintf("OK [%s] Hub ACK: client_id=%s", c.cfg.PrinterName, ack["client_id"]))
			return
		}
	}

	// Parse payload utuh dari Laravel
	var payload HubPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		logStatus(fmt.Sprintf("ERROR [%s] Gagal parse JSON payload: %v\nRaw: %s",
			c.cfg.PrinterName, err, truncate(string(raw), 300)))
		return
	}

	logStatus(fmt.Sprintf("INFO [%s] Payload diterima — from=%s ip_printer=%s",
		c.cfg.PrinterName, payload.From, payload.IPAddress))

	switch payload.From {
	case "note":
		c.handleNotes(payload)
	case "reporting":
		c.handleReports(payload)
	case "order":
		c.handleOrders(payload)
	default:
		logStatus(fmt.Sprintf("WARN [%s] Unknown from=%q, skip", c.cfg.PrinterName, payload.From))
	}
}

// ── Note ──────────────────────────────────────────────────────────────────────

func (c *WSClient) handleNotes(p HubPayload) {
	if p.Data.Notes == nil {
		logStatus(fmt.Sprintf("WARN [%s] Payload note: data.notes kosong", c.cfg.PrinterName))
		return
	}
	for i, note := range p.Data.Notes {
		b64 := note.ImageBase64
		if b64 == "" {
			logStatus(fmt.Sprintf("WARN [%s] Note[%d] image_base64 kosong, skip", c.cfg.PrinterName, i))
			continue
		}
		ip := p.IPAddress
		if ip == "" {
			ip = c.cfg.PrinterIPAddress
		}
		logStatus(fmt.Sprintf("INFO [%s] Print note[%d] → %s:%d", c.cfg.PrinterName, i, ip, c.cfg.GetPort()))
		go c.printBase64Image(ip, b64, fmt.Sprintf("note[%d]", i))
	}
}

// ── Reporting ─────────────────────────────────────────────────────────────────

func (c *WSClient) handleReports(p HubPayload) {
	if p.Data.Reports == nil {
		logStatus(fmt.Sprintf("WARN [%s] Payload reporting: data.reports kosong", c.cfg.PrinterName))
		return
	}
	for i, rep := range p.Data.Reports {
		b64 := rep.ImageBase64
		if b64 == "" {
			logStatus(fmt.Sprintf("WARN [%s] Report[%d] image_base64 kosong, skip", c.cfg.PrinterName, i))
			continue
		}
		ip := p.IPAddress
		if ip == "" {
			ip = c.cfg.PrinterIPAddress
		}
		logStatus(fmt.Sprintf("INFO [%s] Print report[%d] → %s:%d", c.cfg.PrinterName, i, ip, c.cfg.GetPort()))
		go c.printBase64Image(ip, b64, fmt.Sprintf("report[%d]", i))
	}
}

// ── Order ─────────────────────────────────────────────────────────────────────

func (c *WSClient) handleOrders(p HubPayload) {
	if p.Data.Orders == nil {
		logStatus(fmt.Sprintf("WARN [%s] Payload order: data.orders kosong", c.cfg.PrinterName))
		return
	}
	for i, ow := range p.Data.Orders {
		b64 := ow.Order.ImageBase64
		if b64 == "" {
			logStatus(fmt.Sprintf("WARN [%s] Order[%d] image_base64 kosong, skip", c.cfg.PrinterName, i))
			continue
		}
		ip := p.IPAddress
		if ip == "" {
			ip = c.cfg.PrinterIPAddress
		}
		label := fmt.Sprintf("order[%d] id=%d", i, ow.Order.ID)
		logStatus(fmt.Sprintf("INFO [%s] Print %s → %s:%d", c.cfg.PrinterName, label, ip, c.cfg.GetPort()))
		go c.printBase64Image(ip, b64, label)
	}
}

// ── Core: decode base64 → PNG → ESC/POS bitimage → TCP 9100 ──────────────────

func (c *WSClient) printBase64Image(printerIP, b64data, label string) {
	start := time.Now()

	// 1. Strip data URI prefix jika ada (data:image/png;base64,...)
	raw := b64data
	if idx := strings.Index(raw, ","); idx != -1 {
		raw = raw[idx+1:]
	}

	// 2. Decode base64
	imgBytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		logStatus(fmt.Sprintf("ERROR [%s] %s — base64 decode gagal: %v", c.cfg.PrinterName, label, err))
		return
	}

	// 3. Decode PNG → image.Image
	img, err := png.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		// Coba format lain (jpeg, bmp, dll)
		img, _, err = image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			logStatus(fmt.Sprintf("ERROR [%s] %s — decode gambar gagal: %v", c.cfg.PrinterName, label, err))
			return
		}
	}

	logStatus(fmt.Sprintf("INFO [%s] %s — gambar %dx%d px, size=%d bytes",
		c.cfg.PrinterName, label, img.Bounds().Dx(), img.Bounds().Dy(), len(imgBytes)))

	// 4. Koneksi ke printer sesuai conn_type
	printer, err := newPrinterFromConfig(c.cfg)
	if err != nil {
		logStatus(fmt.Sprintf("ERROR [%s] %s — koneksi printer gagal [%s → %s]: %v",
			c.cfg.PrinterName, label, c.cfg.GetConnType(), c.cfg.ConnSummary(), err))
		return
	}
	defer printer.close()

	// 5. Encode ke ESC/POS bit-image dan kirim
	printer.init()
	printer.alignLeft()
	if err := printer.bitImage(img); err != nil {
		logStatus(fmt.Sprintf("ERROR [%s] %s — ESC/POS bitImage gagal: %v", c.cfg.PrinterName, label, err))
		return
	}
	printer.feed()
	printer.feed()
	printer.feed()
	printer.cut()

	if err := printer.flush(); err != nil {
		logStatus(fmt.Sprintf("ERROR [%s] %s — flush ke printer gagal: %v", c.cfg.PrinterName, label, err))
		return
	}

	elapsed := time.Since(start)
	logStatus(fmt.Sprintf("OK [%s] %s — BERHASIL DICETAK via %s (%v)",
		c.cfg.PrinterName, label, c.cfg.ConnSummary(), elapsed.Round(time.Millisecond)))
}

// ── TCP direct fallback (jika newPrinterFromConfig tidak dipakai) ─────────────

// dialPrinterTCP membuka raw TCP ke printer ESC/POS port 9100.
// Dipakai sebagai fallback eksplisit jika perlu.
func dialPrinterTCP(ip string, port int) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, printTCPTimeout)
	if err != nil {
		return nil, fmt.Errorf("TCP dial %s gagal: %w", addr, err)
	}
	return conn, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}