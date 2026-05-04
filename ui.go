package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func openManageWindow() {
	if runtime.GOOS != "windows" {
		logStatus("Manage window only supported on Windows")
		return
	}

	printers, err := loadPrinters()
	if err != nil {
		showErrorDialog("Error loading printers: " + err.Error())
		return
	}

	// Build ListView rows
	listItems := ""
	for _, p := range printers {
		safeName := strings.ReplaceAll(p.PrinterName, "'", "''")
		safeConn := strings.ReplaceAll(p.ConnSummary(), "'", "''")
		safeURL := strings.ReplaceAll(p.ServerURL, "'", "''")
		listItems += fmt.Sprintf(`
$row = New-Object System.Windows.Forms.ListViewItem('%d')
$row.SubItems.Add('%s') | Out-Null
$row.SubItems.Add('%s') | Out-Null
$row.SubItems.Add('%d') | Out-Null
$row.SubItems.Add('%s') | Out-Null
$row.Tag = %d
$lvPrinters.Items.Add($row) | Out-Null
`, p.ID, safeName, safeConn, p.GetPollingSeconds(), safeURL, p.ID)
	}

	exePath := getExePath()

	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# ═══════════════════════════════════════════════════
# FORM
# ═══════════════════════════════════════════════════
$form = New-Object System.Windows.Forms.Form
$form.Text = "Munchii Printer Manager"
$form.Size = New-Object System.Drawing.Size(940, 780)
$form.MinimumSize = New-Object System.Drawing.Size(880, 700)
$form.StartPosition = "CenterScreen"
$form.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)
$form.Font = New-Object System.Drawing.Font("Segoe UI", 9)

# ── Title bar ─────────────────────────────────────────
$pnlTitle = New-Object System.Windows.Forms.Panel
$pnlTitle.Dock = "Top"
$pnlTitle.Height = 64
$pnlTitle.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
$form.Controls.Add($pnlTitle)

$lblBrand = New-Object System.Windows.Forms.Label
$lblBrand.Text = "MUNCHII"
$lblBrand.Font = New-Object System.Drawing.Font("Segoe UI Black", 16, [System.Drawing.FontStyle]::Bold)
$lblBrand.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
$lblBrand.Location = New-Object System.Drawing.Point(16, 8)
$lblBrand.Size = New-Object System.Drawing.Size(180, 30)
$pnlTitle.Controls.Add($lblBrand)

$lblSub = New-Object System.Windows.Forms.Label
$lblSub.Text = "Printer Manager"
$lblSub.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$lblSub.ForeColor = [System.Drawing.Color]::FromArgb(180, 180, 200)
$lblSub.Location = New-Object System.Drawing.Point(16, 36)
$lblSub.Size = New-Object System.Drawing.Size(300, 20)
$pnlTitle.Controls.Add($lblSub)

# ── Section strip ──────────────────────────────────────
$pnlSec = New-Object System.Windows.Forms.Panel
$pnlSec.Location = New-Object System.Drawing.Point(0, 64)
$pnlSec.Size = New-Object System.Drawing.Size(940, 34)
$pnlSec.BackColor = [System.Drawing.Color]::FromArgb(232, 234, 242)
$form.Controls.Add($pnlSec)

$lblHdr = New-Object System.Windows.Forms.Label
$lblHdr.Text = "   CONFIGURED PRINTERS"
$lblHdr.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 8)
$lblHdr.ForeColor = [System.Drawing.Color]::FromArgb(90, 90, 110)
$lblHdr.Location = New-Object System.Drawing.Point(0, 0)
$lblHdr.Size = New-Object System.Drawing.Size(400, 34)
$lblHdr.TextAlign = [System.Drawing.ContentAlignment]::MiddleLeft
$pnlSec.Controls.Add($lblHdr)

# ── ListView ───────────────────────────────────────────
$lvPrinters = New-Object System.Windows.Forms.ListView
$lvPrinters.Location = New-Object System.Drawing.Point(14, 106)
$lvPrinters.Size = New-Object System.Drawing.Size(900, 160)
$lvPrinters.View = [System.Windows.Forms.View]::Details
$lvPrinters.FullRowSelect = $true
$lvPrinters.GridLines = $true
$lvPrinters.MultiSelect = $false
$lvPrinters.BorderStyle = "FixedSingle"
$lvPrinters.BackColor = [System.Drawing.Color]::White
$lvPrinters.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lvPrinters.HeaderStyle = [System.Windows.Forms.ColumnHeaderStyle]::Nonclickable
$lvPrinters.Columns.Add("ID",         40)  | Out-Null
$lvPrinters.Columns.Add("Name",      170)  | Out-Null
$lvPrinters.Columns.Add("Connection",160)  | Out-Null
$lvPrinters.Columns.Add("Poll (s)",   64)  | Out-Null
$lvPrinters.Columns.Add("API URL",   450)  | Out-Null
$form.Controls.Add($lvPrinters)

