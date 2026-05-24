package main

import (
	"fmt"
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

	hubCfg := loadHubConfig()

	// Build ListView rows
	// Tidak pakai fmt.Sprintf di sini agar tidak ada % yang konflik
	// dengan outer fmt.Sprintf template
	listItems := ""
	for _, p := range printers {
		safeName := strings.ReplaceAll(p.PrinterName, "'", "''")
		safeConn := strings.ReplaceAll(p.ConnSummary(), "'", "''")
		safeUUID := strings.ReplaceAll(p.PrinterUUID, "'", "''")
		idStr := strconv.Itoa(p.ID)
		listItems += "$row = New-Object System.Windows.Forms.ListViewItem('" + idStr + "')\n"
		listItems += "$row.SubItems.Add('" + safeName + "') | Out-Null\n"
		listItems += "$row.SubItems.Add('" + safeConn + "') | Out-Null\n"
		listItems += "$row.SubItems.Add('" + safeUUID + "') | Out-Null\n"
		listItems += "$row.Tag = " + idStr + "\n"
		listItems += "$lvPrinters.Items.Add($row) | Out-Null\n"
	}

	exePath := getExePath()
	safeHubURL := strings.ReplaceAll(hubCfg.HubURL, "'", "''")

	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# ═══════════════════════════════════════════════════
# FORM
# ═══════════════════════════════════════════════════
$form = New-Object System.Windows.Forms.Form
$form.Text = "Munchii Printer Manager"
$form.Size = New-Object System.Drawing.Size(980, 800)
$form.MinimumSize = New-Object System.Drawing.Size(920, 720)
$form.StartPosition = "CenterScreen"
$form.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)
$form.Font = New-Object System.Drawing.Font("Segoe UI", 9)

# ── Title bar ──────────────────────────────────────────
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
$lblBrand.Size = New-Object System.Drawing.Size(200, 30)
$pnlTitle.Controls.Add($lblBrand)

$lblSub = New-Object System.Windows.Forms.Label
$lblSub.Text = "Printer Manager"
$lblSub.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$lblSub.ForeColor = [System.Drawing.Color]::FromArgb(180, 180, 200)
$lblSub.Location = New-Object System.Drawing.Point(16, 36)
$lblSub.Size = New-Object System.Drawing.Size(300, 20)
$pnlTitle.Controls.Add($lblSub)

# Hub Settings button di title bar (kanan)
$btnHubSettings = New-Object System.Windows.Forms.Button
$btnHubSettings.Text = "⚙  Hub Settings"
$btnHubSettings.Location = New-Object System.Drawing.Point(820, 16)
$btnHubSettings.Size = New-Object System.Drawing.Size(130, 32)
$btnHubSettings.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnHubSettings.FlatAppearance.BorderColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
$btnHubSettings.FlatAppearance.BorderSize = 1
$btnHubSettings.BackColor = [System.Drawing.Color]::FromArgb(40, 40, 60)
$btnHubSettings.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
$btnHubSettings.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
$btnHubSettings.Cursor = [System.Windows.Forms.Cursors]::Hand
$pnlTitle.Controls.Add($btnHubSettings)

# ── Hub status strip ──────────────────────────────────
$pnlHubStrip = New-Object System.Windows.Forms.Panel
$pnlHubStrip.Location = New-Object System.Drawing.Point(0, 64)
$pnlHubStrip.Size = New-Object System.Drawing.Size(980, 30)
$pnlHubStrip.BackColor = [System.Drawing.Color]::FromArgb(25, 25, 45)
$form.Controls.Add($pnlHubStrip)

