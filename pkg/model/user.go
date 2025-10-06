package model

import (
	"net/url"

	"github.com/golang-jwt/jwt"
)

type UserRegisterRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Phone       string `json:"phone"`
	DeviceId    string `json:"device_id"`
	Role        string `json:"role"`
	Designation string `json:"designation"`
}

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type UserClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.StandardClaims
}
type UserOtp struct {
	Email string `json:"email"`
	Otp   string `json:"otp"`
}

func (u *UserRegisterRequest) Valid() url.Values {
	err := url.Values{}

	// if len(u.Name) < 2 {
	// 	err.Add("name", "invalid name")
	// }

	if len(u.Password) < 6 {
		err.Add("password", "password must be greater than 6")
	}

	return err
}
