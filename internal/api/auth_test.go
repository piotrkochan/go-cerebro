package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAuthMeReturnsIdentity(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	deps := &Deps{
		Auth: authMod,
		Cfg: &config.Config{RBAC: config.RBAC{
			Enabled:     true,
			DefaultRole: "role:viewer",
			Bindings: []config.RBACBinding{
				{Subject: "group:cerebro-admins", Role: "role:admin"},
			},
			Policies: []config.RBACPolicy{
				{Subject: "role:viewer", Resource: "overview", Action: "read", Object: "*", Effect: "allow"},
				{Subject: "role:admin", Resource: "*", Action: "*", Object: "*", Effect: "allow"},
			},
		}},
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	sessionRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionIdentity(sessionRR, sessionReq, auth.Identity{
		Username:   "admin",
		Groups:     []string{"cerebro-admins"},
		Provider:   "basic",
		ProviderID: "local",
	}))
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	for _, cookie := range sessionRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	deps.handleAuthMe(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, true, got["authenticated"])
	assert.Equal(t, "admin", got["user"])
	assert.Equal(t, "basic", got["provider"])
	assert.Equal(t, "local", got["provider_id"])
	assert.Equal(t, []any{"cerebro-admins"}, got["groups"])
	assert.Equal(t, []any{"role:viewer", "role:admin"}, got["roles"])
	assert.Equal(t, []any{
		map[string]any{"action": "read", "effect": "allow", "object": "*", "resource": "overview"},
		map[string]any{"action": "*", "effect": "allow", "object": "*", "resource": "*"},
	}, got["permissions"])
}

func TestHandleAuthMeRequiresSession(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	deps := &Deps{Auth: authMod}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	rr := httptest.NewRecorder()

	deps.handleAuthMe(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleLogoutReturnsProviderRedirectForJSONClients(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Proxy: map[string]config.ProxyAuth{
				"github": {
					Enabled:        true,
					UserHeader:     "X-Forwarded-User",
					LogoutURL:      "/oauth2/sign_out?rd=/oauth2/sign_in",
					TrustedProxies: []string{"127.0.0.1/32"},
				},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	deps := &Deps{Auth: authMod, Cfg: &config.Config{Server: config.Server{BasePath: "/"}}}
	req := httptest.NewRequest(http.MethodPost, "http://example.test/auth/logout", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Forwarded-User", "alice")
	req.RemoteAddr = "127.0.0.1:12345"

	got, err := deps.logout(WithHTTPRequest(context.Background(), req))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, got.Status)
	assert.Equal(t, "http://example.test/schemas/LogoutResponse.json", got.Body.Schema)
	assert.Equal(t, "/oauth2/sign_out?rd=/oauth2/sign_in", got.Body.RedirectURL)
}

func TestLogoutClearsSessionAndReturnsLoginRedirect(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	deps := &Deps{Auth: authMod, Cfg: &config.Config{Server: config.Server{BasePath: "/"}}}
	sessionReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	sessionRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionIdentity(sessionRR, sessionReq, auth.Identity{Username: "admin", Provider: "basic"}))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/auth/logout", nil)
	for _, cookie := range sessionRR.Result().Cookies() {
		req.AddCookie(cookie)
	}

	got, err := deps.logout(WithHTTPRequest(context.Background(), req))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, got.Status)
	assert.Equal(t, "/#/login", got.Body.RedirectURL)
	require.NotEmpty(t, got.SetCookie)
	assert.Contains(t, got.SetCookie[0], "Max-Age=0")
}

func TestHandleAuthStatusReportsOAuthProvider(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			OAuth: map[string]config.OAuthAuth{
				"github": {
					Enabled:      true,
					Name:         "GitHub",
					AuthURL:      "https://github.com/login/oauth/authorize",
					TokenURL:     "https://github.com/login/oauth/access_token",
					UserInfoURL:  "https://api.github.com/user",
					ClientID:     "client",
					ClientSecret: "secret",
				},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	deps := &Deps{Auth: authMod, Cfg: &config.Config{}}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	rr := httptest.NewRecorder()

	deps.handleAuthStatus(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, map[string]any{"entraid": false, "oauth": true, "password": false}, got["providers"])
	assert.Equal(t, map[string]any{"oauth": "GitHub"}, got["provider_names"])
	assert.Equal(t, []any{map[string]any{
		"id":         "github",
		"kind":       "oauth",
		"login_path": "/auth/oauth/github/login",
		"name":       "GitHub",
	}}, got["external_providers"])
}

func TestAuthRolesAndPermissionsUseSamePrincipalsAsRBAC(t *testing.T) {
	rbac := config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "user:alice", Role: "role:user-prefix"},
			{Subject: "operators", Role: "role:raw-group"},
			{Subject: "group:cerebro-admins", Role: "role:admin"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:user-prefix", Resource: "overview", Action: "read", Object: "*", Effect: "allow"},
			{Subject: "role:raw-group", Resource: "nodes", Action: "read", Object: "*", Effect: "allow"},
			{Subject: "role:admin", Resource: "*", Action: "*", Object: "*", Effect: "allow"},
		},
	}
	identity := auth.Identity{Username: "alice", Groups: []string{"operators", "cerebro-admins"}}

	assert.Equal(t, []string{"role:user-prefix", "role:raw-group", "role:admin"}, authRoles(rbac, identity))
	assert.Equal(t, []authPermission{
		{Resource: "overview", Action: "read", Object: "*", Effect: "allow"},
		{Resource: "nodes", Action: "read", Object: "*", Effect: "allow"},
		{Resource: "*", Action: "*", Object: "*", Effect: "allow"},
	}, authPermissions(rbac, identity))
}
