package model

import "time"

type QuotationItem struct {
	QuotationItemID string    `json:"quotation_item_id"`
	QuotationID     string    `json:"quotation_id"`
	ItemName        string    `json:"item_name"`
	Description     string    `json:"description"`
	RentAmount      int       `json:"rent_amount"`
	StartDate       string    `json:"start_date"`
	EndDate         string    `json:"end_date"`
	CreatedAt       time.Time `json:"created_at"`
}

type Quotation struct {
	QuotationID     string          `json:"quotation_id"`
	CustomerID      string          `json:"customer_id"`
	ContactName     string          `json:"contact_name"`
	ContactNumber   string          `json:"contact_number"`
	ShippingAddress string          `json:"shipping_address"`
	QuotationNumber string          `json:"quotation_number"`
	TotalAmount     int             `json:"total_amount"`
	PlacedAt        time.Time       `json:"placed_at"`
	StartDate       string          `json:"start_date"`
	ReturnDate      string          `json:"return_date"`
	Items           []QuotationItem `json:"items"`
}

type QuotationItemRequest struct {
	ItemName    string `json:"item_name"`
	Description string `json:"description"`
	RentAmount  int    `json:"rent_amount"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type QuotationRequest struct {
	CustomerID      string                 `json:"customer_id"`
	ContactName     string                 `json:"contact_name"`
	ContactNumber   string                 `json:"contact_number"`
	ShippingAddress string                 `json:"shipping_address"`
	Items           []QuotationItemRequest `json:"items"`
}
