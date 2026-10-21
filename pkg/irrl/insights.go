package irrl

import (
	"fmt"
	"myproject/pkg/model"
)

// Read-only report queries served to the ai-chatbot service on /irrl/chat-reports.
// Each returns rows as maps so they can be passed on as JSON without a struct per report.
//
// These only read values the app stores when orders and items change. Anything the app
// computes at read time (invoice amounts in GetSubTransactions) is applied in Go instead,
// in chat_reports.go, so the reports cannot drift from it.
//
// Date ranges are [from, to) as 'YYYY-MM-DD' strings; callers pass `to` already exclusive.

// rentedDaysSQL is the number of days delivery item `di` was out on rent inside [$1, $2).
// An item counts from the day it was placed up to its planned end date (rent stops accruing
// there, matching the nightly billing job), and never past today.
const rentedDaysSQL = `GREATEST(0,
		LEAST($2::date,
		      GREATEST(di.placed_at::date + 1,
		               LEAST(COALESCE(di.returned_at::date, CURRENT_DATE + 1), CURRENT_DATE + 1)))
		- GREATEST($1::date, di.placed_at::date))`

// incomeByProductCTE sums rental income and rented unit-days per product inside [$1, $2).
const incomeByProductCTE = `
	income AS (
		SELECT i.product_code,
		       SUM(di.rent_amount * ` + rentedDaysSQL + `)::float8 AS income,
		       SUM(` + rentedDaysSQL + `) AS rented_unit_days
		FROM delivery_items di
		JOIN delivery_chelan dc ON dc.delivery_id = di.order_id
		JOIN items i ON i.item_id = di.item_id
		WHERE dc.status NOT IN ('DELETED', 'RESERVED')
		  AND COALESCE(di.status, '') NOT IN ('DELETED', 'RESERVED')
		  AND di.placed_at IS NOT NULL
		GROUP BY i.product_code
	),
	fleet AS (
		SELECT product_code, MAX(item_name) AS item_name,
		       COUNT(*) FILTER (WHERE COALESCE(category, '') <> 'EXPIRED') AS units_owned,
		       COUNT(*) FILTER (WHERE category = 'DAMAGED')   AS currently_damaged,
		       COUNT(*) FILTER (WHERE category = 'REPAIRING') AS currently_repairing
		FROM items
		GROUP BY product_code
	)`

// GetBusinessSnapshot returns headline numbers for a period plus the current state of
// rentals. Money owed is added by the service from order balances.
func (r *repository) GetBusinessSnapshot(from, to string) ([]map[string]interface{}, error) {
	query := `
	SELECT
		(SELECT COALESCE(SUM(di.rent_amount * ` + rentedDaysSQL + `), 0)::float8
		   FROM delivery_items di
		   JOIN delivery_chelan dc ON dc.delivery_id = di.order_id
		  WHERE dc.status NOT IN ('DELETED', 'RESERVED')
		    AND COALESCE(di.status, '') NOT IN ('DELETED', 'RESERVED')
		    AND di.placed_at IS NOT NULL) AS rental_income_earned,
		(SELECT COUNT(*) FROM delivery_chelan
		  WHERE status NOT IN ('DELETED', 'RESERVED') AND placed_at >= $1::date AND placed_at < $2::date) AS orders_placed,
		(SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_chelan
		  WHERE status NOT IN ('DELETED', 'RESERVED') AND placed_at >= $1::date AND placed_at < $2::date) AS orders_placed_value,
		(SELECT COALESCE(SUM(advance_amount), 0) FROM delivery_chelan
		  WHERE status NOT IN ('DELETED', 'RESERVED') AND placed_at >= $1::date AND placed_at < $2::date) AS advances_received,
		(SELECT COUNT(*) FROM delivery_chelan
		  WHERE declined_at >= $1::date AND declined_at < $2::date) AS cancelled_orders,
		(SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_chelan
		  WHERE declined_at >= $1::date AND declined_at < $2::date) AS cancelled_orders_value,
		(SELECT COUNT(*) FROM repair_history
		  WHERE created_at >= $1::date AND created_at < $2::date) AS repair_events,
		(SELECT COALESCE(SUM(amount), 0) FROM repair_history
		  WHERE created_at >= $1::date AND created_at < $2::date) AS repair_cost,
		(SELECT COUNT(*) FROM items WHERE category = 'RENTED') AS units_on_rent_now,
		(SELECT COUNT(*) FROM items WHERE category = 'AVAILABLE') AS units_available_now,
		(SELECT COUNT(*) FROM delivery_items di
		   JOIN delivery_chelan dc ON dc.delivery_id = di.order_id
		  WHERE COALESCE(di.status, '') NOT IN ('COMPLETED', 'RETURNED', 'DELETED', 'RESERVED', 'BLOCKED')
		    AND dc.status NOT IN ('COMPLETED', 'DELETED', 'RESERVED')
		    AND di.returned_at < CURRENT_DATE) AS units_overdue_for_return_now;`
	return r.queryMaps(query, from, to)
}

