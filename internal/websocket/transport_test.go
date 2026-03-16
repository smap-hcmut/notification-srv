package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"notification-srv/internal/alert"
	wsConfig "notification-srv/internal/websocket/delivery/http" // Alias to avoid conflict
	"notification-srv/internal/websocket/usecase"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(ctx context.Context, args ...any)                     {}
func (m *MockLogger) Infof(ctx context.Context, template string, args ...any)   {}
func (m *MockLogger) Warn(ctx context.Context, args ...any)                     {}
func (m *MockLogger) Warnf(ctx context.Context, template string, args ...any)   {}
func (m *MockLogger) Error(ctx context.Context, args ...any)                    {}
func (m *MockLogger) Errorf(ctx context.Context, template string, args ...any)  {}
func (m *MockLogger) Fatal(ctx context.Context, args ...any)                    {}
func (m *MockLogger) Fatalf(ctx context.Context, template string, args ...any)  {}
func (m *MockLogger) Debug(ctx context.Context, args ...any)                    {}
func (m *MockLogger) Debugf(ctx context.Context, template string, args ...any)  {}
func (m *MockLogger) DPanic(ctx context.Context, args ...any)                   {}
func (m *MockLogger) DPanicf(ctx context.Context, template string, args ...any) {}
func (m *MockLogger) Panic(ctx context.Context, args ...any)                    {}
func (m *MockLogger) Panicf(ctx context.Context, template string, args ...any)  {}

func (m *MockLogger) WithTrace(ctx context.Context) log.Logger {
	return m
}

type MockAlertUC struct {
	mock.Mock
}

func (m *MockAlertUC) DispatchCrisisAlert(ctx context.Context, input alert.CrisisAlertInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockAlertUC) DispatchDataOnboarding(ctx context.Context, input alert.DataOnboardingInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockAlertUC) DispatchCampaignEvent(ctx context.Context, input alert.CampaignEventInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

type MockScopeManager struct {
	mock.Mock
}

func (m *MockScopeManager) Verify(token string) (auth.Payload, error) {
	args := m.Called(token)
	return args.Get(0).(auth.Payload), args.Error(1)
}

func (m *MockScopeManager) VerifyWithTrace(ctx context.Context, token string) (auth.Payload, context.Context, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(auth.Payload), ctx, args.Error(1)
}

func (m *MockScopeManager) CreateToken(payload auth.Payload) (string, error) {
	args := m.Called(payload)
	return args.String(0), args.Error(1)
}

func (m *MockScopeManager) CreateTokenWithTrace(ctx context.Context, payload auth.Payload) (string, context.Context, error) {
	args := m.Called(ctx, payload)
	return args.String(0), ctx, args.Error(1)
}

func (m *MockScopeManager) VerifyScope(scopeHeader string) (auth.Scope, error) {
	args := m.Called(scopeHeader)
	return args.Get(0).(auth.Scope), args.Error(1)
}

func (m *MockScopeManager) VerifyScopeWithTrace(ctx context.Context, scopeHeader string) (auth.Scope, error) {
	args := m.Called(ctx, scopeHeader)
	return args.Get(0).(auth.Scope), args.Error(1)
}

// --- Tests ---

func TestWebSocketConnection(t *testing.T) {
	// Setup
	logger := &MockLogger{}
	alertUC := &MockAlertUC{}
	scopeMgr := &MockScopeManager{}

	// Mock Verify Token
	scopeMgr.On("Verify", "valid_token").Return(auth.Payload{
		UserID: "user_123",
	}, nil)

	// Init UseCase
	uc := usecase.New(logger, 100, alertUC)
	go uc.Run()
	// defer uc.Shutdown(context.Background())

	// Init Handler
	handler := wsConfig.New(
		uc,
		scopeMgr,
		logger,
		wsConfig.WSConfig{
			MaxConnections:  10,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{"*"},
		},
		wsConfig.CookieConfig{},
		"test",
	)

	// Setup Router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.RegisterRoutes(r.Group(""), nil)

	// Test Server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=valid_token"

	// Connect
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Should connect successfully")
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	if conn != nil {
		conn.Close()
	}

	// Verify Expectations
	scopeMgr.AssertExpectations(t)
}

func TestWebSocketMissingToken(t *testing.T) {
	// Setup
	logger := &MockLogger{}
	alertUC := &MockAlertUC{}
	scopeMgr := &MockScopeManager{}

	uc := usecase.New(logger, 100, alertUC)
	handler := wsConfig.New(
		uc,
		scopeMgr,
		logger,
		wsConfig.WSConfig{},
		wsConfig.CookieConfig{},
		"test",
	)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.RegisterRoutes(r.Group(""), nil)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws" // No token

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
	// Expect 400 Bad Request because 'token' is binding required/bound via validation?
	// Or 401? presenters.go: toInput validates it.
	// process_request.go: binds params.
	// If presenters.go validation fails, it returns error.

	// Let's assert strictly on error existence first. status might be 400.
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
