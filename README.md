# Kypesen Printer — Windows Tray App

Aplikasi Windows yang berjalan di system tray untuk menerima job cetak secara **real-time via WebSocket** dari server Go Hub, lalu meneruskannya langsung ke printer ESC/POS via jaringan lokal.

---

## Arsitektur Sistem

```
┌─────────────────────────────────────────────────────────────────────┐
│  SERVER (VPS/AWS)                                                   │
│                                                                     │
│  ┌─────────────────┐    push JSON     ┌──────────────────────────┐  │
│  │  Laravel        │ ─────────────→  │  Go Hub (port 8080)      │  │
│  │  PrintController│  + image_base64  │  WebSocket Broker        │  │
│  │                 │  http POST       │  /api/push-print         │  │
│  │  wkhtmltoimage  │                  │  /ws?client_id=UUID      │  │
│  │  (generate PNG) │                  └──────────┬───────────────┘  │
│  └─────────────────┘                             │ WebSocket        │
└──────────────────────────────────────────────────┼─────────────────┘
                                                   │ (persistent conn)
┌──────────────────────────────────────────────────┼─────────────────┐
│  PC KASIR / RESTORAN (Windows)                   │                 │
│                                                  ▼                 │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  kypesen-printer.exe (System Tray)                            │ │
│  │                                                               │ │
│  │  WSClient ──→ decode base64 ──→ EscPrinter ──→ TCP port 9100 │ │
│  │  (per UUID)                                  (network/BT/USB) │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                          │              │              │            │
│                    Printer Kasir   Printer Dapur   Printer Bar     │
└─────────────────────────────────────────────────────────────────────┘
```

**Flow per job cetak:**
1. Order masuk → Laravel `PrintController` jalankan `wkhtmltoimage` di server → hasilkan PNG
2. PNG di-encode ke Base64 → di-push via HTTP POST ke Go Hub (`/api/push-print`)
3. Go Hub forward payload ke `kypesen-printer.exe` yang terkoneksi (match by UUID)
4. `.exe` decode Base64 → gambar → ESC/POS bit-image → kirim TCP ke printer fisik

> **Catatan:** Proses generate gambar sepenuhnya di server. PC kasir hanya terima, decode, dan cetak.

---

## Fitur

- 🖥️ Berjalan di system tray Windows (tidak ada jendela)
- ⚡ Real-time via WebSocket — cetak langsung saat order masuk, tanpa delay polling
- 🔁 Auto-reconnect jika koneksi ke Hub terputus (delay bertahap 3s → max 60s)
- 🖨️ Support banyak printer sekaligus (goroutine per printer, non-blocking)
- 🔌 3 jenis koneksi printer: **Network (TCP)**, **Bluetooth (COM)**, **USB (COM / Windows Printer)**
- ⚙️ UI manage printer dari tray icon (tambah/edit/hapus)
- 📋 View log aktivitas real-time dengan color-coding per level
- 🗓️ Log otomatis dihapus setelah 1 bulan
- 🔄 Auto-start saat Windows login (toggle dari tray menu)
- 🧪 Test print per printer (semua jenis koneksi)

---

## Struktur File

```
kypesen-printer/
│
├── main.go                   ← Entry point + systray setup + tray menu event loop
├── websocket.go              ← WebSocket client (WSManager, WSClient, auto-reconnect,
│                               parse payload, dispatch ke printer)
├── kypesen.go                ← Struct payload JSON dari Go Hub (HubPayload, Order,
│                               NoteItem, ReportItem, dll)
├── printer.go                ← ESC/POS engine: koneksi Network/BT/USB, bitImage(),
│                               sendTestPrint(), text fallback kitchen & cashier
├── config.go                 ← Baca/tulis printers.json (PrinterConfig, CRUD printer)
├── logger.go                 ← Logging ke buffer + file, auto-cleanup log > 1 bulan
├── ui.go                     ← Manage Printers window (PowerShell Forms), CLI dispatch
│                               (add/edit/delete/get/testprint/listcom/listprinters)
├── icon.go                   ← Generate tray icon (ICO) secara programatik
│
├── autostart_windows.go      ← Registry auto-start (build tag: windows)
├── autostart_stub.go         ← Stub kosong untuk non-Windows
├── windows_helpers.go        ← newPSCommand(), listCOMPorts(), listWindowsPrinters()
│                               (build tag: windows)
├── windows_helpers_stub.go   ← Stub untuk non-Windows
├── polling.go                ← [DEPRECATED] Digantikan websocket.go, dibiarkan kosong
│
├── go.mod                    ← Dependencies Go
├── build.bat                 ← Script build Windows → kypesen-printer.exe
├── versioninfo.json          ← Metadata EXE (versi, company, dll)
└── app.manifest              ← Windows app manifest (DPI awareness, dll)
```

