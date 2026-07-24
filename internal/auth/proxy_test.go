package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyAuthenticatorIdentity(t *testing.T) {
	proxy, err := NewProxyAuthenticator(config.ProxyAuth{
		UserHeader:     "X-Forwarded-User",
		GroupsHeader:   "X-Forwarded-Groups",
		GroupSeparator: ",",
		DefaultGroups:  []string{"cerebro-viewers", "developers"},
		TrustedProxies: []string{"127.0.0.1/32"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "http://example.test/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-User", "alice@example.com")
	req.Header.Add("X-Forwarded-Groups", "cerebro-admins, developers")
	req.Header.Add("X-Forwarded-Groups", "developers")

	identity, ok := proxy.Identity(req)

	require.True(t, ok)
	assert.Equal(t, "alice@example.com", identity.Username)
	assert.Equal(t, []string{"cerebro-viewers", "developers", "cerebro-admins"}, identity.Groups)
}

func TestProxyAuthenticatorUsesOAuthProxyFallbackHeaders(t *testing.T) {
	proxy, err := NewProxyAuthenticator(config.ProxyAuth{
		UserHeader:     "X-Forwarded-User",
		TrustedProxies: []string{"127.0.0.1/32"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "http://example.test/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-Preferred-Username", "alice")

	identity, ok := proxy.Identity(req)

	require.True(t, ok)
	assert.Equal(t, "alice", identity.Username)
	assert.Equal(t, "proxy", identity.Provider)
	assert.Empty(t, identity.ProviderID)
}

func TestNamedProxyAuthenticatorReportsProviderID(t *testing.T) {
	proxy, err := NewNamedProxyAuthenticator("github", config.ProxyAuth{
		UserHeader:     "X-Forwarded-User",
		TrustedProxies: []string{"127.0.0.1/32"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "http://example.test/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-User", "alice")

	identity, ok := proxy.Identity(req)

	require.True(t, ok)
	assert.Equal(t, "proxy", identity.Provider)
	assert.Equal(t, "github", identity.ProviderID)
}

func TestProxyAuthenticatorRejectsUntrustedRemote(t *testing.T) {
	proxy, err := NewProxyAuthenticator(config.ProxyAuth{
		UserHeader:     "X-Forwarded-User",
		TrustedProxies: []string{"10.0.0.0/8"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "http://example.test/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-User", "alice")

	_, ok := proxy.Identity(req)

	assert.False(t, ok)
}

func TestProxyAuthenticatorRequiresTrustedProxies(t *testing.T) {
	_, err := NewProxyAuthenticator(config.ProxyAuth{UserHeader: "X-Forwarded-User"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "trusted_proxies")
}
