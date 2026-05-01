package irrl

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"myproject/pkg/model"
	"myproject/pkg/util"
	"strings"
	"time"

	"github.com/lib/pq"
)

// ListWish
type Repository interface {
	Login(ctx context.Context, email string) (model.UserRegisterRequest, error)
	Register(ctx context.Context, request model.UserRegisterRequest) (string, error)
	UpdateUser(ctx context.Context, query string, args []interface{}) error

	//cart and wishlist

	Listing(ctx context.Context) ([]model.ProductListingUsers, error)
	AddProductsBulk(products []model.Item) error
	GetAttributes(ctx context.Context, typ string) ([]model.Attribute, error)
	Getid(ctx context.Context, username string) string

	VerifyOtp(ctx context.Context, email string)
	ApiQuery(apiType string) (string, error)
	ExecuteUnJoinQuery(query string) ([]map[string]interface{}, error)
	RetrieveSingleVal(query string) (map[string]interface{}, error)
	Exists(query string) (bool, error)
	StartTransaction() (*sql.Tx, error)
	AddMainOrder(request model.DeliveryChelan) (string, string, error)
	UpdateOrderToInitiated(orderID string, guaranteeImages []string) error
	AddDeliveryItem(item model.DeliveryItemHandler, orderId, customerID, inventoryId string) (string, error)
	GetOrderItems(ctx context.Context, query string) ([]string, error)
	AreAllItemsCompleted(orderID string) (bool, error)
	DeleteEntry(table, id, key string) error
	AddMainTransaction(orderID, status string) error
	AddSubTransaction(mainTransactionID, amount int, image, status, typeT string) error
	CreateCustomer(customer model.Customer) error
	UpdateOrderItemImg(itemID string, status string, afterImages []string) error
	GetDeliveryReport(filter model.DeliveryReportFilter) ([]model.ReportRow, error)
	UpdateCurrentAmounts() error
	MarkItemDamage(itemID string, itemDeliveryId string, damageImages []string) error
	ClearItemDamage(itemID string) error
	MarkItemRepairing(itemID string, repairImages []string, amount int) error
	ClearItemRepairing(itemID string) error
	GetRepairHistory(itemID string) ([]model.RepairHistory, error)
	GetDamagedGrouped() ([]model.ItemGroupCount, error)
	GetDamagedList() ([]model.Item, error)
	GetRepairingGrouped() ([]model.ItemGroupCount, error)
	GetRepairingList() ([]model.Item, error)
}

type repository struct {
	sql  *sql.DB
	util *util.Initiator
}

func NewRepository(sqlDB *sql.DB, util *util.Initiator) Repository {
	return &repository{
		sql:  sqlDB,
		util: util,
	}
}
func (r *repository) VerifyOtp(ctx context.Context, email string) {
	query := `
	UPDATE users
	SET verification =true
	WHERE email = $1
	`

	_, err := r.sql.ExecContext(ctx, query, email)

	if err != nil {
		fmt.Errorf("failed to execute update query: %w", err)
	}

}

func (r *repository) Register(ctx context.Context, request model.UserRegisterRequest) (string, error) {
	fmt.Println("this is in the repository Register")
	var id string
	query := `INSERT INTO users (firstname, lastname, email, password) VALUES ($1, $2, $3, $4) Returning id`
	err := r.sql.QueryRowContext(ctx, query, request.FirstName, request.LastName, request.Email, request.Password).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to execute insert query: %w", err)
	}

	return id, nil
}

func (r *repository) Getid(ctx context.Context, username string) string {
	var id string
	fmt.Println("this is in the repository Register !!!")
	query := `select id from users where email=$1;`
	fmt.Println(query, username)
	row := r.sql.QueryRowContext(ctx, query, username)
	err := row.Scan(&id)
	fmt.Println(err)
	fmt.Println("this is id returning from Getid:::", id)

	return id
}

