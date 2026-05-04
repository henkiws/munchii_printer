package main

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── ESC/POS byte constants ────────────────────────────────────────────────────

var (
	ESC_INIT         = []byte{0x1B, 0x40}
	ESC_ALIGN_LEFT   = []byte{0x1B, 0x61, 0x00}
	ESC_ALIGN_CENTER = []byte{0x1B, 0x61, 0x01}
	ESC_ALIGN_RIGHT  = []byte{0x1B, 0x61, 0x02}
	ESC_BOLD_ON      = []byte{0x1B, 0x45, 0x01}
	ESC_BOLD_OFF     = []byte{0x1B, 0x45, 0x00}
	ESC_FONT_A       = []byte{0x1B, 0x4D, 0x00}
	ESC_FONT_B       = []byte{0x1B, 0x4D, 0x01}
	ESC_SIZE_NORMAL  = []byte{0x1D, 0x21, 0x00}
	ESC_SIZE_DOUBLE  = []byte{0x1D, 0x21, 0x11}
	ESC_CUT          = []byte{0x1D, 0x56, 0x42, 0x00}
	ESC_FEED         = []byte{0x0A}
	ESC_UTF8         = []byte{0x1C, 0x26}
)

// ── Printer connection ────────────────────────────────────────────────────────

type EscPrinter struct {
	conn io.ReadWriteCloser
	buf  []byte
}

// newPrinterFromConfig opens the right connection based on ConnType
func newPrinterFromConfig(cfg PrinterConfig) (*EscPrinter, error) {
	switch cfg.GetConnType() {
	case ConnNetwork:
		return newNetworkPrinter(cfg.PrinterIPAddress, cfg.GetPort())
	case ConnBluetooth, ConnUSB:
		if cfg.WindowsPrinter != "" {
			return newWindowsPrinter(cfg.WindowsPrinter)
		}
		return newCOMPrinter(cfg.COMPort, cfg.GetBaudRate())
	default:
		return newNetworkPrinter(cfg.PrinterIPAddress, cfg.GetPort())
	}
}

// newNetworkPrinter connects via TCP
func newNetworkPrinter(ip string, port int) (*EscPrinter, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to printer %s: %w", addr, err)
	}
	return &EscPrinter{conn: conn}, nil
}

// newCOMPrinter opens a serial COM port (Bluetooth SPP or USB-Serial)
func newCOMPrinter(comPort string, baudRate int) (*EscPrinter, error) {
	if comPort == "" {
		return nil, fmt.Errorf("COM port not specified")
	}
	conn, err := openCOMPort(comPort, baudRate)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", comPort, err)
	}
	return &EscPrinter{conn: conn}, nil
}

// newWindowsPrinter sends to a named Windows printer via RAW spooler
func newWindowsPrinter(printerName string) (*EscPrinter, error) {
	if printerName == "" {
		return nil, fmt.Errorf("windows printer name not specified")
	}
	return &EscPrinter{conn: &winPrinterConn{name: printerName}}, nil
}

// ── COM port (CGO-free) ───────────────────────────────────────────────────────

type comConn struct {
	file *os.File
}