$lblHubStatus = New-Object System.Windows.Forms.Label
$hubDisplay = if ('%s' -ne '') { '%s' } else { '(belum dikonfigurasi — klik Hub Settings)' }
$hubColor = if ('%s' -ne '') { [System.Drawing.Color]::FromArgb(80, 220, 120) } else { [System.Drawing.Color]::FromArgb(255, 150, 50) }
$lblHubStatus.Text = "  Hub WS: " + $hubDisplay
$lblHubStatus.ForeColor = $hubColor
$lblHubStatus.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblHubStatus.Location = New-Object System.Drawing.Point(0, 0)
$lblHubStatus.Size = New-Object System.Drawing.Size(900, 30)
$lblHubStatus.TextAlign = [System.Drawing.ContentAlignment]::MiddleLeft
$pnlHubStrip.Controls.Add($lblHubStatus)

# ── Section strip ──────────────────────────────────────
$pnlSec = New-Object System.Windows.Forms.Panel
$pnlSec.Location = New-Object System.Drawing.Point(0, 94)
$pnlSec.Size = New-Object System.Drawing.Size(980, 34)
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
$lvPrinters.Location = New-Object System.Drawing.Point(14, 136)
$lvPrinters.Size = New-Object System.Drawing.Size(940, 160)
$lvPrinters.View = [System.Windows.Forms.View]::Details
$lvPrinters.FullRowSelect = $true
$lvPrinters.GridLines = $true
$lvPrinters.MultiSelect = $false
$lvPrinters.BorderStyle = "FixedSingle"
$lvPrinters.BackColor = [System.Drawing.Color]::White
$lvPrinters.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lvPrinters.HeaderStyle = [System.Windows.Forms.ColumnHeaderStyle]::Nonclickable
$lvPrinters.Columns.Add("ID",         40)  | Out-Null
$lvPrinters.Columns.Add("Name",      180)  | Out-Null
$lvPrinters.Columns.Add("Connection",180)  | Out-Null
$lvPrinters.Columns.Add("UUID",      510)  | Out-Null
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

$btnTest   = MakeBtn "  Test Print"  14  304  148 34 $cBlue   $cWhite
$btnEdit   = MakeBtn "  Edit"       170  304  110 34 $cOrange $cWhite
$btnDelete = MakeBtn "  Delete"     288  304  110 34 $cRed    $cWhite
$form.Controls.Add($btnTest)
$form.Controls.Add($btnEdit)
$form.Controls.Add($btnDelete)

# ── Status label ───────────────────────────────────────
$lblStatus = New-Object System.Windows.Forms.Label
$lblStatus.Location = New-Object System.Drawing.Point(14, 344)
$lblStatus.Size = New-Object System.Drawing.Size(940, 24)
$lblStatus.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblStatus.ForeColor = $cGreen
$form.Controls.Add($lblStatus)

$sep = New-Object System.Windows.Forms.Label
$sep.BorderStyle = "Fixed3D"
$sep.Location = New-Object System.Drawing.Point(14, 372)
$sep.Size = New-Object System.Drawing.Size(940, 2)
$form.Controls.Add($sep)

# ═══════════════════════════════════════════════════
# ADD/EDIT PANEL
# ═══════════════════════════════════════════════════
$grp = New-Object System.Windows.Forms.GroupBox
$grp.Text = "  Add New Printer"
$grp.Location = New-Object System.Drawing.Point(14, 380)
$grp.Size = New-Object System.Drawing.Size(940, 380)
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
$txtName = MkTxt $tx 32 740
$grp.Controls.Add($txtName)

# Row 2: Connection Type
$grp.Controls.Add((MkLbl "Connection Type :" $lx 68))
$cboType = MkCombo $tx 68 200
$cboType.Items.Add("network")   | Out-Null
$cboType.Items.Add("bluetooth") | Out-Null
$cboType.Items.Add("usb")       | Out-Null
$cboType.SelectedIndex = 0
$grp.Controls.Add($cboType)

