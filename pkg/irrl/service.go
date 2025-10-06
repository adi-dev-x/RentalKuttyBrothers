package irrl

import (
	"context"
	"fmt"
	services "myproject/pkg/client"
	"myproject/pkg/config"
	"myproject/pkg/model"
	"myproject/pkg/util"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, request model.UserRegisterRequest) error
	Login(ctx context.Context, request model.UserLoginRequest) error
	//Product listing
	Listing(ctx context.Context) ([]model.ProductListingUsers, error)
	AddProduct(ctx context.Context, product model.Product) ([]model.Attribute, error)
	AddOrder(order model.DeliveryOrder) error
	GetAttributes(ctx context.Context, typ string) ([]model.Attribute, error)
	OtpLogin(ctx context.Context, request model.UserOtp) error
	UpdateUser(ctx context.Context, updatedData model.UserRegisterRequest) error

	VerifyOtp(ctx context.Context, email string)
	GenericApi(ctx context.Context, apiType, endQuery string) ([]map[string]interface{}, error)
	GenericStatusUpdate(update model.GenericUpdate) error
	UpdateMainOrderStatus(orderID, typestat string) error
	UpdateItemStatus(orderID, typestat string) error
	InsertGeneric(genReq map[string]interface{}) error
	AddSubTransaction(transaction model.Transaction) error
	UpdateOrderItemStatus(req model.OrderItemUpdate) error
	CreateCustomer(customer model.Customer) error
}
type service struct {
	repo     Repository
	Config   config.Config
	services services.Services
	util     *util.Initiator
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
	fmt.Println("-----reg req----", request)
	if request.Name == "" || request.Email == "" || request.Password == "" {
		fmt.Println("this is in the service error value missing")
		err = fmt.Errorf("missing values")
		return err
	}
	if !isValidEmail(request.Email) {
		fmt.Println("this is in the service error invalid email")
		err = fmt.Errorf("invalid email")
		return err
	}
	// validate phone only if provided
	if request.Phone != "" {
		if !isValidPhoneNumber(request.Phone) {
			fmt.Println("this is in the service error invalid phone number")
			err = fmt.Errorf("invalid phone number")
			return err
		}
	}
	fmt.Println("this is the dataaa ", request.Email)
	existingUser, err := s.repo.Login(ctx, request.Username)
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
	id, regErr := s.repo.Register(ctx, request)
	fmt.Println("d", id)
	if regErr != nil {
		return fmt.Errorf("failed to register user: %w", regErr)
	}
	return nil
}

