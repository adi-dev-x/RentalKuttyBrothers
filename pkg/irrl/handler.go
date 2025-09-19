package irrl

import (
	"encoding/json"
	"fmt"
	"html/template"

	"myproject/pkg/util"

	"time"

	// db "myproject/pkg/database"
	services "myproject/pkg/client"
	"myproject/pkg/config"
	db "myproject/pkg/database"

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
}

func NewHandler(service Service, srv services.Services, adTK Adminjwt, cnf config.Config, initiator *util.Initiator) *Handler {

	return &Handler{
		service:  service,
		services: srv,
		adminjw:  adTK,
		cnf:      cnf,
		init:     initiator,
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
	applicantApi.POST("/addProduct", h.addProduct)
	applicantApi.POST("/addOrder", h.addOrder)
	applicantApi.POST("/genericStatusUpdate", h.genericStatusUpdate)
	//// wallet transactions

	//}

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
	endQuery, err := h.init.Initiator(c, typeApi)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch keys", "details": err.Error()})
	}
	genericResult, err := h.service.GenericApi(ctx, typeApi, endQuery)
	if err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch genericResult", "details": err.Error()})
	}
	fmt.Println("this is the data ", genericResult)
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
	if len(req.Valid()) > 0 {
		return h.respondWithError(c, http.StatusBadRequest, map[string]interface{}{"invalid-request": req.Valid()})
	}
	//	ctx := c.Request().Context()
	if err := h.service.AddOrder(req); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"db-error": err.Error()})
	}

	return h.respondWithData(c, http.StatusOK, "success", "products")
}
