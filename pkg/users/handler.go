package users

import (
	"fmt"
	"net/http"

	"myproject/pkg/irrl"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service  Service
	services Services
	adminjw  Adminjwt
}

func NewHandler(service Service, srv Services, adTK Adminjwt) *Handler {
	return &Handler{
		service:  service,
		services: srv,
		adminjw:  adTK,
	}
}

func (h *Handler) MountRoutes(engine *echo.Echo) {
	grp := engine.Group("/users")
	grp.POST("/register", h.Register)
	grp.POST("/login", h.Login)
	grp.POST("/otpLogin", h.OtpLogin)
	grp.POST("/update", h.UpdateUser)
}

func (h *Handler) respondWithError(c echo.Context, code int, msg interface{}) error {
	resp := map[string]interface{}{"msg": msg}
	return c.JSON(code, resp)
}

func (h *Handler) respondWithData(c echo.Context, code int, message interface{}, data interface{}) error {
	if data == nil {
		data = "Succesfully done"
		resp := map[string]interface{}{"msg": message, "Process": data}
		return c.JSON(code, resp)
	}
	resp := map[string]interface{}{"msg": message, "data": data}
	return c.JSON(code, resp)
}

func (h *Handler) Register(c echo.Context) error {
	fmt.Println("users: Register handler")
	var request irrl.UserRegisterRequest
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
	return h.respondWithData(c, http.StatusOK, "success", nil)
}

func (h *Handler) Login(c echo.Context) error {
	fmt.Println("users: Login handler")
	var request irrl.UserLoginRequest
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}
	ctx := c.Request().Context()
	if err := h.service.Login(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	token := ""
	if h.adminjw != nil {
		var err error
		token, err = h.adminjw.GenerateAdminToken(request.Username)
		if err != nil {
			return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"token-generation": err.Error()})
		}
	}
	return h.respondWithData(c, http.StatusOK, "success", map[string]string{"token": token})
}

func (h *Handler) OtpLogin(c echo.Context) error {
	fmt.Println("users: OtpLogin handler")
	var request irrl.UserOtp
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}
	// For package independence we avoid direct redis access here.
	// Assume service.OtpLogin handles verification.
	ctx := c.Request().Context()
	if err := h.service.OtpLogin(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	h.service.VerifyOtp(ctx, request.Email)
	return h.respondWithData(c, http.StatusOK, "success", nil)
}

func (h *Handler) UpdateUser(c echo.Context) error {
	fmt.Println("users: UpdateUser handler")
	var request irrl.UserRegisterRequest
	if err := c.Bind(&request); err != nil {
		return h.respondWithError(c, http.StatusBadRequest, map[string]string{"request-parse": err.Error()})
	}
	ctx := c.Request().Context()
	if err := h.service.UpdateUser(ctx, request); err != nil {
		return h.respondWithError(c, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return h.respondWithData(c, http.StatusOK, "success", nil)
}