func (s *service) Login(ctx context.Context, request model.UserLoginRequest) error {
	fmt.Println("this is in the service Login", request.Password)
	var err error
	if request.Username == "" || request.Password == "" {
		fmt.Println("this is in the service error value missing")
		err = fmt.Errorf("missing values")
		return err
	}
	storedUser, err := s.repo.Login(ctx, request.Username)
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

	if updatedData.Name != "" {
		query += " name = ?,"
		args = append(args, updatedData.Name)
	}
	// if updatedData.LastName != "" {
	// 	query += " lastname = ?,"
	// 	args = append(args, updatedData.LastName)
	// }
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
func (s *service) AddProduct(ctx context.Context, product model.Product) ([]model.Attribute, error) {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	// Get latest sub_code
	query := fmt.Sprintf(
		"SELECT item_code FROM items WHERE sub_code = '%s' ORDER BY created_at DESC LIMIT 1;",
		product.SubCode,
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
			return nil, fmt.Errorf("failed to add new ItemMainType")
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
			return nil, fmt.Errorf("failed to add new sub code")
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
			return nil, fmt.Errorf("failed to add new brand")
		}

	}
	// prefixes
	brandPrefix := strings.ToUpper(product.Brand)
	categoryPrefix := strings.ToUpper(product.ItemMainType)
	if len(brandPrefix) > 2 {
		brandPrefix = brandPrefix[:2]
	}
	if len(categoryPrefix) > 2 {
		categoryPrefix = categoryPrefix[:2]
	}
	year := time.Now().Format("2006")

	startSeq := 0
	if latestRow != nil {
		latestSub := latestRow["item_code"].(string)
		lastSeq := latestSub[len(latestSub)-4:] // last 4 digits
		startSeq, _ = strconv.Atoi(lastSeq)
	}

	// prepare bulk items
	var items []model.Item
	var createdSubs []model.Attribute

	for i := 1; i <= product.Unit; i++ {
		newSeq := startSeq + i
		newSubCode := fmt.Sprintf("%s%s%s%04d", brandPrefix, categoryPrefix, year, newSeq)

		item := model.Item{
			ItemCode:     newSubCode,
			SubCode:      product.SubCode,
			ItemName:     product.Name,
			ItemMainType: product.ItemMainType,
			ItemSubType:  product.NewSubCode,
			Brand:        product.Brand,
			Category:     product.Category,
			Description:  product.Description,
			InventoryID:  product.InventoryID,
			CreatedAt:    time.Now(),
			MainCode:     product.MainCode,
		}
		items = append(items, item)

	}

	// bulk insert
	if err := s.repo.AddProductsBulk(items); err != nil {
		fmt.Println("----addd errrr---", err.Error())
		return nil, err
	}

	return createdSubs, nil
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
func (s *service) AddOrder(order model.DeliveryOrder) error {
	fmt.Println("Add order------")
	if !(order.Status == "INITIATED" || order.Status == "RESERVED") {

		return fmt.Errorf("invalid order type: %s", order.Status)

	}
	// Prepare main order
	ch := &model.DeliveryChelan{
		CustomerID:      order.CustomerID,
		InventoryID:     order.InventoryID,
		AdvanceAmount:   order.AdvanceAmount,
		Status:          order.Status,
		ContactName:     order.ContactName,
		ContactNumber:   order.ContactNumber,
		ShippingAddress: order.ShippingAddress,
		//PlacedAt:        time.Now(), // set main order placed time
	}
	if order.Status == "INITIATED" {
		ch.PlacedAt = time.Now()
	}

	// Update amounts for items
	updateGeneratedAmount(order.Items)

	// Calculate totals for main order
	ch.GeneratedAmount, ch.CurrentAmount = retriveMainOrderAmt(order.Items)

	// Add main order
	id, err := s.repo.AddMainOrder(*ch)
	if err != nil {
		fmt.Println("Error adding main order:", err)
		return err
	}

	// Add delivery items
	for _, item := range order.Items {
		_, err := s.repo.AddDeliveryItem(item, id, order.CustomerID, order.InventoryID)
		if err != nil {
			fmt.Println("Error adding delivery item:", err)
			return fmt.Errorf("failed to add delivery item: %w", err)
		}

		query := fmt.Sprintf(
			"update items set category ='RENTED' where item_id='%s';",
			item.ItemNewId,
		)
		err = s.util.UtilRepository.ExecQuery(query)
		if err != nil {
			return err
		}
	}

	err = s.repo.AddMainTransaction(id, "PENDING")
	if err != nil {
		fmt.Println("Error adding main transaction:", err)
		return err
	}
	return nil
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
		date := time.Now()
		query = fmt.Sprintf(`
		UPDATE delivery_chelan
		SET placed_at = '%s', status = '%s'
		WHERE delivery_id = '%s' AND status = 'RESERVED';
	`, date.Format("2006-01-02 15:04:05"), typestat, orderID)
		itemquery = fmt.Sprintf(`
		UPDATE delivery_items
		SET placed_at = '%s', status = '%s'
		WHERE order_id = '%s' AND status = 'RESERVED';
	`, date.Format("2006-01-02 15:04:05"), typestat, orderID)
	} else {
		query = fmt.Sprintf(
			"update delivery_chelan set status ='%s' where delivery_id='%s';",
			typestat, orderID,
		)
		itemquery = fmt.Sprintf(
			"update delivery_items set status ='%s' where order_id='%s';",
			typestat, orderID,
		)
	}
	err = s.util.UtilRepository.ExecQuery(query)
	if err != nil {
		return err
	}
	err = s.util.UtilRepository.ExecQuery(itemquery)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) UpdateItemStatus(itemID, typestat string) error {
	fmt.Println("UpdateMainOrderStatus:", itemID, typestat)
	if !(typestat == "DAMAGED" || typestat == "REPAIRING" || typestat == "EXPIRED" || typestat == "AVAILABLE") {

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

func (s *service) AddSubTransaction(trx model.Transaction) error {
	fmt.Println("Add sub transaction------")

	err := s.repo.AddSubTransaction(trx.MainTransactionID, trx.Amount, trx.Image, trx.Status, trx.Type)
	if err != nil {
		fmt.Println("Error adding sub transaction:", err)
		return err
	}

	return nil
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
	//  retriveitemquery:= fmt.Sprintf(
	// 	 "SELECT item_id FROM public.delivery_items where delivery_item_id='%s';;",
	// 	 item,
	//  )
	// retriveItemid:=
	// if req.Status == "COMPLETED" {
	// 	query := fmt.Sprintf(
	// 		"update items set category ='AVAILABLE' where item_id='%s';",
	// 		item,
	// 	)
	// 	checkNewBrand := fmt.Sprintf(
	// 		"SELECT item_id FROM public.delivery_items where order_id='%s';",
	// 		orderID,
	// 	)
	// }
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