// GetIncomeByEquipment returns rental income, rented unit-days and utilization per product.
func (r *repository) GetIncomeByEquipment(from, to string, ascending bool) ([]map[string]interface{}, error) {
	order := "DESC"
	if ascending {
		order = "ASC"
	}
	query := `WITH ` + incomeByProductCTE + `
	SELECT f.product_code, f.item_name, f.units_owned,
	       COALESCE(inc.income, 0) AS rental_income,
	       COALESCE(inc.rented_unit_days, 0) AS rented_unit_days,
	       ROUND(100.0 * COALESCE(inc.rented_unit_days, 0)
	             / NULLIF(f.units_owned * ($2::date - $1::date), 0), 1)::float8 AS utilization_pct
	FROM fleet f
	LEFT JOIN income inc ON inc.product_code = f.product_code
	ORDER BY rental_income ` + order + `, f.product_code;`
	return r.queryMaps(query, from, to)
}

// GetFleetStatus returns unit counts per status for each product, with an overall total row first.
func (r *repository) GetFleetStatus() ([]map[string]interface{}, error) {
	query := `
	SELECT COALESCE(product_code, 'ALL PRODUCTS') AS product_code,
	       CASE WHEN product_code IS NULL THEN '' ELSE MAX(item_name) END AS item_name,
	       COUNT(*) AS total_units,
	       COUNT(*) FILTER (WHERE category = 'AVAILABLE') AS available,
	       COUNT(*) FILTER (WHERE category = 'RENTED')    AS rented,
	       COUNT(*) FILTER (WHERE category = 'RESERVED')  AS reserved,
	       COUNT(*) FILTER (WHERE category = 'SITE')      AS at_site,
	       COUNT(*) FILTER (WHERE category = 'DAMAGED')   AS damaged,
	       COUNT(*) FILTER (WHERE category = 'REPAIRING') AS repairing,
	       COUNT(*) FILTER (WHERE category = 'BLOCKED')   AS blocked,
	       COUNT(*) FILTER (WHERE category = 'EXPIRED')   AS worn_out,
	       ROUND(100.0 * COUNT(*) FILTER (WHERE category = 'RENTED')
	             / NULLIF(COUNT(*) FILTER (WHERE COALESCE(category, '') <> 'EXPIRED'), 0), 1)::float8 AS utilization_now_pct
	FROM items
	GROUP BY GROUPING SETS ((product_code), ())
	ORDER BY (product_code IS NULL) DESC, total_units DESC;`
	return r.queryMaps(query)
}

// GetIdleEquipment returns, per product, available units not rented in the last minDays days.
func (r *repository) GetIdleEquipment(minDays int) ([]map[string]interface{}, error) {
	query := `
	WITH last_rent AS (
		SELECT item_id, MAX(placed_at) AS last_rented
		FROM delivery_items
		WHERE COALESCE(status, '') NOT IN ('DELETED', 'RESERVED')
		GROUP BY item_id
	)
	SELECT i.product_code, MAX(i.item_name) AS item_name,
	       COUNT(*) AS idle_units,
	       COUNT(*) FILTER (WHERE lr.last_rented IS NULL) AS never_rented_units,
	       COALESCE(TO_CHAR(MIN(lr.last_rented), 'DD-MM-YYYY'), '') AS oldest_last_rented,
	       MAX(CURRENT_DATE - lr.last_rented::date) AS max_days_idle_of_rented_units
	FROM items i
	LEFT JOIN last_rent lr ON lr.item_id = i.item_id
	WHERE i.category = 'AVAILABLE'
	  AND (lr.last_rented IS NULL OR lr.last_rented < NOW() - make_interval(days => $1))
	GROUP BY i.product_code
	ORDER BY idle_units DESC;`
	return r.queryMaps(query, minDays)
}

// GetRentalReturns returns rented units past their end date, or due within dueWithinDays days.
func (r *repository) GetRentalReturns(dueWithinDays int) ([]map[string]interface{}, error) {
	query := `
	SELECT c.name AS customer_name, COALESCE(dc.order_number, '') AS order_number,
	       i.product_code, MAX(i.item_name) AS item_name, COUNT(*) AS units,
	       TO_CHAR(di.returned_at::date, 'DD-MM-YYYY') AS due_date,
	       CURRENT_DATE - di.returned_at::date AS days_overdue,
	       SUM(di.rent_amount) AS daily_rent,
	       (SUM(di.rent_amount) * GREATEST(0, CURRENT_DATE - di.returned_at::date))::float8 AS unbilled_rent_for_overdue_days
	FROM delivery_items di
	JOIN delivery_chelan dc ON dc.delivery_id = di.order_id
	JOIN customer c ON c.customer_id = di.customer_id
	JOIN items i ON i.item_id = di.item_id
	WHERE COALESCE(di.status, '') NOT IN ('COMPLETED', 'RETURNED', 'DELETED', 'RESERVED', 'BLOCKED')
	  AND dc.status NOT IN ('COMPLETED', 'DELETED', 'RESERVED')
	  AND di.returned_at IS NOT NULL
	  AND di.returned_at::date <= CURRENT_DATE + $1::int
	GROUP BY c.name, dc.order_number, i.product_code, di.returned_at::date
	ORDER BY di.returned_at::date, c.name;`
	return r.queryMaps(query, dueWithinDays)
}