# Help button for connection type
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
        "> Setelah pair, Windows membuat port COM virtual"
        ""
        "## Cara Pair Printer Bluetooth"
        "1. Nyalakan printer Bluetooth"
        "2. Buka Settings Windows > Bluetooth & devices"
        "3. Klik [Add device] dan pilih printer"
        "4. Buka Device Manager > Ports (COM & LPT)"
        "5. Catat nomor COM yang muncul (misal COM3)"
        ""
        "! Jika COM Port tidak muncul, pastikan printer"
        "! sudah di-pair dan driver terinstall."
    )
    $usbLines = @(
        "# USB"
        ""
        "## Cara 1: Windows Printer (Direkomendasikan)"
        "Install driver printer lalu pilih nama printer dari dropdown."
        "> Lebih stabil dan mudah digunakan"
        ""
        "## Cara 2: USB via COM Port"
        "Beberapa printer USB muncul sebagai port COM virtual."
        "> Cocok untuk printer tanpa driver Windows"
        ""
        "## Cara Cek di Device Manager"
        "1. Klik kanan [This PC] > Device Manager"
        "2. Lihat di [Ports (COM & LPT)]"
        "3. Catat nomor COM saat USB dicolok"
        ""
        "! Jika tidak terdeteksi, install driver dari"
        "! website produsen printer."
    )

    $tabs.TabPages.Add((MakeTab "  Network/IP  " $netLines))
    $tabs.TabPages.Add((MakeTab "  Bluetooth  " $btLines))
    $tabs.TabPages.Add((MakeTab "  USB  " $usbLines))

    if ($t -eq "bluetooth") { $tabs.SelectedIndex = 1 }
    elseif ($t -eq "usb")   { $tabs.SelectedIndex = 2 }
    else                     { $tabs.SelectedIndex = 0 }

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

# ── Network fields ─────────────────────────────────────
$pnlNet = New-Object System.Windows.Forms.Panel
$pnlNet.Location = New-Object System.Drawing.Point(0, 104)
$pnlNet.Size = New-Object System.Drawing.Size(920, 70)
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

# ── COM port fields (Bluetooth / USB-Serial) ───────────
$pnlCOM = New-Object System.Windows.Forms.Panel
$pnlCOM.Location = New-Object System.Drawing.Point(0, 104)
$pnlCOM.Size = New-Object System.Drawing.Size(920, 70)
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

# ── USB Windows Printer fields ─────────────────────────
$pnlUSBWin = New-Object System.Windows.Forms.Panel
$pnlUSBWin.Location = New-Object System.Drawing.Point(0, 178)
$pnlUSBWin.Size = New-Object System.Drawing.Size(920, 36)
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

# ── UUID field ────────────────────────────────────────
$grp.Controls.Add((MkLbl "Printer UUID :" $lx 226))
$txtUUID = MkTxt $tx 226 560
$txtUUID.Text = ""
$txtUUID.Font = New-Object System.Drawing.Font("Consolas", 9)
$grp.Controls.Add($txtUUID)

# UUID help button
$btnUUIDHelp = New-Object System.Windows.Forms.Button
$btnUUIDHelp.Text = " ? "
$btnUUIDHelp.Location = New-Object System.Drawing.Point(($tx + 568), 226)
$btnUUIDHelp.Size = New-Object System.Drawing.Size(32, 26)
$btnUUIDHelp.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnUUIDHelp.FlatAppearance.BorderColor = [System.Drawing.Color]::FromArgb(0, 120, 212)
$btnUUIDHelp.FlatAppearance.BorderSize = 1
$btnUUIDHelp.BackColor = [System.Drawing.Color]::FromArgb(240, 248, 255)
$btnUUIDHelp.ForeColor = [System.Drawing.Color]::FromArgb(0, 120, 212)
$btnUUIDHelp.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
$btnUUIDHelp.Cursor = [System.Windows.Forms.Cursors]::Hand
$grp.Controls.Add($btnUUIDHelp)