func (r *repository) Login(ctx context.Context, email string) (model.UserRegisterRequest, error) {
	fmt.Println("theee !!!!!!!!!!!  LLLLoginnnnnn  ", email)
	query := `SELECT firstname, lastname, email, password FROM users WHERE email = $1 AND verification=true`
	fmt.Println(`SELECT firstname, lastname, email, password FROM users WHERE email = 'adithyanunni258@gmail.com' ;`)

	var user model.UserRegisterRequest
	err := r.sql.QueryRowContext(ctx, query, email).Scan(&user.FirstName, &user.LastName, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.UserRegisterRequest{}, nil
		}
		return model.UserRegisterRequest{}, fmt.Errorf("failed to find user by email: %w", err)
	}
	fmt.Println("the data !!!! ", user)

	return user, nil
}
func (r *repository) UpdateUser(ctx context.Context, query string, args []interface{}) error {
	queryWithParams := query
	for _, arg := range args {
		queryWithParams = strings.Replace(queryWithParams, "?", fmt.Sprintf("'%v'", arg), 1)
	}
	fmt.Println("Executing update with query:", queryWithParams)
	fmt.Println("Arguments:", args)
	fmt.Println("Executing update for email:", args[len(args)-1]) // Email is the last argument
	_, err := r.sql.ExecContext(ctx, queryWithParams)
	if err != nil {
		return fmt.Errorf("failed to execute update query: %w", err)
	}
	return nil
}

