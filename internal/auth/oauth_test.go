package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthLoginWithMockProvider(t *testing.T) {
	var providerURL string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
			assert.Equal(t, "test-code", r.FormValue("code"))
			writeJSON(t, w, map[string]any{
				"access_token": "access-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
		case "/userinfo":
			assert.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
			writeJSON(t, w, map[string]any{
				"groups": []string{"cerebro-admins", "operators"},
				"login":  "alice",
				"sub":    "alice-subject",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	providerURL = provider.URL

	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			OAuth: config.OAuthAuth{
				Enabled:      true,
				Name:         "Test OAuth",
				AuthURL:      providerURL + "/authorize",
				TokenURL:     providerURL + "/token",
				UserInfoURL:  providerURL + "/userinfo",
				ClientID:     "cerebro-client",
				ClientSecret: "client-secret",
				GroupsClaim:  "groups",
			},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	beginReq := httptest.NewRequest(http.MethodGet, "http://cerebro.test/auth/oauth/login", nil)
	beginRR := httptest.NewRecorder()
	authURL, err := mod.BeginOAuthLogin(beginRR, beginReq, "http://cerebro.test/auth/oauth/callback", "/#/overview")
	require.NoError(t, err)
	parsedAuthURL, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, providerURL+"/authorize", parsedAuthURL.Scheme+"://"+parsedAuthURL.Host+parsedAuthURL.Path)
	require.NotEmpty(t, parsedAuthURL.Query().Get("state"))

	callbackReq := requestWithCookies(beginRR, http.MethodGet, "http://cerebro.test/auth/oauth/callback?code=test-code&state="+url.QueryEscape(parsedAuthURL.Query().Get("state")))
	callbackRR := httptest.NewRecorder()
	redirect, err := mod.CompleteOAuthLogin(callbackRR, callbackReq, "http://cerebro.test/auth/oauth/callback")
	require.NoError(t, err)
	assert.Equal(t, "/#/overview", redirect)

	sessionReq := requestWithCookies(callbackRR, http.MethodGet, "http://cerebro.test/auth/status")
	identity, ok := mod.SessionIdentity(sessionReq)
	require.True(t, ok)
	assert.Equal(t, "alice", identity.Username)
	assert.Equal(t, []string{"cerebro-admins", "operators"}, identity.Groups)
	assert.Equal(t, "oauth", identity.Provider)
}

func TestOAuthLoginRejectsInvalidState(t *testing.T) {
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

	beginReq := httptest.NewRequest(http.MethodGet, "http://cerebro.test/auth/oauth/login", nil)
	beginRR := httptest.NewRecorder()
	_, err = mod.BeginOAuthLogin(beginRR, beginReq, "http://cerebro.test/auth/oauth/callback", "/")
	require.NoError(t, err)

	callbackReq := requestWithCookies(beginRR, http.MethodGet, "http://cerebro.test/auth/oauth/callback?code=test-code&state=wrong")
	callbackRR := httptest.NewRecorder()
	_, err = mod.CompleteOAuthLogin(callbackRR, callbackReq, "http://cerebro.test/auth/oauth/callback")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestOAuthUserInfoRequiresUsername(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
			})
		case "/userinfo":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			OAuth: config.OAuthAuth{
				Enabled:      true,
				AuthURL:      provider.URL + "/authorize",
				TokenURL:     provider.URL + "/token",
				UserInfoURL:  provider.URL + "/userinfo",
				ClientID:     "client",
				ClientSecret: "secret",
			},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	beginReq := httptest.NewRequest(http.MethodGet, "http://cerebro.test/auth/oauth/login", nil)
	beginRR := httptest.NewRecorder()
	authURL, err := mod.BeginOAuthLogin(beginRR, beginReq, "http://cerebro.test/auth/oauth/callback", "/")
	require.NoError(t, err)
	parsedAuthURL, err := url.Parse(authURL)
	require.NoError(t, err)

	callbackReq := requestWithCookies(beginRR, http.MethodGet, "http://cerebro.test/auth/oauth/callback?code=test-code&state="+url.QueryEscape(parsedAuthURL.Query().Get("state")))
	callbackRR := httptest.NewRecorder()
	_, err = mod.CompleteOAuthLogin(callbackRR, callbackReq, "http://cerebro.test/auth/oauth/callback")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "username claim")
}
