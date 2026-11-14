package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasePathFrontend(t *testing.T) {
	// Keep routing tests independent of whether Vite has been built locally.
	originalFS := embeddedStatic
	embeddedStatic = fstest.MapFS{
		"index.html":       {Data: []byte(`<script type="module" src="./assets/index.js"></script>`)},
		"assets/index.js":  {Data: []byte(`import('./editor.js')`)},
		"assets/editor.js": {Data: []byte(`export const editor = true`)},
		"assets/index.css": {Data: []byte(`body { color: black }`)},
	}
	t.Cleanup(func() { embeddedStatic = originalFS })

	for _, base := range []string{"", "/", "/cerebro", "/tools/cerebro/"} {
		t.Run(base, func(t *testing.T) {
			cfg := &config.Config{Server: config.Server{BasePath: base}}
			authMod, err := auth.NewModule(cfg)
			require.NoError(t, err)
			handler := New(Options{Cfg: cfg, Auth: authMod}).Handler()
			prefix := strings.TrimRight(base, "/")
			pageURL := "http://example.test" + prefix + "/"

			if prefix != "" {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://example.test"+prefix+"?next=a%2Fb&x=1", nil))
				assert.Equal(t, http.StatusPermanentRedirect, rr.Code)
				assert.Equal(t, prefix+"/?next=a%2Fb&x=1", rr.Header().Get("Location"))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, pageURL, nil))
			require.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), `src="./assets/index.js"`)
			page, err := url.Parse(pageURL)
			require.NoError(t, err)
			for _, asset := range []string{"./assets/index.js", "./assets/index.css", "./assets/editor.js"} {
				ref, err := url.Parse(asset)
				require.NoError(t, err)
				assetRR := httptest.NewRecorder()
				handler.ServeHTTP(assetRR, httptest.NewRequest(http.MethodGet, page.ResolveReference(ref).String(), nil))
				assert.Equal(t, http.StatusOK, assetRR.Code, asset)
				assert.NotContains(t, assetRR.Body.String(), "<script", asset)
			}
			for _, path := range []string{prefix + "/assets/missing.js", prefix + "/index.html", prefix + "/../other/assets/index.js"} {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil))
				assert.Equal(t, http.StatusNotFound, rr.Code, path)
			}
		})
	}
}

func TestBasePathAuthentication(t *testing.T) {
	for _, base := range []string{"/", "/cerebro", "/tools/cerebro/"} {
		t.Run(base, func(t *testing.T) {
			prefix := strings.TrimRight(base, "/")
			cfg := &config.Config{
				Server: config.Server{BasePath: base, CSRFEnabled: true, MaxRequestBytes: 1 << 20},
				Auth: config.Auth{Basic: map[string]config.BasicAuth{
					config.DefaultAuthProviderID: {Enabled: true, Users: []config.BasicAuthUser{{Username: "viewer", Password: "secret"}}},
				}},
				RBAC: config.RBAC{Enabled: true, Policies: []config.RBACPolicy{
					{Subject: "viewer", Resource: "*", Action: "read", Object: "*", Effect: "allow"},
				}},
			}
			authMod, err := auth.NewModule(cfg)
			require.NoError(t, err)
			handler := New(Options{Cfg: cfg, Auth: authMod}).Handler()
			request := func(method, path, body string, cookies []*http.Cookie, token string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, "http://example.test"+prefix+path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Cerebro-CSRF", token)
				for _, cookie := range cookies {
					req.AddCookie(cookie)
				}
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				return rr
			}

			for _, path := range []string{"/connect/hosts", "/auth/me", "/clusters/local/overview"} {
				assert.Equal(t, http.StatusUnauthorized, request(http.MethodGet, path, "", nil, "").Code, path)
			}
			assert.Equal(t, http.StatusUnauthorized, request(http.MethodPost, "/auth/logout", "", nil, "").Code)

			assert.Equal(t, http.StatusOK, request(http.MethodGet, "/auth/status", "", nil, "").Code)
			login := request(http.MethodPost, "/auth/login", `{"user":"viewer","password":"secret"}`, nil, "")
			require.Equal(t, http.StatusSeeOther, login.Code)
			assert.Equal(t, prefix+"/", login.Header().Get("Location"))
			status := request(http.MethodGet, "/auth/status", "", login.Result().Cookies(), "")
			require.Equal(t, http.StatusOK, status.Code)
			var statusBody struct {
				Token string `json:"csrf_token"`
			}
			require.NoError(t, json.Unmarshal(status.Body.Bytes(), &statusBody))
			require.NotEmpty(t, statusBody.Token)
			cookies := login.Result().Cookies()
			for _, cookie := range cookies {
				assert.Equal(t, prefix+"/", cookie.Path)
			}

			for _, path := range []string{"/connect/hosts", "/auth/me"} {
				assert.Equal(t, http.StatusOK, request(http.MethodGet, path, "", cookies, "").Code, path)
			}
			assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/auth/logout", "", cookies, "").Code)
			denied := request(http.MethodDelete, "/clusters/local/overview/indices/logs", "", cookies, statusBody.Token)
			assert.Equal(t, http.StatusForbidden, denied.Code)
			assert.Contains(t, denied.Body.String(), "Permission denied.")
			logout := request(http.MethodPost, "/auth/logout", "", cookies, statusBody.Token)
			assert.Equal(t, http.StatusOK, logout.Code)
			assert.Contains(t, logout.Body.String(), prefix+"/#/login")
		})
	}
}
