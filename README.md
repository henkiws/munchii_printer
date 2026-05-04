# Kypesen Printer - Windows Tray App

Aplikasi Windows untuk polling order dari API Kypesen dan print otomatis ke printer ESC/POS via jaringan (TCP port 9100).

## Fitur
- 🖥️ Berjalan di system tray Windows (tidak ada jendela)
- 🔄 Auto-start saat Windows login (bisa toggle dari tray menu)
- ⏱️ Polling tiap X detik per printer (default 5 detik, bisa custom)
- 🖨️ Support banyak printer sekaligus
- ⚙️ UI manage printer (tambah/hapus) dari tray icon
- 📋 View log aktivitas dari tray icon
- 🔁 Fallback ke mode text jika wkhtmltoimage tidak tersedia

## Cara Build

### Requirement
- [Go 1.21+](https://go.dev/dl/) — hanya untuk development/build
- Windows 10/11

### Langkah Build
1. Install Go dari https://go.dev/dl/
2. Buka folder project ini
3. Double-click `build.bat`
4. File `kypesen-printer.exe` akan terbuat

## Cara Deploy ke PC Target

### Yang perlu dikopi ke PC target:
```
📁 folder bebas/
├── kypesen-printer.exe    ← hasil build
├── wkhtmltoimage.exe      ← download dari wkhtmltopdf.org
└── printers.json          ← auto-dibuat saat pertama jalan
```

### Download wkhtmltoimage
- Pergi ke https://wkhtmltopdf.org/downloads.html
- Download versi Windows 64-bit
- Extract, ambil file `wkhtmltoimage.exe`
- Taruh di folder yang sama dengan `kypesen-printer.exe`

> **Catatan:** PC target TIDAK perlu install Go, PHP, Laravel, atau apapun.
> Cukup 2 file `.exe` di atas.

## Cara Pakai

### Pertama kali jalankan
1. Double-click `kypesen-printer.exe`
2. Ikon printer akan muncul di system tray (pojok kanan bawah)
3. Klik kanan → **Manage Printers** → Tambah printer

### Tambah Printer
Isi form:
| Field | Contoh |
|---|---|
| Printer Name | `Cashier 1` |
| Printer IP Address | `192.168.1.100` |
| Kypesen API URL | `http://192.168.1.x:8001/api/v1/print/UUID` |
| Polling (seconds) | `5` |

### Auto-start Windows
Klik kanan tray icon → centang **Auto-start with Windows**

### Melihat Log
Klik kanan tray icon → **View Log**

Log juga tersimpan di file `kypesen-printer.log` di folder yang sama.

## Struktur File

```
kypesen-printer/
├── main.go              ← entry point + tray menu
├── config.go            ← baca/tulis printers.json
├── kypesen.go           ← hit API + struct response
├── printer.go           ← ESC/POS printing + wkhtmltoimage
├── polling.go           ← goroutine polling per printer
├── ui.go                ← manage window (PowerShell Forms)
├── logger.go            ← logging
├── icon.go              ← tray icon bytes
├── autostart_windows.go ← Windows registry auto-start
├── autostart_stub.go    ← stub untuk non-Windows
├── windows_helpers.go   ← Windows exec helpers
├── windows_helpers_stub.go
├── go.mod
└── build.bat
```

## Format printers.json

```json
[
    {
        "id": 0,
        "printer_name": "Cashier 1",
        "printer_conn": "local_network",
        "printer_ip_address": "192.168.1.100",
        "server_url": "http://192.168.1.x:8001/api/v1/print/UUID-DISINI",
        "polling_time": 5
    }
]
```

`polling_time` bisa diisi `null` untuk pakai default 5 detik.

## Troubleshooting

**Printer tidak print:**
- Pastikan IP printer benar dan bisa di-ping dari PC
- Pastikan port 9100 tidak diblok firewall
- Cek log di tray icon → View Log

**wkhtmltoimage error:**
- Pastikan `wkhtmltoimage.exe` ada di folder yang sama
- App akan fallback ke mode text ESC/POS jika wkhtmltoimage gagal

**App tidak muncul di tray:**
- Cek di hidden tray icons (panah ^ di taskbar)
- Cek `kypesen-printer.log` untuk error