$btnUUIDHelp.Add_Click({
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -AssemblyName System.Drawing
    $hw = New-Object System.Windows.Forms.Form
    $hw.Text = "Apa itu Printer UUID?"
    $hw.Size = New-Object System.Drawing.Size(520, 380)
    $hw.StartPosition = "CenterParent"
    $hw.FormBorderStyle = "FixedDialog"
    $hw.MaximizeBox = $false
    $hw.MinimizeBox = $false
    $hw.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)

    $ht = New-Object System.Windows.Forms.Panel
    $ht.Dock = "Top"; $ht.Height = 52
    $ht.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
    $hl = New-Object System.Windows.Forms.Label
    $hl.Text = "  Printer UUID"
    $hl.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 12)
    $hl.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
    $hl.Location = New-Object System.Drawing.Point(8, 12)
    $hl.Size = New-Object System.Drawing.Size(480, 28)
    $ht.Controls.Add($hl)
    $hw.Controls.Add($ht)

    $rtb = New-Object System.Windows.Forms.RichTextBox
    $rtb.Location = New-Object System.Drawing.Point(14, 66)
    $rtb.Size = New-Object System.Drawing.Size(476, 260)
    $rtb.ReadOnly = $true
    $rtb.BorderStyle = "None"
    $rtb.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)
    $rtb.Font = New-Object System.Drawing.Font("Segoe UI", 9)
    $rtb.ScrollBars = "Vertical"

    $lines = @(
        "## Apa itu UUID Printer?"
        "UUID adalah identitas unik untuk setiap printer fisik"
        "di sistem Munchii. Setiap printer punya UUID berbeda."
        ""
        "## Cara Mendapatkan UUID"
        "> Login ke dashboard Munchii"
        "> Buka menu Printers / Perangkat"
        "> Pilih printer yang ingin dikonfigurasi"
        "> Salin UUID yang tertera"
        ""
        "## Format UUID"
        "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        "Contoh:"
        "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        ""
        "## Catatan"
        "! UUID ini dipakai sebagai client_id saat"
        "! konek ke Go Hub WebSocket server."
        "! Pastikan UUID benar agar printer menerima"
        "! job cetak yang ditujukan untuknya."
    )
    foreach ($line in $lines) {
        if ($line.StartsWith("##")) {
            $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 10)
            $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(0, 90, 180)
            $rtb.AppendText($line.Substring(2).Trim() + [char]10)
        } elseif ($line.StartsWith(">")) {
            $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI", 9)
            $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(16, 130, 16)
            $rtb.AppendText("  " + $line.Substring(1).Trim() + [char]10)
        } elseif ($line.StartsWith("!")) {
            $rtb.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
            $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(180, 60, 0)
            $rtb.AppendText("  " + $line.Substring(1).Trim() + [char]10)
        } elseif ($line -eq "") {
            $rtb.AppendText([char]10)
        } else {
            $rtb.SelectionFont = New-Object System.Drawing.Font("Consolas", 9)
            $rtb.SelectionColor = [System.Drawing.Color]::FromArgb(40, 40, 60)
            $rtb.AppendText("  " + $line + [char]10)
        }
    }
    $hw.Controls.Add($rtb)

    $hClose = New-Object System.Windows.Forms.Button
    $hClose.Text = "Tutup"
    $hClose.Location = New-Object System.Drawing.Point(390, 332)
    $hClose.Size = New-Object System.Drawing.Size(90, 32)
    $hClose.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
    $hClose.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
    $hClose.ForeColor = [System.Drawing.Color]::White
    $hClose.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
    $hClose.Add_Click({ $hw.Close() })
    $hw.Controls.Add($hClose)
    $hw.ShowDialog() | Out-Null
})

# ── WS URL preview (read-only) ────────────────────────
$lblWSPreview = New-Object System.Windows.Forms.Label
$lblWSPreview.Text = "WS URL Preview :"
$lblWSPreview.Location = New-Object System.Drawing.Point($lx, 262)
$lblWSPreview.Size = New-Object System.Drawing.Size(148, 26)
$lblWSPreview.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(70, 70, 90)
$lblWSPreview.TextAlign = [System.Drawing.ContentAlignment]::MiddleRight
$grp.Controls.Add($lblWSPreview)

