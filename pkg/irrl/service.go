package irrl

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	services "myproject/pkg/client"
	"myproject/pkg/config"
	"myproject/pkg/model"
	"myproject/pkg/util"
	"myproject/pkg/util/constants"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, request model.UserRegisterRequest) error
	Login(ctx context.Context, request model.UserLoginRequest) error
	//Product listing
	Listing(ctx context.Context) ([]model.ProductListingUsers, error)
	AddProduct(ctx context.Context, product model.Product) (model.AddProductResult, error)
	UploadItemSheet(ctx context.Context, rows []model.ItemSheetRow) (model.UploadItemSheetResult, error)
	GetProductStockInfo(productCode string) (model.ProductStockInfo, error)
	AddOrder(order model.DeliveryOrder) (string, error)
	GetAttributes(ctx context.Context, typ string) ([]model.Attribute, error)
	OtpLogin(ctx context.Context, request model.UserOtp) error
	UpdateUser(ctx context.Context, updatedData model.UserRegisterRequest) error

	VerifyOtp(ctx context.Context, email string)
	GenericApi(ctx context.Context, apiType, endQuery string) ([]map[string]interface{}, error)
	GenericStatusUpdate(update model.GenericUpdate) error
	UpdateMainOrderStatus(orderID, typestat string) error
	InitiateMainOrder(orderID string, guaranteeImages []string) error
	UpdateItemStatus(orderID, typestat string) error
	InsertGeneric(genReq map[string]interface{}) error
	AddSubTransaction(transaction model.Transaction) (string, error)
	GetSubTransactions(orderID string) ([]model.Transaction, error)
	UpdateOrderItemStatus(req model.OrderItemUpdate) error
	CreateCustomer(customer model.Customer) error
	GenerateDeliveryReportExcel(filter model.DeliveryReportFilter) ([]byte, error)
	GenericDelete(req model.GenericDelete) error
	UpdateCurrentAmounts() error
	MarkItemDamage(itemID, itemDeliveryId string, damageImages []string, description string) error
	ClearItemDamage(itemID string) error
	MarkItemRepairing(itemID string, repairImages []string, amount int, description string) error
	ClearItemRepairing(itemID string) error
	GetRepairHistory(itemID string) ([]model.RepairHistory, error)
	GetDamagedGrouped() ([]model.ItemGroupCount, error)
	GetDamagedList(productCode string) ([]model.Item, error)
	GetRepairingGrouped() ([]model.ItemGroupCount, error)
	GetRepairingList(productCode string) ([]model.Item, error)
	UpdateOrderPass(req model.OrderPassRequest) error
	UpdateOrderReturnedAt(orderID string, returnedAt time.Time) error
	UpdateOrderItemReturnedAt(deliveryItemID string, returnedAt time.Time) error

	GetCustomerDamageAnalytics() ([]model.CustomerDamageAnalytics, error)
	GetItemDamageAnalytics() ([]model.ItemDamageAnalytics, error)
	GetCustomerBlockedAnalytics() ([]model.CustomerBlockedAnalytics, error)
	GetItemRentalAnalytics() ([]model.ItemRentalAnalytics, error)
	GetOrderStatusSummary() ([]model.OrderStatusSummary, error)
	GetRevenueByCustomer() ([]model.RevenueByCustomer, error)

	GetCostSummary() (model.CostSummary, error)
	GetRevenueTrend(period string, limit int) ([]model.RevenueTrend, error)
	GetRepairCostAnalytics() ([]model.RepairCostAnalytics, error)
	GetOutstandingBalances() ([]model.OutstandingBalance, error)
	GetInventoryRevenue() ([]model.InventoryRevenue, error)

	CreateQuotation(req model.QuotationRequest) (string, error)
	GetQuotations() ([]model.Quotation, error)
	GetQuotationByID(id string) (model.Quotation, error)

	GetOrderDetails(orderID string) (model.OrderDetails, error)

	MarkItemSite(itemID string) error
	ClearItemSite(itemID string) error
	GetSiteGrouped() ([]model.ItemGroupCount, error)
	GetSiteList(productCode string) ([]model.Item, error)
	GetAvailableItemByProductCode(productCode string) (model.Item, error)
}
type service struct {
	repo     Repository
	Config   config.Config
	services services.Services
	util     *util.Initiator

	// addProductMu serializes AddProduct calls so two concurrent requests
	// can't read the same "latest code" and hand out duplicate
	// product_code/item_code values.
	addProductMu sync.Mutex
}

func NewService(repo Repository, services services.Services, util *util.Initiator) Service {
	return &service{
		repo:     repo,
		services: services,
		util:     util,
	}
}
func (s *service) VerifyOtp(ctx context.Context, email string) {
	s.repo.VerifyOtp(ctx, email)

}

// ///
type PageVariable struct {
	AppointmentID string
}

func (s *service) Register(ctx context.Context, request model.UserRegisterRequest) error {
	var err error
	if request.FirstName == "" || request.Email == "" || request.Password == "" || request.Phone == "" {
		fmt.Println("this is in the service error value missing")
		err = fmt.Errorf("missing values")
		return err
	}
	if !isValidEmail(request.Email) {
		fmt.Println("this is in the service error invalid email")
		err = fmt.Errorf("invalid email")
		return err
	}
	if !isValidPhoneNumber(request.Phone) {
		fmt.Println("this is in the service error invalid phone number")
		err = fmt.Errorf("invalid phone number")
		return err
	}
	fmt.Println("this is the dataaa ", request.Email)
	existingUser, err := s.repo.Login(ctx, request.Email)
	fmt.Println("there may be a user", existingUser)
	if err != nil && err != gorm.ErrRecordNotFound {
		fmt.Println("this is in the service error checking existing user")
		err = fmt.Errorf("failed to check existing user: %w", err)
		return err
	}
	if existingUser.Email != "" {
		fmt.Println("this is in the service user already exists")
		err = fmt.Errorf("user already exists")
		return err
	}
	fmt.Println("this is in the service Register", request.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("this is in the service error hashing password")
		err = fmt.Errorf("failed to hash password: %w", err)
		return err
	}
	request.Password = string(hashedPassword)
	fmt.Println("this is in the service Register", request.Password)
	id, _ := s.repo.Register(ctx, request)
	fmt.Println("d", id)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	return nil
}

