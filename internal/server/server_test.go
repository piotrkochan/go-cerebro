package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeaders_SetHSTSForHTTPSWhenEnabled(t *testing.T) {
	handler := securityHeaders(config.Server{
		HSTSEnabled:           true,
		HSTSMaxAgeSeconds:     123,
		HSTSIncludeSubDomains: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=123; includeSubDomains", rr.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_SetHSTSForForwardedHTTPSWhenEnabled(t *testing.T) {
	handler := securityHeaders(config.Server{
		HSTSEnabled:       true,
		HSTSMaxAgeSeconds: 456,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=456", rr.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_OmitHSTSWhenDisabled(t *testing.T) {
	handler := securityHeaders(config.Server{
		HSTSEnabled:           false,
		HSTSMaxAgeSeconds:     31536000,
		HSTSIncludeSubDomains: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestServer_SchemeReflectsTLSConfig(t *testing.T) {
	httpServer := &Server{cfg: &config.Config{Server: config.Server{Port: 9000}}}
	assert.Equal(t, "http", httpServer.Scheme())

	httpsServer := &Server{cfg: &config.Config{Server: config.Server{
		Port:        9000,
		TLSCertFile: "/tmp/tls.crt",
		TLSKeyFile:  "/tmp/tls.key",
	}}}
	assert.Equal(t, "https", httpsServer.Scheme())
}

func TestServerTLSConfig_RequiresTLS12OrNewer(t *testing.T) {
	assert.Equal(t, uint16(tls.VersionTLS12), serverTLSConfig().MinVersion)
}

func TestRequestLoggerUsesSlog(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	handler := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest(http.MethodPost, "http://example.test/clusters/local/rest/history", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logLine := buf.String()
	assert.Contains(t, logLine, `msg="http request"`)
	assert.Contains(t, logLine, "method=POST")
	assert.Contains(t, logLine, "path=/clusters/local/rest/history")
	assert.Contains(t, logLine, "status=201")
	assert.Contains(t, logLine, "bytes=2")
	assert.Contains(t, logLine, "remote_addr=127.0.0.1:12345")
	assert.True(t, strings.Contains(logLine, "duration="))
}

func TestShouldGate_ProtectsClusterAPI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)

	assert.True(t, shouldGate(req))
}

func TestShouldGate_LeavesReactShellPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)

	assert.False(t, shouldGate(req))
}

func TestShouldGate_LeavesAuthStatusPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)

	assert.False(t, shouldGate(req))
}

func TestShouldGate_ProtectsAuthMe(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)

	assert.True(t, shouldGate(req))
}

func TestShouldGate_ProtectsLogout(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.test/auth/logout", nil)

	assert.True(t, shouldGate(req))
}

func TestShouldGate_LeavesEntraIDRedirectFlowPublic(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "login", path: "/auth/entraid/login"},
		{name: "callback", path: "/auth/entraid/callback?code=test-code&state=test-state"},
		{name: "oauth login", path: "/auth/oauth/login"},
		{name: "oauth callback", path: "/auth/oauth/callback?code=test-code&state=test-state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.path, nil)

			assert.False(t, shouldGate(req))
		})
	}
}

func TestMountBasePath(t *testing.T) {
	app := chi.NewRouter()
	app.Get("/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := mountBasePath("/cerebro", app)

	mountedReq := httptest.NewRequest(http.MethodGet, "http://example.test/cerebro/auth/status", nil)
	mountedRR := httptest.NewRecorder()
	handler.ServeHTTP(mountedRR, mountedReq)
	assert.Equal(t, http.StatusNoContent, mountedRR.Code)

	rootReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	rootRR := httptest.NewRecorder()
	handler.ServeHTTP(rootRR, rootReq)
	assert.Equal(t, http.StatusNotFound, rootRR.Code)
}

func TestRequestOriginUsesPublicURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://internal.test/auth/status", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "forwarded.example.org")

	assert.Equal(t, "https://public.example.org", requestOrigin(config.Server{PublicURL: "https://public.example.org"}, req))
}

func TestRequestOriginIgnoresForwardedHeadersFromUntrustedRemote(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://internal.test/auth/status", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "evil.example.org")

	assert.Equal(t, "http://internal.test", requestOrigin(config.Server{TrustedProxies: []string{"127.0.0.1/32"}}, req))
}

func TestRequestOriginUsesForwardedHeadersFromTrustedRemote(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://internal.test/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "cerebro.example.org")

	assert.Equal(t, "https://cerebro.example.org", requestOrigin(config.Server{TrustedProxies: []string{"127.0.0.1/32"}}, req))
}

func TestAPIAuthGate_RequiresCSRFWhenAuthDisabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIAuthGate_RBACRequiresAuthenticationProvider(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
		},
	})
	handler := apiAuthGate(authMod, config.Server{}, authorizer, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAPIAuthGate_AllowsValidCSRFWhenAuthDisabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	req.Header.Set("X-Cerebro-CSRF", token)
	for _, cookie := range tokenRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAPIAuthGate_LogoutRequiresCSRF(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "admin"))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/auth/logout", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIAuthGate_LogoutBypassesRBAC(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "viewer", Password: "secret"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "viewer", Role: "role:viewer"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
		},
	})
	handler := apiAuthGate(authMod, config.Server{}, authorizer, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusSeeOther)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "viewer"))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/auth/logout", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
}

