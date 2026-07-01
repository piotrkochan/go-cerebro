package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
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
		Auth:   config.Auth{Type: "disabled"},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAPIAuthGate_AllowsValidCSRFWhenAuthDisabled(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth:   config.Auth{Type: "disabled"},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		Auth:   config.Auth{Type: "disabled"},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	tokenReq := httptest.NewRequest(http.MethodGet, "http://example.test/auth/status", nil)
	tokenRR := httptest.NewRecorder()
	token, err := authMod.EnsureCSRFToken(tokenRR, tokenReq)
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			Type:     "basic",
			Settings: config.AuthSettings{Username: "admin", Password: "admin123"},
		},
		Server: config.Server{Secret: "test-secret", BasePath: "/"},
	})
	require.NoError(t, err)
	handler := apiAuthGate(authMod, config.Server{CSRFEnabled: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/clusters/local-cluster/overview", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