# Populate
%s

# ── Colors ─────────────────────────────────────────────
$cBlue   = [System.Drawing.Color]::FromArgb(0,   120, 212)
$cOrange = [System.Drawing.Color]::FromArgb(210,  95,   0)
$cRed    = [System.Drawing.Color]::FromArgb(190,  40,  30)
$cGreen  = [System.Drawing.Color]::FromArgb( 16, 130,  16)
$cWhite  = [System.Drawing.Color]::White

function MakeBtn($text, $x, $y, $w, $h, $bg, $fg) {
    $b = New-Object System.Windows.Forms.Button
    $b.Text = $text; $b.Location = New-Object System.Drawing.Point($x,$y)
    $b.Size = New-Object System.Drawing.Size($w,$h)
    $b.BackColor = $bg; $b.ForeColor = $fg
    $b.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
    $b.FlatAppearance.BorderSize = 0
    $b.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
    $b.Cursor = [System.Windows.Forms.Cursors]::Hand
    return $b
}

$btnTest   = MakeBtn "  Test Print"  14  274  148 34 $cBlue   $cWhite
$btnEdit   = MakeBtn "  Edit"       170  274  110 34 $cOrange $cWhite
$btnDelete = MakeBtn "  Delete"     288  274  110 34 $cRed    $cWhite
$form.Controls.Add($btnTest)
$form.Controls.Add($btnEdit)
$form.Controls.Add($btnDelete)

# ── Status label ───────────────────────────────────────
$lblStatus = New-Object System.Windows.Forms.Label
$lblStatus.Location = New-Object System.Drawing.Point(14, 314)
$lblStatus.Size = New-Object System.Drawing.Size(900, 24)
$lblStatus.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblStatus.ForeColor = $cGreen
$form.Controls.Add($lblStatus)

$sep = New-Object System.Windows.Forms.Label
$sep.BorderStyle = "Fixed3D"
$sep.Location = New-Object System.Drawing.Point(14, 342)
$sep.Size = New-Object System.Drawing.Size(900, 2)
$form.Controls.Add($sep)

# ═══════════════════════════════════════════════════
# ADD/EDIT PANEL
# ═══════════════════════════════════════════════════
$grp = New-Object System.Windows.Forms.GroupBox
$grp.Text = "  Add New Printer"
$grp.Location = New-Object System.Drawing.Point(14, 350)
$grp.Size = New-Object System.Drawing.Size(900, 370)
$grp.BackColor = [System.Drawing.Color]::White
$grp.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
$form.Controls.Add($grp)

$Script:editID = -1

function MkLbl($t,$x,$y) {
    $l = New-Object System.Windows.Forms.Label
    $l.Text=$t; $l.Location=New-Object System.Drawing.Point($x,$y)
    $l.Size=New-Object System.Drawing.Size(148,26)
    $l.Font=New-Object System.Drawing.Font("Segoe UI",9)
    $l.ForeColor=[System.Drawing.Color]::FromArgb(70,70,90)
    $l.TextAlign=[System.Drawing.ContentAlignment]::MiddleRight
    return $l
}
function MkTxt($x,$y,$w) {
    $t = New-Object System.Windows.Forms.TextBox
    $t.Location=New-Object System.Drawing.Point($x,$y)
    $t.Size=New-Object System.Drawing.Size($w,26)
    $t.Font=New-Object System.Drawing.Font("Segoe UI",9)
    $t.BorderStyle="FixedSingle"
    $t.BackColor=[System.Drawing.Color]::FromArgb(250,250,252)
    return $t
}
function MkCombo($x,$y,$w) {
    $c = New-Object System.Windows.Forms.ComboBox
    $c.Location=New-Object System.Drawing.Point($x,$y)
    $c.Size=New-Object System.Drawing.Size($w,26)
    $c.Font=New-Object System.Drawing.Font("Segoe UI",9)
    $c.DropDownStyle=[System.Windows.Forms.ComboBoxStyle]::DropDownList
    $c.FlatStyle=[System.Windows.Forms.FlatStyle]::Flat
    return $c
}

$lx = 14; $tx = 170

# Row 1: Printer Name
$grp.Controls.Add((MkLbl "Printer Name :" $lx 32))
$txtName = MkTxt $tx 32 700
$grp.Controls.Add($txtName)

# Row 2: Connection Type
$grp.Controls.Add((MkLbl "Connection Type :" $lx 68))
$cboType = MkCombo $tx 68 200
$cboType.Items.Add("network")   | Out-Null
$cboType.Items.Add("bluetooth") | Out-Null
$cboType.Items.Add("usb")       | Out-Null
$cboType.SelectedIndex = 0
$grp.Controls.Add($cboType)

