package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIMiddleware_ReturnsUnauthorizedWithoutSession(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			Basic: config.BasicAuth{Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
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

func TestNewEntraIDProviderRequiresSettings(t *testing.T) {
	_, err := NewEntraIDProvider(config.EntraIDAuth{TenantID: "tenant", ClientID: "client"})
	assert.EqualError(t, err, "entra id auth requires tenant_id, client_id and client_secret")
}

func TestNewModuleEnablesEntraID(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			EntraID: config.EntraIDAuth{Enabled: true, TenantID: "example.onmicrosoft.com", ClientID: "client", ClientSecret: "secret"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	assert.True(t, mod.Enabled())
	assert.True(t, mod.EntraIDEnabled())
	assert.False(t, mod.PasswordLoginEnabled())
}

func TestNewOAuthProviderRequiresSettings(t *testing.T) {
	_, err := NewOAuthProvider(config.OAuthAuth{ClientID: "client", ClientSecret: "secret"})
	assert.EqualError(t, err, "oauth auth requires issuer_url or auth_url, token_url and userinfo_url")
}

func TestNewModuleEnablesOAuth(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			OAuth: config.OAuthAuth{
				Enabled:      true,
				AuthURL:      "https://auth.example.org/authorize",
				TokenURL:     "https://auth.example.org/token",
				UserInfoURL:  "https://auth.example.org/userinfo",
				ClientID:     "client",
				ClientSecret: "secret",
			},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	assert.True(t, mod.Enabled())
	assert.True(t, mod.OAuthEnabled())
	assert.Equal(t, "OAuth", mod.OAuthName())
	assert.False(t, mod.PasswordLoginEnabled())
}

func TestEntraIDClaimHelpers(t *testing.T) {
	claims := map[string]any{
		"preferred_username": "alice@example.org",
		"groups":             []any{"admins", "operators"},
	}

	assert.Equal(t, "alice@example.org", firstClaimString(claims, "missing", "preferred_username"))
	assert.Equal(t, []string{"admins", "operators"}, claimStringSlice(claims["groups"]))
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
	identity, ok := mod.SessionIdentity(req)
	require.True(t, ok)
	assert.Equal(t, "admin", identity.Username)
	assert.Equal(t, "", identity.Provider)

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

func TestSessionIdentityExpiresByMaxLifetime(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth:   config.Auth{Session: config.AuthSession{MaxLifetimeSeconds: 60}},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, mod.SetSessionIdentity(rr, req, Identity{Username: "admin", Provider: "basic"}))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	rr = httptest.NewRecorder()
	sess, err := mod.Store().Get(req, SessionName)
	require.NoError(t, err)
	sess.Values[SessionIssuedAtKey] = time.Now().Add(-2 * time.Minute).Unix()
	require.NoError(t, sess.Save(req, rr))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	_, ok := mod.SessionIdentity(req)
	assert.False(t, ok)
}

func TestSessionIdentityExpiresByIdleTimeout(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth:   config.Auth{Session: config.AuthSession{IdleTimeoutSeconds: 60}},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, mod.SetSessionIdentity(rr, req, Identity{Username: "admin", Provider: "basic"}))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	rr = httptest.NewRecorder()
	sess, err := mod.Store().Get(req, SessionName)
	require.NoError(t, err)
	sess.Values[SessionLastSeenKey] = time.Now().Add(-2 * time.Minute).Unix()
	require.NoError(t, sess.Save(req, rr))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	_, ok := mod.SessionIdentity(req)
	assert.False(t, ok)
}

func TestTouchSessionRefreshesIdleTimestamp(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth:   config.Auth{Session: config.AuthSession{IdleTimeoutSeconds: 3600}},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, mod.SetSessionIdentity(rr, req, Identity{Username: "admin", Provider: "basic"}))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	rr = httptest.NewRecorder()
	sess, err := mod.Store().Get(req, SessionName)
	require.NoError(t, err)
	oldSeen := time.Now().Add(-time.Minute).Unix()
	sess.Values[SessionLastSeenKey] = oldSeen
	require.NoError(t, sess.Save(req, rr))

	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	rr = httptest.NewRecorder()
	require.NoError(t, mod.TouchSession(rr, req))
	req = requestWithCookies(rr, http.MethodGet, "http://example.test/api")
	sess, err = mod.Store().Get(req, SessionName)
	require.NoError(t, err)
	assert.Greater(t, sess.Values[SessionLastSeenKey].(int64), oldSeen)
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
	require.NoError(t, mod.SetSessionIdentity(sessionRR, sessionReq, Identity{Username: "admin", Groups: []string{"cerebro-admins"}}))

	handler := mod.APIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "admin", UserFrom(r.Context()))
		assert.Equal(t, []string{"cerebro-admins"}, GroupsFrom(r.Context()))
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
			Basic: config.BasicAuth{Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
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
