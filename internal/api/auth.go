package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterAuth registers POST /auth/login, POST /auth/logout. The GET /login screen is served
// by the static handler in the server package. Login accepts both JSON and form-encoded bodies
// to match the original Cerebro form-based login.
func (d *Deps) RegisterAuth(api huma.API, mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}) {
	// We register login/logout directly on chi (mux) instead of via Huma — they need flexible
	// content-type handling (form vs JSON) and cookie writing on the http.ResponseWriter.
	mux.HandleFunc("GET /auth/status", d.handleAuthStatus)
	mux.HandleFunc("POST /auth/login", d.handleLogin)
	mux.HandleFunc("GET /auth/entraid/login", d.handleEntraIDLogin)
	mux.HandleFunc("GET /auth/entraid/callback", d.handleEntraIDCallback)
	mux.HandleFunc("POST /auth/logout", d.handleLogout)
}

func (d *Deps) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	user, authenticated := d.Auth.SessionUser(r)
	csrfToken := ""
	if d.Cfg.Server.CSRFEnabled {
		var err error
		csrfToken, err = d.Auth.EnsureCSRFToken(w, r)
		if err != nil {
			http.Error(w, "csrf token error", http.StatusInternalServerError)
			return
		}
	}
	if !d.Auth.Enabled() {
		authenticated = false
		user = ""
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": authenticated,
		"csrf_token":    csrfToken,
		"enabled":       d.Auth.Enabled(),
		"providers": map[string]bool{
			"entraid":  d.Auth.EntraIDEnabled(),
			"password": d.Auth.PasswordLoginEnabled(),
		},
		"user": user,
	})
}

func (d *Deps) handleEntraIDLogin(w http.ResponseWriter, r *http.Request) {
	callbackURL := d.entraIDCallbackURL(r)
	returnPath := r.URL.Query().Get("redirect")
	if returnPath == "" {
		returnPath = "/"
	}
	authURL, err := d.Auth.BeginEntraIDLogin(w, r, callbackURL, returnPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (d *Deps) handleEntraIDCallback(w http.ResponseWriter, r *http.Request) {
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	redirect, err := d.Auth.CompleteEntraIDLogin(w, r, d.entraIDCallbackURL(r))
	if err != nil {
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect is same-origin absolute path from auth.CompleteEntraIDLogin.
}

func (d *Deps) handleLogin(w http.ResponseWriter, r *http.Request) {
	var user, password string
	r.Body = http.MaxBytesReader(w, r.Body, d.Cfg.Server.MaxRequestBytes)
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid login form data", http.StatusBadRequest)
			return
		}
		user = r.FormValue("user")
		password = r.FormValue("password")
	} else {
		var payload struct {
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid login json data", http.StatusBadRequest)
			return
		}
		user = payload.User
		password = payload.Password
	}
	if user == "" || password == "" {
		http.Error(w, "invalid login form data", http.StatusBadRequest)
		return
	}
	identity, err := d.Auth.Authenticate(user, password)
	if err != nil {
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	if err := d.Auth.SetSessionIdentity(w, r, identity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect := d.Auth.ConsumeRedirect(w, r)
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect comes from auth.ConsumeRedirect, which accepts only same-origin absolute paths.
}

func (d *Deps) handleLogout(w http.ResponseWriter, r *http.Request) {
	_ = d.Auth.ClearSession(w, r)
	http.Redirect(w, r, basePathFor(d, "/#/login"), http.StatusSeeOther)
}

func basePathFor(d *Deps, suffix string) string {
	prefix := strings.TrimRight(d.Cfg.Server.BasePath, "/")
	if prefix == "" {
		return suffix
	}
	return prefix + suffix
}

func (d *Deps) entraIDCallbackURL(r *http.Request) string {
	if configured := d.Auth.EntraIDRedirectURL(); configured != "" {
		return configured
	}
	return requestOrigin(r) + basePathFor(d, "/auth/entraid/callback")
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}
