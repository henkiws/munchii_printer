package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/getlantern/systray"
)

func main() {
	// CLI mode: dipanggil dari PowerShell UI
	if len(os.Args) > 1 && handleCLI(os.Args) {
		return
	}

	// Normal mode: tray app
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIconBytes())
	systray.SetTitle("Kypesen Printer")
	systray.SetTooltip("Kypesen Printer — starting...")

	// ── Menu ──────────────────────────────────────────────────────────────────
	mStatus := systray.AddMenuItem("● Starting...", "Status koneksi")
	mStatus.Disable()

	systray.AddSeparator()

	mManage  := systray.AddMenuItem("⚙   Manage Printers", "Tambah, edit, atau hapus printer")
	mViewLog := systray.AddMenuItem("📋  View Log", "Lihat aktivitas terakhir")

	systray.AddSeparator()

	mAutoStart := systray.AddMenuItemCheckbox(
		"🔄  Auto-start dengan Windows",
		"Jalankan otomatis saat login Windows",
		isAutoStartEnabled(),
	)

	systray.AddSeparator()
	mExit := systray.AddMenuItem("✖   Exit", "Hentikan dan keluar")

	// ── Startup ───────────────────────────────────────────────────────────────
	StartLogCleanup() // auto-cleanup log > 1 bulan

	printers, err := loadPrinters()
	if err != nil {
		logStatus("ERROR Gagal load printers: " + err.Error())
	} else if len(printers) == 0 {
		logStatus("WARN Belum ada printer dikonfigurasi. Gunakan Manage Printers.")
		systray.SetTooltip("Kypesen Printer — belum ada printer")
	}

	wsManager.StartAll()
	updateStatus(mStatus)

	// ── Event loop ────────────────────────────────────────────────────────────
	go func() {
		for {
			select {
			case <-mManage.ClickedCh:
				openManageWindow()
				go func() {
					wsManager.Restart()
					updateStatus(mStatus)
				}()

			case <-mViewLog.ClickedCh:
				openLogWindow()

			case <-mAutoStart.ClickedCh:
				if mAutoStart.Checked() {
					mAutoStart.Uncheck()
					unregisterAutoStart()
					logStatus("INFO Auto-start dinonaktifkan")
				} else {
					mAutoStart.Check()
					registerAutoStart()
					logStatus("INFO Auto-start diaktifkan")
				}

			case <-mExit.ClickedCh:
				wsManager.StopAll()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logStatus("INFO Kypesen Printer keluar")
}

func updateStatus(mStatus *systray.MenuItem) {
	printers, err := loadPrinters()
	if err != nil || len(printers) == 0 {
		mStatus.SetTitle("● Belum ada printer")
		systray.SetTooltip("Kypesen Printer — belum ada printer")
		return
	}

	statuses := wsManager.GetStatuses()
	lines := []string{}
	allOK := true

	for _, p := range printers {
		st := statuses[p.ID]
		icon := "✅"
		if strings.Contains(st, "error") || strings.Contains(st, "disconnected") {
			icon = "❌"
			allOK = false
		} else if strings.Contains(st, "connecting") {
			icon = "🔄"
		}
		lines = append(lines, fmt.Sprintf("%s %s", icon, p.PrinterName))
	}

	title := "● " + strings.Join(lines, "  |  ")
	if len(title) > 64 {
		if allOK {
			title = fmt.Sprintf("● %d printer(s) connected", len(printers))
		} else {
			title = fmt.Sprintf("● %d printer(s) — cek koneksi", len(printers))
		}
	}
	mStatus.SetTitle(title)

	if allOK {
		systray.SetTooltip(fmt.Sprintf("Kypesen Printer — %d printer(s) OK", len(printers)))
	} else {
		systray.SetTooltip("Kypesen Printer — ada printer terputus!")
	}
}

func openLogWindow() {
	logs := getRecentLogs(100)
	content := strings.Join(logs, "\\n")
	if content == "" {
		content = "(Belum ada log)"
	}

	// Color-code log levels untuk tampilan
	// OK → hijau, ERROR → merah, WARN → kuning, INFO → abu
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$form = New-Object System.Windows.Forms.Form
$form.Text = "Kypesen Printer - Activity Log"
$form.Size = New-Object System.Drawing.Size(900, 600)
$form.StartPosition = "CenterScreen"
$form.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 30)

$pnlTitle = New-Object System.Windows.Forms.Panel
$pnlTitle.Dock = "Top"
$pnlTitle.Height = 48
$pnlTitle.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
$lbl = New-Object System.Windows.Forms.Label
$lbl.Text = "  Activity Log  (100 baris terakhir — log disimpan 1 bulan)"
$lbl.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 10)
$lbl.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
$lbl.Location = New-Object System.Drawing.Point(10, 12)
$lbl.Size = New-Object System.Drawing.Size(860, 26)
$pnlTitle.Controls.Add($lbl)
$form.Controls.Add($pnlTitle)

$rtb = New-Object System.Windows.Forms.RichTextBox
$rtb.Multiline = $true
$rtb.ScrollBars = "Vertical"
$rtb.ReadOnly = $true
$rtb.Location = New-Object System.Drawing.Point(10, 58)
$rtb.Size = New-Object System.Drawing.Size(864, 490)
$rtb.Font = New-Object System.Drawing.Font("Consolas", 9)
$rtb.BackColor = [System.Drawing.Color]::FromArgb(18, 18, 28)
$rtb.BorderStyle = "None"
$form.Controls.Add($rtb)

$lines = "%s" -split "\\n"
foreach ($line in $lines) {
    if ($line -match "^\[.*?\] OK") {
        $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(80, 220, 120)
    } elseif ($line -match "^\[.*?\] ERROR") {
        $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(255, 90, 80)
    } elseif ($line -match "^\[.*?\] WARN") {
        $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(255, 200, 60)
    } elseif ($line -match "^\[.*?\] INFO") {
        $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(140, 180, 255)
    } else {
        $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(180, 180, 200)
    }
    $rtb.AppendText($line + [char]10)
}
$rtb.ScrollToCaret()
$form.ShowDialog() | Out-Null
`, content)

	runPS(ps)
}

func runPS(script string) {
	cmd := newPSCommand(script)
	cmd.Start()
}