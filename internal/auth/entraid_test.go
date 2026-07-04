package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntraIDLoginWithMockOIDCProvider(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var issuer string
	var nonce string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, map[string]any{
				"issuer":                 issuer,
				"authorization_endpoint": issuer + "/authorize",
				"token_endpoint":         issuer + "/token",
				"jwks_uri":               issuer + "/keys",
			})
		case "/keys":
			writeJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
				Key:       key.Public(),
				KeyID:     "test-key",
				Algorithm: string(jose.RS256),
				Use:       "sig",
			}}})
		case "/token":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "authorization_code", r.FormValue("grant_type"))
			require.Equal(t, "test-code", r.FormValue("code"))
			idToken := signedIDToken(t, issuer, key, nonce)
			writeJSON(t, w, map[string]any{
				"access_token": "access-token",
				"expires_in":   3600,
				"id_token":     idToken,
				"token_type":   "Bearer",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	issuer = provider.URL

	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			EntraID: config.EntraIDAuth{
				Enabled:      true,
				IssuerURL:    issuer,
				ClientID:     "cerebro-client",
				ClientSecret: "client-secret",
				GroupsClaim:  "groups",
			},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)

	beginReq := httptest.NewRequest(http.MethodGet, "http://cerebro.test/auth/entraid/login", nil)
	beginRR := httptest.NewRecorder()
	authURL, err := mod.BeginEntraIDLogin(beginRR, beginReq, "http://cerebro.test/auth/entraid/callback", "/#/overview")
	require.NoError(t, err)
	parsedAuthURL, err := url.Parse(authURL)
	require.NoError(t, err)
	nonce = parsedAuthURL.Query().Get("nonce")
	require.NotEmpty(t, nonce)

	callbackReq := requestWithCookies(beginRR, http.MethodGet, "http://cerebro.test/auth/entraid/callback?code=test-code&state="+url.QueryEscape(parsedAuthURL.Query().Get("state")))
	callbackRR := httptest.NewRecorder()
	redirect, err := mod.CompleteEntraIDLogin(callbackRR, callbackReq, "http://cerebro.test/auth/entraid/callback")
	require.NoError(t, err)
	assert.Equal(t, "/#/overview", redirect)

	sessionReq := requestWithCookies(callbackRR, http.MethodGet, "http://cerebro.test/auth/status")
	identity, ok := mod.SessionIdentity(sessionReq)
	require.True(t, ok)
	assert.Equal(t, "alice@example.org", identity.Username)
	assert.Equal(t, []string{"cerebro-admins", "operators"}, identity.Groups)
	assert.Equal(t, "entra_id", identity.Provider)
}

func signedIDToken(t *testing.T, issuer string, key *rsa.PrivateKey, nonce string) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	require.NoError(t, err)
	now := time.Now()
	claims := jwt.Claims{
		Issuer:    issuer,
		Subject:   "alice-subject",
		Audience:  jwt.Audience{"cerebro-client"},
		Expiry:    jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
	}
	privateClaims := struct {
		PreferredUsername string   `json:"preferred_username"`
		Groups            []string `json:"groups"`
		Nonce             string   `json:"nonce"`
	}{
		PreferredUsername: "alice@example.org",
		Groups:            []string{"cerebro-admins", "operators"},
		Nonce:             nonce,
	}
	raw, err := jwt.Signed(signer).Claims(claims).Claims(privateClaims).Serialize()
	require.NoError(t, err)
	return raw
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

func TestEntraIDProviderRequiresTenantOrIssuer(t *testing.T) {
	_, err := NewEntraIDProvider(config.EntraIDAuth{ClientID: "client", ClientSecret: "secret"})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "tenant_id") || strings.Contains(err.Error(), "issuer"))
}

func TestEntraIDGroupsOverageDetection(t *testing.T) {
	assert.False(t, entraIDGroupsOverage(map[string]any{"groups": []any{"admins"}}, "groups"))
	assert.False(t, entraIDGroupsOverage(map[string]any{}, "groups"))
	assert.True(t, entraIDGroupsOverage(map[string]any{
		"_claim_names": map[string]any{"groups": "src1"},
	}, "groups"))
}
