package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/getlantern/systray"
)

func main() {
	// CLI mode: called from PowerShell UI for add/edit/delete/get/testprint
	if len(os.Args) > 1 && handleCLI(os.Args) {
		return
	}

	// Normal mode: run as tray app (no console window when built with -H windowsgui)
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIconBytes())
	systray.SetTitle("Munchii Printer")
	systray.SetTooltip("Munchii Printer — starting...")

	// ── Menu ──────────────────────────────────────────────────────────────────
	mStatus := systray.AddMenuItem("● Starting...", "Polling status")
	mStatus.Disable()

	systray.AddSeparator()

	mManage  := systray.AddMenuItem("⚙   Manage Printers", "Add, edit or remove printers")
	mViewLog := systray.AddMenuItem("📋  View Log", "View recent activity")

	systray.AddSeparator()

	mAutoStart := systray.AddMenuItemCheckbox(
		"🔄  Auto-start with Windows",
		"Launch automatically on Windows login",
		isAutoStartEnabled(),
	)

	systray.AddSeparator()
	mExit := systray.AddMenuItem("✖   Exit", "Stop polling and quit")

	// ── Boot ──────────────────────────────────────────────────────────────────
	printers, err := loadPrinters()
	if err != nil {
		logStatus("Error loading printers: " + err.Error())
	} else if len(printers) == 0 {
		logStatus("No printers configured. Use Manage Printers to add one.")
		systray.SetTooltip("Munchii Printer — no printers configured")
	}

	manager.StartAll()
	updateStatus(mStatus)

	// ── Event loop ────────────────────────────────────────────────────────────
	go func() {
		for {
			select {
			case <-mManage.ClickedCh:
				openManageWindow()
				go func() {
					manager.Restart()
					updateStatus(mStatus)
				}()

			case <-mViewLog.ClickedCh:
				openLogWindow()

			case <-mAutoStart.ClickedCh:
				if mAutoStart.Checked() {
					mAutoStart.Uncheck()
					unregisterAutoStart()
					logStatus("Auto-start disabled")
				} else {
					mAutoStart.Check()
					registerAutoStart()
					logStatus("Auto-start enabled")
				}

			case <-mExit.ClickedCh:
				manager.StopAll()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logStatus("Munchii Printer exiting")
}

func updateStatus(mStatus *systray.MenuItem) {
	printers, err := loadPrinters()
	if err != nil || len(printers) == 0 {
		mStatus.SetTitle("● No printers configured")
		systray.SetTooltip("Munchii Printer — no printers")
		return
	}

	statuses := manager.GetStatuses()
	lines := []string{}
	allOK := true

	for _, p := range printers {
		st := statuses[p.ID]
		icon := "✅"
		if strings.Contains(st, "error") {
			icon = "❌"
			allOK = false
		}
		lines = append(lines, fmt.Sprintf("%s %s", icon, p.PrinterName))
	}

	title := "● " + strings.Join(lines, "  |  ")
	if len(title) > 64 {
		if allOK {
			title = fmt.Sprintf("● %d printer(s) running", len(printers))
		} else {
			title = fmt.Sprintf("● %d printer(s) — check errors", len(printers))
		}
	}
	mStatus.SetTitle(title)

	if allOK {
		systray.SetTooltip(fmt.Sprintf("Munchii Printer — %d printer(s) OK", len(printers)))
	} else {
		systray.SetTooltip("Munchii Printer — errors detected!")
	}
}

func openLogWindow() {
	logs := getRecentLogs(60)
	content := strings.Join(logs, "\\n")
	if content == "" {
		content = "(No logs yet)"
	}

	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$form = New-Object System.Windows.Forms.Form
$form.Text = "Munchii Printer - Activity Log"
$form.Size = New-Object System.Drawing.Size(760, 540)
$form.StartPosition = "CenterScreen"
$form.BackColor = [System.Drawing.Color]::FromArgb(240, 242, 245)

$pnlTitle = New-Object System.Windows.Forms.Panel
$pnlTitle.Dock = "Top"
$pnlTitle.Height = 48
$pnlTitle.BackColor = [System.Drawing.Color]::FromArgb(0, 120, 212)
$lbl = New-Object System.Windows.Forms.Label
$lbl.Text = "  Activity Log"
$lbl.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 13)
$lbl.ForeColor = [System.Drawing.Color]::White
$lbl.Location = New-Object System.Drawing.Point(10, 10)
$lbl.Size = New-Object System.Drawing.Size(400, 30)
$pnlTitle.Controls.Add($lbl)
$form.Controls.Add($pnlTitle)

$txt = New-Object System.Windows.Forms.TextBox
$txt.Multiline = $true
$txt.ScrollBars = "Vertical"
$txt.ReadOnly = $true
$txt.Location = New-Object System.Drawing.Point(10, 58)
$txt.Size = New-Object System.Drawing.Size(724, 430)
$txt.Font = New-Object System.Drawing.Font("Consolas", 9)
$txt.BackColor = [System.Drawing.Color]::FromArgb(30, 30, 30)
$txt.ForeColor = [System.Drawing.Color]::FromArgb(180, 255, 180)
$txt.BorderStyle = "None"
$txt.Text = "%s"
$form.Controls.Add($txt)
$form.ShowDialog() | Out-Null
`, content)

	runPS(ps)
}

func runPS(script string) {
	cmd := newPSCommand(script)
	cmd.Start()
}