$btnHelp = New-Object System.Windows.Forms.Button
$btnHelp.Text = " ? "
$btnHelp.Location = New-Object System.Drawing.Point(($tx + 208), 68)
$btnHelp.Size = New-Object System.Drawing.Size(32, 26)
$btnHelp.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnHelp.FlatAppearance.BorderColor = [System.Drawing.Color]::FromArgb(0, 120, 212)
$btnHelp.FlatAppearance.BorderSize = 1
$btnHelp.BackColor = [System.Drawing.Color]::FromArgb(240, 248, 255)
$btnHelp.ForeColor = [System.Drawing.Color]::FromArgb(0, 120, 212)
$btnHelp.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
$btnHelp.Cursor = [System.Windows.Forms.Cursors]::Hand
$grp.Controls.Add($btnHelp)

$btnHelp.Add_Click({
    $t = $cboType.SelectedItem
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -AssemblyName System.Drawing

    $hw = New-Object System.Windows.Forms.Form
    $hw.Text = "Panduan Koneksi Printer"
    $hw.Size = New-Object System.Drawing.Size(560, 520)
    $hw.StartPosition = "CenterParent"
    $hw.FormBorderStyle = "FixedDialog"
    $hw.MaximizeBox = $false
    $hw.MinimizeBox = $false
    $hw.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)
    $hw.Font = New-Object System.Drawing.Font("Segoe UI", 9)

    # Title bar
    $ht = New-Object System.Windows.Forms.Panel
    $ht.Dock = "Top"; $ht.Height = 52
    $ht.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
    $hl = New-Object System.Windows.Forms.Label
    $hl.Text = "  Panduan Koneksi Printer"
    $hl.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 12)
    $hl.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
    $hl.Location = New-Object System.Drawing.Point(8, 12)
    $hl.Size = New-Object System.Drawing.Size(500, 28)
    $ht.Controls.Add($hl)
    $hw.Controls.Add($ht)

    # Tab control
    $tabs = New-Object System.Windows.Forms.TabControl
    $tabs.Location = New-Object System.Drawing.Point(10, 62)
    $tabs.Size = New-Object System.Drawing.Size(524, 390)
    $tabs.Font = New-Object System.Drawing.Font("Segoe UI", 9)
    $hw.Controls.Add($tabs)

    function MakeTab($title, $lines) {
        $tab = New-Object System.Windows.Forms.TabPage
        $tab.Text = $title
        $tab.BackColor = [System.Drawing.Color]::White
        $tab.Padding = New-Object System.Windows.Forms.Padding(6)

        $rtb = New-Object System.Windows.Forms.RichTextBox
        $rtb.Dock = "Fill"
        $rtb.ReadOnly = $true
        $rtb.BorderStyle = "None"
        $rtb.BackColor = [System.Drawing.Color]::White
        $rtb.Font = New-Object System.Drawing.Font("Segoe UI", 9)
        $rtb.ScrollBars = "Vertical"

        foreach ($line in $lines) {
            if ($line.StartsWith("##")) {
                $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 10)
                $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(0, 90, 180)
                $rtb.AppendText($line.Substring(2).Trim() + [char]10)
            } elseif ($line.StartsWith("#")) {
                $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI Black", 11)
                $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
                $rtb.AppendText($line.Substring(1).Trim() + [char]10)
            } elseif ($line.StartsWith("!")) {
                $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
                $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(180, 60, 0)
                $rtb.AppendText("  " + $line.Substring(1).Trim() + [char]10)
            } elseif ($line.StartsWith(">")) {
                $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI", 9)
                $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(16, 130, 16)
                $rtb.AppendText("  " + $line.Substring(1).Trim() + [char]10)
            } elseif ($line -eq "") {
                $rtb.AppendText([char]10)
            } else {
                $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI", 9)
                $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(40, 40, 60)
                $rtb.AppendText("  " + $line + [char]10)
            }
        }
        $tab.Controls.Add($rtb)
        return $tab
    }

    $netLines = @(
        "# Network / IP (LAN)"
        ""
        "## Apa itu?"
        "Printer terhubung ke jaringan WiFi atau kabel LAN,"
        "dan bisa diakses melalui alamat IP."
        ""
        "## Persyaratan"
        "> Printer harus mendukung koneksi jaringan (LAN/WiFi)"
        "> Printer dan PC harus berada di jaringan yang sama"
        "> Port TCP 9100 tidak diblokir firewall"
        ""
        "## Cara Mengisi"
        "IP Address  : Alamat IP printer di jaringan"
        "              Contoh: 192.168.1.100"
        "Port        : Biasanya 9100 (default ESC/POS)"
        ""
        "## Cara Cek IP Printer"
        "Kebanyakan printer bisa cetak halaman konfigurasi"
        "dengan menahan tombol FEED saat menyalakan printer."
        "IP address akan tercetak di kertas."
        ""
        "! Pastikan printer bisa di-ping dari PC:"
        "! Buka CMD, ketik: ping 192.168.1.100"
    )

    $btLines = @(
        "# Bluetooth"
        ""
        "## Apa itu?"
        "Printer terhubung via Bluetooth dan muncul sebagai"
        "port COM virtual di Windows."
        ""
        "## Persyaratan"
        "> PC harus punya Bluetooth"
        "> Printer sudah di-pair dengan PC lewat Settings Windows"
        "> Setelah pair, Windows membuat port COM virtual (misal COM3)"
        ""
        "## Cara Pair Printer Bluetooth"
        "1. Nyalakan printer Bluetooth"
        "2. Buka Settings Windows > Bluetooth & devices"
        "3. Klik [Add device] dan pilih printer"
        "4. Setelah terhubung, buka Device Manager"
        "5. Lihat di Ports (COM & LPT)"
        "6. Catat nomor COM yang muncul (misal COM3)"
        ""
        "## Cara Mengisi"
        "COM Port    : Pilih dari dropdown (klik Refresh)"
        "              Contoh: COM3, COM5"
        "Baud Rate   : Sesuaikan dengan printer"
        "              Biasanya 9600 atau 115200"
        ""
        "! Jika COM Port tidak muncul, pastikan printer"
        "! sudah di-pair dan driver terinstall."
    )

    $usbLines = @(
        "# USB"
        ""
        "## Apa itu?"
        "Printer terhubung langsung via kabel USB ke PC."
        "Ada dua cara koneksi USB yang didukung:"
        ""
        "## Cara 1: Windows Printer (Direkomendasikan)"
        "Install driver printer dari CD atau website produsen,"
        "lalu pilih nama printer dari dropdown."
        ""
        "> Lebih stabil dan mudah digunakan"
        "> Driver otomatis menangani komunikasi"
        ""
        "Cara mengisi:"
        "- Pilih nama printer dari dropdown [Windows Printer]"
        "- Klik Refresh jika printer tidak muncul"
        "- Pastikan status printer Online di Windows"
        ""
        "## Cara 2: USB via COM Port"
        "Beberapa printer USB muncul sebagai port COM virtual."
        "Biasanya terjadi jika printer menggunakan chip serial USB."
        ""
        "> Cocok untuk printer yang tidak punya driver Windows"
        ""
        "Cara mengisi:"
        "- Pilih COM Port dari dropdown (klik Refresh)"
        "- Atur Baud Rate sesuai spesifikasi printer"
        ""
        "## Cara Cek di Device Manager"
        "1. Klik kanan [This PC] > Properties > Device Manager"
        "2. Lihat di [Ports (COM & LPT)]"
        "3. Catat nomor COM yang muncul saat USB dicolok"
        ""
        "! Jika printer tidak terdeteksi, coba install"
        "! driver dari website produsen printer."
    )

    $tabs.TabPages.Add((MakeTab "  Network/IP  " $netLines))
    $tabs.TabPages.Add((MakeTab "  Bluetooth  " $btLines))
    $tabs.TabPages.Add((MakeTab "  USB  " $usbLines))

    # Select tab matching current selection
    if ($t -eq "bluetooth") { $tabs.SelectedIndex = 1 }
    elseif ($t -eq "usb")   { $tabs.SelectedIndex = 2 }
    else                     { $tabs.SelectedIndex = 0 }

    # Close button
    $hClose = New-Object System.Windows.Forms.Button
    $hClose.Text = "Tutup"
    $hClose.Location = New-Object System.Drawing.Point(430, 460)
    $hClose.Size = New-Object System.Drawing.Size(90, 32)
    $hClose.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
    $hClose.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
    $hClose.ForeColor = [System.Drawing.Color]::White
    $hClose.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
    $hClose.Add_Click({ $hw.Close() })
    $hw.Controls.Add($hClose)

    $hw.ShowDialog() | Out-Null
})