func TestAPIAuthGate_RejectsCrossSiteFetchMetadata(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("X-Cerebro-CSRF", token)
	for _, cookie := range tokenRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIAuthGate_AuthenticatesBeforeCSRF(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAPIAuthGate_AuthMeSkipsCSRFToken(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "admin"))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAPIAuthGate_ConnectHostsSkipsCSRFToken(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "admin"))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/connect/hosts", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAPIAuthGate_ConnectRequiresSessionWhenAuthEnabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{}, rbac.New(config.RBAC{}), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/connect", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAPIAuthGate_ConnectBypassesRBACAfterAuthentication(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "viewer", Password: "secret"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Policies: []config.RBACPolicy{
			{Subject: "viewer", Resource: "overview", Action: "read", Object: "local-cluster", Effect: "allow"},
		},
	})
	handler := apiAuthGate(authMod, config.Server{}, authorizer, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "viewer"))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/connect", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAPIAuthGate_EnforcesRBAC(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "alice", Password: "secret"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)
	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "alice", Role: "role:viewer"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
		},
	})
	handler := apiAuthGate(authMod, config.Server{}, authorizer, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "alice"))

	readReq := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		readReq.AddCookie(cookie)
	}
	readRR := httptest.NewRecorder()
	handler.ServeHTTP(readRR, readReq)
	assert.Equal(t, http.StatusNoContent, readRR.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, "http://example.test/clusters/local-cluster/overview/indices/logs-000001", nil)
	for _, cookie := range loginRR.Result().Cookies() {
		deleteReq.AddCookie(cookie)
	}
	deleteRR := httptest.NewRecorder()
	handler.ServeHTTP(deleteRR, deleteReq)
	assert.Equal(t, http.StatusForbidden, deleteRR.Code)
	assert.Equal(t, "application/problem+json", deleteRR.Header().Get("Content-Type"))
	assert.Equal(t, `<http://example.test/schemas/ErrorModel.json>; rel="describedBy"`, deleteRR.Header().Get("Link"))
	var problem map[string]any
	require.NoError(t, json.Unmarshal(deleteRR.Body.Bytes(), &problem))
	assert.Equal(t, "http://example.test/schemas/ErrorModel.json", problem["$schema"])
	assert.Equal(t, float64(http.StatusForbidden), problem["status"])
	assert.Equal(t, "Forbidden", problem["title"])
	assert.Equal(t, "Permission denied.", problem["detail"])
}

func TestAPIAuthGate_EnforcesRBACIndexObjectsAndActions(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: map[string]config.BasicAuth{
				config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "alice", Password: "secret"}}},
			},
		},
		Server: config.Server{BasePath: "/"},
	})
	require.NoError(t, err)

	authorizer := rbac.New(config.RBAC{
		Enabled: true,
		Bindings: []config.RBACBinding{
			{Subject: "alice", Role: "role:index-maintainer"},
		},
		Policies: []config.RBACPolicy{
			{Subject: "role:index-maintainer", Resource: "indices", Action: "read", Object: "prod/index-*", Effect: "allow"},
			{Subject: "role:index-maintainer", Resource: "indices", Action: "delete", Object: "prod/index-*", Effect: "allow"},
			{Subject: "role:index-maintainer", Resource: "documents", Action: "write", Object: "prod/index-*", Effect: "allow"},
			{Subject: "role:index-maintainer", Resource: "snapshots", Action: "read", Object: "prod/*/*", Effect: "allow"},
		},
	})
	handler := apiAuthGate(authMod, config.Server{}, authorizer, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	loginReq := httptest.NewRequest(http.MethodPost, "http://example.test/auth/login", nil)
	loginRR := httptest.NewRecorder()
	require.NoError(t, authMod.SetSessionUser(loginRR, loginReq, "alice"))
	cookies := loginRR.Result().Cookies()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "can read matching index settings",
			method:     http.MethodGet,
			path:       "/clusters/prod/commons/indices/index-users/settings",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "cannot read non matching index settings",
			method:     http.MethodGet,
			path:       "/clusters/prod/commons/indices/app-users/settings",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "can delete matching index",
			method:     http.MethodDelete,
			path:       "/clusters/prod/overview/indices/index-users",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "cannot delete non matching index",
			method:     http.MethodDelete,
			path:       "/clusters/prod/overview/indices/app-users",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "cannot refresh matching index without refresh action",
			method:     http.MethodPost,
			path:       "/clusters/prod/overview/indices/index-users/refresh",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "can write documents in matching index",
			method:     http.MethodPut,
			path:       "/clusters/prod/data_explorer/index-users/documents",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "cannot write documents in non matching index",
			method:     http.MethodPut,
			path:       "/clusters/prod/data_explorer/app-users/documents",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "can read two level snapshot object",
			method:     http.MethodGet,
			path:       "/clusters/prod/snapshots/repository/snapshot",
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.test"+tt.path, nil)
			for _, cookie := range cookies {
				req.AddCookie(cookie)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}
