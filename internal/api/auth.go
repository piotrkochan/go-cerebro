package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
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
	mux.HandleFunc("GET /auth/me", d.handleAuthMe)
	mux.HandleFunc("POST /auth/login", d.handleLogin)
	mux.HandleFunc("GET /auth/entraid/login", d.handleEntraIDLogin)
	mux.HandleFunc("GET /auth/entraid/callback", d.handleEntraIDCallback)
	mux.HandleFunc("POST /auth/logout", d.handleLogout)
}

func (d *Deps) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	identity, authenticated := d.Auth.SessionIdentity(r)
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
		identity = auth.Identity{}
	}
	if authenticated {
		_ = d.Auth.TouchSession(w, r)
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
		"groups":      identity.Groups,
		"permissions": authPermissions(d.Cfg.RBAC, identity),
		"provider":    identity.Provider,
		"roles":       authRoles(d.Cfg.RBAC, identity),
		"user":        identity.Username,
	})
}

func (d *Deps) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if !d.Auth.Enabled() {
		http.Error(w, "authentication not enabled", http.StatusUnauthorized)
		return
	}
	identity, authenticated := d.Auth.SessionIdentity(r)
	if !authenticated {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_ = d.Auth.TouchSession(w, r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"groups":        identity.Groups,
		"permissions":   authPermissions(d.Cfg.RBAC, identity),
		"provider":      identity.Provider,
		"roles":         authRoles(d.Cfg.RBAC, identity),
		"user":          identity.Username,
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
		d.auditAuth(r, "entraid_login_start_failed", auth.Identity{}, "failure", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.auditAuth(r, "entraid_login_started", auth.Identity{Provider: "entra_id"}, "success", "")
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (d *Deps) handleEntraIDCallback(w http.ResponseWriter, r *http.Request) {
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		d.auditAuth(r, "entraid_login_failed", auth.Identity{Provider: "entra_id"}, "failure", authErr)
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	redirect, err := d.Auth.CompleteEntraIDLogin(w, r, d.entraIDCallbackURL(r))
	if err != nil {
		d.auditAuth(r, "entraid_login_failed", auth.Identity{Provider: "entra_id"}, "failure", err.Error())
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	d.auditAuth(r, "entraid_login_succeeded", auth.Identity{Provider: "entra_id"}, "success", "")
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
		d.auditAuth(r, "password_login_failed", auth.Identity{Username: user, Provider: "password"}, "failure", "missing credentials")
		http.Error(w, "invalid login form data", http.StatusBadRequest)
		return
	}
	identity, err := d.Auth.Authenticate(user, password)
	if err != nil {
		d.auditAuth(r, "password_login_failed", auth.Identity{Username: user, Provider: "password"}, "failure", "invalid credentials")
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	if err := d.Auth.SetSessionIdentity(w, r, identity); err != nil {
		d.auditAuth(r, "password_login_failed", identity, "failure", "session error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.auditAuth(r, "password_login_succeeded", identity, "success", "")
	redirect := d.Auth.ConsumeRedirect(w, r)
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect comes from auth.ConsumeRedirect, which accepts only same-origin absolute paths.
}

func (d *Deps) handleLogout(w http.ResponseWriter, r *http.Request) {
	identity, _ := d.Auth.SessionIdentity(r)
	_ = d.Auth.ClearSession(w, r)
	d.auditAuth(r, "logout", identity, "success", "")
	http.Redirect(w, r, basePathFor(d, "/#/login"), http.StatusSeeOther)
}

func (d *Deps) auditAuth(r *http.Request, event string, identity auth.Identity, outcome, reason string) {
	if d.Cfg != nil && !d.Cfg.Logging.AuthLogEnabled {
		return
	}
	attrs := []any{
		"event", event,
		"outcome", outcome,
		"user", identity.Username,
		"provider", identity.Provider,
		"remote_addr", r.RemoteAddr,
	}
	if reason != "" {
		attrs = append(attrs, "reason", reason)
	}
	slog.InfoContext(r.Context(), "auth audit", attrs...)
}

func authRoles(rbac config.RBAC, identity auth.Identity) []string {
	if !rbac.Enabled || identity.Username == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(role string) {
		if role != "" && !seen[role] {
			seen[role] = true
			out = append(out, role)
		}
	}
	add(rbac.DefaultRole)
	groupSubjects := map[string]bool{}
	for _, group := range identity.Groups {
		groupSubjects["group:"+group] = true
	}
	for _, binding := range rbac.Bindings {
		if binding.Subject == identity.Username || groupSubjects[binding.Subject] {
			add(binding.Role)
		}
	}
	return out
}

type authPermission struct {
	Action   string `json:"action"`
	Effect   string `json:"effect"`
	Object   string `json:"object"`
	Resource string `json:"resource"`
}

func authPermissions(rbac config.RBAC, identity auth.Identity) []authPermission {
	if !rbac.Enabled || identity.Username == "" {
		return nil
	}
	subjects := map[string]bool{identity.Username: true}
	for _, group := range identity.Groups {
		if group != "" {
			subjects["group:"+group] = true
		}
	}
	for _, role := range authRoles(rbac, identity) {
		if role != "" {
			subjects[role] = true
		}
	}
	out := make([]authPermission, 0, len(rbac.Policies))
	seen := map[authPermission]bool{}
	for _, policy := range rbac.Policies {
		if !subjects[policy.Subject] {
			continue
		}
		permission := authPermission{
			Action:   policy.Action,
			Effect:   policy.Effect,
			Object:   policy.Object,
			Resource: policy.Resource,
		}
		if !seen[permission] {
			seen[permission] = true
			out = append(out, permission)
		}
	}
	return out
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