# ── Network fields ────────────────────────────────────
$pnlNet = New-Object System.Windows.Forms.Panel
$pnlNet.Location = New-Object System.Drawing.Point(0, 100)
$pnlNet.Size = New-Object System.Drawing.Size(880, 70)
$pnlNet.BackColor = [System.Drawing.Color]::White
$grp.Controls.Add($pnlNet)

$pnlNet.Controls.Add((MkLbl "IP Address :" $lx 4))
$txtIP = MkTxt $tx 4 200
$txtIP.Text = "192.168.1.100"
$pnlNet.Controls.Add($txtIP)

$pnlNet.Controls.Add((MkLbl "Port :" $lx 36))
$txtPort = MkTxt $tx 36 80
$txtPort.Text = "9100"
$pnlNet.Controls.Add($txtPort)

$lblPortNote = New-Object System.Windows.Forms.Label
$lblPortNote.Text = "(default 9100)"
$lblPortNote.Location = New-Object System.Drawing.Point(258, 40)
$lblPortNote.Size = New-Object System.Drawing.Size(120, 18)
$lblPortNote.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$lblPortNote.ForeColor = [System.Drawing.Color]::Gray
$pnlNet.Controls.Add($lblPortNote)

# ── COM port fields (Bluetooth / USB-Serial) ──────────
$pnlCOM = New-Object System.Windows.Forms.Panel
$pnlCOM.Location = New-Object System.Drawing.Point(0, 100)
$pnlCOM.Size = New-Object System.Drawing.Size(880, 70)
$pnlCOM.BackColor = [System.Drawing.Color]::White
$pnlCOM.Visible = $false
$grp.Controls.Add($pnlCOM)

