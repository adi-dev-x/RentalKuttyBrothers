package irrl

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"myproject/pkg/model"

	"github.com/labstack/echo/v4"
)

// Read-only reports for the ai-chatbot service. The chatbot has no database access: every
// number it speaks comes from these endpoints, so the app's own rules apply to it.

const chatReportMaxRows = 20

// ist is India Standard Time; a fixed zone avoids depending on tzdata in the container.
var ist = time.FixedZone("IST", 5*60*60+30*60)

// reportInputError marks a bad request parameter, answered with 400 instead of 500.
type reportInputError struct{ msg string }

func (e reportInputError) Error() string { return e.msg }

func (h *Handler) mountChatReports(engine *echo.Echo) {
	reports := engine.Group("/api/irrl/chat-reports", h.chatbotKeyAuth)

	reports.GET("/snapshot", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetBusinessSnapshot(c.QueryParam("from"), c.QueryParam("to"))
	}))
	reports.GET("/income-by-equipment", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetIncomeByEquipment(c.QueryParam("from"), c.QueryParam("to"), c.QueryParam("order") == "lowest")
	}))
	reports.GET("/fleet-status", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetFleetStatus()
	}))
	reports.GET("/idle-equipment", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetIdleEquipment(queryDays(c, "days", 30))
	}))
	reports.GET("/receivables", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetReceivables(queryDays(c, "min_age_days", 0))
	}))
	reports.GET("/customer-payment-history", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetCustomerPaymentHistory(c.QueryParam("name"), queryLimit(c))
	}))
	reports.GET("/largest-orders", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetLargestOrders(queryLimit(c))
	}))
	reports.GET("/rental-returns", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetRentalReturns(queryDays(c, "days", 0))
	}))
	reports.GET("/customer-activity", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetCustomerActivity(c.QueryParam("sort"))
	}))
	reports.GET("/repair-vs-income", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetRepairVsIncome(c.QueryParam("from"), c.QueryParam("to"))
	}))
	reports.GET("/revenue-trend", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetRevenueTrend(c.QueryParam("period"), queryLimit(c))
	}))
	reports.GET("/order-status-summary", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetOrderStatusSummary()
	}))
	reports.GET("/customer-damage", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetCustomerDamageAnalytics()
	}))
	reports.GET("/customer-details", h.chatReport(func(c echo.Context) (interface{}, error) {
		return h.service.GetCustomerDetails(c.QueryParam("name"), queryLimit(c))
	}))
}

// chatbotKeyAuth only lets through requests carrying the shared X-Chatbot-Key.
func (h *Handler) chatbotKeyAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := h.cnf.ChatbotAPIKey
		if key == "" {
			return h.respondWithError(c, http.StatusServiceUnavailable, "chat reports are disabled: CHATBOT_API_KEY is not set")
		}
		given := c.Request().Header.Get("X-Chatbot-Key")
		if subtle.ConstantTimeCompare([]byte(given), []byte(key)) != 1 {
			return h.respondWithError(c, http.StatusUnauthorized, "invalid chatbot key")
		}
		return next(c)
	}
}

func (h *Handler) chatReport(run func(c echo.Context) (interface{}, error)) echo.HandlerFunc {
	return func(c echo.Context) error {
		data, err := run(c)
		if err != nil {
			var inputErr reportInputError
			if errors.As(err, &inputErr) {
				return h.respondWithError(c, http.StatusBadRequest, err.Error())
			}
			return h.respondWithError(c, http.StatusInternalServerError, err.Error())
		}
		return h.respondWithData(c, http.StatusOK, "success", data)
	}
}

func queryLimit(c echo.Context) int {
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > chatReportMaxRows {
		return chatReportMaxRows
	}
	return limit
}

func queryDays(c echo.Context, name string, fallback int) int {
	days, err := strconv.Atoi(c.QueryParam(name))
	if err != nil || days < 0 {
		return fallback
	}
	if days > 3650 {
		return 3650
	}
	return days
}

