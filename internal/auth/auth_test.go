package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIMiddleware_ReturnsUnauthorizedWithoutSession(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			Type:     "basic",
			Settings: config.AuthSettings{Username: "admin", Password: "admin123"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	handler := mod.APIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestBasicServiceAuthenticate(t *testing.T) {
	service, err := NewBasicService(config.AuthSettings{Username: "admin", Password: "admin123"})
	require.NoError(t, err)

	user, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, "admin", user)

	_, err = service.Authenticate("admin", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestNewBasicServiceRequiresCredentials(t *testing.T) {
	_, err := NewBasicService(config.AuthSettings{Username: "admin"})
	assert.EqualError(t, err, "basic auth requires username and password settings")
}

func TestSessionUserCSRFAndClearSession(t *testing.T) {
	mod := testModule(t)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()

	require.NoError(t, mod.SetSessionUser(rr, req, "admin"))
	req = requestWithCookies(rr, http.MethodPost, "http://example.test/api")

	user, ok := mod.SessionUser(req)
	require.True(t, ok)
	assert.Equal(t, "admin", user)

	token, ok := mod.CSRFToken(req)
	require.True(t, ok)
	assert.NotEmpty(t, token)
	assert.False(t, mod.ValidCSRF(req))

	req.Header.Set("X-Cerebro-CSRF", token)
	assert.True(t, mod.ValidCSRF(req))

	rr = httptest.NewRecorder()
	require.NoError(t, mod.ClearSession(rr, req))
	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	_, ok = mod.SessionUser(req)
	assert.False(t, ok)
}

func TestEnsureCSRFTokenReusesExistingToken(t *testing.T) {
	mod := testModule(t)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()

	first, err := mod.EnsureCSRFToken(rr, req)
	require.NoError(t, err)
	assert.NotEmpty(t, first)

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	rr = httptest.NewRecorder()
	second, err := mod.EnsureCSRFToken(rr, req)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestRedirectIsSameOriginOnlyAndConsumedOnce(t *testing.T) {
	mod := testModule(t)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/login", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, mod.SetRedirectIfSafe(rr, req, "/#/overview"))
	req = requestWithCookies(rr, http.MethodGet, "http://example.test/login")

	rr = httptest.NewRecorder()
	assert.Equal(t, "/#/overview", mod.ConsumeRedirect(rr, req))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/login")
	rr = httptest.NewRecorder()
	assert.Empty(t, mod.ConsumeRedirect(rr, req))

	for _, unsafe := range []string{"", "https://evil.test", "//evil.test", `/\evil`} {
		req := httptest.NewRequest(http.MethodGet, "http://example.test/login", nil)
		rr := httptest.NewRecorder()
		require.NoError(t, mod.SetRedirectIfSafe(rr, req, unsafe))
		req = requestWithCookies(rr, http.MethodGet, "http://example.test/login")
		assert.Empty(t, mod.ConsumeRedirect(httptest.NewRecorder(), req))
	}
}

func TestAPIMiddlewareAddsUserToContext(t *testing.T) {
	mod := testModule(t)
	sessionReq := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	sessionRR := httptest.NewRecorder()
	require.NoError(t, mod.SetSessionUser(sessionRR, sessionReq, "admin"))

	handler := mod.APIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "admin", UserFrom(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}))
	req := requestWithCookies(sessionRR, http.MethodGet, "http://example.test/api")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func testModule(t *testing.T) *Module {
	t.Helper()
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			Type:     "basic",
			Settings: config.AuthSettings{Username: "admin", Password: "admin123"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	return mod
}

func requestWithCookies(rr *httptest.ResponseRecorder, method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}