$txtWSPreview = MkTxt $tx 262 740
$txtWSPreview.ReadOnly = $true
$txtWSPreview.BackColor = [System.Drawing.Color]::FromArgb(235, 240, 248)
$txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(0, 90, 160)
$txtWSPreview.Font = New-Object System.Drawing.Font("Consolas", 8)
$txtWSPreview.Text = "(isi Hub Settings dan UUID untuk melihat preview)"
$grp.Controls.Add($txtWSPreview)

# Update WS preview saat UUID berubah
$Script:currentHubURL = '%s'
$txtUUID.Add_TextChanged({
    $uuid = $txtUUID.Text.Trim()
    $hub = $Script:currentHubURL
    if ($uuid -ne "" -and $hub -ne "") {
        $base = $hub.TrimEnd('/')
        $txtWSPreview.Text = $base + "?client_id=" + $uuid
        $txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(0, 130, 0)
    } elseif ($hub -eq "") {
        $txtWSPreview.Text = "(Hub Settings belum diisi — klik Hub Settings)"
        $txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(200, 100, 0)
    } else {
        $txtWSPreview.Text = "(isi UUID printer)"
        $txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(150, 150, 150)
    }
})

# ── Form status + buttons ──────────────────────────────
$lblFrmStatus = New-Object System.Windows.Forms.Label
$lblFrmStatus.Location = New-Object System.Drawing.Point($tx, 300)
$lblFrmStatus.Size = New-Object System.Drawing.Size(740, 22)
$lblFrmStatus.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$lblFrmStatus.ForeColor = $cGreen
$grp.Controls.Add($lblFrmStatus)

$btnSave  = MakeBtn "  Add Printer" $tx 326 160 36 $cGreen  $cWhite
$btnClear = New-Object System.Windows.Forms.Button
$btnClear.Text = "Clear"
$btnClear.Location = New-Object System.Drawing.Point(($tx+168), 326)
$btnClear.Size = New-Object System.Drawing.Size(80, 36)
$btnClear.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
$btnClear.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$btnClear.ForeColor = [System.Drawing.Color]::FromArgb(60,60,60)
$grp.Controls.Add($btnSave)
$grp.Controls.Add($btnClear)