// parsePeriod validates an inclusive YYYY-MM-DD range and returns it as [from, to) with
// `to` capped at tomorrow, so periods that run into the future only count days so far.
func parsePeriod(from, to string) (string, string, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return "", "", reportInputError{fmt.Sprintf("from must be YYYY-MM-DD, got %q", from)}
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return "", "", reportInputError{fmt.Sprintf("to must be YYYY-MM-DD, got %q", to)}
	}
	end = end.AddDate(0, 0, 1)

	tomorrow := todayIST().AddDate(0, 0, 1)
	if end.After(tomorrow) {
		end = tomorrow
	}
	if !end.After(start) {
		return "", "", reportInputError{fmt.Sprintf("the period %s to %s has no days up to today", from, to)}
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

// todayIST is today's date in India, as midnight UTC so it compares cleanly with dates.
func todayIST() time.Time {
	now := time.Now().In(ist)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// daysSince counts whole calendar days from t's date to today.
func daysSince(t time.Time) int {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return int(todayIST().Sub(day).Hours() / 24)
}

func (s *service) GetBusinessSnapshot(from, to string) (map[string]interface{}, error) {
	start, end, err := parsePeriod(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.GetBusinessSnapshot(start, end)
	if err != nil {
		return nil, err
	}
	balances, err := s.orderBalances()
	if err != nil {
		return nil, err
	}

	snapshot := map[string]interface{}{"period_from": from, "period_to": to}
	if len(rows) > 0 {
		for k, v := range rows[0] {
			snapshot[k] = v
		}
	}
	owed, owing := 0, map[string]bool{}
	for _, o := range balances {
		if o.Owed > 0 {
			owed += o.Owed
			owing[o.CustomerID] = true
		}
	}
	snapshot["total_owed_by_customers_now"] = owed
	snapshot["customers_owing_now"] = len(owing)
	return snapshot, nil
}

func (s *service) GetIncomeByEquipment(from, to string, ascending bool) ([]map[string]interface{}, error) {
	start, end, err := parsePeriod(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.GetIncomeByEquipment(start, end, ascending)
}

func (s *service) GetRepairVsIncome(from, to string) ([]map[string]interface{}, error) {
	start, end, err := parsePeriod(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.GetRepairVsIncome(start, end)
}

func (s *service) GetFleetStatus() ([]map[string]interface{}, error) {
	return s.repo.GetFleetStatus()
}

func (s *service) GetIdleEquipment(minDays int) ([]map[string]interface{}, error) {
	return s.repo.GetIdleEquipment(minDays)
}

func (s *service) GetRentalReturns(dueWithinDays int) ([]map[string]interface{}, error) {
	return s.repo.GetRentalReturns(dueWithinDays)
}

// orderBalances returns every live order with what has been paid and what is still owed.
// Invoice amounts come from GetSubTransactions, so its daily_amount × days rule applies.
func (s *service) orderBalances() ([]model.OrderBalance, error) {
	orders, err := s.repo.GetOrderBalanceBase()
	if err != nil {
		return nil, err
	}
	invoices, err := s.repo.GetSubTransactions("")
	if err != nil {
		return nil, err
	}

	paid, pending := map[string]int{}, map[string]int{}
	for _, t := range invoices {
		switch t.Status {
		case "COMPLETED":
			paid[t.OrderID] += t.Amount
		case "PENDING":
			pending[t.OrderID] += t.Amount
		}
	}

	for i := range orders {
		o := &orders[i]
		o.InvoicesPaid = paid[o.OrderID]
		o.InvoicesPending = pending[o.OrderID]
		o.Owed = o.OrderAmount - o.AdvanceAmount - o.InvoicesPaid
		if o.Owed < 0 {
			o.Owed = 0
		}
		if o.PlacedAt != nil {
			o.PlacedOn = o.PlacedAt.Format("02-01-2006")
			o.AgeDays = daysSince(*o.PlacedAt)
		}
	}
	return orders, nil
}

// GetReceivables returns money owed per customer on orders at least minAgeDays old,
// highest first, with an overall total row first.
func (s *service) GetReceivables(minAgeDays int) ([]model.CustomerReceivable, error) {
	balances, err := s.orderBalances()
	if err != nil {
		return nil, err
	}

	total := model.CustomerReceivable{CustomerName: "ALL CUSTOMERS"}
	byCustomer := map[string]*model.CustomerReceivable{}
	for _, o := range balances {
		if o.Owed <= 0 || o.AgeDays < minAgeDays {
			continue
		}
		r, ok := byCustomer[o.CustomerID]
		if !ok {
			r = &model.CustomerReceivable{CustomerName: o.CustomerName}
			byCustomer[o.CustomerID] = r
		}
		for _, acc := range []*model.CustomerReceivable{r, &total} {
			acc.OrdersWithBalance++
			acc.Owed += o.Owed
			acc.PendingInvoiceAmount += o.InvoicesPending
			if o.AgeDays > acc.OldestUnpaidOrderDays {
				acc.OldestUnpaidOrderDays = o.AgeDays
			}
		}
	}

	result := make([]model.CustomerReceivable, 0, len(byCustomer)+1)
	for _, r := range byCustomer {
		result = append(result, *r)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Owed > result[j].Owed })
	return append([]model.CustomerReceivable{total}, result...), nil
}

// GetCustomerPaymentHistory returns the most recent orders of customers whose name matches.
func (s *service) GetCustomerPaymentHistory(name string, limit int) ([]model.OrderBalance, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, reportInputError{"name is required"}
	}
	balances, err := s.orderBalances()
	if err != nil {
		return nil, err
	}

	var matches []model.OrderBalance
	for _, o := range balances {
		if strings.Contains(strings.ToLower(o.CustomerName), name) ||
			strings.Contains(strings.ToLower(o.CustomerShortName), name) {
			matches = append(matches, o)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return placedAfter(matches[i], matches[j]) })
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

// GetLargestOrders returns individual orders ranked by order amount.
func (s *service) GetLargestOrders(limit int) ([]model.OrderBalance, error) {
	balances, err := s.orderBalances()
	if err != nil {
		return nil, err
	}
	sort.Slice(balances, func(i, j int) bool { return balances[i].OrderAmount > balances[j].OrderAmount })
	if len(balances) > limit {
		balances = balances[:limit]
	}
	return balances, nil
}

// GetCustomerActivity returns lifetime revenue, balance and recency per customer.
// sortBy is "revenue" (highest first) or "inactive" (longest since last order first).
func (s *service) GetCustomerActivity(sortBy string) ([]model.CustomerActivity, error) {
	balances, err := s.orderBalances()
	if err != nil {
		return nil, err
	}

	byCustomer := map[string]*model.CustomerActivity{}
	lastOrder := map[string]time.Time{}
	for _, o := range balances {
		a, ok := byCustomer[o.CustomerID]
		if !ok {
			a = &model.CustomerActivity{CustomerName: o.CustomerName, CustomerFlag: o.CustomerFlag}
			byCustomer[o.CustomerID] = a
		}
		a.Orders++
		a.TotalRevenue += o.OrderAmount
		a.Owed += o.Owed
		if o.PlacedAt != nil && o.PlacedAt.After(lastOrder[o.CustomerID]) {
			lastOrder[o.CustomerID] = *o.PlacedAt
			a.LastOrderDate = o.PlacedOn
			a.DaysSinceLastOrder = o.AgeDays
		}
	}

	result := make([]model.CustomerActivity, 0, len(byCustomer))
	for _, a := range byCustomer {
		result = append(result, *a)
	}
	if sortBy == "inactive" {
		sort.Slice(result, func(i, j int) bool { return result[i].DaysSinceLastOrder > result[j].DaysSinceLastOrder })
	} else {
		sort.Slice(result, func(i, j int) bool { return result[i].TotalRevenue > result[j].TotalRevenue })
	}
	return result, nil
}

// GetCustomerDetails returns how many customers match search by name, short name or phone
// (all customers when empty) and the contact and GST details of the first ones.
func (s *service) GetCustomerDetails(search string, limit int) (model.CustomerDirectory, error) {
	return s.repo.GetCustomerDetails(strings.TrimSpace(search), limit)
}

// placedAfter orders balances newest first, with undated orders last.
func placedAfter(a, b model.OrderBalance) bool {
	if a.PlacedAt == nil || b.PlacedAt == nil {
		return a.PlacedAt != nil
	}
	return a.PlacedAt.After(*b.PlacedAt)
}
