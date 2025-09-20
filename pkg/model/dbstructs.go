package model

import (
	"time"

	"github.com/google/uuid"
)

// Users
type User struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Designation string    `json:"designation"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
}

// Inventory (renamed from site)
type Inventory struct {
	InventoryID uuid.UUID `json:"inventory_id"`
	SiteName    string    `json:"sitename"`
	Status      string    `json:"status,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Customer
type Customer struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	Name         string    `json:"name"`
	ShortName    string    `json:"short_name"`
	Phone        string    `json:"phone"`
	Type         string    `json:"type"`
	GST          string    `json:"gst"`
	Address      string    `json:"address"`
	Email        string    `json:"email,omitempty"`
	CustomerFlag string    `json:"customer_flag,omitempty"`
	Status       string    `json:"status,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Attribute
type Attribute struct {
	AttributesID uuid.UUID `json:"attributes_id"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	Status       string    `json:"status,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DeliveryChelan (keeps original name delivery_chelan)
type DeliveryChelan struct {
	DeliveryID      uuid.UUID  `json:"delivery_id"`
	CustomerID      string     `json:"customer_id"`
	InventoryID     string     `json:"inventory_id"`
	AdvanceAmount   int        `json:"advance_amount"`
	GeneratedAmount int        `json:"generaed_amount"`
	CurrentAmount   int        `json:"current_amount"`
	ContactName     string     `json:"contact_name"`
	ContactNumber   string     `json:"contact_number"`
	ShippingAddress string     `json:"shipping_address"`
	PlacedAt        time.Time  `json:"placed_at"`
	ExpiryAt        *time.Time `json:"expiry_at,omitempty"`
	DeclinedAt      *time.Time `json:"declined_at,omitempty"`
	Status          string     `json:"status,omitempty"`
}

// DeliveryItem
type DeliveryItem struct {
	DeliveryItemID  string     `json:"delivery_item_id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	InventoryID     uuid.UUID  `json:"inventory_id"`
	RentAmount      int        `json:"rent_amount"`
	GeneratedAmount int        `json:"generated_amount"`
	CurrentAmount   int        `json:"current_amount"`
	BeforeImages    []string   `json:"before_images,omitempty"`
	AfterImages     []string   `json:"after_images,omitempty"`
	ConditionOut    string     `json:"condition_out,omitempty"`
	ConditionIn     string     `json:"condition_in,omitempty"`
	PlacedAt        *time.Time `json:"placed_at,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
	ReturnedStr     string     `json:"returned_str"`
	DeclinedAt      *time.Time `json:"declined_at,omitempty"`
	Status          string     `json:"status,omitempty"`
	ItemID          string     `json:"item_id,omitempty"`
}

// Item
type Item struct {
	ItemID       uuid.UUID  `json:"item_id"`
	ItemCode     string     `json:"item_code"`
	SubCode      string     `json:"sub_code"`
	ItemName     string     `json:"item_name"`
	ItemMainType string     `json:"item_main_type"`
	ItemSubType  string     `json:"item_sub_type,omitempty"`
	Brand        string     `json:"brand,omitempty"`
	Category     string     `json:"category,omitempty"`
	Description  string     `json:"description,omitempty"`
	InventoryID  *uuid.UUID `json:"inventory_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ItemStatusHistory
type ItemStatusHistory struct {
	HistoryID  uuid.UUID  `json:"history_id"`
	ItemID     uuid.UUID  `json:"item_id"`
	OldStatus  string     `json:"old_status,omitempty"`
	NewStatus  string     `json:"new_status,omitempty"`
	ChangedAt  time.Time  `json:"changed_at"`
	ChangedBy  *uuid.UUID `json:"changed_by,omitempty"`
	DeliveryID *uuid.UUID `json:"delivery_id,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}
