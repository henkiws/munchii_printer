package main

// ── Payload dari Go Hub (dikirim Laravel setelah generate image) ──────────────
//
// Format JSON yang diterima via WebSocket:
// {
//   "status":     "success",
//   "from":       "order" | "note" | "reporting",
//   "client_id":  "UUID-PRINTER",
//   "ip_address": "192.168.1.100",
//   "data": {
//     "printer": { ... },
//     "orders":  [ { "order": { ..., "image_base64": "data:image/png;base64,..." } } ]
//     "notes":   [ { ..., "image_base64": "..." } ]
//     "reports": [ { ..., "image_base64": "..." } ]
//   }
// }

// HubPayload — root payload dari Go Hub
type HubPayload struct {
	Status    string      `json:"status"`
	From      string      `json:"from"`       // "order" | "note" | "reporting"
	ClientID  string      `json:"client_id"`  // UUID printer
	IPAddress string      `json:"ip_address"` // IP printer fisik
	Data      HubData     `json:"data"`
}

// HubData — isi data tergantung from
type HubData struct {
	Printer PrinterInfo    `json:"printer"`
	Orders  []OrderWrapper `json:"orders"`  // from=order
	Notes   []NoteItem     `json:"notes"`   // from=note
	Reports []ReportItem   `json:"reports"` // from=reporting
}

// ── Printer info ──────────────────────────────────────────────────────────────

type PrinterInfo struct {
	ID           int          `json:"id"`
	IPAddress    string       `json:"ip_address"`
	PrinterGroup PrinterGroup `json:"printer_group"`
}

type PrinterGroup struct {
	ID         int        `json:"id"`
	PrinterJob PrinterJob `json:"printer_job"`
}

type PrinterJob struct {
	PaperSize       int             `json:"paper_size"`
	PrinterTemplate PrinterTemplate `json:"printer_template"`
}

type PrinterTemplate struct {
	Name string `json:"name"`
}

// ── Order ─────────────────────────────────────────────────────────────────────

type OrderWrapper struct {
	FkPrinter  int   `json:"fk_printer"`
	PrintCount int   `json:"print_count"`
	OnlyOrder  int   `json:"only_order"`
	OnlyVoid   int   `json:"only_void"`
	Order      Order `json:"order"`
}

type Order struct {
	ID               int        `json:"id"`
	Ref              string     `json:"ref"`
	FkTableCod       *int       `json:"fk_table_cod"`
	Name             string     `json:"name"`
	CreatedAt        string     `json:"created_at"`
	DeliveryMethod   int        `json:"delivery_method"`
	PaymentMethod    string     `json:"payment_method"`
	OrderPrice       float64    `json:"order_price"`
	DeliveryPrice    float64    `json:"delivery_price"`
	TotalRefundPrice float64    `json:"total_refund_price"`
	OrderPriceCancel float64    `json:"order_price_cancel"`
	SurchargeValue   float64    `json:"surcharge_value"`
	DiscountValue    float64    `json:"discount_value"`
	Surcharge        float64    `json:"surcharge"`
	Vatvalue         float64    `json:"vatvalue"`
	Discount         float64    `json:"discount"`
	Restorant        Restaurant `json:"restorant"`
	Table            *TableInfo `json:"table"`
	TableCod         *TableCod  `json:"table_cod"`
	Items            []Item     `json:"items"`
	ItemsCod         []Item     `json:"items_cod"`

	// Disisipkan oleh Laravel setelah wkhtmltoimage sukses
	ImageBase64 string `json:"image_base64"`
}

// ── Note ──────────────────────────────────────────────────────────────────────

type NoteItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	FkPrinter    int    `json:"fk_printer"`
	PrintAttempt int    `json:"print_attempt"`
	PrintCount   int    `json:"print_count"`
	ImageBase64  string `json:"image_base64"`
	ImgPrint     string `json:"img_print"`
}

// ── Report ────────────────────────────────────────────────────────────────────

type ReportItem struct {
	ID           int    `json:"id"`
	Type         string `json:"type"`
	FkCompany    int    `json:"fk_company"`
	PrintAttempt int    `json:"print_attempt"`
	PrintCount   int    `json:"print_count"`
	ImageBase64  string `json:"image_base64"`
	ImgPrint     string `json:"img_print"`
}

// ── Supporting types ──────────────────────────────────────────────────────────

type Restaurant struct {
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	Phone        string  `json:"phone"`
	Currency     string  `json:"currency"`
	InclusiveTax float64 `json:"inclusive_tax"`
}

type TableInfo struct {
	Name      string    `json:"name"`
	Restoarea *AreaInfo `json:"restoarea"`
}

type AreaInfo struct {
	Name string `json:"name"`
}

type TableCod struct {
	Name       string `json:"name"`
	ManyPeople int    `json:"many_people"`
}

type Item struct {
	Name         string            `json:"name"`
	Price        float64           `json:"price"`
	Pivot        ItemPivot         `json:"pivot"`
	Options      []ItemOption      `json:"options"`
	PrinterGroup *ItemPrinterGroup `json:"printer_group"`
}

type ItemPivot struct {
	Qty          int     `json:"qty"`
	VariantPrice float64 `json:"variant_price"`
	VariantName  string  `json:"variant_name"`
	Modifiers    string  `json:"modifiers"`
	Upsell       string  `json:"upsell"`
	Extras       string  `json:"extras"`
}

type ItemOption struct {
	Name string `json:"name"`
}

type ItemPrinterGroup struct {
	DataPrinterGroup *DataPrinterGroup `json:"data_printer_group"`
}

type DataPrinterGroup struct {
	ID int `json:"id"`
}