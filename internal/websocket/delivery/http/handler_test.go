package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notification-srv/internal/websocket"

	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"
	sharedauth "github.com/smap-hcmut/shared-libs/go/auth"
	sharederrors "github.com/smap-hcmut/shared-libs/go/errors"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type httpTestLogger struct{}

func (httpTestLogger) Debug(context.Context, ...any)           {}
func (httpTestLogger) Debugf(context.Context, string, ...any)  {}
func (httpTestLogger) Info(context.Context, ...any)            {}
func (httpTestLogger) Infof(context.Context, string, ...any)   {}
func (httpTestLogger) Warn(context.Context, ...any)            {}
func (httpTestLogger) Warnf(context.Context, string, ...any)   {}
func (httpTestLogger) Error(context.Context, ...any)           {}
func (httpTestLogger) Errorf(context.Context, string, ...any)  {}
func (httpTestLogger) DPanic(context.Context, ...any)          {}
func (httpTestLogger) DPanicf(context.Context, string, ...any) {}
func (httpTestLogger) Panic(context.Context, ...any)           {}
func (httpTestLogger) Panicf(context.Context, string, ...any)  {}
func (httpTestLogger) Fatal(context.Context, ...any)           {}
func (httpTestLogger) Fatalf(context.Context, string, ...any)  {}
func (l httpTestLogger) WithTrace(context.Context) log.Logger  { return l }

type fakeJWTManager struct {
	payload sharedauth.Payload
	err     error
	tokens  []string
}

func (m *fakeJWTManager) Verify(token string) (sharedauth.Payload, error) {
	m.tokens = append(m.tokens, token)
	return m.payload, m.err
}
func (m *fakeJWTManager) VerifyWithTrace(ctx context.Context, token string) (sharedauth.Payload, context.Context, error) {
	payload, err := m.Verify(token)
	return payload, ctx, err
}
func (m *fakeJWTManager) CreateToken(sharedauth.Payload) (string, error) { return "", nil }
func (m *fakeJWTManager) CreateTokenWithTrace(ctx context.Context, payload sharedauth.Payload) (string, context.Context, error) {
	return "", ctx, nil
}
func (m *fakeJWTManager) VerifyScope(string) (sharedauth.Scope, error) {
	return sharedauth.Scope{}, nil
}
func (m *fakeJWTManager) VerifyScopeWithTrace(context.Context, string) (sharedauth.Scope, error) {
	return sharedauth.Scope{}, nil
}

type fakeWSUseCase struct {
	registerErr error
	inputs      []websocket.ConnectionInput
}

func (u *fakeWSUseCase) Run()                           {}
func (u *fakeWSUseCase) Shutdown(context.Context) error { return nil }
func (u *fakeWSUseCase) Register(_ context.Context, input websocket.ConnectionInput) error {
	u.inputs = append(u.inputs, input)
	return u.registerErr
}
func (u *fakeWSUseCase) Unregister(context.Context, websocket.ConnectionInput) error {
	return nil
}
func (u *fakeWSUseCase) GetStats(context.Context) (websocket.HubStats, error) {
	return websocket.HubStats{}, nil
}
func (u *fakeWSUseCase) ProcessMessage(context.Context, websocket.ProcessMessageInput) error {
	return nil
}
func (u *fakeWSUseCase) OnUserConnected(context.Context, string) error { return nil }
func (u *fakeWSUseCase) OnUserDisconnected(context.Context, string, bool) error {
	return nil
}

func newTestHandler(uc websocket.UseCase, jwtMgr sharedauth.Manager) *handler {
	return New(
		uc,
		jwtMgr,
		httpTestLogger{},
		WSConfig{ReadBufferSize: 1024, WriteBufferSize: 1024},
		CookieConfig{Name: "access_token"},
		"test",
	).(*handler)
}

func ginContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, nil)
	c.Request = req
	return c, w
}

func TestNew(t *testing.T) {
	uc := &fakeWSUseCase{}
	jwtMgr := &fakeJWTManager{}
	h := newTestHandler(uc, jwtMgr)

	require.Equal(t, uc, h.uc)
	require.Equal(t, jwtMgr, h.jwtMgr)
	require.Equal(t, "test", h.environment)
}

func TestUpgradeReqValidateAndToInput(t *testing.T) {
	require.ErrorIs(t, UpgradeReq{}.validate(), websocket.ErrMissingToken)
	require.NoError(t, UpgradeReq{Token: "token"}.validate())

	conn := &gorilla.Conn{}
	got := UpgradeReq{ProjectID: "project-1"}.toInput(conn, "user-1")
	require.Equal(t, "user-1", got.UserID)
	require.Equal(t, "project-1", got.ProjectID)
	require.Equal(t, conn, got.Conn)
}

