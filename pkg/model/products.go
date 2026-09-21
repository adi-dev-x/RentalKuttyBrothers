package model

import (
	"github.com/google/uuid"
	"net/url"
)

// Product is the AddProduct request. There is no product_code field: the
// server resolves it from Name — reusing the existing code if this product
// name has been added before, or assigning the next IRR#### code if not.
// See service.resolveProductCode.
type Product struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Unit     int    `json:"units"`
	HsnCode  string `json:"hsn_code"`

	Status       string     `json:"status"`
	Description  string     `json:"description"`
	Brand        string     `json:"brand"`
	InventoryID  *uuid.UUID `json:"inventory_id,omitempty"`
	Year         string     `json:"year,omitempty"`
	ItemMainType string     `json:"item_main_type"`
	NewSubCode   string     `json:"new_sub_code"`
}

// AddProductResult is what AddProduct hands back: the product_code that was
// resolved (reused or newly assigned) and the item_code of every unit it
// just created under that product_code.
type AddProductResult struct {
	ProductCode string   `json:"product_code"`
	ItemCodes   []string `json:"item_codes"`
}

// ProductStockInfo is the shared, product-level info for a product_code —
// item_name/brand/description/hsn_code are identical across every unit of a
// product, so this is looked up from a single row. Used by
// GET /listStockCnt/:product_code.
type ProductStockInfo struct {
	ItemName    string `json:"item_name"`
	Brand       string `json:"brand"`
	ProductCode string `json:"product_code"`
	Description string `json:"description"`
	HsnCode     string `json:"hsn_code"`
}

// ItemSheetRow pairs a parsed /uploadItemSheet row's Product with its
// 1-indexed row number in the sheet (header is row 1), so results and
// errors can be reported against the row the user will actually see.
type ItemSheetRow struct {
	Row     int     `json:"row"`
	Product Product `json:"product"`
}

// UploadItemSheetRowResult is what happened for one row of an uploaded item
// sheet — the product_code it resolved to (reused or newly assigned) and the
// item_code of every unit that row added.
type UploadItemSheetRowResult struct {
	Row         int      `json:"row"`
	ItemName    string   `json:"item_name"`
	ProductCode string   `json:"product_code"`
	ItemCodes   []string `json:"item_codes"`
}

// UploadItemSheetResult is the response for POST /uploadItemSheet.
type UploadItemSheetResult struct {
	RowsProcessed int                        `json:"rows_processed"`
	ItemsCreated  int                        `json:"items_created"`
	Results       []UploadItemSheetRowResult `json:"results"`
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

	if !(u.Category == "RESERVED" || u.Category == "RENTED" || u.Category == "DAMAGED" || u.Category == "AVAILABLE" || u.Category == "WORN_OUT" || u.Category == "REPAIRING" || u.Category == "EXPIRED" || u.Category == "SITE") {
		err.Add("CATEGORY", "Give valid Category")
	}

	// if u.Description == "" {
	// 	err.Add("description", "Description is required")
	// }

	return err
}

type OrderItemUpdate struct {
	ItemID     string `json:"delivery_item_id"`
	Status     string `json:"status"`
	AfterImage string `json:"after_images"`
}

func (u *OrderItemUpdate) Valid() url.Values {
	err := url.Values{}
	if u.ItemID == "" {
		err.Add("item_id", "ItemId is required")
	}
	if !(u.Status == "BLOCKED" || u.Status == "COMPLETED" || u.Status == "INITIATED") {
		err.Add("item_id", "ItemId is required")
	}
	return err
}
