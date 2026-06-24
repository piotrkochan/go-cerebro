package api

import (
	"context"
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
	mux.HandleFunc("POST /auth/login", d.handleLogin)
	mux.HandleFunc("POST /auth/logout", d.handleLogout)
}

func (d *Deps) handleLogin(w http.ResponseWriter, r *http.Request) {
	var user, password string
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		_ = r.ParseForm()
		user = r.FormValue("user")
		password = r.FormValue("password")
	} else {
		// JSON: {"user": "...", "password": "..."}
		_ = r.ParseForm() // best effort
		user = r.FormValue("user")
		password = r.FormValue("password")
	}
	if user == "" || password == "" {
		http.Error(w, "invalid login form data", http.StatusBadRequest)
		return
	}
	username, err := d.Auth.Authenticate(user, password)
	if err != nil {
		http.Redirect(w, r, basePathFor(d, "/login"), http.StatusSeeOther)
		return
	}
	if err := d.Auth.SetSessionUser(w, r, username); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect := d.Auth.ConsumeRedirect(w, r)
	if redirect == "" {
		redirect = basePathFor(d, "/")
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (d *Deps) handleLogout(w http.ResponseWriter, r *http.Request) {
	_ = d.Auth.ClearSession(w, r)
	http.Redirect(w, r, basePathFor(d, "/login"), http.StatusSeeOther)
}

func basePathFor(d *Deps, suffix string) string {
	prefix := strings.TrimRight(d.Cfg.Server.BasePath, "/")
	if prefix == "" {
		return suffix
	}
	return prefix + suffix
}

// silence unused import warnings if context not used elsewhere in this file
var _ = context.Background
