package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"myproject/pkg/irrl"

	"github.com/labstack/echo/v4"
)

// mockService implements minimal methods used by the handler
type mockService struct{}

func (m *mockService) Register(ctx context.Context, request irrl.UserRegisterRequest) error {
	return nil
}
func (m *mockService) Login(ctx context.Context, request irrl.UserLoginRequest) error { return nil }
func (m *mockService) OtpLogin(ctx context.Context, request irrl.UserOtp) error       { return nil }
func (m *mockService) UpdateUser(ctx context.Context, updatedData irrl.UserRegisterRequest) error {
	return nil
}
func (m *mockService) VerifyOtp(ctx context.Context, email string) {}

// simpleAdminJWT implements Adminjwt for tests
type simpleAdminJWT struct {
	key string
}

func (s simpleAdminJWT) GenerateAdminToken(username string) (string, error) {
	// produce a deterministic token for test
	return "tok-" + username, nil
}

func TestLoginHandler(t *testing.T) {
	e := echo.New()

	svc := &mockService{}
	adm := simpleAdminJWT{key: "testkey"}
	handler := NewHandler(svc, Services{}, adm)
	handler.MountRoutes(e)

	// prepare request body
	body := map[string]string{"username": "tester", "password": "secret"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data in response: %v", resp)
	}
	if token, ok := data["token"].(string); !ok || token == "" {
		t.Fatalf("expected token in response data, got: %v", data)
	}
}
