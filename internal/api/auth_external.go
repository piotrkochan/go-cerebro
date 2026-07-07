package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lmenezes/cerebro/internal/auth"
)

func (d *Deps) handleEntraIDLogin(w http.ResponseWriter, r *http.Request) {
	providerID := d.externalProviderID(r, "entra_id")
	callbackURL := d.entraIDCallbackURL(r, providerID)
	returnPath := r.URL.Query().Get("redirect")
	if returnPath == "" {
		returnPath = "/"
	}
	authURL, err := d.Auth.BeginEntraIDLogin(providerID, w, r, callbackURL, returnPath)
	if err != nil {
		d.auditAuth(r, "entraid_login_start_failed", auth.Identity{}, "failure", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.auditAuth(r, "entraid_login_started", auth.Identity{Provider: "entra_id", ProviderID: providerID}, "success", "")
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (d *Deps) handleEntraIDCallback(w http.ResponseWriter, r *http.Request) {
	providerID := d.externalProviderID(r, "entra_id")
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		d.auditAuth(r, "entraid_login_failed", auth.Identity{Provider: "entra_id", ProviderID: providerID}, "failure", authErr)
		http.Redirect(w, r, basePathFor(d, "/#/login?error=external"), http.StatusSeeOther)
		return
	}
	redirect, err := d.Auth.CompleteEntraIDLogin(providerID, w, r, d.entraIDCallbackURL(r, providerID))
	if err != nil {
		d.auditAuth(r, "entraid_login_failed", auth.Identity{Provider: "entra_id", ProviderID: providerID}, "failure", err.Error())
		http.Redirect(w, r, basePathFor(d, "/#/login?error=external"), http.StatusSeeOther)
		return
	}
	d.auditAuth(r, "entraid_login_succeeded", auth.Identity{Provider: "entra_id", ProviderID: providerID}, "success", "")
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect is same-origin absolute path from auth.CompleteEntraIDLogin.
}

func (d *Deps) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	providerID := d.externalProviderID(r, "oauth")
	callbackURL := d.oauthCallbackURL(r, providerID)
	returnPath := r.URL.Query().Get("redirect")
	if returnPath == "" {
		returnPath = "/"
	}
	authURL, err := d.Auth.BeginOAuthLogin(providerID, w, r, callbackURL, returnPath)
	if err != nil {
		d.auditAuth(r, "oauth_login_start_failed", auth.Identity{}, "failure", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.auditAuth(r, "oauth_login_started", auth.Identity{Provider: "oauth", ProviderID: providerID}, "success", "")
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (d *Deps) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerID := d.externalProviderID(r, "oauth")
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		d.auditAuth(r, "oauth_login_failed", auth.Identity{Provider: "oauth", ProviderID: providerID}, "failure", authErr)
		http.Redirect(w, r, basePathFor(d, "/#/login?error=external"), http.StatusSeeOther)
		return
	}
	redirect, err := d.Auth.CompleteOAuthLogin(providerID, w, r, d.oauthCallbackURL(r, providerID))
	if err != nil {
		d.auditAuth(r, "oauth_login_failed", auth.Identity{Provider: "oauth", ProviderID: providerID}, "failure", err.Error())
		http.Redirect(w, r, basePathFor(d, "/#/login?error=external"), http.StatusSeeOther)
		return
	}
	d.auditAuth(r, "oauth_login_succeeded", auth.Identity{Provider: "oauth", ProviderID: providerID}, "success", "")
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther) // #nosec G710 -- redirect is same-origin absolute path from auth.CompleteOAuthLogin.
}

func (d *Deps) entraIDCallbackURL(r *http.Request, providerID string) string {
	if configured := d.Auth.EntraIDRedirectURL(providerID); configured != "" {
		return configured
	}
	if providerID == "" {
		return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/entraid/callback")
	}
	return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/entraid/"+providerID+"/callback")
}

func (d *Deps) oauthCallbackURL(r *http.Request, providerID string) string {
	if configured := d.Auth.OAuthRedirectURL(providerID); configured != "" {
		return configured
	}
	if providerID == "" {
		return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/oauth/callback")
	}
	return requestOrigin(d.Cfg.Server, r) + basePathFor(d, "/auth/oauth/"+providerID+"/callback")
}

func (d *Deps) externalProviderID(r *http.Request, kind string) string {
	if providerID := chi.URLParam(r, "provider"); providerID != "" {
		return providerID
	}
	for _, provider := range d.Auth.ExternalLoginProviders() {
		if provider.Kind == kind {
			return provider.ID
		}
	}
	return ""
}
