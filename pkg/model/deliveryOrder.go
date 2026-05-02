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

func (s *DeliveryOrder) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "INITIATED" || s.Status == "DECLINED" || s.Status == "DECLINED" || s.Status == "RESERVED") {
		err.Add("Status", "Enter valid status")
	}
	return err
}