$pnlCOM.Controls.Add((MkLbl "COM Port :" $lx 4))
$cboCOM = MkCombo $tx 4 120
$pnlCOM.Controls.Add($cboCOM)

$btnRefreshCOM = New-Object System.Windows.Forms.Button
$btnRefreshCOM.Text = "Refresh"
$btnRefreshCOM.Location = New-Object System.Drawing.Point(298, 4)
$btnRefreshCOM.Size = New-Object System.Drawing.Size(70, 26)
$btnRefreshCOM.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnRefreshCOM.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$pnlCOM.Controls.Add($btnRefreshCOM)

$pnlCOM.Controls.Add((MkLbl "Baud Rate :" $lx 36))
$cboBaud = MkCombo $tx 36 120
@("9600","19200","38400","57600","115200") | ForEach-Object { $cboBaud.Items.Add($_) | Out-Null }
$cboBaud.SelectedIndex = 0
$pnlCOM.Controls.Add($cboBaud)

# ── USB Windows Printer fields ────────────────────────
$pnlUSBWin = New-Object System.Windows.Forms.Panel
$pnlUSBWin.Location = New-Object System.Drawing.Point(0, 174)
$pnlUSBWin.Size = New-Object System.Drawing.Size(880, 36)
$pnlUSBWin.BackColor = [System.Drawing.Color]::White
$pnlUSBWin.Visible = $false
$grp.Controls.Add($pnlUSBWin)

$pnlUSBWin.Controls.Add((MkLbl "Windows Printer :" $lx 4))
$cboWinPrinter = MkCombo $tx 4 300
$pnlUSBWin.Controls.Add($cboWinPrinter)

$btnRefreshWin = New-Object System.Windows.Forms.Button
$btnRefreshWin.Text = "Refresh"
$btnRefreshWin.Location = New-Object System.Drawing.Point(478, 4)
$btnRefreshWin.Size = New-Object System.Drawing.Size(70, 26)
$btnRefreshWin.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnRefreshWin.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$pnlUSBWin.Controls.Add($btnRefreshWin)

$lblOrCOM = New-Object System.Windows.Forms.Label
$lblOrCOM.Text = "or use COM port above"
$lblOrCOM.Location = New-Object System.Drawing.Point(556, 8)
$lblOrCOM.Size = New-Object System.Drawing.Size(200, 18)
$lblOrCOM.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$lblOrCOM.ForeColor = [System.Drawing.Color]::Gray
$pnlUSBWin.Controls.Add($lblOrCOM)

# ── API URL + Poll ─────────────────────────────────────
$grp.Controls.Add((MkLbl "Kypesen API URL :" $lx 218))
$txtURL = MkTxt $tx 218 700
$txtURL.Text = "http://localhost:8001/api/v1/print/UUID"
$grp.Controls.Add($txtURL)

$grp.Controls.Add((MkLbl "Poll Interval :" $lx 256))
$txtPoll = MkTxt $tx 256 80
$txtPoll.Text = "5"
$grp.Controls.Add($txtPoll)

$lblPollNote = New-Object System.Windows.Forms.Label
$lblPollNote.Text = "seconds (default: 5)"
$lblPollNote.Location = New-Object System.Drawing.Point(258, 260)
$lblPollNote.Size = New-Object System.Drawing.Size(200, 18)
$lblPollNote.Font = New-Object System.Drawing.Font("Segoe UI", 8)
$lblPollNote.ForeColor = [System.Drawing.Color]::Gray
$grp.Controls.Add($lblPollNote)

