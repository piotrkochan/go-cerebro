//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyAuthThroughTrustedReverseProxy(t *testing.T) {
	authMod, err := auth.NewModule(&config.Config{
		Auth: config.Auth{
			Proxy: config.ProxyAuth{
				Enabled:        true,
				UserHeader:     "X-Forwarded-User",
				GroupsHeader:   "X-Forwarded-Groups",
				GroupSeparator: ",",
				TrustedProxies: []string{"127.0.0.1/32"},
			},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	upstream := httptest.NewServer(authMod.APIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := auth.IdentityFrom(r.Context())
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":   identity.Username,
			"groups": identity.Groups,
		})
	})))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Forwarded-User", "alice@example.org")
		r.Header.Set("X-Forwarded-Groups", "cerebro-admins,operators")
		httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
	}))
	defer proxy.Close()

	res, err := http.Get(proxy.URL + "/api")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	var body struct {
		User   string   `json:"user"`
		Groups []string `json:"groups"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	assert.Equal(t, "alice@example.org", body.User)
	assert.Equal(t, []string{"cerebro-admins", "operators"}, body.Groups)
}
