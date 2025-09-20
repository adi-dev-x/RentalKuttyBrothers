package model

import (
	"github.com/google/uuid"
	"net/url"
)

type Product struct {
	Name         string     `json:"name"`
	MainCode     string     `json:"main_code"`
	SubCode      string     `json:"sub_code"`
	Category     string     `json:"category"`
	Unit         int        `json:"units"`
	Price        float64    `json:"amount"`
	Status       string     `json:"status"`
	Description  string     `json:"description"`
	Brand        string     `json:"brand"`
	InventoryID  *uuid.UUID `json:"inventory_id,omitempty"`
	Year         string     `json:"year,omitempty"`
	ItemMainType string     `json:"item_main_type"`
	NewSubCode   string     `json:"new_sub_code"`
}
type UpdateProduct struct {
	Unit  float64 `json:"units"`
	Tax   float64 `json:"tax"`
	Price float64 `json:"amount"`

	Status          bool    `json:"status"`
	Discount        float64 `json:"discount"`
	Description     string  `json:"description"`
	Pid             string  `json:"pid"`
	ClearDiscount   string  `json:"clrdis"`
	ClearUnit       string  `json:"clrunit"`
	ClProductStatus string  `json:"p_status"`
}

func (u *UpdateProduct) Valid() url.Values {
	err := url.Values{}
	if !(u.ClearDiscount == "Yes" || u.ClearDiscount == "No") {
		err.Add("ClearDiscount", "Should be Valid Yes or No")

	}
	if !(u.ClearUnit == "Yes" || u.ClearUnit == "No") {
		err.Add("ClearUnit", "Should be Valid Yes or No")

	}
	if !(u.ClProductStatus == "Yes" || u.ClProductStatus == "No") {
		err.Add("Clear ProductStatus", "Should be Valid Yes or No")

	}
	if u.Pid == "" {
		err.Add("Product id", "Product id id null")
	}
	return err
}

type ProductList struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Unit       float64 `json:"units"`
	Tax        float64 `json:"tax"`
	Price      float64 `json:"amount"`
	VendorName string  `json:"vendorName"`
	Status     bool    `json:"status"`
	Discount   float64 `json:"discount"`
}

type VendorProductList struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Unit     float64 `json:"units"`
	Tax      float64 `json:"tax"`
	Price    float64 `json:"amount"`

	Status    bool    `json:"status"`
	Discount  float64 `json:"discount"`
	TotalSold int     `json:"total_sold"`
}
type ProductListUsers struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Unit       float64 `json:"units"`
	Tax        float64 `json:"tax"`
	Price      float64 `json:"product_price"`
	VendorName string  `json:"vendorName"`
	Status     bool    `json:"status"`
	Discount   float64 `json:"discount"`
	Pid        string  `json:"pid"`
}
type ProductListingUsers struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Unit     float64 `json:"units"`
	Tax      float64 `json:"tax"`
	Price    float64 `json:"product_price"`
	Status   bool    `json:"status"`
	Discount float64 `json:"discount"`
	Pid      string  `json:"pid"`
	Pdetail  string  `json:"pdetail"`
}
type ProductListDetailed struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Unit       float64 `json:"units"`
	Tax        float64 `json:"tax"`
	Price      float64 `json:"product_price"`
	VendorName string  `json:"vendorName"`
	Status     bool    `json:"status"`
	Discount   float64 `json:"discount"`
	Pid        string  `json:"pid"`
	VEmail     string  `json:"vendorEmail"`
	VGst       string  `json:"vendorgst"`
	VId        string  `json:"vendorid"`
	Pds        string  `json:"pds"`
	Brand      string  `json:"brand"`
}

func (u *Product) Valid() url.Values {
	err := url.Values{}

	if u.Name == "" {
		err.Add("name", "Name is required")
	}
	if u.Category == "" {
		err.Add("category", "Category is required")
	}
	if u.Unit <= 0 {
		err.Add("unit", "Unit must be greater than zero")
	}
	if u.Price <= 0 {

		err.Add("price", "Price must be greater than zero")
	}
	if !(u.Category == "NOT_INITIATED" || u.Category == "RENTED" || u.Category == "DAMAGED" || u.Category == "AVAILABLE" || u.Category == "WORN_OUT") {
		err.Add("CATEGORY", "Give valid Category")
	}

	if u.Description == "" {
		err.Add("description", "Description is required")
	}

	return err
}
