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
type Transaction struct {
	ID                int    `json:"id"`
	MainTransactionID int    `json:"main_transaction_id"`
	Amount            int    `json:"amount"`
	Image             string `json:"image"`
	Status            string `json:"status"` // PENDING, COMPLETED, FAILED
	Type              string `json:"type"`
}

func (s *Transaction) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "PENDING" || s.Status == "COMPLETED" || s.Status == "FAILED") {
		err.Add("Status", "Enter valid status")
	}
	if s.Amount == 0 || s.Amount < 0 {
		err.Add("Amount", "greater than zero")
	}
	if s.MainTransactionID == 0 || s.MainTransactionID < 0 {
		err.Add("MainTransactionID", "give a valid value")
	}

	return err
}
