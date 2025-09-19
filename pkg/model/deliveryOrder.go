package model

import "net/url"

type DeliveryOrder struct {
	CustomerID    string `json:"customer_id"`
	InventoryID   string `json:"inventory_id,omitempty"`
	AdvanceAmount int    `json:"advance_amount"`
	//GeneratedAmount int            `json:"generated_amount"`
	//	PlacedAt        string        `json:"placed_at,omitempty"`
	//ExpiryAt        string         `json:"expiry_at"`
	Status          string         `json:"status,omitempty"`
	Notes           *string        `json:"notes,omitempty"`
	ContactName     string         `json:"contact_name"`
	ContactNumber   string         `json:"contact_number"`
	ShippingAddress string         `json:"shipping_address"`
	Items           []DeliveryItem `json:"items"`
}

func (s *DeliveryOrder) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "NOT_INITIATED" || s.Status == "INITIATED" || s.Status == "DECLINED" || s.Status == "DECLINED") {
		err.Add("Status", "Enter valid status")
	}
	return err
}
