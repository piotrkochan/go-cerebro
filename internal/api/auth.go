package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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
	mux.HandleFunc("GET /auth/oauth/login", d.handleOAuthLogin)
	mux.HandleFunc("GET /auth/oauth/callback", d.handleOAuthCallback)
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
			"oauth":    d.Auth.OAuthEnabled(),
			"password": d.Auth.PasswordLoginEnabled(),
		},
		"provider_names": map[string]string{
			"oauth": d.Auth.OAuthName(),
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

func (d *Deps) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	callbackURL := d.oauthCallbackURL(r)
	returnPath := r.URL.Query().Get("redirect")
	if returnPath == "" {
		returnPath = "/"
	}
	authURL, err := d.Auth.BeginOAuthLogin(w, r, callbackURL, returnPath)
	if err != nil {
		d.auditAuth(r, "oauth_login_start_failed", auth.Identity{}, "failure", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.auditAuth(r, "oauth_login_started", auth.Identity{Provider: "oauth"}, "success", "")
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (d *Deps) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		d.auditAuth(r, "oauth_login_failed", auth.Identity{Provider: "oauth"}, "failure", authErr)
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	redirect, err := d.Auth.CompleteOAuthLogin(w, r, d.oauthCallbackURL(r))
	if err != nil {
		d.auditAuth(r, "oauth_login_failed", auth.Identity{Provider: "oauth"}, "failure", err.Error())
		http.Redirect(w, r, basePathFor(d, "/#/login?error=invalid"), http.StatusSeeOther)
		return
	}
	d.auditAuth(r, "oauth_login_succeeded", auth.Identity{Provider: "oauth"}, "success", "")
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect is same-origin absolute path from auth.CompleteOAuthLogin.
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
	subjects := authPrincipalSet(identity)
	if rbac.DefaultRole != "" {
		subjects[rbac.DefaultRole] = true
	}
	changed := true
	for changed {
		changed = false
		for _, binding := range rbac.Bindings {
			if !subjects[binding.Subject] || subjects[binding.Role] {
				continue
			}
			subjects[binding.Role] = true
			if binding.Role != "" && !seen[binding.Role] {
				add(binding.Role)
				changed = true
			}
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
	subjects := authPrincipalSet(identity)
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

func authPrincipalSet(identity auth.Identity) map[string]bool {
	subjects := map[string]bool{}
	if identity.Username != "" {
		subjects[identity.Username] = true
		subjects["user:"+identity.Username] = true
	}
	for _, group := range identity.Groups {
		if group == "" {
			continue
		}
		subjects[group] = true
		subjects["group:"+group] = true
	}
	return subjects
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
	return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/entraid/callback")
}

func (d *Deps) oauthCallbackURL(r *http.Request) string {
	if configured := d.Auth.OAuthRedirectURL(); configured != "" {
		return configured
	}
	return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/oauth/callback")
}

func requestOrigin(serverCfg config.Server, r *http.Request) string {
	if serverCfg.PublicURL != "" {
		if origin := publicOrigin(serverCfg.PublicURL); origin != "" {
			return origin
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if trustedRemote(r.RemoteAddr, serverCfg.TrustedProxies) {
		if forwardedProto := forwardedProto(r); forwardedProto != "" {
			scheme = strings.ToLower(forwardedProto)
		}
		if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			host = forwardedHost
		}
	}
	return scheme + "://" + host
}

func forwardedProto(r *http.Request) string {
	value := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if value == "http" || value == "https" {
		return value
	}
	return ""
}

func publicOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func trustedRemote(remoteAddr string, trustedProxies []string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	for _, raw := range trustedProxies {
		network, err := trustedProxyNet(raw)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func trustedProxyNet(raw string) (*net.IPNet, error) {
	value := strings.TrimSpace(raw)
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		return network, err
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil, http.ErrNotSupported
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}