# ── Form status + buttons ──────────────────────────────
$lblFrmStatus = New-Object System.Windows.Forms.Label
$lblFrmStatus.Location = New-Object System.Drawing.Point($tx, 294)
$lblFrmStatus.Size = New-Object System.Drawing.Size(700, 22)
$lblFrmStatus.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblFrmStatus.ForeColor = $cGreen
$grp.Controls.Add($lblFrmStatus)

$btnSave  = MakeBtn "  Add Printer" $tx 320 160 36 $cGreen  $cWhite
$btnClear = New-Object System.Windows.Forms.Button
$btnClear.Text = "Clear"
$btnClear.Location = New-Object System.Drawing.Point(($tx+168), 320)
$btnClear.Size = New-Object System.Drawing.Size(80, 36)
$btnClear.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnClear.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$btnClear.ForeColor = [System.Drawing.Color]::FromArgb(60,60,60)
$grp.Controls.Add($btnSave)
$grp.Controls.Add($btnClear)

# ═══════════════════════════════════════════════════
# HELPERS
# ═══════════════════════════════════════════════════
function ShowStatus($msg, $isErr) {
    if ($isErr) { $lblStatus.ForeColor = $cRed } else { $lblStatus.ForeColor = $cGreen }
    $lblStatus.Text = "  " + $msg
}

function GetSelectedID() {
    if ($lvPrinters.SelectedItems.Count -eq 0) { return -1 }
    return [int]$lvPrinters.SelectedItems[0].Tag
}

function LoadCOMPorts() {
    $cboCOM.Items.Clear()
    $ports = & '%s' listcom 2>&1
    if ($ports) {
        "$ports".Split([char]10) | ForEach-Object {
            $p = $_.Trim()
            if ($p -ne "") { $cboCOM.Items.Add($p) | Out-Null }
        }
    }
    if ($cboCOM.Items.Count -gt 0) { $cboCOM.SelectedIndex = 0 }
}

function LoadWinPrinters() {
    $cboWinPrinter.Items.Clear()
    $list = & '%s' listprinters 2>&1
    if ($list) {
        "$list".Split([char]10) | ForEach-Object {
            $p = $_.Trim()
            if ($p -ne "") { $cboWinPrinter.Items.Add($p) | Out-Null }
        }
    }
    if ($cboWinPrinter.Items.Count -gt 0) { $cboWinPrinter.SelectedIndex = 0 }
}

function UpdatePanels() {
    $t = $cboType.SelectedItem
    $pnlNet.Visible    = ($t -eq "network")
    $pnlCOM.Visible    = ($t -eq "bluetooth" -or $t -eq "usb")
    $pnlUSBWin.Visible = ($t -eq "usb")
    if ($t -eq "bluetooth" -or $t -eq "usb") { LoadCOMPorts }
    if ($t -eq "usb") { LoadWinPrinters }
}

function ClearForm() {
    $Script:editID = -1
    $txtName.Text = ""
    $cboType.SelectedIndex = 0
    $txtIP.Text = "192.168.1.100"
    $txtPort.Text = "9100"
    $txtURL.Text = "http://localhost:8001/api/v1/print/UUID"
    $txtPoll.Text = "5"
    $lblFrmStatus.Text = ""
    $btnSave.Text = "  Add Printer"
    $btnSave.BackColor = $cGreen
    $grp.Text = "  Add New Printer"
    $lvPrinters.SelectedItems.Clear()
    UpdatePanels
}

function GetConnArgs() {
    $t = $cboType.SelectedItem
    if ($t -eq "network") {
        $port = $txtPort.Text.Trim(); if ($port -eq "") { $port = "9100" }
        return @($t, $txtIP.Text.Trim(), $port, "", "9600", "")
    } elseif ($t -eq "bluetooth") {
        $com = if ($cboCOM.SelectedItem) { $cboCOM.SelectedItem } else { $cboCOM.Text }
        $baud = if ($cboBaud.SelectedItem) { $cboBaud.SelectedItem } else { "9600" }
        return @($t, "", "9100", $com, $baud, "")
    } else {
        $com = if ($cboCOM.SelectedItem) { $cboCOM.SelectedItem } else { $cboCOM.Text }
        $baud = if ($cboBaud.SelectedItem) { $cboBaud.SelectedItem } else { "9600" }
        $winp = if ($cboWinPrinter.SelectedItem) { $cboWinPrinter.SelectedItem } else { "" }
        return @($t, "", "9100", $com, $baud, $winp)
    }
}

# Init panels
UpdatePanels

# ═══════════════════════════════════════════════════
# EVENTS
# ═══════════════════════════════════════════════════
$cboType.Add_SelectedIndexChanged({ UpdatePanels })
$btnRefreshCOM.Add_Click({ LoadCOMPorts })
$btnRefreshWin.Add_Click({ LoadWinPrinters })