func (r *repository) Listing(ctx context.Context) ([]model.ProductListingUsers, error) {
	query := `
		SELECT 
			product_models.name,
			product_models.category,
			product_models.units,
			product_models.tax,
			product_models.amount,
			product_models.status,
			product_models.discount,
			
			 product_models.id AS pid 
		FROM 
			product_models 
		INNER JOIN 
			vendor ON product_models.vendor_id = vendor.id WHERE product_models.units > 0;`

	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	var products []model.ProductListingUsers
	for rows.Next() {
		var product model.ProductListingUsers
		err := rows.Scan(&product.Name, &product.Category, &product.Unit, &product.Tax, &product.Price, &product.Status, &product.Discount, &product.Pid)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		product.Pdetail = "https://adiecom.gitfunswokhu.in/user/listingSingleProduct/" + product.Pid
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return products, nil
}
func (r *repository) GetAttributes(ctx context.Context, typ string) ([]model.Attribute, error) {
	query := `
		SELECT DISTINCT attributes_id,type, name, status, created_at
		FROM attributes
		WHERE type = $1;`

	rows, err := r.sql.QueryContext(ctx, query, typ)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	var attributes []model.Attribute
	for rows.Next() {
		var attr model.Attribute
		err := rows.Scan(&attr.AttributesID, &attr.Type, &attr.Name, &attr.Status, &attr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		attributes = append(attributes, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return attributes, nil
}
func (r *repository) ApiQuery(apiType string) (string, error) {
	var query string
	fmt.Println("this is the query---", apiType)
	err := r.sql.QueryRow("SELECT query_text FROM api_registry WHERE api_type=$1", apiType).Scan(&query)
	if err != nil {
		fmt.Println("fmt error----", err.Error())
		return "", fmt.Errorf("failed to retrive api's query: %w", err)
	}
	return query, nil
}
func (r *repository) ExecuteUnJoinQuery(query string) ([]map[string]interface{}, error) {
	fmt.Println("this is the query---", query)
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, err
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

		rowMap := make(map[string]interface{})
		for i, col := range cols {
			rowMap[col] = values[i]
		}
		results = append(results, rowMap)
	}

	return results, nil
}
func (r *repository) RetrieveSingleVal(query string) (map[string]interface{}, error) {
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range cols {
			rowMap[col] = values[i]
		}
		return rowMap, nil
	}

	return nil, sql.ErrNoRows
}

func (r *repository) AddProductsBulk(products []model.Item) error {
	fmt.Println("got in reppppp")
	if len(products) == 0 {
		return nil
	}

	query := `INSERT INTO items
	(item_code, sub_code, item_name, item_main_type, item_sub_type, brand, category, description, inventory_id, created_at,main_code,hsn_code)
	VALUES `

	values := []interface{}{}
	placeholders := []string{}
	for i, p := range products {
		n := i*12 + 1
		placeholders = append(placeholders,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11))
		values = append(values,
			p.ItemCode, p.SubCode, p.ItemName, p.ItemMainType, p.ItemSubType,
			p.Brand, p.Category, p.Description, p.InventoryID, p.CreatedAt, p.MainCode, p.HsnCode)
	}

	query += strings.Join(placeholders, ",")
	_, err := r.sql.Exec(query, values...)
	return err
}
func (r *repository) Exists(query string) (bool, error) {
	var exists bool
	err := r.sql.QueryRow(query).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
func (r *repository) StartTransaction() (*sql.Tx, error) {
	tx, err := r.sql.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *repository) AddMainOrder(request model.DeliveryChelan) (string, string, error) {
	var id, invoiceNumber string

	query := `
		INSERT INTO delivery_chelan 
		(customer_id, inventory_id, advance_amount, generated_amount, current_amount, 
		 contact_name, contact_number, shipping_address, placed_at, status,delivery_chelan_number,order_number,invoice_number, discount, guarantee_images) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,$11,$12,$13, $14, $15)
		RETURNING delivery_id, invoice_number
	`

	err := r.sql.QueryRow(

		query,

		request.CustomerID,
		request.InventoryID,
		request.AdvanceAmount,
		request.GeneratedAmount,
		request.CurrentAmount,
		request.ContactName,
		request.ContactNumber,
		request.ShippingAddress,
		request.PlacedAt,
		request.Status,
		generateDeliveryChelanId(),
		generateOrderNumber(),
		generateInvoiceNumber(),
		request.Discount,
		pq.Array(request.GuaranteeImages),
	).Scan(&id, &invoiceNumber)

	if err != nil {
		return "", "", fmt.Errorf("failed to execute insert query: %w", err)
	}

	return id, invoiceNumber, nil
}
func generateDeliveryChelanId() string {
	timestamp := time.Now().Format("20060102150405")    // YYYYMMDDHHMMSS
	randomPart := fmt.Sprintf("%04d", rand.Intn(10000)) // 4 digit random number
	return fmt.Sprintf("DC%s%s", timestamp, randomPart)
}

// Helper function to generate order_number
func generateOrderNumber() string {
	timestamp := time.Now().Format("20060102150405")    // YYYYMMDDHHMMSS
	randomPart := fmt.Sprintf("%04d", rand.Intn(10000)) // 4 digit random number
	return fmt.Sprintf("ORD%s%s", timestamp, randomPart)
}

// generateInvoiceNumber produces a unique invoice number: INV-YYYYMMDD-XXXX
func generateInvoiceNumber() string {
	date := time.Now().Format("20060102")
	randomPart := fmt.Sprintf("%04d", rand.Intn(10000))
	return fmt.Sprintf("INV-%s-%s", date, randomPart)
}
func (r *repository) AddDeliveryItem(item model.DeliveryItemHandler, orderId, customerID, inventoryId string) (string, error) {
	var id string
	fmt.Println("this is the setttt----", item.ItemNewId)

	query := `
		INSERT INTO delivery_items 
		(customer_id, inventory_id, rent_amount, generated_amount, current_amount, 
		 before_images, after_images, condition_out, condition_in, placed_at, 
		 returned_at, status,order_id,item_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,$13,$14)
		RETURNING delivery_item_id
	`

	err := r.sql.QueryRow(
		query,
		customerID,
		inventoryId,
		item.RentAmount,
		item.GeneratedAmount,
		item.CurrentAmount,
		pq.Array(item.BeforeImages), // PostgreSQL array
		pq.Array(item.AfterImages),  // PostgreSQL array
		item.ConditionOut,
		item.ConditionIn,
		item.PlacedAt,
		item.ReturnedAt, // if you want ReturnedStr instead, parse before inserting
		item.Status,
		orderId,
		item.ItemNewId,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to execute insert query: %w", err)
	}

	return id, nil
}

func (r *repository) GetOrderItems(ctx context.Context, query string) ([]string, error) {
	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	var orderitems []string
	for rows.Next() {
		var orderitem string
		err := rows.Scan(
			&orderitem,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		orderitems = append(orderitems, orderitem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return orderitems, nil
}

func (r *repository) AreAllItemsCompleted(orderID string) (bool, error) {
	query := `
	SELECT CASE 
		WHEN COUNT(*) = SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END) 
		THEN TRUE ELSE FALSE 
	END AS all_completed
	FROM public.delivery_items
	WHERE order_id = $1;
	`

	var allCompleted bool
	err := r.sql.QueryRow(query, orderID).Scan(&allCompleted)
	if err != nil {
		return false, fmt.Errorf("failed to check item statuses: %w", err)
	}

	return allCompleted, nil
}

func (r *repository) DeleteEntry(table, key, id string) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1;",
		table, key,
	)
	fmt.Println("this is the query to delete entry", query)
	_, err := r.sql.Exec(query, id)
	return err
}

//func (r *repository) CreateCustomer(ctx context.Context, customer model.Customer) error {
//	query := `
//	INSERT INTO customer (customer_id, name, short_name, phone, type, gst, address, email, customer_flag, status, created_at)
//	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
//	`
//
//	customer.CustomerID = uuid.New()
//	customer.CreatedAt = time.Now()
//
//	_, err := r.sql.ExecContext(ctx, query,
//		customer.CustomerID,
//		customer.Name,
//		customer.ShortName,
//		customer.Phone,
//		customer.Type,
//		customer.GST,
//		customer.Address,
//		customer.Email,
//		customer.CustomerFlag,
//		customer.Status,
//		customer.CreatedAt,
//	)
//	return err
//}

func (r *repository) AddMainTransaction(orderID, status string) error {
	var id int64

	query := `
		INSERT INTO main_transaction (order_id, status)
		VALUES ($1, $2)
		RETURNING id
	`

	err := r.sql.QueryRow(query, orderID, status).Scan(&id)
	if err != nil {
		fmt.Println("main transaction error--", err.Error())
		return fmt.Errorf("failed to insert main_transaction: %w", err)
	}

	return nil
}
func (r *repository) AddSubTransaction(mainTransactionID, amount int, image, status, typ string) error {
	var id int64

	query := `
		INSERT INTO transaction (main_transaction_id,amount, image,status,transaction_type)
		VALUES ($1, $2,$3,$4,$5)
		RETURNING id
	`

	err := r.sql.QueryRow(query, mainTransactionID, amount, image, status, typ).Scan(&id)
	if err != nil {
		fmt.Println("main transaction error--", err.Error())
		return fmt.Errorf("failed to insert main_transaction: %w", err)
	}

	return nil
}
func (r *repository) CreateCustomer(customer model.Customer) error {
	query := `
		INSERT INTO customer (name, short_name, phone, type, gst, address, email, customer_flag, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.sql.Exec(query,
		customer.Name,
		customer.ShortName,
		customer.Phone,
		customer.Type,
		customer.GST,
		customer.Address,
		customer.Email,
		customer.CustomerFlag,
		customer.Status,
	)

	return err
}

func (r *repository) UpdateOrderItemImg(itemID string, status string, afterImages []string) error {
	query := `
        UPDATE delivery_items
        SET status = $1,
            after_images = $2,
            returned_at = CASE WHEN $1 = 'RETURNED' THEN NOW() ELSE returned_at END
        WHERE item_id = $3
    `
	_, err := r.sql.Exec(query, status, pq.Array(afterImages), itemID)
	if err != nil {
		fmt.Println("update order item image error--", err.Error())
	}
	debugQuery := fmt.Sprintf(`
    UPDATE delivery_items
    SET status = '%s',
        after_images = '{%s}',
        returned_at = CASE WHEN '%s' = 'RETURNED' THEN NOW() ELSE returned_at END
    WHERE item_id = '%s';
`,
		status,
		strings.Join(afterImages, ","), // converts slice → comma-separated string
		status,
		itemID,
	)

	fmt.Println("Debug query:", debugQuery)
	return err
}

func (r *repository) GetDeliveryReport(filter model.DeliveryReportFilter) ([]model.ReportRow, error) {
	query := `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY dc.placed_at) AS s_no,
			COALESCE(dc.delivery_chelan_number || ' / ' || TO_CHAR(dc.placed_at, 'DD-MM-YYYY'), '') AS dc_no_date,
			COALESCE(c.name, '') AS party_name,
			COALESCE(STRING_AGG(i.item_name, E'\n'), '') AS items,
			COALESCE(MAX(di.rent_amount), 0) AS rate,
			COALESCE(dc.advance_amount, 0) AS advance_recd,
			COALESCE(SUM(di.generated_amount), 0) AS tool_hire,
			COALESCE(SUM(di.generated_amount) - dc.advance_amount, 0) AS balance_due,
			COALESCE(dc.placed_at::date::text, '') AS date,
			COALESCE(dc.status, '') AS remarks
		FROM public.delivery_items di
		INNER JOIN public.customer c 
			ON di.customer_id = c.customer_id
		INNER JOIN public.delivery_chelan dc 
			ON di.order_id = dc.delivery_id
		INNER JOIN public.items i
			ON di.item_id = i.item_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	// Date filter
	if filter.DateRange != "" {
		var interval string
		switch filter.DateRange {
		case "7days":
			interval = "7 days"
		case "14days":
			interval = "14 days"
		case "21days":
			interval = "21 days"
		case "1month":
			interval = "1 month"
		case "2months":
			interval = "2 months"
		case "1year":
			interval = "1 year"
		case "2year":
			interval = "2 years"
		}
		if interval != "" {
			query += fmt.Sprintf(" AND dc.placed_at >= NOW() - INTERVAL '%s' ", interval)
		}
	}

	// Remark/Status filter
	if filter.Remark != "" {
		query += fmt.Sprintf(" AND dc.status = $%d ", argIdx)
		args = append(args, filter.Remark)
		argIdx++
	}

	query += `
		GROUP BY 
			dc.delivery_chelan_number, 
			dc.placed_at, 
			c.name, 
			dc.advance_amount, 
			dc.status
		ORDER BY dc.placed_at;
	`

	rows, err := r.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.ReportRow
	for rows.Next() {
		var row model.ReportRow
		err := rows.Scan(
			&row.SNo,
			&row.DCNoDate,
			&row.PartyName,
			&row.Items,
			&row.Rate,
			&row.AdvanceRecd,
			&row.ToolHire,
			&row.BalanceDue,
			&row.Date,
			&row.Remarks,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, row)
	}
	return reports, nil
}

// UpdateCurrentAmounts recalculates current_amount for all active delivery items
// and rolls the totals up to their parent delivery_chelan orders.
// Runs nightly at midnight IST.
func (r *repository) UpdateCurrentAmounts() error {
	// Step 1: update current_amount on each active delivery item.
	// current_amount = rent_amount * days elapsed since placed_at (capped at total rental days).
	itemQuery := `
		UPDATE delivery_items
		SET current_amount = rent_amount * GREATEST(1,
			CASE
				WHEN returned_at IS NOT NULL AND NOW() > returned_at
					THEN EXTRACT(DAY FROM (returned_at - placed_at))::int
				ELSE EXTRACT(DAY FROM (NOW() - placed_at))::int
			END
		),
		generated_amount = rent_amount * GREATEST(1,
			CASE
				WHEN returned_at IS NOT NULL AND NOW() > returned_at
					THEN EXTRACT(DAY FROM (returned_at - placed_at))::int
				ELSE EXTRACT(DAY FROM (NOW() - placed_at))::int
			END
		)
		WHERE status NOT IN ('COMPLETED', 'DELETED', 'BLOCKED')
		  AND placed_at IS NOT NULL;
	`
	if _, err := r.sql.Exec(itemQuery); err != nil {
		return fmt.Errorf("cron: failed to update delivery_items current_amount: %w", err)
	}

	// Step 2: roll up the summed item current_amounts and generated_amounts to the main order.
	orderQuery := `
		UPDATE delivery_chelan dc
		SET current_amount = (
			SELECT COALESCE(SUM(di.current_amount), 0)
			FROM delivery_items di
			WHERE di.order_id = dc.delivery_id
		),
		generated_amount = GREATEST(0, (
			SELECT COALESCE(SUM(di.generated_amount), 0)
			FROM delivery_items di
			WHERE di.order_id = dc.delivery_id
		) - COALESCE(dc.discount, 0))
		WHERE dc.status NOT IN ('COMPLETED', 'DELETED');
	`
	if _, err := r.sql.Exec(orderQuery); err != nil {
		return fmt.Errorf("cron: failed to update delivery_chelan current_amount: %w", err)
	}

	return nil
}

// MarkItemDamage flags an item as damaged, updates its category, and inserts a repair_history record.
func (r *repository) MarkItemDamage(itemID, itemDeliveryId string, damageImages []string) error {
	// 1. Look up the active delivery order and customer name for this item.
	type orderInfo struct {
		OrderID      string
		CustomerName string
	}
	var info orderInfo
	infoQuery := `
		SELECT di.order_id, COALESCE(c.name, '') AS customer_name
		FROM public.delivery_items di
		LEFT JOIN public.customer c ON di.customer_id = c.customer_id
		WHERE di.item_id = $1
		  AND di.status NOT IN ('COMPLETED', 'DELETED')
		ORDER BY di.placed_at DESC
		LIMIT 1;
	`
	err := r.sql.QueryRow(infoQuery, itemID).Scan(&info.OrderID, &info.CustomerName)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("failed to fetch order info for item: %w", err)
	}

	// 2. Mark the item as DAMAGED in the items table and set damage flag on delivery_items.
	_, err = r.sql.Exec(
		"UPDATE items SET category = 'DAMAGED' WHERE item_id = $1;",
		itemID,
	)
	if err != nil {
		return fmt.Errorf("failed to update item category to DAMAGED: %w", err)
	}

	_, err = r.sql.Exec(
		"UPDATE delivery_items SET damage = true WHERE delivery_item_id = $1;",
		itemDeliveryId,
	)
	if err != nil {
		return fmt.Errorf("failed to set damage flag on delivery_items: %w", err)
	}

	// 3. Insert a repair_history record.
	insertQuery := `
		INSERT INTO repair_history (item_id, order_item_id, damage_images, created_at)
		VALUES ($1, $2, $3, NOW());
	`
	_, err = r.sql.Exec(insertQuery, itemID, itemDeliveryId, pq.Array(damageImages))
	if err != nil {
		return fmt.Errorf("failed to insert repair_history: %w", err)
	}

	return nil
}

// ClearItemDamage removes the damage flag from an item (no images needed).
func (r *repository) ClearItemDamage(itemID string) error {
	_, err := r.sql.Exec(
		"UPDATE delivery_items SET damage = false WHERE item_id = $1 AND status NOT IN ('COMPLETED','DELETED');",
		itemID,
	)
	if err != nil {
		return err
	}
	_, err = r.sql.Exec(
		"UPDATE items SET category = 'AVAILABLE' WHERE item_id = $1;",
		itemID,
	)
	return err
}

func (r *repository) UpdateOrderToInitiated(orderID string, guaranteeImages []string) error {
	fmt.Println("images.. here:::", guaranteeImages)
	query := `
		UPDATE delivery_chelan
		SET placed_at = NOW(), status = 'INITIATED',
			guarantee_images = $1,
		    generated_amount = GREATEST(0, (SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_items WHERE order_id = $2) - COALESCE(discount, 0)),
		    current_amount = (SELECT COALESCE(SUM(current_amount), 0) FROM delivery_items WHERE order_id = $2)
		WHERE delivery_id = $2 AND status = 'RESERVED';
	`
	fmt.Println("qryyy.. here:::", query)
	_, err := r.sql.Exec(query, pq.Array(guaranteeImages), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order to initiated: %w", err)
	}
	return nil
}

// MarkItemRepairing flags an item as repairing, updates its category, and inserts a repair_history record.
func (r *repository) MarkItemRepairing(itemID string, repairImages []string, amount int) error {
	// 1. Look up the active delivery order and customer name for this item.
	type orderInfo struct {
		OrderID      string
		CustomerName string
	}
	var info orderInfo
	infoQuery := `
		SELECT di.order_id, COALESCE(c.name, '') AS customer_name
		FROM public.delivery_items di
		LEFT JOIN public.customer c ON di.customer_id = c.customer_id
		WHERE di.item_id = $1
		  AND di.status NOT IN ('COMPLETED', 'DELETED')
		ORDER BY di.placed_at DESC
		LIMIT 1;
	`
	err := r.sql.QueryRow(infoQuery, itemID).Scan(&info.OrderID, &info.CustomerName)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("failed to fetch order info for item: %w", err)
	}

	// 2. Mark the item as REPAIRING in the items table.
	_, err = r.sql.Exec(
		"UPDATE items SET category = 'REPAIRING' WHERE item_id = $1;",
		itemID,
	)
	if err != nil {
		return fmt.Errorf("failed to update item category to REPAIRING: %w", err)
	}

	// 3. Insert a repair_history record.
	insertQuery := `
		INSERT INTO repair_history (item_id, order_id, customer_name, damage_images, amount, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW());
	`
	_, err = r.sql.Exec(insertQuery, itemID, info.OrderID, info.CustomerName, pq.Array(repairImages), amount)
	if err != nil {
		return fmt.Errorf("failed to insert repair_history: %w", err)
	}

	return nil
}

