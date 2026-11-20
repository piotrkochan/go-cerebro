package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trustedProxyAddr = "10.0.0.10"
	untrustedPeer    = "198.51.100.7:4444"
)

func proxyAuthHandler(t *testing.T, serverTrustedProxies ...string) http.Handler {
	t.Helper()
	cfg := &config.Config{
		Server: config.Server{
			BasePath:       "/",
			Secret:         "test-secret-not-default",
			TrustedProxies: serverTrustedProxies,
		},
		Auth: config.Auth{
			Proxy: map[string]config.ProxyAuth{
				"": {
					Enabled:        true,
					UserHeader:     "X-Forwarded-User",
					GroupsHeader:   "X-Forwarded-Groups",
					TrustedProxies: []string{trustedProxyAddr + "/32"},
				},
			},
		},
	}
	authMod, err := auth.NewModule(cfg)
	require.NoError(t, err)
	return New(Options{Cfg: cfg, Auth: authMod}).Handler()
}

func TestProxyAuth_IgnoresSpoofedClientIPHeaders(t *testing.T) {
	handler := proxyAuthHandler(t)

	for _, header := range []string{"True-Client-IP", "X-Real-IP", "X-Forwarded-For"} {
		t.Run(header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
			req.RemoteAddr = untrustedPeer
			req.Header.Set(header, trustedProxyAddr)
			req.Header.Set("X-Forwarded-User", "mallory")
			req.Header.Set("X-Forwarded-Groups", "cerebro-admins")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.NotContains(t, rr.Body.String(), "mallory")
		})
	}
}

func TestProxyAuth_AcceptsTrustedProxyForwardingClientAddress(t *testing.T) {
	handler := proxyAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	req.RemoteAddr = trustedProxyAddr + ":4444"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "alice")
}

func TestRequestOrigin_IgnoresForwardedHostFromSpoofedPeer(t *testing.T) {
	handler := proxyAuthHandler(t, trustedProxyAddr)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/auth/me", nil)
	req.RemoteAddr = untrustedPeer
	req.Header.Set("True-Client-IP", trustedProxyAddr)
	req.Header.Set("X-Forwarded-Host", "evil.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Header().Get("Link"), "http://example.test/schemas/")
	assert.NotContains(t, rr.Header().Get("Link"), "evil.test")
}

func TestRequestLogger_ReportsClientIPOnlyThroughTrustedProxies(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	cfg := &config.Config{
		Server:  config.Server{BasePath: "/", TrustedProxies: []string{trustedProxyAddr}},
		Logging: config.Logging{RequestLogEnabled: true},
	}
	authMod, err := auth.NewModule(cfg)
	require.NoError(t, err)
	handler := New(Options{Cfg: cfg, Auth: authMod}).Handler()

	viaProxy := httptest.NewRequest(http.MethodGet, "http://example.test/openapi.json", nil)
	viaProxy.RemoteAddr = trustedProxyAddr + ":4444"
	viaProxy.Header.Set("X-Forwarded-For", "203.0.113.5")
	handler.ServeHTTP(httptest.NewRecorder(), viaProxy)
	assert.Contains(t, buf.String(), "remote_addr="+trustedProxyAddr+":4444")
	assert.Contains(t, buf.String(), "client_ip=203.0.113.5")

	buf.Reset()
	direct := httptest.NewRequest(http.MethodGet, "http://example.test/openapi.json", nil)
	direct.RemoteAddr = untrustedPeer
	direct.Header.Set("X-Forwarded-For", "203.0.113.5")
	direct.Header.Set("True-Client-IP", "203.0.113.6")
	handler.ServeHTTP(httptest.NewRecorder(), direct)
	assert.Contains(t, buf.String(), "remote_addr="+untrustedPeer)
	assert.Contains(t, buf.String(), "client_ip=198.51.100.7")
	assert.NotContains(t, buf.String(), "203.0.113")
}