func TestProcessUpgradeRequest(t *testing.T) {
	verifyErr := errors.New("verify failed")
	tcs := map[string]struct {
		target      string
		cookieToken string
		verifyErr   error
		wantReq     UpgradeReq
		wantUserID  string
		wantErr     error
		wantTokens  []string
	}{
		"query token": {
			target:     "/ws?token=query-token&project_id=project-1",
			wantReq:    UpgradeReq{Token: "query-token", ProjectID: "project-1"},
			wantUserID: "user-1",
			wantTokens: []string{"query-token"},
		},
		"cookie token fallback": {
			target:      "/ws",
			cookieToken: "cookie-token",
			wantReq:     UpgradeReq{Token: "cookie-token"},
			wantUserID:  "user-1",
			wantTokens:  []string{"cookie-token"},
		},
		"bind error": {
			target:  "/ws?bad=%zz",
			wantErr: websocket.ErrInvalidMessage,
		},
		"missing token": {
			target:  "/ws",
			wantErr: websocket.ErrMissingToken,
		},
		"invalid token": {
			target:     "/ws?token=bad-token",
			verifyErr:  verifyErr,
			wantErr:    websocket.ErrInvalidToken,
			wantTokens: []string{"bad-token"},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			jwtMgr := &fakeJWTManager{
				payload: sharedauth.Payload{UserID: "user-1"},
				err:     tc.verifyErr,
			}
			h := newTestHandler(&fakeWSUseCase{}, jwtMgr)
			c, _ := ginContext("GET", tc.target)
			if tc.cookieToken != "" {
				c.Request.AddCookie(&http.Cookie{Name: "access_token", Value: tc.cookieToken})
			}

			gotReq, gotUserID, err := h.processUpgradeRequest(c)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Equal(t, tc.wantTokens, jwtMgr.tokens)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantReq, gotReq)
			require.Equal(t, tc.wantUserID, gotUserID)
			require.Equal(t, tc.wantTokens, jwtMgr.tokens)
		})
	}
}

func TestMapError(t *testing.T) {
	h := newTestHandler(&fakeWSUseCase{}, &fakeJWTManager{})
	tcs := map[string]struct {
		input      error
		wantCode   int
		wantStatus int
		wantMsg    string
	}{
		"invalid token":   {input: websocket.ErrInvalidToken, wantCode: 401, wantStatus: 401, wantMsg: "Invalid or expired token"},
		"missing token":   {input: websocket.ErrMissingToken, wantCode: 401, wantStatus: 401, wantMsg: "Missing authentication token"},
		"max connections": {input: websocket.ErrMaxConnectionsReached, wantCode: 503, wantStatus: 503, wantMsg: "Maximum connections reached"},
		"user not found":  {input: websocket.ErrUserNotFound, wantCode: 404, wantStatus: 404, wantMsg: "User not found"},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got := h.mapError(tc.input)
			httpErr, ok := got.(*sharederrors.HTTPError)
			require.True(t, ok)
			require.Equal(t, tc.wantCode, httpErr.Code)
			require.Equal(t, tc.wantStatus, httpErr.StatusCode)
			require.Equal(t, tc.wantMsg, httpErr.Message)
		})
	}

	unknownErr := errors.New("unknown")
	require.PanicsWithValue(t, unknownErr, func() {
		h.mapError(unknownErr)
	})
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newTestHandler(&fakeWSUseCase{}, &fakeJWTManager{})
	h.RegisterRoutes(router.Group(""), nil)

	routes := router.Routes()
	require.Len(t, routes, 1)
	require.Equal(t, "GET", routes[0].Method)
	require.Equal(t, "/ws", routes[0].Path)
}

func TestHandleWebSocket(t *testing.T) {
	tcs := map[string]struct {
		target      string
		verifyErr   error
		registerErr error
		useDial     bool
		wantStatus  int
		wantInputs  int
	}{
		"process error": {
			target:     "/ws",
			wantStatus: 401,
		},
		"upgrade error": {
			target:     "/ws?token=valid",
			wantStatus: 400,
		},
		"register error": {
			target:      "/ws?token=valid&project_id=project-1",
			registerErr: errors.New("register failed"),
			useDial:     true,
			wantInputs:  1,
		},
		"success": {
			target:     "/ws?token=valid&project_id=project-1",
			useDial:    true,
			wantInputs: 1,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := &fakeWSUseCase{registerErr: tc.registerErr}
			jwtMgr := &fakeJWTManager{payload: sharedauth.Payload{UserID: "user-1"}, err: tc.verifyErr}
			h := newTestHandler(uc, jwtMgr)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			h.RegisterRoutes(router.Group(""), nil)
			server := httptest.NewServer(router)
			defer server.Close()

			if tc.useDial {
				conn, _, err := gorilla.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+tc.target, nil)
				if conn != nil {
					conn.Close()
				}
				if tc.registerErr != nil {
					require.NoError(t, err)
				} else {
					require.NoError(t, err)
				}
			} else {
				resp := httptest.NewRecorder()
				req := httptest.NewRequest("GET", tc.target, nil)
				router.ServeHTTP(resp, req)
				require.Equal(t, tc.wantStatus, resp.Code)
			}

			require.Len(t, uc.inputs, tc.wantInputs)
			if tc.wantInputs > 0 {
				require.Equal(t, "user-1", uc.inputs[0].UserID)
				require.Equal(t, "project-1", uc.inputs[0].ProjectID)
				require.IsType(t, &gorilla.Conn{}, uc.inputs[0].Conn)
			}
		})
	}
}