// GetRepairVsIncome compares repair cost with rental income per product inside [from, to).
func (r *repository) GetRepairVsIncome(from, to string) ([]map[string]interface{}, error) {
	query := `WITH ` + incomeByProductCTE + `,
	repairs AS (
		SELECT i.product_code,
		       COUNT(*) AS repair_events,
		       COUNT(DISTINCT rh.item_id) AS units_repaired,
		       COALESCE(SUM(rh.amount), 0) AS repair_cost
		FROM repair_history rh
		JOIN items i ON i.item_id::TEXT = rh.item_id
		WHERE rh.created_at >= $1::date AND rh.created_at < $2::date
		GROUP BY i.product_code
	)
	SELECT f.product_code, f.item_name, f.units_owned,
	       COALESCE(rp.repair_events, 0) AS repair_events,
	       COALESCE(rp.units_repaired, 0) AS units_repaired,
	       COALESCE(rp.repair_cost, 0) AS repair_cost,
	       COALESCE(inc.income, 0) AS rental_income,
	       ROUND((100.0 * COALESCE(rp.repair_cost, 0) / NULLIF(inc.income, 0))::numeric, 1)::float8 AS repair_cost_pct_of_income,
	       f.currently_damaged, f.currently_repairing
	FROM fleet f
	LEFT JOIN repairs rp ON rp.product_code = f.product_code
	LEFT JOIN income inc ON inc.product_code = f.product_code
	WHERE COALESCE(rp.repair_events, 0) > 0 OR f.currently_damaged > 0 OR f.currently_repairing > 0
	ORDER BY repair_cost DESC, repair_events DESC;`
	return r.queryMaps(query, from, to)
}

// GetOrderBalanceBase returns every live order with its customer, before payments are applied.
func (r *repository) GetOrderBalanceBase() ([]model.OrderBalance, error) {
	query := `
	SELECT dc.delivery_id::TEXT, COALESCE(dc.order_number, ''), COALESCE(dc.status, ''),
	       dc.placed_at, COALESCE(dc.generated_amount, 0), COALESCE(dc.advance_amount, 0),
	       c.customer_id::TEXT, c.name, COALESCE(c.short_name, ''), COALESCE(c.customer_flag, '')
	FROM delivery_chelan dc
	JOIN customer c ON c.customer_id = dc.customer_id
	WHERE dc.status NOT IN ('DELETED', 'RESERVED');`
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}
	defer rows.Close()

	var orders []model.OrderBalance
	for rows.Next() {
		var o model.OrderBalance
		if err := rows.Scan(&o.OrderID, &o.OrderNumber, &o.Status, &o.PlacedAt,
			&o.OrderAmount, &o.AdvanceAmount,
			&o.CustomerID, &o.CustomerName, &o.CustomerShortName, &o.CustomerFlag); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetCustomerDetails returns customers whose name, short name or phone contains search
// (all customers when search is empty), alphabetically. Deleted customers are left out.
func (r *repository) GetCustomerDetails(search string, limit int) ([]model.Customer, error) {
	query := `
	SELECT customer_id, name, short_name, phone, type, gst, address,
	       COALESCE(email, ''), COALESCE(customer_flag, ''), COALESCE(status, ''),
	       COALESCE(created_at, CURRENT_TIMESTAMP)
	FROM customer
	WHERE COALESCE(status, '') <> 'DELETED'
	  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR short_name ILIKE '%' || $1 || '%' OR phone ILIKE '%' || $1 || '%')
	ORDER BY name
	LIMIT $2;`
	rows, err := r.sql.Query(query, search, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customers: %w", err)
	}
	defer rows.Close()

	customers := []model.Customer{}
	for rows.Next() {
		var c model.Customer
		if err := rows.Scan(&c.CustomerID, &c.Name, &c.ShortName, &c.Phone, &c.Type, &c.GST, &c.Address,
			&c.Email, &c.CustomerFlag, &c.Status, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// queryMaps runs a parameterised query and returns each row as column -> value.
func (r *repository) queryMaps(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := r.sql.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("insight query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			// numeric and text columns can arrive as []byte; keep them readable in JSON.
			if b, ok := values[i].([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = values[i]
			}
		}
		results = append(results, rowMap)
	}
	return results, rows.Err()
}