func (s *service) Login(ctx context.Context, request model.UserLoginRequest) error {
	fmt.Println("this is in the service Login", request.Password)
	var err error
	if request.Email == "" || request.Password == "" {
		fmt.Println("this is in the service error value missing")
		err = fmt.Errorf("missing values")
		return err
	}
	storedUser, err := s.repo.Login(ctx, request.Email)
	fmt.Println("thisss is the dataaa ", storedUser)
	if err != nil {
		fmt.Println("this is in the service user not found")
		return fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(request.Password)); err != nil {
		fmt.Println("this is in the service incorrect password")
		return fmt.Errorf("incorrect password: %w", err)
	}

	return nil
}

func (s *service) OtpLogin(ctx context.Context, request model.UserOtp) error {
	fmt.Println("this is in the service Login", request.Otp)
	var err error
	if request.Email == "" || request.Otp == "" {
		fmt.Println("this is in the service error value missing")
		err = fmt.Errorf("missing values")
		return err
	}
	return nil
}

func (s *service) UpdateUser(ctx context.Context, updatedData model.UserRegisterRequest) error {
	var query string
	var args []interface{}

	query = "UPDATE users SET"

	if updatedData.FirstName != "" {
		query += " firstname = ?,"
		args = append(args, updatedData.FirstName)
	}
	if updatedData.LastName != "" {
		query += " lastname = ?,"
		args = append(args, updatedData.LastName)
	}
	if updatedData.Password != "" {
		// Hash the password before updating it
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedData.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		query += " password = ?,"
		args = append(args, string(hashedPassword))
	}
	if updatedData.Phone != "" && isValidPhoneNumber(updatedData.Phone) {
		query += " phone = ?,"
		args = append(args, updatedData.Phone)
	}

	query = strings.TrimSuffix(query, ",")

	query += " WHERE email = ?"
	args = append(args, updatedData.Email)
	fmt.Println("this is the UpdateUser ", query, " kkk ", args)

	return s.repo.UpdateUser(ctx, query, args)
}
func (s *service) GenericApi(ctx context.Context, apiType, endQuery string) ([]map[string]interface{}, error) {
	query, err := s.repo.ApiQuery(apiType)
	fmt.Println("this is the GenericApi query", query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrive api's query: %w", err)
	}
	return s.repo.ExecuteUnJoinQuery(query + endQuery)

}
func (s *service) Listing(ctx context.Context) ([]model.ProductListingUsers, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:

		return s.repo.Listing(ctx)
	}
}
func (s *service) GetAttributes(ctx context.Context, attrType string) ([]model.Attribute, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		switch attrType {
		case "brand":
			return s.repo.GetAttributes(ctx, "brand")
		case "category":
			return s.repo.GetAttributes(ctx, "category")
		case "ItemSubType":
			return s.repo.GetAttributes(ctx, "ItemSubType")
		case "ItemMainType":
			return s.repo.GetAttributes(ctx, "ItemMainType")
			//case "category":
			return s.repo.GetAttributes(ctx, "category")
		default:
			return nil, fmt.Errorf("invalid attribute type: %s", attrType)
		}
	}
}

func isValidEmail(email string) bool {
	// Simple regex pattern for basic email validation
	fmt.Println(" check email validity")
	const emailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}
