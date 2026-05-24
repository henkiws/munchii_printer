# Munchii Printer — Windows Tray App

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
│  │  munchii-printer.exe (System Tray)                            │ │
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
3. Go Hub forward payload ke `munchii-printer.exe` yang terkoneksi, dicocokkan by UUID
4. `.exe` decode Base64 → gambar → ESC/POS bit-image → kirim ke printer fisik

> **Catatan:** Generate gambar sepenuhnya di server. PC kasir hanya terima, decode, dan cetak. Tidak perlu `wkhtmltoimage.exe` di PC target.

---

## Fitur

- 🖥️ Berjalan di system tray Windows (tidak ada jendela)
- ⚡ Real-time via WebSocket — cetak langsung saat order masuk, tanpa delay polling
- 🔁 Auto-reconnect jika koneksi ke Hub terputus (exponential backoff: 3s → max 60s)
- 🖨️ Support banyak printer sekaligus (goroutine per printer, non-blocking)
- 🔌 3 jenis koneksi printer: **Network (TCP)**, **Bluetooth (COM)**, **USB (COM / Windows Printer)**
- ⚙️ UI manage printer dari tray icon (tambah/edit/hapus)
- 📊 Kolom status printer real-time di UI (Connected / Disconnected / Connecting)
- 📋 View log aktivitas dengan color-coding per level (OK / ERROR / WARN / INFO)
- 🗓️ Log otomatis dihapus setelah 1 bulan
- 🔄 Auto-start saat Windows login (toggle dari tray menu)
- 🧪 Test print per printer — tampilkan detail error jika gagal

---

## Struktur File

```
munchii-printer/
│
├── main.go                   ← Entry point + systray setup + tray menu event loop
├── websocket.go              ← WebSocket client (WSManager, WSClient, auto-reconnect,
│                               parse payload, dispatch ke printer)
├── kypesen.go                ← Struct payload JSON dari Go Hub (HubPayload, Order,
│                               NoteItem, ReportItem, dll)
├── printer.go                ← ESC/POS engine: koneksi Network/BT/USB, bitImage(),
│                               sendTestPrint(), text fallback kitchen & cashier
├── config.go                 ← Baca/tulis printers.json + hub.json
│                               (PrinterConfig, HubConfig, CRUD printer)
├── logger.go                 ← Logging ke buffer + file, auto-cleanup log > 1 bulan
├── ui.go                     ← Manage Printers window (PowerShell Forms), CLI dispatch
│                               (add/edit/get/delete/testprint/getstatus/listcom/listprinters)
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
├── build.bat                 ← Script build Windows → munchii-printer.exe
├── versioninfo.json          ← Metadata EXE (versi, company, dll)
└── app.manifest              ← Windows app manifest (DPI awareness, dll)
```

**File yang dibuat otomatis saat runtime** (di folder yang sama dengan `.exe`):

```
📁 folder instalasi/
├── munchii-printer.exe
├── printers.json             ← config semua printer (auto-dibuat)
├── hub.json                  ← config Go Hub WS URL (auto-dibuat via Hub Settings)
└── munchii-printer.log       ← log aktivitas (auto-dibuat, auto-cleanup 1 bulan)
```

---

## Penjelasan per File

### `main.go`
Entry point aplikasi. Menginisialisasi systray, membangun tray menu, dan menjalankan event loop. Memanggil `wsManager.StartAll()` saat startup dan `wsManager.Restart()` setelah perubahan konfigurasi printer. Menjalankan `StartLogCleanup()` untuk jadwal pembersihan log otomatis.

### `websocket.go`
Inti dari sistem real-time. Berisi:
- **`WSManager`** — mengelola map `printer_id → WSClient`, menyediakan `StartAll()`, `Stop()`, `Restart()`, `GetStatuses()`
- **`WSClient`** — satu instance per printer config; menjalankan loop koneksi WS dengan exponential backoff auto-reconnect
- **`buildHubWSURL(cfg)`** — membaca `hub.json` dan UUID printer, membangun `ws://HOST:8080/ws?client_id=UUID`
- **`handleMessage()`** — parse JSON payload, dispatch ke `handleNotes()` / `handleReports()` / `handleOrders()`
- **`printBase64Image()`** — decode Base64 → PNG → `bitImage()` → flush ke printer (goroutine, non-blocking)

### `kypesen.go`
Definisi semua struct Go yang memetakan payload JSON dari Go Hub:
- `HubPayload` — root payload (`status`, `from`, `client_id`, `ip_address`, `data`)
- `HubData` — container untuk `Printer`, `Orders`, `Notes`, `Reports`
- `Order`, `NoteItem`, `ReportItem` — masing-masing menyertakan field `ImageBase64`
- Supporting types: `Restaurant`, `TableInfo`, `Item`, `ItemPivot`, dll