$btnTest.Add_Click({
    $id = GetSelectedID
    if ($id -lt 0) { ShowStatus "Select a printer first." $true; return }
    $nm = $lvPrinters.SelectedItems[0].SubItems[1].Text
    $lblStatus.ForeColor = $cBlue
    $lblStatus.Text = "   Sending test print to '$nm'..."
    $form.Refresh()
    $r = & '%s' testprint "$id" 2>&1
    if ($LASTEXITCODE -eq 0 -or "$r" -match "OK") {
        ShowStatus "Test print sent to '$nm' — check the printer!" $false
    } else {
        ShowStatus "Test print failed: $r" $true
    }
})

$btnEdit.Add_Click({
    $id = GetSelectedID
    if ($id -lt 0) { ShowStatus "Select a printer to edit." $true; return }
    $raw = & '%s' get "$id" 2>&1
    # raw = "name|conntype|ip|port|com|baud|winprinter|url|poll"
    $parts = "$raw" -split "\|"
    if ($parts.Count -ge 9) {
        $Script:editID = $id
        $txtName.Text = $parts[0]
        $idx = $cboType.Items.IndexOf($parts[1])
        if ($idx -ge 0) { $cboType.SelectedIndex = $idx }
        UpdatePanels
        $txtIP.Text   = $parts[2]
        $txtPort.Text = $parts[3]
        if ($cboCOM.Items.Count -eq 0) { $cboCOM.Items.Add($parts[4]) | Out-Null }
        $cboCOM.Text  = $parts[4]
        $cboBaud.Text = $parts[5]
        if ($cboWinPrinter.Items.Count -eq 0 -and $parts[6] -ne "") { $cboWinPrinter.Items.Add($parts[6]) | Out-Null }
        $cboWinPrinter.Text = $parts[6]
        $txtURL.Text  = $parts[7]
        $txtPoll.Text = $parts[8]
        $btnSave.Text = "  Save Changes"
        $btnSave.BackColor = $cOrange
        $grp.Text = "  Edit Printer (ID: $id)"
        $lblStatus.ForeColor = $cOrange
        $lblStatus.Text = "  Editing printer ID $id — modify fields and click Save Changes."
    } else {
        ShowStatus "Could not load printer: $raw" $true
    }
})

$btnDelete.Add_Click({
    $id = GetSelectedID
    if ($id -lt 0) { ShowStatus "Select a printer to delete." $true; return }
    $selRow = $lvPrinters.SelectedItems[0]
    $nm = $selRow.SubItems[1].Text
    $nl = [char]10
    $confirm = [System.Windows.Forms.MessageBox]::Show(
        "Delete printer '$nm' (ID $id)?" + $nl + "This cannot be undone.",
        "Confirm Delete",
        [System.Windows.Forms.MessageBoxButtons]::YesNo,
        [System.Windows.Forms.MessageBoxIcon]::Warning)
    if ($confirm -eq "Yes") {
        & '%s' delete "$id" 2>&1 | Out-Null
        $lvPrinters.Items.Remove($selRow)
        ClearForm
        ShowStatus "Printer '$nm' removed." $false
    }
})

$btnSave.Add_Click({
    $name = $txtName.Text.Trim()
    $url  = $txtURL.Text.Trim()
    $poll = $txtPoll.Text.Trim()
    if ($name -eq "" -or $url -eq "") {
        $lblFrmStatus.ForeColor = $cRed
        $lblFrmStatus.Text = "  Printer Name and API URL are required."
        return
    }
    if ($poll -notmatch "^\d+$" -or [int]$poll -lt 1) { $poll = "5" }
    $args = GetConnArgs
    # args: conntype, ip, port, com, baud, winprinter
    if ($Script:editID -ge 0) {
        & '%s' edit "$($Script:editID)" "$name" "$($args[0])" "$($args[1])" "$($args[2])" "$($args[3])" "$($args[4])" "$($args[5])" "$url" "$poll" 2>&1 | Out-Null
        $row = $lvPrinters.SelectedItems[0]
        if ($null -ne $row) {
            $connStr = "$($args[0])://$($args[1])$($args[3])"
            $row.Text              = "$($Script:editID)"
            $row.SubItems[1].Text = $name
            $row.SubItems[2].Text = $connStr
            $row.SubItems[3].Text = $poll
            $row.SubItems[4].Text = $url
        }
        $lblFrmStatus.ForeColor = $cGreen
        $lblFrmStatus.Text = "  Printer updated."
        ClearForm
    } else {
        & '%s' add "$name" "$($args[0])" "$($args[1])" "$($args[2])" "$($args[3])" "$($args[4])" "$($args[5])" "$url" "$poll" 2>&1 | Out-Null
        $newID = $lvPrinters.Items.Count
        $connStr = "$($args[0])://$($args[1])$($args[3])"
        $row = New-Object System.Windows.Forms.ListViewItem("$newID")
        $row.SubItems.Add($name)    | Out-Null
        $row.SubItems.Add($connStr) | Out-Null
        $row.SubItems.Add($poll)    | Out-Null
        $row.SubItems.Add($url)     | Out-Null
        $row.Tag = $newID
        $lvPrinters.Items.Add($row) | Out-Null
        $lblFrmStatus.ForeColor = $cGreen
        $lblFrmStatus.Text = "  Printer '$name' added!"
        ClearForm
    }
})

