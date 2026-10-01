package model

import (
	"net/url"
	"time"
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
//
// Amount is no longer a client-supplied value: it's derived as
// DailyAmount × the number of days between FromDate and ToDate, both on
// create (AddSubTransaction) and on read (GetSubTransactions), so the two
// can never drift apart. See TransactionDays.
type Transaction struct {
	ID          int    `json:"id"`
	OrderID     string `json:"order_id"`
	InvoiceID   string `json:"invoice_id"`
	DailyAmount int    `json:"daily_amount"`
	Amount      int    `json:"amount"`
	Image       string `json:"image"`
	Status      string `json:"status"` // PENDING, COMPLETED, FAILED
	Type        string `json:"type"`
	FromDate    string `json:"from_date"`
	ToDate      string `json:"to_date"`
	InvoiceType string `json:"invoice_type"`
}

func (s *Transaction) Valid() url.Values {

	err := url.Values{}
	if !(s.Status == "PENDING" || s.Status == "COMPLETED" || s.Status == "FAILED") {
		err.Add("Status", "Enter valid status")
	}
	if s.OrderID == "" {
		err.Add("OrderID", "give a valid value")
	}
	// if s.DailyAmount <= 0 {
	// 	err.Add("DailyAmount", "give a value greater than zero")
	// }
	if s.FromDate == "" || s.ToDate == "" {
		err.Add("FromDate", "from_date and to_date are both required (YYYY-MM-DD)")
	} else if _, ok := TransactionDays(s.FromDate, s.ToDate); !ok {
		err.Add("FromDate", "to_date must be a valid date after from_date (YYYY-MM-DD)")
	}

	return err
}

// TransactionDays returns the whole number of days between fromDate and
// toDate (both "2006-01-02"), and whether the range is valid — both parse,
// and toDate is strictly after fromDate. Used to derive Transaction.Amount
// from DailyAmount; mirrors the day×rate convention already used elsewhere
// (quotation item totals, the nightly current_amount recalculation).
func TransactionDays(fromDate, toDate string) (int, bool) {
	const layout = "2006-01-02"
	start, err1 := time.Parse(layout, fromDate)
	end, err2 := time.Parse(layout, toDate)
	if err1 != nil || err2 != nil || !end.After(start) {
		return 0, false
	}
	return int(end.Sub(start).Hours() / 24), true
}
