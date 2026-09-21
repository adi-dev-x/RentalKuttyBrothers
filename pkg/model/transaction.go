package model

import (
	"net/url"
)

type UserTransactions struct {
	Id     string  `json:"id"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"`
	Des    string  `json:"description"`
}

// Transaction is the AddSubTransaction request/record. It no longer relates
// to main_transaction — it now stands on its own, linked to its order
// directly via OrderID (the transaction table's pre-existing, previously
// unused order_id column). InvoiceID is never sent by the client: the
// server always generates it. See service.AddSubTransaction.
type Transaction struct {
	ID        int    `json:"id"`
	OrderID   string `json:"order_id"`
	InvoiceID string `json:"invoice_id"`
	Amount    int    `json:"amount"`
	Image     string `json:"image"`
	Status    string `json:"status"` // PENDING, COMPLETED, FAILED
	Type      string `json:"type"`
	FromDate  string `json:"from_date"`
	ToDate    string `json:"to_date"`
	// InvoiceType is stored in the invoice_type DB column; the json tag is
	// kept as "tax_type" for wire compatibility with existing clients.
	InvoiceType string `json:"invoice_type"`
}

func (s *Transaction) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "PENDING" || s.Status == "COMPLETED" || s.Status == "FAILED") {
		err.Add("Status", "Enter valid status")
	}
	if s.Amount == 0 || s.Amount < 0 {
		err.Add("Amount", "greater than zero")
	}
	if s.OrderID == "" {
		err.Add("OrderID", "give a valid value")
	}

	return err
}
