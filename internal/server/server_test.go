package server

import (
	"bytes"
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestAPIAuthGate_RequiresCSRFWhenAuthDisabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIAuthGate_AllowsValidCSRFWhenAuthDisabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAPIAuthGate_RejectsCrossSiteFetchMetadata(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			Basic: config.BasicAuth{Enabled: true, Username: "admin", Password: "admin123"},
		},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true}, rbac.New(config.RBAC{}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAPIAuthGate_EnforcesRBAC(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Basic: config.BasicAuth{Enabled: true, Username: "alice", Password: "secret"},
		},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
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
	handler := apiAuthGate(authMod, config.Server{}, authorizer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
}