### Penjelasan per file

#### `main.go`
Entry point aplikasi. Menginisialisasi systray, membangun tray menu, dan menjalankan event loop. Memanggil `wsManager.StartAll()` saat startup dan `wsManager.Restart()` setelah perubahan konfigurasi printer. Menjalankan `StartLogCleanup()` untuk jadwal pembersihan log otomatis.

#### `websocket.go`
Inti dari sistem real-time. Berisi:
- **`WSManager`** — mengelola map `client_id → WSClient`, menyediakan `StartAll()`, `Stop()`, `Restart()`, `GetStatuses()`
- **`WSClient`** — satu instance per printer config; menjalankan loop koneksi WS dengan exponential backoff auto-reconnect
- **`buildHubWSURL()`** — mengekstrak host dan UUID dari `server_url` config, membangun `ws://host:8080/ws?client_id=UUID`
- **`handleMessage()`** — parse JSON payload, dispatch ke `handleNotes()` / `handleReports()` / `handleOrders()`
- **`printBase64Image()`** — decode Base64 → PNG → `bitImage()` → flush ke printer (dijalankan sebagai goroutine)

#### `kypesen.go`
Definisi semua struct Go yang memetakan payload JSON dari Go Hub:
- `HubPayload` — root payload (`status`, `from`, `client_id`, `ip_address`, `data`)
- `HubData` — container untuk `Printer`, `Orders`, `Notes`, `Reports`
- `Order`, `NoteItem`, `ReportItem` — masing-masing menyertakan field `ImageBase64`
- Supporting types: `Restaurant`, `TableInfo`, `Item`, `ItemPivot`, dll

#### `printer.go`
ESC/POS printing engine:
- **`newPrinterFromConfig()`** — factory yang membuka koneksi sesuai `conn_type` (network/bluetooth/usb)
- **`newNetworkPrinter()`** — TCP dial ke IP:9100
- **`newCOMPrinter()`** — buka serial COM port (Bluetooth SPP / USB-Serial)
- **`newWindowsPrinter()`** — kirim ke Windows printer name via RAW spooler
- **`bitImage()`** — konversi `image.Image` → ESC/POS GS v 0 bit-image format
- **`sendTestPrint()`** — cetak halaman test dengan info koneksi, support semua jenis koneksi
- **`printKitchenText()` / `printCashierText()`** — text fallback ESC/POS (dipertahankan)

#### `config.go`
Manajemen konfigurasi printer:
- Struct `PrinterConfig` dengan field: `ConnType`, `PrinterIPAddress`, `PrinterPort`, `COMPort`, `BaudRate`, `WindowsPrinter`, `ServerURL`, `PollingTime`
- `loadPrinters()` / `savePrinters()` / `addPrinter()` / `editPrinter()` / `deletePrinter()`
- Helper: `GetConnType()`, `GetPort()`, `GetBaudRate()`, `ConnSummary()`
- Config disimpan di `printers.json` di folder yang sama dengan `.exe`

#### `logger.go`
- `logStatus(msg)` — tulis ke buffer memori (200 baris) + file `kypesen-printer.log`
- `getRecentLogs(n)` — ambil N baris terakhir untuk ditampilkan di UI
- `StartLogCleanup()` — goroutine yang jalan setiap 24 jam, hapus baris log > 1 bulan
- Format timestamp: `[2006-01-02 15:04:05] LEVEL [context] pesan`

#### `ui.go`
- `openManageWindow()` — UI PowerShell Forms untuk CRUD printer, support semua jenis koneksi, tombol Test Print, panduan koneksi interaktif per tipe
- `openLogWindow()` — tampilkan 100 baris log terakhir dengan color-coding: OK=hijau, ERROR=merah, WARN=kuning, INFO=biru
- `handleCLI()` — dispatch command dari PowerShell ke Go: `add`, `edit`, `get`, `delete`, `testprint`, `listcom`, `listprinters`

---

## Format `printers.json`

```json
[
    {
        "id": 0,
        "printer_name": "Kasir 1",
        "conn_type": "network",
        "printer_ip_address": "192.168.1.100",
        "printer_port": 9100,
        "com_port": "",
        "baud_rate": 0,
        "windows_printer": "",
        "server_url": "http://192.168.1.10:8001/api/v1/print/UUID-PRINTER-DISINI",
        "polling_time": 5
    },
    {
        "id": 1,
        "printer_name": "Dapur (Bluetooth)",
        "conn_type": "bluetooth",
        "printer_ip_address": "",
        "printer_port": 0,
        "com_port": "COM3",
        "baud_rate": 9600,
        "windows_printer": "",
        "server_url": "http://192.168.1.10:8001/api/v1/print/UUID-PRINTER-LAIN",
        "polling_time": 5
    }
]
```

