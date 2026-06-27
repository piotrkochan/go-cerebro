package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
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
