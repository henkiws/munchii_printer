package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ── Response structures ──────────────────────────────────────────────────────

type KypesenResponse struct {
	Response KypesenData `json:"response"`
}

type KypesenData struct {
	Data KypesenInner `json:"data"`
}

type KypesenInner struct {
	Printer PrinterInfo   `json:"printer"`
	Orders  []OrderWrapper `json:"orders"`
}

type PrinterInfo struct {
	ID           int          `json:"id"`
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

type OrderWrapper struct {
	FkPrinter  int   `json:"fk_printer"`
	PrintCount int   `json:"print_count"`
	OnlyOrder  int   `json:"only_order"`
	Order      Order `json:"order"`
}

type Order struct {
	ID              int        `json:"id"`
	FkTableCod      *int       `json:"fk_table_cod"`
	Name            string     `json:"name"`
	CreatedAt       string     `json:"created_at"`
	DeliveryMethod  int        `json:"delivery_method"`
	PaymentMethod   string     `json:"payment_method"`
	OrderPrice      float64    `json:"order_price"`
	DeliveryPrice   float64    `json:"delivery_price"`
	TotalRefundPrice float64   `json:"total_refund_price"`
	OrderPriceCancel float64   `json:"order_price_cancel"`
	SurchargeValue  float64    `json:"surcharge_value"`
	DiscountValue   float64    `json:"discount_value"`
	Surcharge       float64    `json:"surcharge"`
	Vatvalue        float64    `json:"vatvalue"`
	Discount        float64    `json:"discount"`
	Restorant       Restaurant `json:"restorant"`
	Table           *TableInfo `json:"table"`
	TableCod        *TableCod  `json:"table_cod"`
	Items           []Item     `json:"items"`
	ItemsCod        []Item     `json:"items_cod"`
}

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
	Name         string      `json:"name"`
	Price        float64     `json:"price"`
	Pivot        ItemPivot   `json:"pivot"`
	Options      []ItemOption `json:"options"`
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

// ── API call ─────────────────────────────────────────────────────────────────

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func fetchKypesen(serverURL string) (*KypesenResponse, error) {
	resp, err := httpClient.Get(serverURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var result KypesenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &result, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (o Order) GetReceiptNumber() string {
	if o.FkTableCod != nil {
		return fmt.Sprintf("%d", *o.FkTableCod)
	}
	return fmt.Sprintf("%d", o.ID)
}

func (o Order) GetCustomer() string {
	if o.Name == "" {
		return "Guest"
	}
	return o.Name
}

func (o Order) GetTableArea() string {
	if o.Table != nil && o.Table.Restoarea != nil {
		return o.Table.Restoarea.Name
	}
	return "N/A"
}

func (o Order) GetTableName() string {
	if o.Table != nil {
		return o.Table.Name
	}
	return "N/A"
}

func (o Order) GetOrderType() string {
	switch o.DeliveryMethod {
	case 3:
		return "Dine-In"
	case 2:
		return "Pickup"
	default:
		return "Delivery"
	}
}

func (o Order) GetTotal() float64 {
	return o.DeliveryPrice + o.OrderPrice - o.TotalRefundPrice - o.OrderPriceCancel + o.SurchargeValue - o.DiscountValue
}

func (o Order) GetSubtotal() float64 {
	return o.OrderPrice - o.Vatvalue - o.TotalRefundPrice
}
