package irrl

import (
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"myproject/pkg/util"
	"path/filepath"
	"strconv"
	"strings"

	"time"

	// db "myproject/pkg/database"
	services "myproject/pkg/client"
	"myproject/pkg/config"
	db "myproject/pkg/database"

	"context"
	"myproject/pkg/model"

	"net/http"

	// "time"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service   Service
	services  services.Services
	adminjw   Adminjwt
	templates *template.Template
	cnf       config.Config
	init      *util.Initiator
	S3Client  *s3.Client // 👈 add this
	Bucket    string
}

func NewHandler(service Service, srv services.Services,
	adTK Adminjwt, cnf config.Config,
	initiator *util.Initiator, s3Client *s3.Client) *Handler {

	return &Handler{
		service:  service,
		services: srv,
		adminjw:  adTK,
		cnf:      cnf,
		init:     initiator,
		S3Client: s3Client, // 👈 store client
		Bucket:   cnf.BUCKET_ID,
	}
}
func (h *Handler) MountRoutes(engine *echo.Echo) {
	//applicantApi := engine.Group(basePath)
	applicantApi := engine.Group("/irrl")

	//applicantApi.Use(h.adminjw.AdminAuthMiddleware())
	//{

	applicantApi.GET("/listing", h.Listing)
	applicantApi.GET("/attribute/:AttributeType", h.getAttribute)
	applicantApi.GET("/genericApiUnjoin/:ApiType", h.GenericApiUnJoin)
	applicantApi.GET("/genericApiJoin/:ApiType", h.GenericApiJoin)
	applicantApi.POST("/addProduct", h.addProduct)
	applicantApi.POST("/addOrder", h.addOrder)
	applicantApi.POST("/initiateOrder", h.initiateOrder)
	applicantApi.POST("/genericStatusUpdate", h.genericStatusUpdate)
	applicantApi.POST("/upload", h.Upload)
	applicantApi.GET("/updateOrder/:orderID", h.updateformatOrder)
	applicantApi.GET("/updateItem/:itemID", h.updateformatItem)
	applicantApi.GET("/insertGeneric", h.insertGeneric)
	applicantApi.POST("/addSubTransaction", h.addSubTransaction)
	applicantApi.GET("/editTransaction/:transactionID", h.editTransaction)
	applicantApi.POST("/updateOrderItem", h.updateOrderItems)
	applicantApi.POST("/customers", h.addCustomers)
	applicantApi.GET("/reports/delivery", h.DownloadDeliveryReport)
	applicantApi.POST("/genericDelete", h.genericDelete)
	applicantApi.POST("/markDamage", h.markDamage)
	applicantApi.POST("/markRepairing", h.markRepairing)
	applicantApi.GET("/damaged/grouped", h.getDamagedGrouped)
	applicantApi.GET("/damaged/list", h.getDamagedList)
	applicantApi.GET("/repairing/grouped", h.getRepairingGrouped)
	applicantApi.GET("/repairing/list", h.getRepairingList)
	applicantApi.GET("/repairHistory", h.getRepairHistory)
	applicantApi.POST("/updateOrderPass", h.updateOrderPass)

	applicantApi.GET("/analytics/customer-damage", h.getCustomerDamageAnalytics)
	applicantApi.GET("/analytics/item-damage", h.getItemDamageAnalytics)
	applicantApi.GET("/analytics/customer-blocked", h.getCustomerBlockedAnalytics)
	applicantApi.GET("/analytics/item-rental", h.getItemRentalAnalytics)
	applicantApi.GET("/analytics/order-status", h.getOrderStatusSummary)
	applicantApi.GET("/analytics/customer-revenue", h.getRevenueByCustomer)

	applicantApi.GET("/analytics/cost/summary", h.getCostSummary)
	applicantApi.GET("/analytics/cost/revenue-trend", h.getRevenueTrend)
	applicantApi.GET("/analytics/cost/repair-costs", h.getRepairCostAnalytics)
	applicantApi.GET("/analytics/cost/outstanding-balances", h.getOutstandingBalances)
	applicantApi.GET("/analytics/cost/inventory-revenue", h.getInventoryRevenue)

	applicantApi.POST("/quotation", h.createQuotation)
	applicantApi.GET("/quotation", h.getQuotations)
	applicantApi.GET("/quotation/:id", h.getQuotationByID)

	//// wallet transactions

	//}

}
func (h *Handler) markDamage(c echo.Context) error {
	var req struct {
		ItemID         string   `json:"item_id"`
		DamageImages   []string `json:"damage_images"`
		Clear          bool     `json:"clear"` // if true, clears damage instead
		DeliveryItemID string   `json:"delivery_item_id"`
		Description    string   `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	if req.ItemID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{
			"error": "item_id is required",
		})
	}

	var err error
	if req.Clear {
		err = h.service.ClearItemDamage(req.ItemID)
	} else {
		if len(req.DamageImages) == 0 {
			return h.respondWithError(c, http.StatusBadRequest, map[string]string{
				"error": "damage_images is required when marking damage",
			})
		}
		err = h.service.MarkItemDamage(req.ItemID, req.DeliveryItemID, req.DamageImages, req.Description)
	}

	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	action := "marked as damaged"
	if req.Clear {
		action = "damage cleared"
	}
	return h.respondWithData(c, http.StatusOK, "success", action)
}

func (h *Handler) markRepairing(c echo.Context) error {
	var req struct {
		ItemID       string   `json:"item_id"`
		RepairImages []string `json:"repair_images"`
		Description  string   `json:"description"`
		Amount       int      `json:"amount"`
		Clear        bool     `json:"clear"` // if true, clears repairing instead
	}

	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	if req.ItemID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{
			"error": "item_id is required",
		})
	}

	var err error
	if req.Clear {
		err = h.service.ClearItemRepairing(req.ItemID)
	} else {
		if len(req.RepairImages) == 0 {
			return h.respondWithError(c, http.StatusBadRequest, map[string]string{
				"error": "repair_images is required when marking repairing",
			})
		}
		err = h.service.MarkItemRepairing(req.ItemID, req.RepairImages, req.Amount, req.Description)
	}

	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	action := "marked as repairing"
	if req.Clear {
		action = "repairing cleared"
	}
	return h.respondWithData(c, http.StatusOK, "success", action)
}

func (h *Handler) getDamagedGrouped(c echo.Context) error {
	data, err := h.service.GetDamagedGrouped()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getDamagedList(c echo.Context) error {
	subCode := c.QueryParam("item_sub_code")
	data, err := h.service.GetDamagedList(subCode)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRepairingGrouped(c echo.Context) error {
	data, err := h.service.GetRepairingGrouped()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRepairingList(c echo.Context) error {
	subCode := c.QueryParam("item_sub_code")
	data, err := h.service.GetRepairingList(subCode)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRepairHistory(c echo.Context) error {
	itemID := c.QueryParam("item_id")
	if itemID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "item_id is required"})
	}

	data, err := h.service.GetRepairHistory(itemID)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) updateOrderPass(c echo.Context) error {
	var req model.OrderPassRequest
	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "invalid request body", "details": err.Error()})
	}
	if req.OrderID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "order_id is required"})
	}
	if err := h.service.UpdateOrderPass(req); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", "order pass details updated")
}

func (h *Handler) respondWithError(c echo.Context, code int, msg interface{}) error {
	resp := map[string]interface{}{
		"msg": msg,
	}

	return c.JSON(code, resp)
}

func (h *Handler) respondWithData(c echo.Context, code int, message interface{}, data interface{}) error {
	if data == nil {
		data = "Succesfully done"
		resp := map[string]interface{}{
			"msg":     message,
			"Process": data,
		}
		return c.JSON(code, resp)

	}
	resp := map[string]interface{}{
		"msg":  message,
		"data": data,
	}
	return c.JSON(code, resp)
}

// ///

func (h *Handler) Register(c echo.Context) error {

	fmt.Println("this is in the handler Register")
	var request model.UserRegisterRequest
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}

	errVal := request.Valid()
	if len(errVal) > 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"invalid-request": errVal})
	}

	ctx := c.Request().Context()
	if err := h.service.Register(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	fmt.Println("this is in the handler Register")

	otp, err := h.services.SendEmailWithOTP(request.Email)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "error in sending otp"})

	}
	err = db.SetRedis(request.Email, otp, time.Minute*5)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "error in saving otp"})

	}
	storedData, _ := db.GetRedis(request.Email)
	fmt.Println("this is the keyy!!!!!", storedData)

	return h.respondWithData(c, http.StatusOK, "success", nil)
}
func (h *Handler) UpdateUser(c echo.Context) error {

	fmt.Println("this is in the handler UpdateUser")
	var request model.UserRegisterRequest
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}

	// Validate request fields
	//errVal := request.Valid()

	ctx := c.Request().Context()
	if err := h.service.UpdateUser(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	fmt.Println("this is in the handler UpdateUser")

	otp, err := h.services.SendEmailWithOTP(request.Email)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "error in sending otp"})

	}
	err = db.SetRedis(request.Email, otp, time.Minute*5)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "error in saving otp"})

	}
	storedData, _ := db.GetRedis(request.Email)
	fmt.Println("this is the keyy!!!!!", storedData)

	return h.respondWithData(c, http.StatusOK, "success", nil)
}
func (h *Handler) Login(c echo.Context) error {

	fmt.Println("this is in the handler Register")
	var request model.UserLoginRequest
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}

	ctx := c.Request().Context()
	if err := h.service.Login(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	fmt.Println("this is in the handler Register")
	token, err := h.adminjw.GenerateAdminToken(request.Email)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"token-generation": err.Error()})
	}

	fmt.Println("User logged in successfully")
	return h.respondWithData(c, http.StatusOK, "success", map[string]string{"token": token})
}
func (h *Handler) OtpLogin(c echo.Context) error {
	// Parse request body into UserRegisterRequest
	fmt.Println("this is in the handler OtpLogin")
	var request model.UserOtp

	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}
	fmt.Println("this is request", request)

	// Respond with success
	storedData, err := db.GetRedis(request.Email)
	fmt.Println("this is the keyy!!!!!", storedData, err)
	if storedData != request.Otp {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "wrong otp"})

	}
	ctx := c.Request().Context()
	h.service.VerifyOtp(ctx, request.Email)

	return h.respondWithData(c, http.StatusOK, "success", nil)
}

func (h *Handler) Listing(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := h.service.Listing(ctx)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products", "details": err.Error()})
	}
	fmt.Println("this is the data ", products)
	return h.respondWithData(c, http.StatusOK, "success", products)
}
func (h *Handler) GenericApiUnJoin(c echo.Context) error {
	ctx := c.Request().Context()
	typeApi := c.Param("ApiType")
	endQuery, err := h.init.Initiator(c, typeApi, "")
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch keys", "details": err.Error()})
	}
	genericResult, err := h.service.GenericApi(ctx, typeApi, endQuery)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch genericResult", "details": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", genericResult)
}
func (h *Handler) GenericApiJoin(c echo.Context) error {
	ctx := c.Request().Context()
	typeApi := c.Param("ApiType")
	endQuery, err := h.init.Initiator(c, typeApi, "")
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch keys", "details": err.Error()})
	}
	genericResult, err := h.service.GenericApi(ctx, typeApi, endQuery)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch genericResult", "details": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", genericResult)
}
func (h *Handler) getAttribute(c echo.Context) error {
	ctx := c.Request().Context()
	typeAttribute := c.Param("AttributeType")
	products, err := h.service.GetAttributes(ctx, typeAttribute)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products", "details": err.Error()})
	}
	fmt.Println("this is the data ", products)
	return h.respondWithData(c, http.StatusOK, "success", products)
}

func (h *Handler) addProduct(c echo.Context) error {
	ctx := c.Request().Context()
	var request model.Product
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}

	if len(request.Valid()) > 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"invalid-request": request.Valid()})
	}
	products, err := h.service.AddProduct(ctx, request)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products", "details": err.Error()})
	}
	fmt.Println("this is the data ", products)
	return h.respondWithData(c, http.StatusOK, "success", products)
}
func (h *Handler) genericStatusUpdate(c echo.Context) error {
	var request model.GenericUpdate
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}

	// Retrieve stored fields (jsonb) for this API code
	fieldBytes, err := h.init.RetrieveApiFields(request.Code)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"db-error": err.Error()})
	}

	// Unmarshal DB fields into a struct (or map)
	var fields model.Fields
	if err := json.Unmarshal(fieldBytes, &fields); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"unmarshal-error": err.Error()})
	}
	fmt.Println("this is the data ", fields)
	request.Table = fields.Table
	request.Field = fields.Key
	request.ID = fields.ID
	fmt.Println("this is the data ", request)
	h.service.GenericStatusUpdate(request)

	return h.respondWithData(c, http.StatusOK, "success", "updated")
}

func (h *Handler) addOrder(c echo.Context) error {
	var req model.DeliveryOrder
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request", "details": err.Error()})
	}
	if req.Status == "" {
		req.Status = "RESERVED"
	}
	if len(req.Valid()) > 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"invalid-request": req.Valid()})
	}
	//	ctx := c.Request().Context()
	if _, err := h.service.AddOrder(req); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "products")
}

func (h *Handler) addSubTransaction(c echo.Context) error {
	var req model.Transaction
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request", "details": err.Error()})
	}
	if len(req.Valid()) > 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"invalid-request": req.Valid()})
	}
	//	ctx := c.Request().Context()
	if err := h.service.AddSubTransaction(req); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "products")
}
func (h *Handler) Upload(c echo.Context) error {
	// Debug: Print all headers
	fmt.Println("=== Request Headers ===")
	for name, values := range c.Request().Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", name, value)
		}
	}
	fmt.Println("=====================")

	// Check if the request is multipart/form-data
	contentType := c.Request().Header.Get("Content-Type")
	fmt.Printf("Content-Type: '%s'\n", contentType)

	// More lenient check - just try to parse the form
	if contentType == "" || !strings.Contains(contentType, "multipart/form-data") {
		// Try to parse anyway and see what happens
		fmt.Println("Warning: Content-Type not set to multipart/form-data, but attempting to parse...")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Error parsing multipart form: %v", err),
		})
	}
	fmt.Println("this is the form", form)
	files := form.File["images"] // "images" is the form field name
	fmt.Println("this is the files", files)
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No files uploaded in 'images' field",
		})
	}

	var uploadedURLs []string

	for _, fileHeader := range files {
		// Validate file size (e.g., max 10MB)
		if fileHeader.Size > 10*1024*1024 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("File %s is too large (max 10MB)", fileHeader.Filename),
			})
		}

		// Validate file type
		if !isValidImageType(fileHeader.Header.Get("Content-Type")) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("File %s is not a valid image type", fileHeader.Filename),
			})
		}

		src, err := fileHeader.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Error opening file %s: %v", fileHeader.Filename, err),
			})
		}
		defer src.Close()

		// Generate unique filename
		ext := filepath.Ext(fileHeader.Filename)
		name := strings.TrimSuffix(fileHeader.Filename, ext)

		// Use UUID instead of random number for better uniqueness
		uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
		newFileName := fmt.Sprintf("%s_%s%s", name, uniqueID, ext)

		// Get content type safely
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream" // fallback
		}

		// Upload to S3
		fmt.Println("this is the buckkkkk----", h.Bucket)
		_, err = h.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      &h.Bucket,
			Key:         &newFileName,
			Body:        src,
			ACL:         types.ObjectCannedACLPublicRead, // Use types instead of string
			ContentType: &contentType,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Error uploading %s to S3: %v", fileHeader.Filename, err),
			})
		}

		// Construct public URL
		publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", h.Bucket, h.cnf.AWS_REGION, newFileName)
		uploadedURLs = append(uploadedURLs, publicURL)
	}

	// Return JSON response with URLs
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Files uploaded successfully!",
		"count":   len(uploadedURLs),
		"urls":    uploadedURLs,
	})
}

// Helper function to validate image types
func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}
func (h *Handler) updateformatOrder(c echo.Context) error {
	typeApi := c.Param("orderID")
	typestatus := c.QueryParam("status")
	if typestatus == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"missing status": "invalid request"})
	}

	err := h.service.UpdateMainOrderStatus(typeApi, typestatus)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]interface{}{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "order updated")
}
func (h *Handler) updateformatItem(c echo.Context) error {
	typeApi := c.Param("itemID")
	typestatus := c.QueryParam("status")
	if typestatus == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"missing status": "invalid request"})
	}

	err := h.service.UpdateItemStatus(typeApi, typestatus)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]interface{}{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "order updated")
}
func (h *Handler) updateOrderItems(c echo.Context) error {
	var req model.OrderItemUpdate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request", "details": err.Error()})
	}
	fmt.Println("this is the req12----", req)
	err := h.service.UpdateOrderItemStatus(req)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]interface{}{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "order updated")
}
func (h *Handler) editTransaction(c echo.Context) error {
	idstr := c.Param("transactionID")
	if idstr == "" {
		return c.String(http.StatusBadRequest, "id is required")
	}

	id, err := strconv.Atoi(idstr)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid amount")
	}
	fmt.Println("this is the data  trans", id)
	if id <= 0 {
		return c.String(http.StatusBadRequest, "amount must be greater than 0")
	}
	table := c.QueryParam("table")

	if table == "" {

		typestatus := c.QueryParam("status")

		if typestatus == "" {
			return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"missing status": "invalid request"})
		}
		if !(typestatus == "PENDING" || typestatus == "COMPLETED") {
			return c.String(http.StatusBadRequest, "invalid status")
		}
		amtStr := c.QueryParam("amount")
		if amtStr == "" {
			return c.String(http.StatusBadRequest, "amount is required")
		}

		// Convert to integer
		amt, err := strconv.Atoi(amtStr)
		if err != nil {
			return c.String(http.StatusBadRequest, "invalid amount")
		}
		updateQuery := ""
		// Check if > 0
		if amt <= 0 {
			updateQuery = fmt.Sprintf(
				"update main_transaction set status='%s' where id=%d ;",
				typestatus, id,
			)
		} else {
			updateQuery = fmt.Sprintf(
				"update main_transaction set status='%s',amount=%d where id=%d ;",
				typestatus, amt, id,
			)
		}

		err = h.init.Repo.ExecQuery(updateQuery)
		if err != nil {
			return h.respondWithError(c, http.StatusInternalServerError, map[string]interface{}{"db-error failed to update": err.Error()})
		}
	} else {
		typestatus := c.QueryParam("status")

		if typestatus == "" {
			return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"missing status": "invalid request"})
		}
		if !(typestatus == "PENDING" || typestatus == "COMPLETED" || typestatus == "FAILED") {
			return c.String(http.StatusBadRequest, "invalid status")
		}
		updateQuery := fmt.Sprintf(
			"update transaction set status='%s' where id=%d ;",
			typestatus, id,
		)
		err = h.init.Repo.ExecQuery(updateQuery)
		if err != nil {
			return h.respondWithError(c, http.StatusInternalServerError, map[string]interface{}{"db-error failed to update": err.Error()})
		}
	}
	return h.respondWithData(c, http.StatusOK, "success", "order updated")
}
func (h *Handler) insertGeneric(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request", "details": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "products")
}

func (h *Handler) addCustomers(c echo.Context) error {
	var customer model.Customer

	if err := c.Bind(&customer); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := h.service.CreateCustomer(customer); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "customer created successfully"})
}
func (h *Handler) genericDelete(c echo.Context) error {
	var req model.GenericDelete
	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "invalid request body", "details": err.Error()})
	}
	if err := h.service.GenericDelete(req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", "deleted")
}

func (h *Handler) DownloadDeliveryReport(c echo.Context) error {
	filter := model.DeliveryReportFilter{
		DateRange: c.QueryParam("date_range"),
		Remark:    c.QueryParam("remark"),
	}
	fileBytes, err := h.service.GenerateDeliveryReportExcel(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.Blob(
		http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		fileBytes,
	)
}

func (h *Handler) initiateOrder(c echo.Context) error {
	var req model.InitiateOrderRequest
	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "invalid request", "details": err.Error()})
	}

	if req.OrderID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "order_id is required"})
	}

	err := h.service.InitiateMainOrder(req.OrderID, req.GuaranteeImages)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "order initiated")
}

func (h *Handler) getCustomerDamageAnalytics(c echo.Context) error {
	data, err := h.service.GetCustomerDamageAnalytics()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getItemDamageAnalytics(c echo.Context) error {
	data, err := h.service.GetItemDamageAnalytics()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getCustomerBlockedAnalytics(c echo.Context) error {
	data, err := h.service.GetCustomerBlockedAnalytics()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getItemRentalAnalytics(c echo.Context) error {
	data, err := h.service.GetItemRentalAnalytics()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getOrderStatusSummary(c echo.Context) error {
	data, err := h.service.GetOrderStatusSummary()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRevenueByCustomer(c echo.Context) error {
	data, err := h.service.GetRevenueByCustomer()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getCostSummary(c echo.Context) error {
	data, err := h.service.GetCostSummary()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRevenueTrend(c echo.Context) error {
	period := c.QueryParam("period") // monthly | weekly | daily
	limitStr := c.QueryParam("limit")
	limit := 12
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}
	data, err := h.service.GetRevenueTrend(period, limit)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getRepairCostAnalytics(c echo.Context) error {
	data, err := h.service.GetRepairCostAnalytics()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getOutstandingBalances(c echo.Context) error {
	data, err := h.service.GetOutstandingBalances()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getInventoryRevenue(c echo.Context) error {
	data, err := h.service.GetInventoryRevenue()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) createQuotation(c echo.Context) error {
	var req model.QuotationRequest
	if err := c.Bind(&req); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.CustomerID == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "customer_id is required"})
	}
	if len(req.Items) == 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "items are required"})
	}
	id, err := h.service.CreateQuotation(req)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", map[string]string{"quotation_id": id})
}

func (h *Handler) getQuotations(c echo.Context) error {
	fmt.Println("bhbhbhj b--------")

	data, err := h.service.GetQuotations()
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}

func (h *Handler) getQuotationByID(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"error": "id is required"})
	}
	data, err := h.service.GetQuotationByID(id)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", data)
}