func isValidPhoneNumber(phone string) bool {
	// Simple regex pattern for basic phone number validation
	fmt.Println(" check pfone validity")
	const phoneRegex = `^\+?[1-9]\d{1,14}$` // E.164 international phone number format
	re := regexp.MustCompile(phoneRegex)
	return re.MatchString(phone)
}
func (s *service) AddProduct(ctx context.Context, product model.Product) (model.AddProductResult, error) {
	s.addProductMu.Lock()
	defer s.addProductMu.Unlock()

	inventoryID := config.InventoryID
	product.InventoryID = &inventoryID

	// A negative unit count removes that many units of an existing product
	// that are in the given status (category) instead of adding.
	if product.Unit < 0 {
		return s.removeProductUnits(product.Name, product.Category, -product.Unit)
	}

	productCode, err := s.resolveProductCode(product.Name)
	if err != nil {
		return model.AddProductResult{}, err
	}

	// Get the latest item_code issued under this product code, so the new
	// units continue the per-product sequence instead of restarting at 1.
	query := fmt.Sprintf(
		"SELECT item_code FROM items WHERE product_code = '%s' ORDER BY created_at DESC LIMIT 1;",
		productCode,
	)

	latestRow, _ := s.repo.RetrieveSingleVal(query)
	fmt.Println("--lastrww---", latestRow)
	//if err != nil {
	//	return nil, err
	//}
	checkNewMain := fmt.Sprintf(
		"SELECT 1 FROM attributes WHERE type = 'ItemMainType' AND name = '%s' ORDER BY created_at DESC LIMIT 1;",
		product.ItemMainType,
	)

	exists, _ := s.repo.Exists(checkNewMain)
	//if err != nil {
	//	return nil, err
	//}
	if !exists {
		if s.util.UtilRepository.AddAttribute("ItemMainType", product.ItemMainType) != nil {
			return model.AddProductResult{}, fmt.Errorf("failed to add new ItemMainType")
		}

	}
	checkNewSub := fmt.Sprintf(
		"SELECT 1 FROM attributes WHERE type = 'ItemSubType' AND name = '%s' ORDER BY created_at DESC LIMIT 1;",
		product.NewSubCode,
	)

	exists, _ = s.repo.Exists(checkNewSub)
	//if err != nil {
	//	return nil, err
	//}
	if !exists {
		if s.util.UtilRepository.AddAttribute("ItemSubType", product.NewSubCode) != nil {
			return model.AddProductResult{}, fmt.Errorf("failed to add new sub code")
		}

	}
	checkNewBrand := fmt.Sprintf(
		"SELECT 1 FROM attributes WHERE type = 'brand' AND name = '%s' ORDER BY created_at DESC LIMIT 1;",
		product.Brand,
	)

	exists, _ = s.repo.Exists(checkNewBrand)
	//if err != nil {
	//	return nil, err
	//}
	if !exists {
		if s.util.UtilRepository.AddAttribute("brand", product.Brand) != nil {
			return model.AddProductResult{}, fmt.Errorf("failed to add new brand")
		}

	}

	// startSeq is the per-unit sequence already used under this product
	// code, so a second AddProduct call for the same product continues
	// numbering the units instead of restarting at -001.
	startSeq := 0
	if latestRow != nil {
		if latestItemCode, ok := latestRow["item_code"].(string); ok {
			if idx := strings.LastIndex(latestItemCode, "-"); idx != -1 {
				if n, err := strconv.Atoi(latestItemCode[idx+1:]); err == nil {
					startSeq = n
				}
			}
		}
	}

	// prepare bulk items
	var items []model.Item
	var itemCodes []string

	for i := 1; i <= product.Unit; i++ {
		newSeq := startSeq + i
		newItemCode := fmt.Sprintf("%s-%03d", productCode, newSeq)

		item := model.Item{
			ItemCode:     newItemCode,
			ProductCode:  productCode,
			ItemName:     product.Name,
			ItemMainType: product.ItemMainType,
			ItemSubType:  product.NewSubCode,
			Brand:        product.Brand,
			Category:     product.Category,
			Description:  product.Description,
			InventoryID:  product.InventoryID,
			CreatedAt:    time.Now(),
			HsnCode:      product.HsnCode,
		}
		items = append(items, item)
		itemCodes = append(itemCodes, newItemCode)
	}

	// bulk insert
	if err := s.repo.AddProductsBulk(items); err != nil {
		fmt.Println("----addd errrr---", err.Error())
		return model.AddProductResult{}, err
	}

	return model.AddProductResult{ProductCode: productCode, ItemCodes: itemCodes}, nil
}

// ErrItemInOrder is returned by AddProduct when a unit it was asked to
// remove is part of an order.
var ErrItemInOrder = errors.New("item exists in order")

// removeProductUnits deletes the `count` most recently added units of the
// product named itemName that are in `status`. It fails with
// ErrItemInOrder, deleting nothing, if any of those units appears in an
// order.
func (s *service) removeProductUnits(itemName, status string, count int) (model.AddProductResult, error) {
	existingQuery := fmt.Sprintf(
		"SELECT product_code FROM items WHERE LOWER(TRIM(item_name)) = LOWER(TRIM('%s')) ORDER BY created_at DESC LIMIT 1;",
		sqlEscape(itemName),
	)
	row, _ := s.repo.RetrieveSingleVal(existingQuery)
	productCode, _ := row["product_code"].(string)
	if productCode == "" {
		return model.AddProductResult{}, fmt.Errorf("product %q not found", itemName)
	}

	removed, inOrder, err := s.repo.RemoveProductItems(productCode, status, count)
	if err != nil {
		return model.AddProductResult{}, err
	}
	if len(inOrder) > 0 {
		return model.AddProductResult{}, fmt.Errorf("%w: %s", ErrItemInOrder, strings.Join(inOrder, ", "))
	}

	return model.AddProductResult{ProductCode: productCode, ItemCodes: []string{}, RemovedItemCodes: removed}, nil
}

// resolveProductCode returns the product_code for a product name: the
// existing code if a product with this name has been added before (matched
// case-insensitively), or the next sequential IRR#### code if this is the
// first time it's being added.
func (s *service) resolveProductCode(itemName string) (string, error) {
	existingQuery := fmt.Sprintf(
		"SELECT product_code FROM items WHERE LOWER(TRIM(item_name)) = LOWER(TRIM('%s')) ORDER BY created_at DESC LIMIT 1;",
		sqlEscape(itemName),
	)
	if row, _ := s.repo.RetrieveSingleVal(existingQuery); row != nil {
		if code, ok := row["product_code"].(string); ok && code != "" {
			return code, nil
		}
	}

	latestCodeQuery := fmt.Sprintf(
		`SELECT product_code FROM items WHERE product_code ~ '^%s[0-9]{4}$' ORDER BY product_code DESC LIMIT 1;`,
		constants.ProductCodePrefix,
	)
	row, _ := s.repo.RetrieveSingleVal(latestCodeQuery)

	seq := 0
	if row != nil {
		if code, ok := row["product_code"].(string); ok && len(code) > len(constants.ProductCodePrefix) {
			if n, err := strconv.Atoi(code[len(constants.ProductCodePrefix):]); err == nil {
				seq = n
			}
		}
	}

	return fmt.Sprintf("%s%04d", constants.ProductCodePrefix, seq+1), nil
}

