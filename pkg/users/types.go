package users

import (
	"context"

	"myproject/pkg/irrl"
)

// Service defines the methods the handler needs from the business/service layer.
// Use the request models from pkg/irrl so pkg/users doesn't duplicate models.
type Service interface {
	Register(ctx context.Context, request irrl.UserRegisterRequest) error
	Login(ctx context.Context, request irrl.UserLoginRequest) error
	OtpLogin(ctx context.Context, request irrl.UserOtp) error
	UpdateUser(ctx context.Context, updatedData irrl.UserRegisterRequest) error
	VerifyOtp(ctx context.Context, email string)
}

// Services is a placeholder for external client services (email, sms, etc.) used by the handler.
// Keep it empty for now; tests can provide a concrete implementation if needed.
type Services struct{}

// Adminjwt provides token generation used in the handler. Keep as minimal interface.
type Adminjwt interface {
	GenerateAdminToken(username string) (string, error)
}