// ClearItemRepairing removes the repairing flag from an item.
func (r *repository) ClearItemRepairing(itemID string) error {
	_, err := r.sql.Exec(
		"UPDATE items SET category='AVAILABLE' WHERE item_id = $1;",
		itemID,
	)
	return err
}

// GetRepairHistory returns all repair_history records for a given item, newest first.
func (r *repository) GetRepairHistory(itemID string) ([]model.RepairHistory, error) {
	query := `
		SELECT id, item_id, COALESCE(order_id,''), COALESCE(customer_name,''), damage_images, COALESCE(amount,0), created_at
		FROM repair_history
		WHERE item_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.sql.Query(query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repair history: %w", err)
	}
	defer rows.Close()

	var records []model.RepairHistory
	for rows.Next() {
		var rec model.RepairHistory
		if err := rows.Scan(
			&rec.ID,
			&rec.ItemID,
			&rec.OrderID,
			&rec.CustomerName,
			pq.Array(&rec.DamageImages),
			&rec.Amount,
			&rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan repair_history row: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repair_history rows: %w", err)
	}
	return records, nil
}

func (r *repository) GetDamagedList() ([]model.Item, error) {
	query := `
		SELECT item_id, item_code, sub_code, item_name, item_main_type, 
		       COALESCE(item_sub_type, ''), COALESCE(brand, ''), 
		       COALESCE(category, ''), COALESCE(description, ''), 
		       inventory_id, created_at, main_code, COALESCE(hsn_code, '')
		FROM items
		WHERE category = 'DAMAGED'
		ORDER BY created_at DESC;
	`
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch list: %w", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		if err := rows.Scan(
			&item.ItemID, &item.ItemCode, &item.SubCode, &item.ItemName, &item.ItemMainType,
			&item.ItemSubType, &item.Brand, &item.Category, &item.Description,
			&item.InventoryID, &item.CreatedAt, &item.MainCode, &item.HsnCode,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repository) GetRepairingList() ([]model.Item, error) {
	query := `
		SELECT item_id, item_code, sub_code, item_name, item_main_type, 
		       COALESCE(item_sub_type, ''), COALESCE(brand, ''), 
		       COALESCE(category, ''), COALESCE(description, ''), 
		       inventory_id, created_at, main_code, COALESCE(hsn_code, '')
		FROM items
		WHERE category = 'REPAIRING'
		ORDER BY created_at DESC;
	`
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch list: %w", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		if err := rows.Scan(
			&item.ItemID, &item.ItemCode, &item.SubCode, &item.ItemName, &item.ItemMainType,
			&item.ItemSubType, &item.Brand, &item.Category, &item.Description,
			&item.InventoryID, &item.CreatedAt, &item.MainCode, &item.HsnCode,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repository) GetDamagedGrouped() ([]model.ItemGroupCount, error) {
	query := `
		SELECT item_name, COUNT(*) as count
		FROM items
		WHERE category = 'DAMAGED'
		GROUP BY item_name
		ORDER BY item_name ASC;
	`
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grouped: %w", err)
	}
	defer rows.Close()

	var groups []model.ItemGroupCount
	for rows.Next() {
		var g model.ItemGroupCount
		if err := rows.Scan(&g.ItemName, &g.Count); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *repository) GetRepairingGrouped() ([]model.ItemGroupCount, error) {
	query := `
		SELECT item_name, COUNT(*) as count
		FROM items
		WHERE category = 'REPAIRING'
		GROUP BY item_name
		ORDER BY item_name ASC;
	`
	rows, err := r.sql.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grouped: %w", err)
	}
	defer rows.Close()

	var groups []model.ItemGroupCount
	for rows.Next() {
		var g model.ItemGroupCount
		if err := rows.Scan(&g.ItemName, &g.Count); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}