// sqlEscape guards against a quote in a user-supplied value breaking out of
// the quoted literal in this file's hand-built queries. It is not a general
// SQL-injection defense, just protection against an accidental or malicious
// apostrophe in a product name.
func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// UploadItemSheet runs each row of a parsed item sheet through the exact
// same product_code/item_code logic as AddProduct — one call per row, in
// sheet order, so the per-product unit sequence stays correct when two rows
// share a product name. Each row's AddProductsBulk insert commits on its
// own (there is no single transaction spanning the whole sheet), so if a
// row fails partway through, the rows before it have already been saved;
// the returned result always reflects what was actually committed so far.
func (s *service) UploadItemSheet(ctx context.Context, rows []model.ItemSheetRow) (model.UploadItemSheetResult, error) {
	var result model.UploadItemSheetResult

	for _, row := range rows {
		addResult, err := s.AddProduct(ctx, row.Product)
		if err != nil {
			return result, fmt.Errorf("row %d (%q): %w", row.Row, row.Product.Name, err)
		}

		result.Results = append(result.Results, model.UploadItemSheetRowResult{
			Row:         row.Row,
			ItemName:    row.Product.Name,
			ProductCode: addResult.ProductCode,
			ItemCodes:   addResult.ItemCodes,
		})
		result.RowsProcessed++
		result.ItemsCreated += len(addResult.ItemCodes)
	}

	return result, nil
}

// GetProductStockInfo returns the shared, product-level attributes
// (item_name/brand/description/hsn_code) for a product_code.
func (s *service) GetProductStockInfo(productCode string) (model.ProductStockInfo, error) {
	if productCode == "" {
		return model.ProductStockInfo{}, fmt.Errorf("product_code is required")
	}
	info, err := s.repo.GetProductStockInfo(productCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.ProductStockInfo{}, fmt.Errorf("no product found for product code %s", productCode)
		}
		return model.ProductStockInfo{}, err
	}
	return info, nil
}

func (s *service) GenericStatusUpdate(update model.GenericUpdate) error {
	query := fmt.Sprintf(
		"UPDATE %s SET %s = '%s' WHERE %s = '%s';",
		update.Table,
		update.Field,
		update.Status,
		update.Key,
		update.IDvalue,
	)

	return s.util.UtilRepository.ExecQuery(query)
}
func (s *service) AddOrder(order model.DeliveryOrder) (string, error) {
	fmt.Println("Add order------")
	if !(order.Status == "INITIATED" || order.Status == "RESERVED") {
		return "", fmt.Errorf("invalid order type: %s", order.Status)
	}
	order.InventoryID = config.InventoryID.String()

	// Prepare main order
	ch := &model.DeliveryChelan{
		CustomerID:      order.CustomerID,
		InventoryID:     order.InventoryID,
		AdvanceAmount:   order.AdvanceAmount,
		Discount:        order.Discount,
		Status:          order.Status,
		ContactName:     order.ContactName,
		ContactNumber:   order.ContactNumber,
		ShippingAddress: order.ShippingAddress,
	}
	if order.Status == "INITIATED" {
		ch.PlacedAt = time.Now()
	}

	// Update amounts for items
	updateGeneratedAmount(order.Items)

	// Calculate totals for main order
	ch.GeneratedAmount, ch.CurrentAmount = retriveMainOrderAmt(order.Items)
	ch.GeneratedAmount -= ch.Discount
	if ch.GeneratedAmount < 0 {
		ch.GeneratedAmount = 0
	}

	// Add main order — returns delivery_id and auto-generated invoice_number
	id, invoiceNumber, err := s.repo.AddMainOrder(*ch)
	if err != nil {
		fmt.Println("Error adding main order:", err)
		return "", err
	}

	// Add delivery items
	for _, item := range order.Items {
		_, err := s.repo.AddDeliveryItem(item, id, order.CustomerID, order.InventoryID)
		if err != nil {
			fmt.Println("Error adding delivery item:", err)
			return "", fmt.Errorf("failed to add delivery item: %w", err)
		}

		query := fmt.Sprintf(
			"update items set category ='RENTED' where item_id='%s';",
			item.ItemNewId,
		)
		err = s.util.UtilRepository.ExecQuery(query)
		if err != nil {
			return "", err
		}
	}

	err = s.repo.AddMainTransaction(id, "PENDING")
	if err != nil {
		fmt.Println("Error adding main transaction:", err)
		return "", err
	}
	return invoiceNumber, nil
}

// Helper to get pointer to time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}

// Update generated and current amounts safely
func updateGeneratedAmount(items []model.DeliveryItemHandler) {
	now := time.Now()

	for i := range items {
		item := &items[i]

		// Parse ReturnedAt if not set
		if item.ReturnedAt == nil && item.ReturnedStr != "" {
			parsed, err := time.Parse("2006-01-02", item.ReturnedStr)
			if err != nil {
				fmt.Println("parse error for ReturnedStr:", err)
				t := now.AddDate(0, 0, 1)
				item.ReturnedAt = &t
			} else {
				item.ReturnedAt = &parsed
			}
		}

		// Ensure PlacedAt is set
		if item.PlacedAt == nil {
			item.PlacedAt = &now
		}

		// Safety: skip if still nil
		if item.PlacedAt == nil || item.ReturnedAt == nil {
			fmt.Println("Skipping item due to nil timestamps")
			continue
		}

		// Days till expiry
		daysTotal := int(item.ReturnedAt.Sub(*item.PlacedAt).Hours() / 24)
		if daysTotal <= 0 {
			daysTotal = 1
		}
		item.GeneratedAmount = item.RentAmount * daysTotal

		// Days till now
		daysSoFar := int(now.Sub(*item.PlacedAt).Hours() / 24)
		if daysSoFar <= 0 {
			daysSoFar = 1
		}
		if now.After(*item.ReturnedAt) {
			daysSoFar = daysTotal
		}
		item.CurrentAmount = item.RentAmount * daysSoFar
	}
}

