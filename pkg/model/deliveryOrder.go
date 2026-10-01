package model

import (
	"net/url"
	"time"
)

type InitiateOrderRequest struct {
	OrderID         string   `json:"order_id"`
	GuaranteeImages []string `json:"guarantee_images"`
}

type OrderPassRequest struct {
	OrderID       string `json:"order_id"`
	VehicleNumber string `json:"vehicle_number"`
	PassEntryDate string `json:"pass_entry_date"`
	PassEntryTime string `json:"pass_entry_time"`
	PassExitDate  string `json:"pass_exit_date"`
	PassExitTime  string `json:"pass_exit_time"`
}

// UpdateOrderReturnedAtRequest is the request for POST /updateOrderReturnedAt.
// ReturnedAt accepts "2006-01-02" or RFC3339; if left empty, now is used.
type UpdateOrderReturnedAtRequest struct {
	OrderID    string `json:"order_id"`
	ReturnedAt string `json:"returned_at"`
}

// UpdateOrderItemReturnedAtRequest is the request for
// POST /updateOrderItemReturnedAt. ReturnedAt accepts "2006-01-02" or
// RFC3339; if left empty, now is used.
type UpdateOrderItemReturnedAtRequest struct {
	DeliveryItemID string `json:"delivery_item_id"`
	ReturnedAt     string `json:"returned_at"`
}

type DeliveryOrder struct {
	CustomerID    string `json:"customer_id"`
	InventoryID   string `json:"inventory_id,omitempty"`
	AdvanceAmount int    `json:"advance_amount"`
	Discount      int    `json:"discount"`
	//GeneratedAmount int            `json:"generated_amount"`
	//	PlacedAt        string        `json:"placed_at,omitempty"`
	//ExpiryAt        string         `json:"expiry_at"`
	Status          string                `json:"status,omitempty"`
	Notes           *string               `json:"notes,omitempty"`
	ContactName     string                `json:"contact_name"`
	ContactNumber   string                `json:"contact_number"`
	ShippingAddress string                `json:"shipping_address"`
	Items           []DeliveryItemHandler `json:"items"`
}
type DeliveryItemHandler struct {
	RentAmount      int        `json:"rent_amount"`
	BeforeImages    []string   `json:"before_images,omitempty"`
	AfterImages     []string   `json:"after_images,omitempty"`
	GeneratedAmount int        `json:"generated_amount"`
	CurrentAmount   int        `json:"current_amount"`
	ConditionOut    string     `json:"condition_out,omitempty"`
	ConditionIn     string     `json:"condition_in,omitempty"`
	PlacedAt        *time.Time `json:"placed_at,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
	ReturnedStr     string     `json:"returned_str"`
	Status          string     `json:"status,omitempty"`
	ItemNewId       string     `json:"item_newid"`
}

// OrderDetails is the response for GET /orderDetails/:orderID — the full
// delivery_chelan row (plus customer name/gst) with its line items.
type OrderDetails struct {
	DeliveryID           string      `json:"delivery_id"`
	CustomerID           string      `json:"customer_id"`
	CustomerName         string      `json:"customer_name"`
	CustomerGST          string      `json:"customer_gst"`
	InventoryID          string      `json:"inventory_id"`
	AdvanceAmount        int         `json:"advance_amount"`
	GeneratedAmount      int         `json:"generated_amount"`
	CurrentAmount        int         `json:"current_amount"`
	Discount             int         `json:"discount"`
	ContactName          string      `json:"contact_name"`
	ContactNumber        string      `json:"contact_number"`
	ShippingAddress      string      `json:"shipping_address"`
	TransactionID        string      `json:"transaction_id"`
	OrderNumber          string      `json:"order_number"`
	DeliveryChelanNumber string      `json:"delivery_chelan_number"`
	InvoiceNumber        string      `json:"invoice_number"`
	PlacedAt             *time.Time  `json:"placed_at,omitempty"`
	ExpiryAt             *time.Time  `json:"expiry_at,omitempty"`
	ReturnedAt           *time.Time  `json:"returned_at,omitempty"`
	DeclinedAt           *time.Time  `json:"declined_at,omitempty"`
	Status               string      `json:"status"`
	GuaranteeImages      []string    `json:"guarantee_images"`
	VehicleNumber        string      `json:"vehicle_number"`
	PassEntryDate        string      `json:"pass_entry_date"`
	PassEntryTime        string      `json:"pass_entry_time"`
	PassExitDate         string      `json:"pass_exit_date"`
	PassExitTime         string      `json:"pass_exit_time"`
	OrderItems           []OrderItem `json:"order_items"`
}

// OrderItem is one delivery_items row joined with its item, customer and
// order numbers (the orderDetails query).
type OrderItem struct {
	ItemName             string     `json:"item_name"`
	ItemCode             string     `json:"item_code"`
	ProductCode          string     `json:"product_code"`
	CustomerName         string     `json:"customer_name"`
	CustomerGST          string     `json:"customer_gst"`
	DeliveryChelanNumber string     `json:"delivery_chelan_number"`
	OrderNumber          string     `json:"order_number"`
	DeliveryItemID       string     `json:"delivery_item_id"`
	CustomerID           string     `json:"customer_id"`
	InventoryID          string     `json:"inventory_id"`
	RentAmount           int        `json:"rent_amount"`
	GeneratedAmount      int        `json:"generated_amount"`
	CurrentAmount        int        `json:"current_amount"`
	BeforeImages         []string   `json:"before_images"`
	AfterImages          []string   `json:"after_images"`
	ConditionOut         string     `json:"condition_out"`
	ConditionIn          string     `json:"condition_in"`
	PlacedAt             *time.Time `json:"placed_at,omitempty"`
	ReturnedAt           *time.Time `json:"returned_at,omitempty"`
	DeclinedAt           *time.Time `json:"declined_at,omitempty"`
	Status               string     `json:"status"`
	OrderID              string     `json:"order_id"`
	ItemID               string     `json:"item_id"`
	Damage               bool       `json:"damage"`
}

func (s *DeliveryOrder) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "INITIATED" || s.Status == "DECLINED" || s.Status == "DECLINED" || s.Status == "RESERVED") {
		err.Add("Status", "Enter valid status")
	}
	return err
}
