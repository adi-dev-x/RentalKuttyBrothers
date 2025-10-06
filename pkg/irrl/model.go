package irrl

import (
	"myproject/pkg/model"

	"github.com/golang-jwt/jwt"
)

// Re-export common user models from pkg/model so other packages can import
// models from pkg/irrl (keeps a single canonical definition in pkg/model).
type UserRegisterRequest = model.UserRegisterRequest
type UserLoginRequest = model.UserLoginRequest
type UserOtp = model.UserOtp

type AdminClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}