> `server_url` dipakai untuk mengekstrak **host Go Hub** dan **UUID printer**.
> Format: `http://<host-laravel>:<port>/api/v1/print/<UUID>`
> UUID dari URL ini dipakai sebagai `client_id` saat konek ke WebSocket Hub.

---

## Jenis Koneksi Printer

| Tipe | Field yang diisi | Cocok untuk |
|------|-----------------|-------------|
| `network` | `printer_ip_address`, `printer_port` | Printer LAN/WiFi ESC/POS |
| `bluetooth` | `com_port`, `baud_rate` | Printer Bluetooth (pair dulu di Windows) |
| `usb` (via Windows Printer) | `windows_printer` | Printer USB dengan driver Windows |
| `usb` (via COM) | `com_port`, `baud_rate` | Printer USB-Serial tanpa driver |

---

## Cara Build

### Requirement

- [Go 1.21+](https://go.dev/dl/) — hanya di PC developer, **tidak perlu di PC target**
- Windows 10/11 (untuk build native) **atau** Linux/Mac (cross-compile)

### Build di Windows (paling mudah)

```bat
REM Buka folder project, double-click:
build.bat

REM Output: kypesen-printer.exe
```

### Build di Linux / Mac (cross-compile)

```bash
cd kypesen-printer/

# Download dependency dulu
go mod tidy

# Cross-compile → Windows 64-bit
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-H windowsgui -s -w" -o kypesen-printer.exe .
```

### Flag build yang dipakai

| Flag | Keterangan |
|------|------------|
| `GOOS=windows` | Target OS Windows |
| `GOARCH=amd64` | Target 64-bit |
| `CGO_ENABLED=0` | Tidak pakai C library — binary fully static |
| `-H windowsgui` | Tidak muncul console window hitam saat dijalankan |
| `-s -w` | Strip debug info — ukuran binary lebih kecil |

---

## Deploy ke PC Target

### Yang perlu dikopi ke PC target

```
📁 folder bebas (misal C:\KypesenPrinter\)/
└── kypesen-printer.exe    ← satu file ini saja
```

> PC target **tidak perlu** install Go, PHP, wkhtmltoimage, atau apapun.
> Generate gambar dilakukan di server Laravel — PC kasir hanya terima dan cetak.

### Langkah pertama kali

1. Double-click `kypesen-printer.exe`
2. Ikon printer muncul di system tray (pojok kanan bawah)
3. Klik kanan → **Manage Printers** → tambah printer
4. Isi `server_url` dengan URL API printer dari dashboard Kypesen (yang mengandung UUID)
5. Isi IP/COM sesuai jenis koneksi printer fisik

### Auto-start Windows

Klik kanan tray icon → centang **Auto-start dengan Windows**

---

## Troubleshooting

**Printer tidak print / tidak ada response:**
- Cek log: klik kanan tray → **View Log**, cari baris `ERROR`
- Pastikan `server_url` mengandung UUID yang benar
- Pastikan Go Hub berjalan dan bisa diakses dari PC kasir
- Cek baris `OK [NamaPrinter] Terhubung ke Hub ✓` — kalau tidak ada, koneksi WS belum berhasil

**Koneksi WebSocket terputus terus:**
- Cek firewall — port 8080 harus bisa diakses dari PC kasir ke server Hub
- Log akan menampilkan `Reconnect attempt #N dalam Xs...` — ini normal, akan retry otomatis

**Printer network tidak print:**
- Pastikan IP printer benar dan bisa di-ping dari PC
- Pastikan port 9100 tidak diblok firewall lokal
- Cek log: `ERROR [NamaPrinter] koneksi printer gagal [network → IP:9100]: ...`

**Printer Bluetooth tidak terdeteksi:**
- Pastikan printer sudah di-pair di Windows Settings → Bluetooth & devices
- Buka Device Manager → Ports (COM & LPT) → catat nomor COM
- Isi COM port yang benar di Manage Printers

**Printer USB tidak print:**
- Coba gunakan "Windows Printer" (nama printer di Windows) bukan COM port
- Pastikan driver printer terinstall dan status Online

**Log View:**
- File log: `kypesen-printer.log` di folder yang sama dengan `.exe`
- Log otomatis dihapus setelah 1 bulan
- Color-coding: 🟢 OK, 🔴 ERROR, 🟡 WARN, 🔵 INFO