package api

import (
	"encoding/json"
	"net/http"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
)

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
		"external_providers": d.Auth.ExternalLoginProviders(),
		"groups":             identity.Groups,
		"permissions":        authPermissions(d.Cfg.RBAC, identity),
		"provider":           identity.Provider,
		"provider_id":        identity.ProviderID,
		"roles":              authRoles(d.Cfg.RBAC, identity),
		"user":               identity.Username,
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
		"provider_id":   identity.ProviderID,
		"roles":         authRoles(d.Cfg.RBAC, identity),
		"user":          identity.Username,
	})
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
