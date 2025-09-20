package irrl

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"myproject/pkg/model"
	"myproject/pkg/util"
	"strings"
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
	AddMainOrder(request model.DeliveryChelan) (string, error)
	AddDeliveryItem(item model.DeliveryItemHandler, orderId, customerID, inventoryId string) (string, error)
	GetOrderItems(ctx context.Context, query string) ([]model.DeliveryItem, error)
	AreAllItemsCompleted(orderID string) (bool, error)
	DeleteEntry(table, id, key string) error
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
	(item_code, sub_code, item_name, item_main_type, item_sub_type, brand, category, description, inventory_id, created_at)
	VALUES `

	values := []interface{}{}
	placeholders := []string{}

	for i, p := range products {
		n := i*10 + 1
		placeholders = append(placeholders,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9))
		values = append(values,
			p.ItemCode, p.SubCode, p.ItemName, p.ItemMainType, p.ItemSubType,
			p.Brand, p.Category, p.Description, p.InventoryID, p.CreatedAt)
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

func (r *repository) AddMainOrder(request model.DeliveryChelan) (string, error) {
	var id string

	query := `
		INSERT INTO delivery_chelan 
		(customer_id, inventory_id, advance_amount, generated_amount, current_amount, 
		 contact_name, contact_number, shipping_address, placed_at, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING delivery_id
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
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to execute insert query: %w", err)
	}

	return id, nil
}
func (r *repository) AddDeliveryItem(item model.DeliveryItemHandler, orderId, customerID, inventoryId string) (string, error) {
	var id string

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
		item.ItemCode,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to execute insert query: %w", err)
	}

	return id, nil
}

func (r *repository) GetOrderItems(ctx context.Context, query string) ([]model.DeliveryItem, error) {
	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	var orderitems []model.DeliveryItem
	for rows.Next() {
		var orderitem model.DeliveryItem
		err := rows.Scan(
			&orderitem.DeliveryItemID,
			&orderitem.CustomerID,
			&orderitem.InventoryID,
			&orderitem.RentAmount,
			&orderitem.GeneratedAmount,
			&orderitem.CurrentAmount,
			pq.Array(&orderitem.BeforeImages), // requires "github.com/lib/pq"
			pq.Array(&orderitem.AfterImages),
			&orderitem.ConditionOut,
			&orderitem.ConditionIn,
			&orderitem.PlacedAt,
			&orderitem.ReturnedAt,
			&orderitem.ReturnedStr,
			&orderitem.DeclinedAt,
			&orderitem.Status,
			&orderitem.ItemID,
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
	_, err := r.sql.Exec(query, id)
	return err
}