### `printer.go`
ESC/POS printing engine:
- **`newPrinterFromConfig()`** — factory yang membuka koneksi sesuai `conn_type` (network/bluetooth/usb)
- **`newNetworkPrinter()`** — TCP dial ke IP:9100
- **`newCOMPrinter()`** — buka serial COM port (Bluetooth SPP / USB-Serial)
- **`newWindowsPrinter()`** — kirim ke Windows printer name via RAW spooler
- **`bitImage()`** — konversi `image.Image` → ESC/POS GS v 0 bit-image format
- **`sendTestPrint()`** — cetak halaman test dengan info koneksi lengkap, support semua jenis koneksi
- **`printKitchenText()` / `printCashierText()`** — text fallback ESC/POS (dipertahankan)

### `config.go`
Manajemen konfigurasi:
- **`PrinterConfig`** — struct per printer: `ConnType`, `PrinterIPAddress`, `PrinterPort`, `COMPort`, `BaudRate`, `WindowsPrinter`, `PrinterUUID`
- **`HubConfig`** — struct global: `HubURL` (ws://host:port/ws), disimpan di `hub.json`
- `loadPrinters()` / `savePrinters()` / `addPrinter()` / `editPrinter()` / `deletePrinter()`
- `loadHubConfig()` / `saveHubConfig()`
- Helper: `GetConnType()`, `GetPort()`, `GetBaudRate()`, `ConnSummary()`

### `logger.go`
- `logStatus(msg)` — tulis ke buffer memori (200 baris) + file `munchii-printer.log`
- `getRecentLogs(n)` — ambil N baris terakhir untuk ditampilkan di UI
- `StartLogCleanup()` — goroutine yang jalan setiap 24 jam, hapus baris log > 1 bulan
- Format: `[2006-01-02 15:04:05] LEVEL [context] pesan`

### `ui.go`
- `openManageWindow()` — UI PowerShell Forms: tabel printer + kolom status, form add/edit, Hub Settings, panduan koneksi interaktif, UUID help
- `openLogWindow()` — 100 baris log terakhir dengan color-coding
- `handleCLI()` — dispatch command dari PowerShell ke Go:

| Command | Keterangan |
|---------|-----------|
| `add` | Tambah printer baru |
| `edit <id>` | Edit printer |
| `get <id>` | Ambil data printer (pipe-separated) |
| `delete <id>` | Hapus printer |
| `testprint <id>` | Test print, return `OK:...` atau `Error:...` |
| `getstatus [id]` | Status WS connection per printer |
| `savehub <url>` | Simpan Hub WS URL ke hub.json |
| `listcom` | List COM ports yang tersedia |
| `listprinters` | List Windows printer yang terinstall |

---

## Format Config File

### `printers.json`

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
        "printer_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
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
        "printer_uuid": "b2c3d4e5-f6a7-8901-bcde-f12345678901"
    },
    {
        "id": 2,
        "printer_name": "Bar (USB)",
        "conn_type": "usb",
        "printer_ip_address": "",
        "printer_port": 0,
        "com_port": "",
        "baud_rate": 0,
        "windows_printer": "EPSON TM-T82",
        "printer_uuid": "c3d4e5f6-a7b8-9012-cdef-123456789012"
    }
]
```

> **Penting:** Field `printer_uuid` adalah UUID unik printer dari dashboard Munchii. UUID ini dipakai sebagai `client_id` saat konek ke Go Hub WebSocket — pastikan benar dan unik per printer.

### `hub.json`

```json
{
    "hub_url": "ws://123.45.67.89:8080/ws"
}
```

> File ini dibuat dan dikelola otomatis oleh UI (Hub Settings). Tidak perlu diedit manual. Format URL: `ws://IP_SERVER:PORT/ws` — harus diawali `ws://`, bukan `http://`.

---

## Jenis Koneksi Printer

| Tipe | Field yang diisi | Cocok untuk |
|------|-----------------|-------------|
| `network` | `printer_ip_address`, `printer_port` | Printer LAN/WiFi ESC/POS |
| `bluetooth` | `com_port`, `baud_rate` | Printer Bluetooth (pair dulu di Windows) |
| `usb` (Windows Printer) | `windows_printer` | Printer USB dengan driver Windows |
| `usb` (COM) | `com_port`, `baud_rate` | Printer USB-Serial tanpa driver |

---

## Cara Build

### Requirement