func openCOMPort(port string, baud int) (*comConn, error) {
	name := `\\.\` + port
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	// Configure via Windows mode command (best-effort)
	exec.Command("mode", fmt.Sprintf("%s:baud=%d parity=n data=8 stop=1", port, baud)).Run()
	return &comConn{file: f}, nil
}

func (c *comConn) Read(b []byte) (int, error)  { return c.file.Read(b) }
func (c *comConn) Write(b []byte) (int, error) { return c.file.Write(b) }
func (c *comConn) Close() error                { return c.file.Close() }

// ── Windows named printer (RAW spooler) ───────────────────────────────────────

type winPrinterConn struct {
	name string
	buf  []byte
}

func (w *winPrinterConn) Read(b []byte) (int, error) { return 0, nil }
func (w *winPrinterConn) Write(b []byte) (int, error) {
	w.buf = append(w.buf, b...)
	return len(b), nil
}
func (w *winPrinterConn) Close() error {
	tmp, err := os.CreateTemp("", "escpos_*.bin")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Write(w.buf)
	tmp.Close()
	cmd := exec.Command("cmd", "/c",
		fmt.Sprintf(`copy /b "%s" "\\%%computername%%\%s"`, tmpPath, w.name))
	err = cmd.Run()
	os.Remove(tmpPath)
	return err
}

// ── EscPrinter methods ────────────────────────────────────────────────────────

func (p *EscPrinter) write(data []byte) { p.buf = append(p.buf, data...) }
func (p *EscPrinter) text(s string)     { p.write([]byte(s)) }
func (p *EscPrinter) textln(s string)   { p.text(s + "\n") }
func (p *EscPrinter) init()             { p.write(ESC_INIT); p.write(ESC_UTF8) }
func (p *EscPrinter) alignLeft()        { p.write(ESC_ALIGN_LEFT) }
func (p *EscPrinter) alignCenter()      { p.write(ESC_ALIGN_CENTER) }
func (p *EscPrinter) alignRight()       { p.write(ESC_ALIGN_RIGHT) }
func (p *EscPrinter) boldOn()           { p.write(ESC_BOLD_ON) }
func (p *EscPrinter) boldOff()          { p.write(ESC_BOLD_OFF) }
func (p *EscPrinter) fontA()            { p.write(ESC_FONT_A) }
func (p *EscPrinter) fontB()            { p.write(ESC_FONT_B) }
func (p *EscPrinter) sizeNormal()       { p.write(ESC_SIZE_NORMAL) }
func (p *EscPrinter) sizeDouble()       { p.write(ESC_SIZE_DOUBLE) }
func (p *EscPrinter) feed()             { p.write(ESC_FEED) }
func (p *EscPrinter) cut()              { p.write(ESC_CUT) }

func (p *EscPrinter) flush() error {
	_, err := p.conn.Write(p.buf)
	p.buf = nil
	return err
}
func (p *EscPrinter) close() {
	p.flush()
	p.conn.Close()
}

// ── Image printing ────────────────────────────────────────────────────────────

func (p *EscPrinter) printImage(imgPath string) error {
	f, err := os.Open(imgPath)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return err
	}
	return p.bitImage(img)
}

func (p *EscPrinter) bitImage(img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y
	byteWidth := int(math.Ceil(float64(width) / 8.0))
	paddedWidth := byteWidth * 8
	xL := byte(byteWidth & 0xFF)
	xH := byte((byteWidth >> 8) & 0xFF)
	yL := byte(height & 0xFF)
	yH := byte((height >> 8) & 0xFF)
	p.write([]byte{0x1D, 0x76, 0x30, 0x00, xL, xH, yL, yH})
	for y := 0; y < height; y++ {
		row := make([]byte, byteWidth)
		for x := 0; x < paddedWidth; x++ {
			if x < width {
				r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
				gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 257.0
				if gray < 128 {
					row[x/8] |= 1 << uint(7-x%8)
				}
			}
		}
		p.write(row)
	}
	return nil
}

// ── wkhtmltoimage ─────────────────────────────────────────────────────────────

func getWkhtmlPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "wkhtmltoimage"
	}
	local := filepath.Join(filepath.Dir(exe), "wkhtmltoimage.exe")
	if _, err := os.Stat(local); err == nil {
		return local
	}
	return "wkhtmltoimage"
}

func renderURLtoPNG(sourceURL string, width int, destPath string) error {
	wk := getWkhtmlPath()
	cmd := exec.Command(wk, "--width", fmt.Sprintf("%d", width),
		"--quality", "100", "--format", "png", sourceURL, destPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wkhtmltoimage failed: %w\nOutput: %s", err, string(out))
	}
	return nil
}

func getTempImagePath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "escpos_tmp.png")
}

// ── Main print job ────────────────────────────────────────────────────────────

func processPrintJob(cfg PrinterConfig, apiResp *KypesenResponse) error {
	data := apiResp.Response.Data
	if len(data.Orders) == 0 {
		return nil
	}
	printerInfo := data.Printer
	paperSize := printerInfo.PrinterGroup.PrinterJob.PaperSize
	templateName := strings.ToLower(printerInfo.PrinterGroup.PrinterJob.PrinterTemplate.Name)
	width := 550
	if paperSize == 58 {
		width = 350
	}
	printer, err := newPrinterFromConfig(cfg)
	if err != nil {
		return err
	}
	defer printer.close()
	kypesenBaseURL := getKypesenBaseURL(cfg.ServerURL)
	for _, orderWrapper := range data.Orders {
		order := orderWrapper.Order
		receiptNumber := order.GetReceiptNumber()
		orderID := fmt.Sprintf("%d", order.ID)
		if printerInfo.ID != orderWrapper.FkPrinter {
			continue
		}
		var sourceURL string
		if templateName == "kitchen" {
			sourceURL = fmt.Sprintf("%sview/kitchen/order/%s?printer_group=%d",
				kypesenBaseURL, orderID, printerInfo.PrinterGroup.ID)
		} else {
			sourceURL = fmt.Sprintf("%sview/invoice/cod/%s?printer_group=%d",
				kypesenBaseURL, receiptNumber, printerInfo.PrinterGroup.ID)
		}
		destPath := getTempImagePath()
		if err := renderURLtoPNG(sourceURL, width, destPath); err != nil {
			logStatus(fmt.Sprintf("[%s] wkhtmltoimage error, fallback text: %v", cfg.PrinterName, err))
			if templateName == "kitchen" {
				printKitchenText(printer, orderWrapper, printerInfo, paperSize, "", orderWrapper.PrintCount > 1)
			} else {
				printCashierText(printer, orderWrapper, printerInfo, paperSize)
			}
			continue
		}
		if err := printer.printImage(destPath); err != nil {
			return fmt.Errorf("printing image: %w", err)
		}
		printer.feed()
		printer.feed()
		printer.feed()
		printer.cut()
		os.Remove(destPath)
	}
	return printer.flush()
}

func getKypesenBaseURL(serverURL string) string {
	parts := strings.SplitN(serverURL, "/api/", 2)
	if len(parts) == 2 {
		return parts[0] + "/"
	}
	parts = strings.SplitN(serverURL, "/", 4)
	if len(parts) >= 3 {
		return parts[0] + "//" + parts[2] + "/"
	}
	return serverURL
}

// ── Text fallback: Kitchen ────────────────────────────────────────────────────

func printKitchenText(p *EscPrinter, ow OrderWrapper, pi PrinterInfo, paperSize int, uuid string, isReprint bool) {
	o := ow.Order
	sep := "--------------------------------"
	if paperSize == 78 {
		sep = "--------------------------------------------"
	}
	p.init()
	p.fontA()
	p.sizeNormal()
	p.alignCenter()
	p.feed()
	if isReprint {
		p.textln("*** REPRINT ***")
	}
	p.textln("#" + o.GetReceiptNumber())
	p.textln("No." + fmt.Sprintf("%d", o.ID))
	p.textln(o.GetTableArea() + "/" + o.GetTableName())
	p.alignLeft()
	t, _ := time.Parse("2006-01-02T15:04:05.000000Z", o.CreatedAt)
	t = t.In(time.FixedZone("WIB", 7*3600))
	p.textln(t.Format("02 Jan 2006 15:04"))
	p.textln("Customer: " + o.GetCustomer())
	p.alignCenter()
	p.textln(sep)
	p.textln(o.GetOrderType())
	p.textln(sep)
	p.alignLeft()
	items := o.Items
	if len(o.ItemsCod) > 0 && ow.OnlyOrder == 0 {
		items = o.ItemsCod
	}
	for _, item := range items {
		p.boldOn()
		p.text(fmt.Sprintf("%d   ", item.Pivot.Qty))
		p.boldOff()
		p.textln(item.Name)
	}
	p.alignCenter()
	p.textln(sep)
	p.alignLeft()
	p.textln("Printed at : " + time.Now().In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006 15:04"))
	p.feed()
	p.feed()
	p.feed()
	p.cut()
}

// ── Text fallback: Cashier ────────────────────────────────────────────────────

func printCashierText(p *EscPrinter, ow OrderWrapper, pi PrinterInfo, paperSize int) {
	o := ow.Order
	sep := "------------------------------------------"
	total := o.GetTotal()
	newTotal := math.Ceil(total/500.0) * 500.0
	roundingValue := newTotal - total
	total = newTotal
	p.init()
	p.fontB()
	p.sizeNormal()
	p.alignCenter()
	p.textln(o.Restorant.Name)
	p.textln(o.Restorant.Address)
	p.textln(o.Restorant.Phone)
	p.alignLeft()
	p.textln(sep)
	p.textln("Receipt No  : " + o.GetReceiptNumber())
	p.textln("Date        : " + time.Now().In(time.FixedZone("WIB", 7*3600)).Format("02/01/2006 15:04:05"))
	t, _ := time.Parse("2006-01-02T15:04:05.000000Z", o.CreatedAt)
	t = t.In(time.FixedZone("WIB", 7*3600))
	p.textln("Time In     : " + t.Format("02 Jan 2006 15:04"))
	p.textln("Cashier     : Cashier")
	p.boldOn()
	p.textln("No Meja     : " + o.GetTableArea() + "/" + o.GetTableName())
	p.boldOff()
	p.textln(sep)
	p.textln("NAME            QTY  PRICE        SUBTOT")
	p.textln(sep)
	items := o.Items
	if len(o.ItemsCod) > 0 {
		items = o.ItemsCod
	}
	for _, item := range items {
		subTot := item.Pivot.VariantPrice * float64(item.Pivot.Qty)
		p.textln(item.Name)
		p.alignRight()
		p.textln(fmt.Sprintf("%d | %s | %s", item.Pivot.Qty, formatNumber(item.Pivot.VariantPrice), formatNumber(subTot)))
		p.alignLeft()
	}
	p.textln("_________________________________________")
	p.textln("           Subtotal  : " + o.Restorant.Currency + " " + formatNumber(o.GetSubtotal()))
	p.textln("          Surcharge  : " + o.Restorant.Currency + " " + formatNumber(o.Surcharge))
	p.textln(fmt.Sprintf("             PBI %v%%  : %s %s", o.Restorant.InclusiveTax, o.Restorant.Currency, formatNumber(o.Vatvalue)))
	p.textln(sep)
	p.textln("              TOTAL  : " + o.Restorant.Currency + " " + formatNumber(total))
	p.textln("           Rounding  : " + o.Restorant.Currency + " " + formatNumber(roundingValue))
	p.boldOn()
	p.textln(sep)
	p.textln("        GRAND TOTAL  : " + o.Restorant.Currency + " " + formatNumber(total))
	p.boldOff()
	p.textln(sep)
	p.alignCenter()
	p.textln("Thank you for ordering.")
	p.alignLeft()
	p.textln(sep)
	p.textln("Payment Method : " + o.PaymentMethod)
	p.feed()
	p.feed()
	p.feed()
	p.cut()
}

func formatNumber(f float64) string { return fmt.Sprintf("%.2f", f) }

// ── Test Print ────────────────────────────────────────────────────────────────

func sendTestPrint(id int) error {
	printers, err := loadPrinters()
	if err != nil {
		return err
	}
	var cfg *PrinterConfig
	for i, p := range printers {
		if p.ID == id {
			cfg = &printers[i]
			break
		}
	}
	if cfg == nil {
		return fmt.Errorf("printer id %d not found", id)
	}
	p, err := newPrinterFromConfig(*cfg)
	if err != nil {
		return err
	}
	defer p.close()
	sep := "================================"
	p.init()
	p.alignCenter()
	p.boldOn()
	p.sizeDouble()
	p.textln("TEST PRINT")
	p.sizeNormal()
	p.boldOff()
	p.textln(sep)
	p.textln("Munchii Printer")
	p.textln("Name    : " + cfg.PrinterName)
	p.textln("Type    : " + cfg.GetConnType())
	p.textln("Connect : " + cfg.ConnSummary())
	p.textln(fmt.Sprintf("Polling : %ds", cfg.GetPollingSeconds()))
	p.textln(sep)
	p.alignLeft()
	p.textln("Connection: OK")
	p.textln("ESC/POS:   OK")
	p.textln("Printer configured correctly.")
	p.textln(sep)
	p.alignCenter()
	p.textln(time.Now().Format("02/01/2006 15:04:05"))
	p.feed()
	p.feed()
	p.feed()
	p.cut()
	return p.flush()
}