# ═══════════════════════════════════════════════════
# HUB SETTINGS POPUP
# ═══════════════════════════════════════════════════
$btnHubSettings.Add_Click({
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -AssemblyName System.Drawing

    $hw = New-Object System.Windows.Forms.Form
    $hw.Text = "Hub Settings"
    $hw.Size = New-Object System.Drawing.Size(600, 420)
    $hw.StartPosition = "CenterParent"
    $hw.FormBorderStyle = "FixedDialog"
    $hw.MaximizeBox = $false
    $hw.MinimizeBox = $false
    $hw.BackColor = [System.Drawing.Color]::FromArgb(245, 246, 250)
    $hw.Font = New-Object System.Drawing.Font("Segoe UI", 9)

    # Title
    $ht = New-Object System.Windows.Forms.Panel
    $ht.Dock = "Top"; $ht.Height = 56
    $ht.BackColor = [System.Drawing.Color]::FromArgb(20, 20, 35)
    $hl = New-Object System.Windows.Forms.Label
    $hl.Text = "  ⚙  Hub WebSocket Settings"
    $hl.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 12)
    $hl.ForeColor = [System.Drawing.Color]::FromArgb(255, 180, 0)
    $hl.Location = New-Object System.Drawing.Point(8, 14)
    $hl.Size = New-Object System.Drawing.Size(560, 28)
    $ht.Controls.Add($hl)
    $hw.Controls.Add($ht)

    # Info box
    $rtbInfo = New-Object System.Windows.Forms.RichTextBox
    $rtbInfo.Location = New-Object System.Drawing.Point(14, 70)
    $rtbInfo.Size = New-Object System.Drawing.Size(558, 180)
    $rtbInfo.ReadOnly = $true
    $rtbInfo.BorderStyle = "FixedSingle"
    $rtbInfo.BackColor = [System.Drawing.Color]::FromArgb(240, 244, 255)
    $rtbInfo.Font = New-Object System.Drawing.Font("Segoe UI", 9)
    $rtbInfo.ScrollBars = "Vertical"

    $infoLines = @(
        "## Apa itu Hub WS URL?"
        "Alamat WebSocket server Go Hub yang berjalan di VPS/server kamu."
        "Semua printer di PC ini akan konek ke Hub yang sama."
        ""
        "## Format"
        "ws://IP_SERVER_PUBLIK:PORT/ws"
        ""
        "## Contoh"
        "ws://123.45.67.89:8080/ws"
        "ws://hub.munchii.com:8080/ws"
        ""
        "## Catatan"
        "! Port default Go Hub adalah 8080"
        "! Pastikan port ini tidak diblokir firewall server"
        "! URL harus diawali ws:// (bukan http://)"
    )
    foreach ($line in $infoLines) {
        if ($line.StartsWith("##")) {
            $rtbInfo.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 10)
            $rtbInfo.SelectionColor = [System.Drawing.Color]::FromArgb(0, 90, 180)
            $rtbInfo.AppendText($line.Substring(2).Trim() + [char]10)
        } elseif ($line.StartsWith("!")) {
            $rtbInfo.SelectionFont = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
            $rtbInfo.SelectionColor = [System.Drawing.Color]::FromArgb(180, 60, 0)
            $rtbInfo.AppendText("  " + $line.Substring(1).Trim() + [char]10)
        } elseif ($line -eq "") {
            $rtbInfo.AppendText([char]10)
        } else {
            $rtbInfo.SelectionFont = New-Object System.Drawing.Font("Consolas", 9)
            $rtbInfo.SelectionColor = [System.Drawing.Color]::FromArgb(40, 40, 60)
            $rtbInfo.AppendText("  " + $line + [char]10)
        }
    }
    $hw.Controls.Add($rtbInfo)

    # Input
    $lblURL = New-Object System.Windows.Forms.Label
    $lblURL.Text = "Hub WS URL :"
    $lblURL.Location = New-Object System.Drawing.Point(14, 264)
    $lblURL.Size = New-Object System.Drawing.Size(110, 28)
    $lblURL.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
    $lblURL.ForeColor = [System.Drawing.Color]::FromArgb(70, 70, 90)
    $lblURL.TextAlign = [System.Drawing.ContentAlignment]::MiddleRight
    $hw.Controls.Add($lblURL)

    $txtHubURL = New-Object System.Windows.Forms.TextBox
    $txtHubURL.Location = New-Object System.Drawing.Point(132, 264)
    $txtHubURL.Size = New-Object System.Drawing.Size(440, 28)
    $txtHubURL.Font = New-Object System.Drawing.Font("Consolas", 10)
    $txtHubURL.BorderStyle = "FixedSingle"
    $txtHubURL.BackColor = [System.Drawing.Color]::FromArgb(250, 250, 252)
    $txtHubURL.Text = $Script:currentHubURL
    $hw.Controls.Add($txtHubURL)

    # Validation label
    $lblVal = New-Object System.Windows.Forms.Label
    $lblVal.Location = New-Object System.Drawing.Point(132, 296)
    $lblVal.Size = New-Object System.Drawing.Size(440, 20)
    $lblVal.Font = New-Object System.Drawing.Font("Segoe UI", 8)
    $lblVal.ForeColor = [System.Drawing.Color]::Gray
    $lblVal.Text = "Format: ws://IP_SERVER:8080/ws"
    $hw.Controls.Add($lblVal)

    $txtHubURL.Add_TextChanged({
        $v = $txtHubURL.Text.Trim()
        if ($v -match "^ws://") {
            $lblVal.ForeColor = [System.Drawing.Color]::FromArgb(16, 130, 16)
            $lblVal.Text = "  ✓ Format URL valid"
        } elseif ($v -ne "") {
            $lblVal.ForeColor = [System.Drawing.Color]::FromArgb(190, 40, 30)
            $lblVal.Text = "  ✗ Harus diawali ws://"
        } else {
            $lblVal.ForeColor = [System.Drawing.Color]::Gray
            $lblVal.Text = "  Format: ws://IP_SERVER:8080/ws"
        }
    })

    # Buttons
    $btnHubSave = New-Object System.Windows.Forms.Button
    $btnHubSave.Text = "  Simpan"
    $btnHubSave.Location = New-Object System.Drawing.Point(352, 340)
    $btnHubSave.Size = New-Object System.Drawing.Size(100, 36)
    $btnHubSave.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
    $btnHubSave.FlatAppearance.BorderSize = 0
    $btnHubSave.BackColor = [System.Drawing.Color]::FromArgb(16, 130, 16)
    $btnHubSave.ForeColor = [System.Drawing.Color]::White
    $btnHubSave.Font = New-Object System.Drawing.Font("Segoe UI Semibold", 9)
    $btnHubSave.Cursor = [System.Windows.Forms.Cursors]::Hand
    $hw.Controls.Add($btnHubSave)

    $btnHubCancel = New-Object System.Windows.Forms.Button
    $btnHubCancel.Text = "Batal"
    $btnHubCancel.Location = New-Object System.Drawing.Point(460, 340)
    $btnHubCancel.Size = New-Object System.Drawing.Size(80, 36)
    $btnHubCancel.FlatStyle = [System.Windows.Forms.FlatStyle]::Flat
    $btnHubCancel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
    $btnHubCancel.ForeColor = [System.Drawing.Color]::FromArgb(60, 60, 60)
    $btnHubCancel.Add_Click({ $hw.Close() })
    $hw.Controls.Add($btnHubCancel)

    $btnHubSave.Add_Click({
        $url = $txtHubURL.Text.Trim()
        if ($url -eq "" -or -not ($url -match "^ws://")) {
            $lblVal.ForeColor = [System.Drawing.Color]::FromArgb(190, 40, 30)
            $lblVal.Text = "  ✗ URL tidak valid — harus diawali ws://"
            return
        }
        & '%s' savehub "$url" 2>&1 | Out-Null
        $Script:currentHubURL = $url
        # Update hub status strip
        $lblHubStatus.Text = "  Hub WS: $url"
        $lblHubStatus.ForeColor = [System.Drawing.Color]::FromArgb(80, 220, 120)
        # Update WS preview jika UUID sudah diisi
        $uuid = $txtUUID.Text.Trim()
        if ($uuid -ne "") {
            $base = $url.TrimEnd('/')
            $txtWSPreview.Text = $base + "?client_id=" + $uuid
            $txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(0, 130, 0)
        }
        $hw.Close()
    })

    $hw.ShowDialog() | Out-Null
})

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
    $txtUUID.Text = ""
    $txtWSPreview.Text = "(isi Hub Settings dan UUID untuk melihat preview)"
    $txtWSPreview.ForeColor = [System.Drawing.Color]::FromArgb(150, 150, 150)
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
    # raw = "name|conntype|ip|port|com|baud|winprinter|uuid"
    $parts = "$raw" -split "\|"
    if ($parts.Count -ge 8) {
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
        $txtUUID.Text = $parts[7]
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
    $uuid = $txtUUID.Text.Trim()
    if ($name -eq "" -or $uuid -eq "") {
        $lblFrmStatus.ForeColor = $cRed
        $lblFrmStatus.Text = "  Printer Name dan UUID wajib diisi."
        return
    }
    $args = GetConnArgs
    if ($Script:editID -ge 0) {
        & '%s' edit "$($Script:editID)" "$name" "$($args[0])" "$($args[1])" "$($args[2])" "$($args[3])" "$($args[4])" "$($args[5])" "$uuid" 2>&1 | Out-Null
        $row = $lvPrinters.SelectedItems[0]
        if ($null -ne $row) {
            $row.Text              = "$($Script:editID)"
            $row.SubItems[1].Text = $name
            $row.SubItems[2].Text = "$($args[0])://$($args[1])$($args[3])"
            $row.SubItems[3].Text = $uuid
        }
        $lblFrmStatus.ForeColor = $cGreen
        $lblFrmStatus.Text = "  Printer updated."
        ClearForm
    } else {
        & '%s' add "$name" "$($args[0])" "$($args[1])" "$($args[2])" "$($args[3])" "$($args[4])" "$($args[5])" "$uuid" 2>&1 | Out-Null
        $newID = $lvPrinters.Items.Count
        $row = New-Object System.Windows.Forms.ListViewItem("$newID")
        $row.SubItems.Add($name)                                        | Out-Null
        $row.SubItems.Add("$($args[0])://$($args[1])$($args[3])")      | Out-Null
        $row.SubItems.Add($uuid)                                        | Out-Null
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
`,
		safeHubURL, safeHubURL, safeHubURL, // hub status strip: display, color check
		listItems,
		safeHubURL, // WS preview script variable
		exePath,    // savehub
		exePath,    // listcom
		exePath,    // listprinters
		exePath,    // testprint
		exePath,    // get
		exePath,    // delete
		exePath,    // edit
		exePath,    // add
	)

	cmd := newPSCommand(ps)
	cmd.Start()
}

func showErrorDialog(msg string) {
	safe := strings.ReplaceAll(msg, "'", "''")
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.MessageBox]::Show('%s','Munchii Error',[System.Windows.Forms.MessageBoxButtons]::OK,[System.Windows.Forms.MessageBoxIcon]::Error) | Out-Null
`, safe)
	cmd := newPSCommand(ps)
	cmd.Start()
}