$btnClear.Add_Click({ ClearForm })
$lvPrinters.Add_DoubleClick({ $btnEdit.PerformClick() })

$form.ShowDialog() | Out-Null
`, listItems, exePath, exePath, exePath, exePath, exePath, exePath, exePath, exePath)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.Start()
}

func showErrorDialog(msg string) {
	safe := strings.ReplaceAll(msg, "'", "''")
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.MessageBox]::Show('%s','Munchii Error',[System.Windows.Forms.MessageBoxButtons]::OK,[System.Windows.Forms.MessageBoxIcon]::Error) | Out-Null
`, safe)
	exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps).Start()
}

// ── CLI dispatch ──────────────────────────────────────────────────────────────

func handleCLI(args []string) bool {
	if len(args) < 2 {
		return false
	}
	switch args[1] {

	case "add":
		// add <name> <conntype> <ip> <port> <com> <baud> <winprinter> <url> <poll>
		if len(args) < 10 {
			fmt.Println("usage: add <name> <conntype> <ip> <port> <com> <baud> <winprinter> <url> <poll>")
			return true
		}
		polling, _ := strconv.Atoi(args[9])
		if polling <= 0 {
			polling = 5
		}
		port, _ := strconv.Atoi(args[4])
		baud, _ := strconv.Atoi(args[6])
		cfg := PrinterConfig{
			PrinterName:      args[2],
			ConnType:         args[3],
			PrinterIPAddress: args[4],
			PrinterPort:      port,
			COMPort:          args[5],
			BaudRate:         baud,
			WindowsPrinter:   args[7],
			ServerURL:        args[8],
			PollingTime:      &polling,
		}
		if err := addPrinter(cfg); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "edit":
		// edit <id> <name> <conntype> <ip> <port> <com> <baud> <winprinter> <url> <poll>
		if len(args) < 11 {
			fmt.Println("usage: edit <id> <name> <conntype> <ip> <port> <com> <baud> <winprinter> <url> <poll>")
			return true
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid id")
			return true
		}
		polling, _ := strconv.Atoi(args[10])
		if polling <= 0 {
			polling = 5
		}
		port, _ := strconv.Atoi(args[5])
		baud, _ := strconv.Atoi(args[7])
		cfg := PrinterConfig{
			PrinterName:      args[3],
			ConnType:         args[4],
			PrinterIPAddress: args[5],
			PrinterPort:      port,
			COMPort:          args[6],
			BaudRate:         baud,
			WindowsPrinter:   args[8],
			ServerURL:        args[9],
			PollingTime:      &polling,
		}
		if err := editPrinter(id, cfg); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "get":
		// returns "name|conntype|ip|port|com|baud|winprinter|url|poll"
		if len(args) < 3 {
			fmt.Println("usage: get <id>")
			return true
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid id")
			return true
		}
		printers, err := loadPrinters()
		if err != nil {
			fmt.Println("Error:", err)
			return true
		}
		for _, p := range printers {
			if p.ID == id {
				fmt.Printf("%s|%s|%s|%d|%s|%d|%s|%s|%d\n",
					p.PrinterName, p.GetConnType(),
					p.PrinterIPAddress, p.GetPort(),
					p.COMPort, p.GetBaudRate(),
					p.WindowsPrinter, p.ServerURL, p.GetPollingSeconds())
				return true
			}
		}
		fmt.Println("not found")
		return true

	case "delete":
		if len(args) < 3 {
			fmt.Println("usage: delete <id>")
			return true
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid id")
			return true
		}
		if err := deletePrinter(id); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "testprint":
		if len(args) < 3 {
			fmt.Println("usage: testprint <id>")
			return true
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid id")
			return true
		}
		if err := sendTestPrint(id); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Test print sent OK")
		}
		return true

	case "listcom":
		ports := listCOMPorts()
		fmt.Println(strings.Join(ports, "\n"))
		return true

	case "listprinters":
		printers := listWindowsPrinters()
		fmt.Println(strings.Join(printers, "\n"))
		return true
	}
	return false
}