// Retrieve total amounts for main order
func retriveMainOrderAmt(items []model.DeliveryItemHandler) (int, int) {
	generatedAmt, currentAmt := 0, 0
	for _, item := range items {
		generatedAmt += item.GeneratedAmount
		currentAmt += item.CurrentAmount
	}
	return generatedAmt, currentAmt
}
func (s *service) UpdateMainOrderStatus(orderID, typestat string) error {
	fmt.Println("UpdateMainOrderStatus:", orderID, typestat)
	if !(typestat == "DELETED" || typestat == "COMPLETED" || typestat == "BLOCKED" || typestat == "INITIATED" || typestat == "RESERVED") {

		return fmt.Errorf("invalid order type: %s", typestat)

	}
	if typestat == "COMPLETED" {
		flag, err := s.repo.AreAllItemsCompleted(orderID)
		if err != nil {
			return err
		}
		if !flag {
			return fmt.Errorf("order %s items not all completed", orderID)
		}
	}
	// retrive order items
	checkNewBrand := fmt.Sprintf(
		"SELECT item_id FROM public.delivery_items where order_id='%s';",
		orderID,
	)
	ctx := context.Background()
	data, err := s.repo.GetOrderItems(ctx, checkNewBrand)
	if err != nil {
		return err
	}
	if typestat == "DELETED" {
		err = s.repo.DeleteEntry("delivery_items", "order_id", orderID)

	}
	for _, item := range data {
		if typestat == "DELETED" || typestat == "COMPLETED" {
			query := fmt.Sprintf(
				"update items set category ='AVAILABLE' where item_id='%s';",
				item,
			)
			err = s.util.UtilRepository.ExecQuery(query)
			if err != nil {
				return err
			}

		} else if typestat == "RESERVED" || typestat == "INITIATED" {
			query := fmt.Sprintf(
				"update items set category ='%s' where item_id='%s';",
				typestat, item,
			)
			err = s.util.UtilRepository.ExecQuery(query)
			if err != nil {
				return err
			}
		}

	}
	query := ""
	itemquery := ""
	if typestat == "INITIATED" {
		itemquery = fmt.Sprintf(`
		UPDATE delivery_items
		SET placed_at = NOW(),
		    generated_amount = rent_amount * GREATEST(1, EXTRACT(DAY FROM (COALESCE(returned_at, NOW() + INTERVAL '1 day') - NOW()))::int),
		    current_amount = rent_amount
		WHERE order_id = '%s' AND status = 'RESERVED';
	`, orderID)
		query = fmt.Sprintf(`
		UPDATE delivery_chelan
		SET placed_at = NOW(), status = '%s',
		    generated_amount = GREATEST(0, (SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_items WHERE order_id = '%s') - COALESCE(discount, 0)),
		    current_amount = (SELECT COALESCE(SUM(current_amount), 0) FROM delivery_items WHERE order_id = '%s')
		WHERE delivery_id = '%s' AND status = 'RESERVED';
	`, typestat, orderID, orderID, orderID)
	} else if typestat == "COMPLETED" {
		itemquery = ""
		query = fmt.Sprintf(`
		UPDATE delivery_chelan
		SET status = '%s', declined_at = NOW() ,
		    generated_amount = GREATEST(0, (SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_items WHERE order_id = '%s') - COALESCE(discount, 0))
		WHERE delivery_id = '%s';
	`, typestat, orderID, orderID)
	} else {
		itemquery = ""
		query = fmt.Sprintf(
			"update delivery_chelan set status ='%s' where delivery_id='%s';",
			typestat, orderID,
		)
	}
	if itemquery != "" {
		err = s.util.UtilRepository.ExecQuery(itemquery)
		if err != nil {
			return err
		}
	}
	err = s.util.UtilRepository.ExecQuery(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) UpdateItemStatus(itemID, typestat string) error {
	fmt.Println("UpdateMainOrderStatus:", itemID, typestat)
	if !(typestat == "DAMAGED" || typestat == "REPAIRING" || typestat == "EXPIRED" || typestat == "AVAILABLE" || typestat == "SITE") {

		return fmt.Errorf("invalid order type: %s", typestat)

	}
	checkRented := fmt.Sprintf(
		"SELECT 1 FROM public.items where item_id='%s' AND category='RENTED';",
		itemID,
	)
	flg, _ := s.repo.Exists(checkRented)

	if flg {

		return fmt.Errorf("for item %s status can't be changed for rented", itemID)

	}
	query := fmt.Sprintf(
		"update items set category ='%s' where item_id='%s';",
		typestat, itemID,
	)

	err := s.util.UtilRepository.ExecQuery(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) InsertGeneric(genReq map[string]interface{}) error {

	return nil
}

// AddSubTransaction adds a transaction row for an order and returns the
// invoice_id the server generated for it.
func (s *service) AddSubTransaction(trx model.Transaction) (string, error) {
	fmt.Println("Add sub transaction------")

	invoiceID, err := s.repo.AddSubTransaction(trx)
	if err != nil {
		fmt.Println("Error adding sub transaction:", err)
		return "", err
	}

	return invoiceID, nil
}

// GetSubTransactions returns every transaction row, optionally filtered to
// one order.
func (s *service) GetSubTransactions(orderID string) ([]model.Transaction, error) {
	return s.repo.GetSubTransactions(orderID)
}

//	func (s *service) AddOrder(order model.DeliveryOrder) error {
//		fmt.Println(" add order------")
//		ch := &model.DeliveryChelan{
//			//DeliveryID: req.DeliveryID,
//			CustomerID: order.CustomerID,
//			//InventoryID:     order.InventoryID,
//			AdvanceAmount:   order.AdvanceAmount,
//			Status:          order.Status,
//			InventoryID:     order.InventoryID,
//			ContactName:     order.ContactName,
//			ContactNumber:   order.ContactNumber,
//			ShippingAddress: order.ShippingAddress,
//		}
//		//layout := "02012006"
//		//ch.ExpiryAt, err = time.Parse(layout, order.ExpiryAt)
//		//if err != nil {
//		//	log.Fatal(err)
//		//}
//		//	if order.Status == "INITIATED" {
//		ch.PlacedAt = time.Now()
//		updateGeneratedAmount(&order.Items)
//		fmt.Println(" add order2222222")
//		ch.GeneratedAmount, ch.CurrentAmount, _ = retriveMainOrderAmt(order.Items)
//		id, err := s.repo.AddMainOrder(*ch)
//		fmt.Println("-----", id, "---err--", err.Error())
//		if err != nil {
//			return err
//		}
//		for _, item := range order.Items {
//			_, err := s.repo.AddDeliveryItem(item, id)
//			if err != nil {
//				fmt.Println("----addd errrr---", err.Error())
//				return fmt.Errorf("failed to add delivery item: %w", err)
//			}
//		}
//		//	}
//		//tx, err := s.repo.StartTransaction()
//
//		//if err != nil {
//		//	return err
//		//}
//
//		return nil
//	}
//
//	func retriveMainOrderAmt(items []model.DeliveryItem) (int, int, error) {
//		generatedAmt, currentAmt := 0, 0
//		for _, item := range items {
//			generatedAmt = generatedAmt + item.GeneratedAmount
//			currentAmt = currentAmt + item.CurrentAmount
//		}
//		return generatedAmt, currentAmt, nil
//	}
//
//	func updateGeneratedAmount(orders *[]model.DeliveryItem) {
//		for i := range *orders {
//			item := &(*orders)[i]
//			now := time.Now()
//
//			// Parse ReturnedAt from ReturnedStr
//			if item.ReturnedStr != "" && item.ReturnedAt == nil {
//				parsedTime, err := time.Parse("2006-01-02", item.ReturnedStr)
//				if err != nil {
//					fmt.Println("parse error:", err)
//					tmp := now.AddDate(0, 0, 1)
//					item.ReturnedAt = &tmp
//				} else {
//					item.ReturnedAt = &parsedTime
//				}
//			}
//
//			// If PlacedAt is nil, set it to now
//			if item.PlacedAt == nil {
//				item.PlacedAt = &now
//			}
//
//			// Safely calculate daysTotal
//			daysTotal := 1
//			if item.ReturnedAt != nil && item.PlacedAt != nil {
//				daysTotal = int(item.ReturnedAt.Sub(*item.PlacedAt).Hours() / 24)
//				if daysTotal <= 0 {
//					daysTotal = 1
//				}
//			}
//
//			// GeneratedAmount
//			item.GeneratedAmount = item.RentAmount * daysTotal
//
//			// daysSoFar
//			daysSoFar := int(now.Sub(*item.PlacedAt).Hours() / 24)
//			if daysSoFar <= 0 {
//				daysSoFar = 1
//			}
//
//			if item.ReturnedAt != nil && now.After(*item.ReturnedAt) {
//				daysSoFar = daysTotal
//			}
//
//			item.CurrentAmount = item.RentAmount * daysSoFar
//		}
//	}
func (s *service) UpdateOrderItemStatus(req model.OrderItemUpdate) error {
	fmt.Println("UpdateOrderItemStatus:")
	retriveitemquery := fmt.Sprintf(
		"SELECT item_id FROM public.delivery_items where delivery_item_id='%s';",
		req.ItemID,
	)

	retriveItemid, _ := s.repo.RetrieveSingleVal(retriveitemquery)
	if req.Status == "COMPLETED" {
		query := fmt.Sprintf(
			"update items set category ='AVAILABLE' where item_id='%s';",
			retriveItemid["item_id"].(string),
		)
		err := s.util.UtilRepository.ExecQuery(query)
		if err != nil {
			return err
		}
		imgArr := []string{req.AfterImage}
		fmt.Println("------imgarrr", imgArr)
		err = s.repo.UpdateOrderItemImg(req.ItemID, req.Status, imgArr)
		if err != nil {
			fmt.Println("Error updating order item status:", err)
			return err
		}
		debugQuery := fmt.Sprintf(`
    UPDATE delivery_items
    SET status = '%s',
        after_images = '{"%s"}',
        returned_at = NOW(),
        generated_amount = rent_amount * GREATEST(1, EXTRACT(DAY FROM (NOW() - placed_at))::int)
    WHERE delivery_item_id = '%s';
`,
			req.Status,
			req.AfterImage,
			req.ItemID,
		)

		fmt.Println("Debug query:", debugQuery)

		err = s.util.UtilRepository.ExecQuery(debugQuery)
		if err != nil {
			fmt.Println("Error updating order item status:", err)
		}

		// Also update the main order's generated amount
		updateMainOrderQuery := fmt.Sprintf(`
		UPDATE delivery_chelan
		SET generated_amount = GREATEST(0, (SELECT COALESCE(SUM(generated_amount), 0) FROM delivery_items WHERE order_id = (SELECT order_id FROM delivery_items WHERE delivery_item_id = '%s')) - COALESCE(discount, 0))
		WHERE delivery_id = (SELECT order_id FROM delivery_items WHERE delivery_item_id = '%s');
		`, req.ItemID, req.ItemID)
		s.util.UtilRepository.ExecQuery(updateMainOrderQuery)
		//checkNewBrand := fmt.Sprintf(
		//	"SELECT item_id FROM public.delivery_items where order_id='%s';",
		//	orderID,
		//)
	} else if req.Status == "BLOCKED" {
		query := fmt.Sprintf(
			"update items set category ='BLOCKED' where item_id='%s';",
			retriveItemid["item_id"].(string),
		)
		err := s.util.UtilRepository.ExecQuery(query)
		if err != nil {
			return err
		}
		queryN := fmt.Sprintf(
			"update delivery_items set status ='BLOCKED' where delivery_item_id='%s';",
			req.ItemID,
		)
		err = s.util.UtilRepository.ExecQuery(queryN)
		if err != nil {
			return err
		}
	}
	//// retrive order items
	//checkNewBrand := fmt.Sprintf(
	//	"SELECT item_id FROM public.delivery_items where order_id='%s';",
	//	orderID,
	//)
	//ctx := context.Background()
	//data, err := s.repo.GetOrderItems(ctx, checkNewBrand)
	//if err != nil {
	//	return err
	//}
	//if typestat == "DELETED" {
	//	err = s.repo.DeleteEntry("delivery_items", "order_id", orderID)
	//
	//}
	//for _, item := range data {
	//	if typestat == "DELETED" || typestat == "COMPLETED" {
	//		query := fmt.Sprintf(
	//			"update items set category ='AVAILABLE' where item_id='%s';",
	//			item,
	//		)
	//		err = s.util.UtilRepository.ExecQuery(query)
	//		if err != nil {
	//			return err
	//		}
	//		//query1 := fmt.Sprintf(
	//		//	"update delivery_items set status ='AVAILABLE' where item_id='%s' AND order_id='%s';",
	//		//	item,orderID,
	//		//)
	//		//err = s.util.UtilRepository.ExecQuery(query1)
	//		//if err != nil {
	//		//	return err
	//		//}
	//	} else if typestat == "RESERVED" || typestat == "INITIATED" {
	//		query := fmt.Sprintf(
	//			"update items set category ='%s' where item_id='%s';",
	//			typestat, item,
	//		)
	//		err = s.util.UtilRepository.ExecQuery(query)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//
	//}
	//query := ""
	//if typestat == "INITIATED" {
	//	date := time.Now()
	//	query = fmt.Sprintf(`
	//	UPDATE delivery_chelan
	//	SET placed_at = '%s', status = '%s'
	//	WHERE delivery_id = '%s' AND status = 'RESERVED';
	//`, date.Format("2006-01-02 15:04:05"), typestat, orderID)
	//} else {
	//	query = fmt.Sprintf(
	//		"update delivery_chelan set status ='%s' where delivery_id='%s';",
	//		typestat, orderID,
	//	)
	//}
	//err = s.util.UtilRepository.ExecQuery(query)
	//if err != nil {
	//	return err
	//}
	return nil
}
func (s *service) CreateCustomer(customer model.Customer) error {
	// You can add business logic here (validations, transformations, etc.)
	if customer.Name == "" || customer.Phone == "" {
		return fmt.Errorf("name and phone are required")
	}
	return s.repo.CreateCustomer(customer)
}

// allowedDeleteTables is an allowlist of tables that may be deleted via the generic endpoint.
var allowedDeleteTables = map[string]bool{
	"delivery_items":   true,
	"delivery_chelan":  true,
	"items":            true,
	"attributes":       true,
	"customer":         true,
	"main_transaction": true,
	"transaction":      true,
}

func retrieveKey(table string) string {
	switch table {
	case "items":
		return "item_id"
	default:
		return table + "_id"
	}
}

func (s *service) GenericDelete(req model.GenericDelete) error {
	if req.TableName == "" || req.ID == "" {
		return fmt.Errorf("table_name and id are required")
	}
	if !allowedDeleteTables[req.TableName] {
		return fmt.Errorf("delete not allowed on table: %s", req.TableName)
	}
	fmt.Println("this is the query to delete entry", req.TableName, req.TableName+"_id", req.ID)

	return s.repo.DeleteEntry(req.TableName, retrieveKey(req.TableName), req.ID)
}

func (s *service) UpdateCurrentAmounts() error {
	return s.repo.UpdateCurrentAmounts()
}

func (s *service) MarkItemDamage(itemID, itemDeliveryId string, damageImages []string, description string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	return s.repo.MarkItemDamage(itemID, itemDeliveryId, damageImages, description)
}

func (s *service) ClearItemDamage(itemID string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	return s.repo.ClearItemDamage(itemID)
}

func (s *service) MarkItemRepairing(itemID string, repairImages []string, amount int, description string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	return s.repo.MarkItemRepairing(itemID, repairImages, amount, description)
}

func (s *service) ClearItemRepairing(itemID string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	return s.repo.ClearItemRepairing(itemID)
}

func (s *service) GetRepairHistory(itemID string) ([]model.RepairHistory, error) {
	if itemID == "" {
		return nil, fmt.Errorf("item_id is required")
	}
	fmt.Println("this is the item_id to get repair history", itemID)
	return s.repo.GetRepairHistory(itemID)
}

func (s *service) GetDamagedGrouped() ([]model.ItemGroupCount, error) {
	return s.repo.GetDamagedGrouped()
}

func (s *service) GetDamagedList(productCode string) ([]model.Item, error) {
	return s.repo.GetDamagedList(productCode)
}

func (s *service) GetRepairingGrouped() ([]model.ItemGroupCount, error) {
	return s.repo.GetRepairingGrouped()
}

func (s *service) GetRepairingList(productCode string) ([]model.Item, error) {
	return s.repo.GetRepairingList(productCode)
}

func (s *service) GenerateDeliveryReportExcel(filter model.DeliveryReportFilter) ([]byte, error) {
	data, err := s.repo.GetDeliveryReport(filter)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	// Header row
	headers := []string{
		"S No", "DC No/Date", "Name of the Party", "Items",
		"Rate", "Advance recd", "Tool Hire", "Balance due",
		"Date", "Remarks",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Data rows
	for rIndex, row := range data {
		values := []interface{}{
			row.SNo,
			row.DCNoDate,
			row.PartyName,
			row.Items, // newline or comma separated from query
			row.Rate,
			row.AdvanceRecd,
			row.ToolHire,
			row.BalanceDue,
			row.Date,
			row.Remarks,
		}

		for cIndex, v := range values {
			cell, _ := excelize.CoordinatesToCellName(cIndex+1, rIndex+2)
			f.SetCellValue(sheet, cell, v)

			// Allow text wrapping for Items column
			if cIndex == 3 { // "Items" column index
				styleID, err := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{
						WrapText: true,
					},
				})
				if err != nil {
					panic(err)
				}
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
		}
	}

	// --- Auto-size columns ---
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for _, col := range cols {
		maxLen := 10
		for row := 1; row <= len(data)+1; row++ { // +1 for header
			cell, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", col, row))
			if len(cell) > maxLen {
				maxLen = len(cell)
			}
		}
		_ = f.SetColWidth(sheet, col, col, float64(maxLen)+2)
	}

	// Save to memory
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *service) InitiateMainOrder(orderID string, guaranteeImages []string) error {
	typestat := "INITIATED"
	ctx := context.Background()

	// 1. check if items are in order, update category
	checkNewBrand := fmt.Sprintf("SELECT item_id FROM public.delivery_items where order_id='%s';", orderID)
	data, err := s.repo.GetOrderItems(ctx, checkNewBrand)
	if err != nil {
		return err
	}
	for _, item := range data {
		query := fmt.Sprintf("update items set category ='%s' where item_id='%s';", typestat, item)
		err = s.util.UtilRepository.ExecQuery(query)
		if err != nil {
			return err
		}
	}

	// 2. run item query
	itemquery := fmt.Sprintf(`
		UPDATE delivery_items
		SET placed_at = NOW(), status = '%s',
		    generated_amount = rent_amount * GREATEST(1, EXTRACT(DAY FROM (COALESCE(returned_at, NOW() + INTERVAL '1 day') - NOW()))::int),
		    current_amount = rent_amount
		WHERE order_id = '%s' AND status = 'RESERVED';
	`, typestat, orderID)

	err = s.util.UtilRepository.ExecQuery(itemquery)
	if err != nil {
		return err
	}

	// 3. update main order
	err = s.repo.UpdateOrderToInitiated(orderID, guaranteeImages)
	return err
}

func (s *service) UpdateOrderPass(req model.OrderPassRequest) error {
	return s.repo.UpdateOrderPass(req)
}

func (s *service) UpdateOrderReturnedAt(orderID string, returnedAt time.Time) error {
	return s.repo.UpdateOrderReturnedAt(orderID, returnedAt)
}

func (s *service) UpdateOrderItemReturnedAt(deliveryItemID string, returnedAt time.Time) error {
	return s.repo.UpdateOrderItemReturnedAt(deliveryItemID, returnedAt)
}

func (s *service) GetCustomerDamageAnalytics() ([]model.CustomerDamageAnalytics, error) {
	return s.repo.GetCustomerDamageAnalytics()
}

func (s *service) GetItemDamageAnalytics() ([]model.ItemDamageAnalytics, error) {
	return s.repo.GetItemDamageAnalytics()
}

func (s *service) GetCustomerBlockedAnalytics() ([]model.CustomerBlockedAnalytics, error) {
	return s.repo.GetCustomerBlockedAnalytics()
}

func (s *service) GetItemRentalAnalytics() ([]model.ItemRentalAnalytics, error) {
	return s.repo.GetItemRentalAnalytics()
}

func (s *service) GetOrderStatusSummary() ([]model.OrderStatusSummary, error) {
	return s.repo.GetOrderStatusSummary()
}

func (s *service) GetRevenueByCustomer() ([]model.RevenueByCustomer, error) {
	return s.repo.GetRevenueByCustomer()
}

func (s *service) GetCostSummary() (model.CostSummary, error) {
	return s.repo.GetCostSummary()
}

func (s *service) GetRevenueTrend(period string, limit int) ([]model.RevenueTrend, error) {
	return s.repo.GetRevenueTrend(period, limit)
}

func (s *service) GetRepairCostAnalytics() ([]model.RepairCostAnalytics, error) {
	return s.repo.GetRepairCostAnalytics()
}

func (s *service) GetOutstandingBalances() ([]model.OutstandingBalance, error) {
	return s.repo.GetOutstandingBalances()
}

func (s *service) GetInventoryRevenue() ([]model.InventoryRevenue, error) {
	return s.repo.GetInventoryRevenue()
}

func (s *service) CreateQuotation(req model.QuotationRequest) (string, error) {
	return s.repo.CreateQuotation(req)
}

func (s *service) GetQuotations() ([]model.Quotation, error) {
	return s.repo.GetQuotations()
}

func (s *service) GetQuotationByID(id string) (model.Quotation, error) {
	return s.repo.GetQuotationByID(id)
}

func (s *service) GetOrderDetails(orderID string) (model.OrderDetails, error) {
	return s.repo.GetOrderDetails(orderID)
}

func (s *service) MarkItemSite(itemID string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	checkRented := fmt.Sprintf(
		"SELECT 1 FROM public.items where item_id='%s' AND category='RENTED';",
		itemID,
	)
	if rented, _ := s.repo.Exists(checkRented); rented {
		return fmt.Errorf("item %s is currently rented and cannot be moved to SITE", itemID)
	}
	return s.repo.MarkItemSite(itemID)
}

func (s *service) ClearItemSite(itemID string) error {
	if itemID == "" {
		return fmt.Errorf("item_id is required")
	}
	return s.repo.ClearItemSite(itemID)
}

func (s *service) GetSiteGrouped() ([]model.ItemGroupCount, error) {
	return s.repo.GetSiteGrouped()
}

func (s *service) GetSiteList(productCode string) ([]model.Item, error) {
	return s.repo.GetSiteList(productCode)
}

func (s *service) GetAvailableItemByProductCode(productCode string) (model.Item, error) {
	if productCode == "" {
		return model.Item{}, fmt.Errorf("product_code is required")
	}
	item, err := s.repo.ReserveRandomAvailableItem(productCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Item{}, fmt.Errorf("no available item found for product code %s", productCode)
		}
		return model.Item{}, err
	}
	return item, nil
}
