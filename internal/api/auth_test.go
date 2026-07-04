package api

import (
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
			Basic: config.BasicAuth{Enabled: true, Username: "admin", Password: "admin123"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
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
		}},
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	sessionRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionIdentity(sessionRR, sessionReq, auth.Identity{
		Username: "admin",
		Groups:   []string{"cerebro-admins"},
		Provider: "basic",
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
	assert.Equal(t, []any{"cerebro-admins"}, got["groups"])
	assert.Equal(t, []any{"role:viewer", "role:admin"}, got["roles"])
}

func TestHandleAuthMeRequiresSession(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: config.BasicAuth{Enabled: true, Username: "admin", Password: "admin123"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	deps := &Deps{Auth: authMod}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	rr := httptest.NewRecorder()

	deps.handleAuthMe(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
