package irrl

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	services "myproject/pkg/client"
	"myproject/pkg/config"
	"myproject/pkg/model"
	"myproject/pkg/util"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service interface {
	Register(ctx context.Context, request model.UserRegisterRequest) error
	Login(ctx context.Context, request model.UserLoginRequest) error
	//Product listing
	Listing(ctx context.Context) ([]model.ProductListingUsers, error)
	AddProduct(ctx context.Context, product model.Product) ([]model.Attribute, error)
	GetAttributes(ctx context.Context, typ string) ([]model.Attribute, error)
	OtpLogin(ctx context.Context, request model.UserOtp) error
	UpdateUser(ctx context.Context, updatedData model.UserRegisterRequest) error

	VerifyOtp(ctx context.Context, email string)
	GenericApi(ctx context.Context, apiType, endQuery string) ([]map[string]interface{}, error)
	GenericStatusUpdate(update model.GenericUpdate) error
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
func (s *service) AddProduct(ctx context.Context, product model.Product) ([]model.Attribute, error) {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	// Get latest sub_code
	query := fmt.Sprintf(
		"SELECT sub_code FROM items WHERE item_code = '%s' ORDER BY created_at DESC LIMIT 1;",
		product.ItemCode,
	)

	latestRow, _ := s.repo.RetrieveSingleVal(query)
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
	// prefixes
	brandPrefix := strings.ToUpper(product.Brand)
	categoryPrefix := strings.ToUpper(product.Category)
	if len(brandPrefix) > 2 {
		brandPrefix = brandPrefix[:2]
	}
	if len(categoryPrefix) > 2 {
		categoryPrefix = categoryPrefix[:2]
	}
	year := time.Now().Format("2006")

	startSeq := 0
	if latestRow != nil {
		latestSub := latestRow["sub_code"].(string)
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
			ItemCode:     product.ItemCode,
			SubCode:      newSubCode,
			ItemName:     product.Name,
			ItemMainType: product.ItemMainType,
			ItemSubType:  product.NewSubCode,
			Brand:        product.Brand,
			Category:     product.Status,
			Description:  product.Description,
			InventoryID:  product.InventoryID,
			CreatedAt:    time.Now(),
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