// ── CLI dispatch ──────────────────────────────────────────────────────────────

func handleCLI(args []string) bool {
	if len(args) < 2 {
		return false
	}
	switch args[1] {

	case "savehub":
		// savehub <url>
		if len(args) < 3 {
			fmt.Println("usage: savehub <ws://host:port/ws>")
			return true
		}
		cfg := HubConfig{HubURL: args[2]}
		if err := saveHubConfig(cfg); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "add":
		// add <name> <conntype> <ip> <port> <com> <baud> <winprinter> <uuid>
		if len(args) < 9 {
			fmt.Println("usage: add <name> <conntype> <ip> <port> <com> <baud> <winprinter> <uuid>")
			return true
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
			PrinterUUID:      args[8],
		}
		if err := addPrinter(cfg); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "edit":
		// edit <id> <name> <conntype> <ip> <port> <com> <baud> <winprinter> <uuid>
		if len(args) < 10 {
			fmt.Println("usage: edit <id> <name> <conntype> <ip> <port> <com> <baud> <winprinter> <uuid>")
			return true
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid id")
			return true
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
			PrinterUUID:      args[9],
		}
		if err := editPrinter(id, cfg); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("OK")
		}
		return true

	case "get":
		// returns "name|conntype|ip|port|com|baud|winprinter|uuid"
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
				fmt.Printf("%s|%s|%s|%d|%s|%d|%s|%s\n",
					p.PrinterName, p.GetConnType(),
					p.PrinterIPAddress, p.GetPort(),
					p.COMPort, p.GetBaudRate(),
					p.WindowsPrinter, p.PrinterUUID)
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