- [Go 1.21+](https://go.dev/dl/) — hanya di PC developer, **tidak perlu di PC target**
- Windows 10/11 (untuk build native) **atau** Linux/Mac (cross-compile)

### Build di Windows

```bat
REM Buka folder project, double-click:
build.bat

REM Output: munchii-printer.exe
```

### Build di Linux / Mac (cross-compile untuk Windows)

```bash
cd munchii-printer/

# Download dependency
go mod tidy

# Cross-compile → Windows 64-bit
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-H windowsgui -s -w" -o munchii-printer.exe .
```

### Flag build

| Flag | Keterangan |
|------|------------|
| `GOOS=windows` | Target OS Windows |
| `GOARCH=amd64` | Target 64-bit |
| `CGO_ENABLED=0` | Tidak pakai C library — binary fully static |
| `-H windowsgui` | Tidak muncul console window hitam saat dijalankan |
| `-s -w` | Strip debug info — ukuran binary lebih kecil |

---

## Deploy ke PC Target

### Yang perlu dikopi

```
📁 folder bebas (misal C:\MunchiiPrinter\)/
└── munchii-printer.exe    ← satu file ini saja
```

> PC target **tidak perlu** install Go, PHP, wkhtmltoimage, atau apapun. Generate gambar dilakukan di server Laravel.

### Setup pertama kali

1. Double-click `munchii-printer.exe`
2. Ikon printer muncul di system tray (pojok kanan bawah)
3. Klik kanan → **Manage Printers**
4. Klik **Hub Settings** (pojok kanan atas dialog) → isi URL Go Hub → Simpan
   - Format: `ws://IP_SERVER:8080/ws`
   - Contoh: `ws://123.45.67.89:8080/ws`
5. Klik **Add Printer** → isi form:
   - **Printer Name** — nama bebas, misal `Kasir 1`
   - **Connection Type** — pilih `network` / `bluetooth` / `usb`
   - **IP / COM / Windows Printer** — sesuai jenis koneksi
   - **Printer UUID** — UUID dari dashboard Munchii (klik `?` untuk panduan)
6. Klik **Add Printer** → printer muncul di list dengan status `Connecting...`
7. Tunggu beberapa detik → status berubah jadi `● Connected`
8. Klik **Test Print** untuk verifikasi koneksi printer fisik

### Auto-start Windows

Klik kanan tray icon → centang **Auto-start dengan Windows**

---

## Status Printer di UI

| Status | Warna | Arti |
|--------|-------|------|
| `● Connected` | 🟢 Hijau | Terkoneksi ke Hub, siap terima job cetak |
| `⟳ Connecting...` | 🟠 Oranye | Sedang mencoba konek ke Hub |
| `✗ Disconnected` | 🔴 Merah | Koneksi terputus, sedang auto-reconnect |
| `✗ Error: ...` | 🔴 Merah | Gagal konek, lihat detail di View Log |
| `! UUID belum diisi` | 🔴 Merah | UUID printer belum dikonfigurasi |
| `✓ Test Print OK` | 🟢 Hijau | Test print terakhir berhasil |
| `✗ Test Print Gagal` | 🔴 Merah | Test print terakhir gagal — klik Test Print lagi untuk detail |

Klik **Refresh Status** untuk update kolom status secara manual.

---

## Troubleshooting

**Status tetap `Connecting...` / tidak pernah `Connected`:**
- Pastikan Go Hub berjalan di server dan port 8080 bisa diakses dari PC kasir
- Cek Hub URL di Hub Settings — harus `ws://` bukan `http://`
- Cek log: `ERROR [NamaPrinter] Koneksi WS gagal: ...`
- Coba ping IP server dari CMD: `ping IP_SERVER`

**UUID salah / tidak terima job cetak:**
- Pastikan UUID di Manage Printers sama persis dengan UUID di dashboard Munchii
- UUID bersifat case-sensitive
- Cek log: seharusnya ada baris `OK [NamaPrinter] Hub ACK: client_id=...` saat connect

**Test print gagal:**
- Klik Test Print → popup error detail akan muncul dengan info spesifik
- Untuk network: pastikan IP printer benar dan bisa di-ping
- Untuk Bluetooth: pastikan printer sudah di-pair dan COM port benar
- Untuk USB: pastikan driver terinstall dan printer status Online di Windows
- Cek log: `ERROR [NamaPrinter] koneksi printer gagal [...]: ...`

**Koneksi WebSocket terputus-putus:**
- Normal jika jaringan tidak stabil — app akan auto-reconnect otomatis
- Log akan tampilkan `Reconnect attempt #N dalam Xs...`
- Pastikan tidak ada firewall yang memutus koneksi idle

**App tidak muncul di tray:**
- Cek hidden tray icons (panah `^` di taskbar)
- Buka `munchii-printer.log` untuk cek error startup

**Log View:**
- File: `munchii-printer.log` di folder yang sama dengan `.exe`
- Auto-cleanup setelah 1 bulan
- Color-coding: 🟢 `OK`, 🔴 `ERROR`, 🟡 `WARN`, 🔵 `INFO`